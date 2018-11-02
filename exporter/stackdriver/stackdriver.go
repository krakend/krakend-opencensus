package stackdriver

import (
	"context"
	"errors"
	"time"

	ocStackdriver "contrib.go.opencensus.io/exporter/stackdriver"
	opencensus "github.com/devopsfaith/krakend-opencensus"
	"google.golang.org/api/option"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*ocStackdriver.Exporter, error) {
	if cfg.Exporters.Stackdriver == nil {
		return nil, errors.New("stackdriver exporter disabled")
	}
	if cfg.Exporters.Stackdriver.MetricPrefix == "" {
		cfg.Exporters.Stackdriver.MetricPrefix = "KrakenD"
	}
	labels := &ocStackdriver.Labels{}

	for k, v := range cfg.Exporters.Stackdriver.DefaultLabels {
		labels.Set(k, v, "")
	}

	var options []option.ClientOption
	if cfg.Exporters.Stackdriver.WithCredentials != "" {
		options = append(options, option.WithCredentialsFile(cfg.Exporters.Stackdriver.WithCredentials))
	}
	return ocStackdriver.NewExporter(ocStackdriver.Options{
		ProjectID:               cfg.Exporters.Stackdriver.ProjectID,
		MetricPrefix:            cfg.Exporters.Stackdriver.MetricPrefix,
		BundleDelayThreshold:    time.Duration(cfg.ReportingPeriod),
		BundleCountThreshold:    cfg.Exporters.Stackdriver.CountThreshold,
		DefaultMonitoringLabels: labels,
		MonitoringClientOptions: options,
	})
}
