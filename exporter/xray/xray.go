package xray

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/xray"
	ocAws "github.com/census-ecosystem/opencensus-go-exporter-aws"
	opencensus "github.com/devopsfaith/krakend-opencensus"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*ocAws.Exporter, error) {
	if cfg.Exporters.Xray == nil {
		return nil, errors.New("xray exporter disabled")
	}
	if cfg.Exporters.Xray.Version == "" {
		cfg.Exporters.Xray.Version = "KrakenD-opencensus"
	}

	if !cfg.Exporters.Xray.UseEnv {
		mySession := setupAWSSession(cfg.Exporters.Xray.AccessKey, cfg.Exporters.Xray.SecretKey, cfg.Exporters.Xray.Region)
		return ocAws.NewExporter(
			ocAws.WithAPI(xray.New(mySession, aws.NewConfig().WithRegion(cfg.Exporters.Xray.Region))),
			ocAws.WithRegion(cfg.Exporters.Xray.Region),
			ocAws.WithInterval(time.Duration(cfg.ReportingPeriod)),
			ocAws.WithBufferSize(cfg.SampleRate),
			ocAws.WithVersion(cfg.Exporters.Xray.Version),
		)
	}

	return ocAws.NewExporter(
		ocAws.WithRegion(cfg.Exporters.Xray.Region),
		ocAws.WithInterval(time.Duration(cfg.ReportingPeriod)),
		ocAws.WithBufferSize(cfg.SampleRate),
		ocAws.WithVersion(cfg.Exporters.Xray.Version),
	)
}

func setupAWSSession(id, secret, region string) *session.Session {
	return session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(id, secret, ""),
		Region:      aws.String(region),
	}))
}
