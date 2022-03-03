package datadog

import (
	"context"
	"errors"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	opencensus "github.com/devopsfaith/krakend-opencensus/v2"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*datadog.Exporter, error) {
	if cfg.Exporters.DataDog == nil {
		return nil, errDisabled
	}
	e, err := datadog.NewExporter(datadog.Options{
		Namespace:              cfg.Exporters.DataDog.Namespace,
		Service:                cfg.Exporters.DataDog.Service,
		TraceAddr:              cfg.Exporters.DataDog.TraceAddr,
		StatsAddr:              cfg.Exporters.DataDog.StatsAddr,
		Tags:                   cfg.Exporters.DataDog.Tags,
		GlobalTags:             cfg.Exporters.DataDog.GlobalTags,
		DisableCountPerBuckets: cfg.Exporters.DataDog.DisableCountPerBuckets,
	})
	if err != nil {
		return e, err
	}

	go func() {
		<-ctx.Done()
		e.Stop()
	}()

	return e, nil
}

var errDisabled = errors.New("opencensus datadog exporter disabled")
