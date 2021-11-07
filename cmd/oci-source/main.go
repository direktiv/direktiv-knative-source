package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

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

		res.WriteHeader(http.StatusOK)
		return
	}

	for name, values := range req.Header {
		for _, value := range values {
			oci.esr.Logger().Infof(">>1 %v %v", name, value)
		}
	}

	body, err := ioutil.ReadAll(req.Body)
	oci.esr.Logger().Infof(">>2 %v %v", string(body), err)

	// check if it is a cloudevent
	// respond to oci

	fmt.Fprint(res, "Hello, World!")
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

func (oci *ociHandler) startOCIReceiver() {

	router := mux.NewRouter()
	router.HandleFunc("/", oci.indexHandler).Methods("POST")
	router.Use(basicAuthHandler)

	http.ListenAndServe(":8000", router)

}

func main() {

	esr := direktivsource.NewEventSourceReceiver("oci-source")
	esr.Logger().Infof("ESR %v", esr)
	// start mux

	oci := &ociHandler{
		esr,
	}

	go oci.startOCIReceiver()

	time.Sleep(60 * time.Second)

}
