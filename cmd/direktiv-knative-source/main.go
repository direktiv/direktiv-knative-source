package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	format "github.com/cloudevents/sdk-go/binding/format/protobuf/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	ceclient "github.com/cloudevents/sdk-go/v2/client"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	igrpc "github.com/vorteil/direktiv/pkg/flow/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
)

const (
	lc = `{
          "level": "info",
	        "development": false,
	        "outputPaths": ["stdout"],
	        "errorOutputPaths": ["stderr"],
	        "encoding": "json",
	        "encoderConfig": {
	          "timeKey": "ts",
	          "levelKey": "level",
	          "nameKey": "logger",
	          "callerKey": "caller",
	          "messageKey": "msg",
	          "stacktraceKey": "stacktrace",
	          "lineEnding": "",
	          "levelEncoder": "",
	          "timeEncoder": "iso8601",
	          "durationEncoder": "",
	          "callerEncoder": ""
	        }
      	}`
)

type envConfig struct {
	// Sink URL where to send heartbeat cloudevents
	Sink string `envconfig:"K_SINK"`

	// CEOverrides are the CloudEvents overrides to be applied to the outbound event.
	CEOverrides string `envconfig:"K_CE_OVERRIDES"`
}

type eventSourceReceiver struct {
	logger *zap.SugaredLogger
	uuid   string
	client igrpc.EventingClient
	conn   *grpc.ClientConn

	ceclient ceclient.Client
}

var direktiv string

func (esr *eventSourceReceiver) connect() error {

	esr.logger.Infof("connecting to %s", direktiv)

	// insecure, linkerd adds mtls/tls if required
	conn, err := grpc.Dial(direktiv, []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}...)
	if err != nil {
		esr.logger.Fatalf("can not connect to %s: %v", direktiv, err)
	}

	esr.conn = conn
	esr.client = igrpc.NewEventingClient(conn)

	return nil
}

func (esr *eventSourceReceiver) getStream() (igrpc.Eventing_RequestEventsClient, error) {
	return esr.client.RequestEvents(context.Background(), &igrpc.EventingRequest{Uuid: esr.uuid})
}

func init() {
	flag.StringVar(&direktiv, "direktiv", "", "direktiv address, e.g. direktiv-flow.default:3333")
}

func main() {

	flag.Parse()

	logger, _ := logging.NewLogger(lc, "")
	logger.Infof("starting direktiv event source with direktiv: %s", direktiv)

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		logger.Fatalf("can not create cloud event client: %s", err.Error())
	}

	esr := &eventSourceReceiver{
		logger:   logger,
		uuid:     uuid.New().String(),
		ceclient: c,
	}

	logger.Infof("instance uuid %s", esr.uuid)

	esr.connect()

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		logger.Fatalf("can not process environment: %s", err.Error())
	}

	logger.Infof("using sink %s", env.Sink)

	var ceOverrides *duckv1.CloudEventOverrides
	if len(env.CEOverrides) > 0 {
		overrides := duckv1.CloudEventOverrides{}
		err := json.Unmarshal([]byte(env.CEOverrides), &overrides)
		if err != nil {
			log.Printf("[ERROR] Unparseable CloudEvents overrides %s: %v", env.CEOverrides, err)
			os.Exit(1)
		}
		ceOverrides = &overrides
	}

	var stream igrpc.Eventing_RequestEventsClient

	for {
		if stream == nil {
			if stream, err = esr.getStream(); err != nil {
				logger.Errorf("failed to connect to direktiv: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
		}

		response, err := stream.Recv()
		if err != nil {
			logger.Errorf("failed to receive message: %v", err)
			stream = nil
			time.Sleep(3 * time.Second)
			continue
		}

		var ev event.Event
		err = format.Protobuf.Unmarshal(response.Ce, &ev)
		if err != nil {
			logger.Errorf("failed to unmarshal message: %v", err)
			continue
		}

		if ceOverrides != nil && ceOverrides.Extensions != nil {
			for n, v := range ceOverrides.Extensions {
				ev.SetExtension(n, v)
			}
		}

		ctx := cloudevents.ContextWithTarget(context.Background(), env.Sink)
		if result := c.Send(ctx, ev); cloudevents.IsUndelivered(result) {
			logger.Errorf("failed to send, %v", result)
		}
	}

}
