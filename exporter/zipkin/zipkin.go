package zipkin

import (
	"context"
	"errors"
	"net"

	"contrib.go.opencensus.io/exporter/zipkin"
	opencensus "github.com/krakend/krakend-opencensus/v2"
	"github.com/openzipkin/zipkin-go/model"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(_ context.Context, cfg opencensus.Config) (*zipkin.Exporter, error) {
	if cfg.Exporters.Zipkin == nil {
		return nil, errDisabled
	}
	return zipkin.NewExporter(
		httpreporter.NewReporter(cfg.Exporters.Zipkin.CollectorURL),
		&model.Endpoint{
			ServiceName: cfg.Exporters.Zipkin.ServiceName,
			IPv4:        net.ParseIP(cfg.Exporters.Zipkin.IP),
			Port:        uint16(cfg.Exporters.Zipkin.Port),
		},
	), nil
}

var errDisabled = errors.New("opencensus zipkin exporter disabled")
