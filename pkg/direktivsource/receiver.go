package direktivsource

import (
	"encoding/json"
	"log"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	ceclient "github.com/cloudevents/sdk-go/v2/client"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

type EnvConfig struct {
	// Sink URL where to send heartbeat cloudevents
	Sink string `envconfig:"K_SINK"`

	// CEOverrides are the CloudEvents overrides to be applied to the outbound event.
	CEOverrides string `envconfig:"K_CE_OVERRIDES"`
}

type EventSourceReceiver struct {
	logger      *zap.SugaredLogger
	uuid        string
	ceclient    ceclient.Client
	env         EnvConfig
	ceOverrides *duckv1.CloudEventOverrides
}

func NewEventSourceReceiver(t string) *EventSourceReceiver {

	logger := customLogger(t)

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		logger.Fatalf("can not create cloud event client: %s", err.Error())
	}

	esr := &EventSourceReceiver{
		logger:   logger,
		uuid:     uuid.New().String(),
		ceclient: c,
	}

	logger.Infof("instance uuid %s", esr.uuid)

	if err := envconfig.Process("", &esr.env); err != nil {
		logger.Fatalf("can not process environment: %s", err.Error())
	}

	if len(esr.env.CEOverrides) > 0 {
		overrides := duckv1.CloudEventOverrides{}
		err := json.Unmarshal([]byte(esr.env.CEOverrides), &overrides)
		if err != nil {
			log.Printf("[ERROR] Unparseable CloudEvents overrides %s: %v", esr.env.CEOverrides, err)
			os.Exit(1)
		}
		esr.ceOverrides = &overrides
	}

	logger.Infof("using sink %s", esr.env.Sink)

	return esr

}

func (esr *EventSourceReceiver) OverridesApply(ev *event.Event) {
	if esr.Overrides() != nil && esr.Overrides().Extensions != nil {
		for n, v := range esr.Overrides().Extensions {
			ev.SetExtension(n, v)
		}
	}
}

func (esr *EventSourceReceiver) ID() string {
	return esr.uuid
}

func (esr *EventSourceReceiver) Client() ceclient.Client {
	return esr.ceclient
}

func (esr *EventSourceReceiver) Env() EnvConfig {
	return esr.env
}

func (esr *EventSourceReceiver) Overrides() *duckv1.CloudEventOverrides {
	return esr.ceOverrides
}

func (esr *EventSourceReceiver) Logger() *zap.SugaredLogger {
	return esr.logger
}

func customLogger(t string) *zap.SugaredLogger {

	l := os.Getenv("DEBUG")

	inLvl := zapcore.InfoLevel
	if l == "true" {
		inLvl = zapcore.DebugLevel
	}

	errOut := zapcore.Lock(os.Stderr)

	logLvl := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= inLvl
	})

	// console
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, errOut, logLvl),
	)

	if os.Getenv("LOG") == "json" {
		core = zapcore.NewTee(
			zapcore.NewCore(jsonEncoder, errOut, logLvl),
		)
	}

	return zap.New(core,
		zap.AddCaller()).With(zap.String("component", t)).Sugar()

}
