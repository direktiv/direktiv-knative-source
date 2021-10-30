package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	gosnmp "github.com/gosnmp/gosnmp"
	"github.com/vorteil/direktiv-knative-source/pkg/direktivsource"
)

type snmpHandler struct {
	esr *direktivsource.EventSourceReceiver
}

func (sh *snmpHandler) snmpTrapHandler(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {

	if os.Getenv("DEBUG") == "true" {
		prettyPrint, _ := json.MarshalIndent(packet, "", "\t")
		sh.esr.Logger().Debugf(string(prettyPrint))
	}

	// var event event.Event
	ev := cloudevents.NewEvent()
	ev.SetSource("direktiv/snmp/source")
	ev.SetType("direktiv.snmp")
	ev.SetData(cloudevents.ApplicationJSON, packet)
	ev.SetID(uuid.New().String())
	ev.SetTime(time.Now())

	sh.esr.OverridesApply(&ev)

	ctx := cloudevents.ContextWithTarget(context.Background(), sh.esr.Env().Sink)
	if result := sh.esr.Client().Send(ctx, ev); cloudevents.IsUndelivered(result) {
		sh.esr.Logger().Errorf("failed to send, %v", result)
	}

}

func (sh *snmpHandler) listenUDP(tl *gosnmp.TrapListener, ch chan error) {
	sh.esr.Logger().Infof("starting snmp udp listener")
	err := tl.Listen("udp://0.0.0.0:8000")
	if err != nil {
		ch <- err
	}
}

func (sh *snmpHandler) listenTCP(tl *gosnmp.TrapListener, ch chan error) {
	sh.esr.Logger().Infof("starting snmp tcp listener")
	err := tl.Listen("tcp://0.0.0.0:8000")
	if err != nil {
		ch <- err
	}
}

func main() {

	esr := direktivsource.NewEventSourceReceiver("snmp-source")

	th := &snmpHandler{
		esr,
	}

	tl := gosnmp.NewTrapListener()
	tl.OnNewTrap = th.snmpTrapHandler
	tl.Params = gosnmp.Default

	errCh := make(chan error)
	go th.listenTCP(tl, errCh)
	go th.listenUDP(tl, errCh)

	select {
	case err := <-errCh:
		{
			esr.Logger().Fatalf("%v", err)
		}
	}

}
