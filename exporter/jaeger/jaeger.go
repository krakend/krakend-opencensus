package jaeger

import (
	"context"
	"errors"

	opencensus "github.com/devopsfaith/krakend-opencensus"
	"go.opencensus.io/exporter/jaeger"
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
		Endpoint:    cfg.Exporters.Jaeger.Endpoint,
		ServiceName: cfg.Exporters.Jaeger.ServiceName,
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
