package main

import (
	"context"
	"flag"
	"time"

	format "github.com/cloudevents/sdk-go/binding/format/protobuf/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/vorteil/direktiv-knative-source/pkg/direktivsource"
	igrpc "github.com/vorteil/direktiv/pkg/flow/grpc"
	"google.golang.org/grpc"
)

type direktivConnector struct {
	esr    *direktivsource.EventSourceReceiver
	client igrpc.EventingClient
	conn   *grpc.ClientConn
}

var direktiv string

func (dc *direktivConnector) connect() error {

	dc.esr.Logger().Infof("connecting to %s", direktiv)

	// insecure, linkerd adds mtls/tls if required
	conn, err := grpc.Dial(direktiv, []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}...)
	if err != nil {
		dc.esr.Logger().Fatalf("can not connect to %s: %v", direktiv, err)
	}

	dc.conn = conn
	dc.client = igrpc.NewEventingClient(conn)

	return nil
}

func (dc *direktivConnector) getStream() (igrpc.Eventing_RequestEventsClient, error) {
	return dc.client.RequestEvents(context.Background(), &igrpc.EventingRequest{Uuid: dc.esr.ID()})
}

func init() {
	flag.StringVar(&direktiv, "direktiv", "", "direktiv address, e.g. direktiv-flow.default:3333")
}

func main() {

	esr := direktivsource.NewEventSourceReceiver("direktiv-source")

	dc := &direktivConnector{
		esr: esr,
	}

	flag.Parse()

	dc.connect()

	var (
		stream igrpc.Eventing_RequestEventsClient
		err    error
	)

	for {
		if stream == nil {
			if stream, err = dc.getStream(); err != nil {
				esr.Logger().Errorf("failed to connect to direktiv: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
		}

		esr.Logger().Infof("waiting for event")
		response, err := stream.Recv()
		if err != nil {
			esr.Logger().Errorf("failed to receive message: %v", err)
			stream = nil
			time.Sleep(3 * time.Second)
			continue
		}

		var ev event.Event
		err = format.Protobuf.Unmarshal(response.Ce, &ev)
		if err != nil {
			esr.Logger().Errorf("failed to unmarshal message: %v", err)
			continue
		}

		dc.esr.OverridesApply(&ev)

		if dc.esr.Overrides() != nil && dc.esr.Overrides().Extensions != nil {
			for n, v := range dc.esr.Overrides().Extensions {
				ev.SetExtension(n, v)
			}
		}

		ctx := cloudevents.ContextWithTarget(context.Background(), esr.Env().Sink)
		if result := esr.Client().Send(ctx, ev); cloudevents.IsUndelivered(result) {
			esr.Logger().Errorf("failed to send, %v", result)
		}
	}

}
