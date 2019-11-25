package xray

import (
	"context"
	"errors"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	opencensus "github.com/devopsfaith/krakend-opencensus"
	// Auto-import to enable grpc compression
	_ "google.golang.org/grpc/encoding/gzip"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*ocagent.Exporter, error) {

	options := []ocagent.ExporterOption{}
	if cfg.Exporters.Ocagent == nil {
		return nil, errors.New("ocagent exporter disabled")
	}

	if cfg.Exporters.Ocagent.Address == "" {
		return nil, errors.New("missing ocagent address")
	}
	options = append(options, ocagent.WithAddress(cfg.Exporters.Ocagent.Address))

	if cfg.Exporters.Ocagent.ServiceName == "" {
		cfg.Exporters.Ocagent.ServiceName = "KrakenD-Opencensus"
	}
	options = append(options, ocagent.WithServiceName(cfg.Exporters.Ocagent.ServiceName))

	if cfg.Exporters.Ocagent.Headers != nil {
		options = append(options, ocagent.WithHeaders(cfg.Exporters.Ocagent.Headers))
	}

	if cfg.Exporters.Ocagent.Insecure {
		options = append(options, ocagent.WithInsecure())
	}

	if cfg.Exporters.Ocagent.EnaableCompression {
		options = append(options, ocagent.UseCompressor("gzip"))
	}

	if cfg.Exporters.Ocagent.Reconnection != "" {
		period, err := time.ParseDuration(cfg.Exporters.Ocagent.Reconnection)
		if err != nil {
			return nil, errors.New("cannot parse reconnection period")
		}
		options = append(options, ocagent.WithReconnectionPeriod(period))
	}
	return ocagent.NewExporter(options...)
}
