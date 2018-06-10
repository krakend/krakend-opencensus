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
		return nil, nil
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" || os.Getenv("AWS_DEFAULT_REGION") == "" {
		return nil, errors.New("You need to setup ENV vars for AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and AWS_DEFAULT_REGION to use the Opencensus Xray exporter.")
	}
	os.Getenv("AWS_DEFAULT_REGION")
	if cfg.Exporters.Xray.Version == "" {
		cfg.Exporters.Xray.Version = "KrakenD-opencensus"
	}
	exporter, err := aws.NewExporter(
		aws.WithRegion(cfg.Exporters.Xray.Region),
		aws.WithInterval(time.Duration(cfg.ReportingPeriod)),
		aws.WithBufferSize(cfg.SampleRate),
		aws.WithVersion(cfg.Exporters.Xray.Version),
	)
	if nil != err {
		return nil, err
	}
	return exporter, nil
}
