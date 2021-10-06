package newrelic

import (
	"context"
	"errors"
	"log"

	opencensus "github.com/devopsfaith/krakend-opencensus"
	newrelic "github.com/newrelic/newrelic-opencensus-exporter-go/nrcensus"
)

func init() {
	log.Println("newrelic.init")
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*newrelic.Exporter, error) {
	log.Println("newrelic.Exporter")
	if cfg.Exporters.NewRelic == nil {
		return nil, errDisabled
	}
	return newrelic.NewExporter(cfg.Exporters.NewRelic.ServiceName, cfg.Exporters.NewRelic.NewRelicInsightsApiKey)
}

var errDisabled = errors.New("opencensus newrelic exporter disabled")
