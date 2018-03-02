package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	bt "github.com/opentracing/basictracer-go"
	btevents "github.com/opentracing/basictracer-go/events"
	ot "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
)

func newBasicTracer() {
	opts := bt.DefaultOptions()
	opts.Recorder = bt.NewInMemoryRecorder()
	opts.NewSpanEventListener = btevents.NetTraceIntegrator
	tracer := bt.NewWithOptions(opts)
	ot.SetGlobalTracer(tracer)
}

func newJaegerTracer(agentAddr string) io.Closer {
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agentAddr,
		},
	}

	closer, err := cfg.InitGlobalTracer("aragorn", jaegercfg.Logger(jaegerLoggerAdapter{}))
	if err != nil {
		log.Fatal("Could not initialize jaeger tracer", zap.Error(err))
	}
	return closer
}

type jaegerLoggerAdapter struct{}

func (jaegerLoggerAdapter) Error(msg string) {
	log.Error("jaeger tracer", zap.String("error", msg))
}

func (jaegerLoggerAdapter) Infof(msg string, args ...interface{}) {
	msg = strings.TrimSuffix(msg, "\n")
	details := fmt.Sprintf(msg, args...)
	log.Debug("jaeger tracer", zap.String("details", details))
}
