package stackdriver

import (
	"context"
	"errors"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	opencensus "github.com/devopsfaith/krakend-opencensus"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*stackdriver.Exporter, error) {
	if cfg.Exporters.Stackdriver == nil {
		return nil, errors.New("stackdriver exporter disabled")
	}
	if cfg.Exporters.Stackdriver.MetricPrefix == "" {
		cfg.Exporters.Stackdriver.MetricPrefix = "KrakenD"
	}
	labels := &stackdriver.Labels{}

	for k, v := range cfg.Exporters.Stackdriver.DefaultLabels {
		labels.Set(k, v, "")
	}

	return stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               cfg.Exporters.Stackdriver.ProjectID,
		MetricPrefix:            cfg.Exporters.Stackdriver.MetricPrefix,
		BundleDelayThreshold:    time.Duration(cfg.ReportingPeriod) * time.Second,
		BundleCountThreshold:    cfg.Exporters.Stackdriver.CountThreshold,
		DefaultMonitoringLabels: labels,
	})
}
