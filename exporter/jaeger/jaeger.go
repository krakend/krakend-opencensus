package jaeger

import (
	"context"
	"errors"

	"contrib.go.opencensus.io/exporter/jaeger"
	opencensus "github.com/krakend/krakend-opencensus/v2"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*jaeger.Exporter, error) {
	if cfg.Exporters.Jaeger == nil {
		return nil, errDisabled
	}
	e, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     cfg.Exporters.Jaeger.AgentEndpoint,
		CollectorEndpoint: cfg.Exporters.Jaeger.Endpoint,
		BufferMaxCount:    cfg.Exporters.Jaeger.BufferMaxCount,
		Process: jaeger.Process{
			ServiceName: cfg.Exporters.Jaeger.ServiceName,
		},
	})
	if err != nil {
		return e, err
	}

	go func() {
		<-ctx.Done()
		e.Flush()
	}()

	return e, nil
}

var errDisabled = errors.New("opencensus jaeger exporter disabled")
