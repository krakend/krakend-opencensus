package xray

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/census-ecosystem/opencensus-go-exporter-aws"
	opencensus "github.com/devopsfaith/krakend-opencensus"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*aws.Exporter, error) {
	if cfg.Exporters.Xray == nil {
		return nil, errors.New("xray exporter disabled")
	}
	if cfg.Exporters.Xray.AccessKey == "" || cfg.Exporters.Xray.SecretKey == "" || cfg.Exporters.Xray.Region == "" {
		return nil, errors.New("aws access_key_id, secret_access_key and region needs to be defined")
	}
	if err := os.Setenv("AWS_ACCESS_KEY_ID", cfg.Exporters.Xray.AccessKey); err != nil {
		return nil, errors.New("problem setting environment")
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.Exporters.Xray.SecretKey); err != nil {
		return nil, errors.New("problem setting environment")
	}
	if err := os.Setenv("AWS_DEFAULT_REGION", cfg.Exporters.Xray.Region); err != nil {
		return nil, errors.New("problem setting environment")
	}
	if cfg.Exporters.Xray.Version == "" {
		cfg.Exporters.Xray.Version = "KrakenD-opencensus"
	}

	exporter, err := aws.NewExporter(
		aws.WithRegion(cfg.Exporters.Xray.Region),
		aws.WithInterval(time.Duration(cfg.ReportingPeriod)),
		aws.WithBufferSize(cfg.SampleRate),
		aws.WithVersion(cfg.Exporters.Xray.Version),
	)
	if err != nil {
		return nil, err
	}
	return exporter, nil
}
