package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/gorilla/mux"
	"github.com/vorteil/direktiv-knative-source/pkg/direktivsource"
)

const (
	topicNameHeader = "X-Oci-Ns-Topicname"
	topciURLHeader  = "X-Oci-Ns-Confirmationurl"
)

type ociHandler struct {
	esr *direktivsource.EventSourceReceiver
}

// oracle uses cloudevent spec 0.1, convert to 1.0
func (oci *ociHandler) convertCloudEvent(old []byte) (*event.Event, error) {

	ev := &event.Event{}

	var sourceEvent map[string]interface{}
	err := json.Unmarshal(old, &sourceEvent)
	if err != nil {
		return ev, err
	}

	// it is not an oci event
	if sourceEvent["cloudEventsVersion"] == nil {
		return ev, fmt.Errorf("not an oci event")
	}

	// rename old to new attributes
	renameList := map[string]string{
		"cloudEventsVersion": "specversion",
		"eventID":            "id",
		"eventTime":          "time",
		"eventTypeVersion":   "",
		"contentType":        "datacontenttype",
		"eventType":          "type",
	}

	for k, v := range renameList {
		data := sourceEvent[k]
		delete(sourceEvent, k)
		if v != "" {
			sourceEvent[v] = data
		}
	}

	// update spec version
	sourceEvent["specversion"] = "1.0"

	// move out extensions
	extensions := sourceEvent["extensions"]

	if extensions != nil {
		delete(sourceEvent, "extensions")
		for k, v := range extensions.(map[string]interface{}) {
			sourceEvent[k] = v
		}
	}

	b, err := json.MarshalIndent(sourceEvent, "", "\t")
	err = json.Unmarshal(b, ev)

	if os.Getenv("DEBUG") != "" {
		oci.esr.Logger().Debugf("%v", string(b))
	}

	return ev, err

}

func (oci *ociHandler) indexHandler(res http.ResponseWriter, req *http.Request) {

	// check if this is the confirmation url request
	if req.Header.Get(topicNameHeader) != "" && req.Header.Get(topciURLHeader) != "" {

		url := req.Header.Get(topciURLHeader)
		oci.esr.Logger().Infof("got confirmation url: %v", url)

		_, err := http.Get(url)
		if err != nil {
			oci.esr.Logger().Errorf("can not send confirmation: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		oci.esr.Logger().Infof("subscription confirmed")
		res.WriteHeader(http.StatusOK)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		oci.esr.Logger().Errorf("can not read body: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// check if it is a cloudevent
	ev, err := oci.convertCloudEvent(body)
	if err != nil {
		oci.esr.Logger().Errorf("can not convert cloudevent: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	oci.esr.OverridesApply(ev)
	ctx := cloudevents.ContextWithTarget(context.Background(), oci.esr.Env().Sink)
	if result := oci.esr.Client().Send(ctx, *ev); cloudevents.IsUndelivered(result) {
		oci.esr.Logger().Errorf("failed to send, %v", result)
	}

	res.WriteHeader(http.StatusOK)
}

func basicAuthHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username, password, ok := r.BasicAuth()

		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(os.Getenv("BASICAUTH_USERNAME")))
			expectedPasswordHash := sha256.Sum256([]byte(os.Getenv("BASICAUTH_PASSWORD")))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				h.ServeHTTP(w, r)
				return
			}
		}

		// not authenticated
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)

	})
}

func (oci *ociHandler) startOCIReceiver() error {

	router := mux.NewRouter()
	router.HandleFunc("/", oci.indexHandler).Methods("POST")
	router.Use(basicAuthHandler)

	return http.ListenAndServe(":8000", router)

}

func main() {

	esr := direktivsource.NewEventSourceReceiver("oci-source")

	oci := &ociHandler{
		esr,
	}

	// start mux
	err := oci.startOCIReceiver()
	if err != nil {
		log.Fatalf("can not run oci source: %v", err)
	}

}
