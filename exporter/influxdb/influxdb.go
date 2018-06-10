package influxdb

import (
	"context"
	"errors"
	"time"

	opencensus "github.com/devopsfaith/krakend-opencensus"
	"github.com/kpacha/opencensus-influxdb"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*influxdb.Exporter, error) {
	if cfg.Exporters.InfluxDB == nil {
		return nil, errDisabled
	}
	timeout, err := time.ParseDuration(cfg.Exporters.InfluxDB.Timeout)
	if err != nil {
		timeout = 0
	}
	return influxdb.NewExporter(ctx, influxdb.Options{
		Address:         cfg.Exporters.InfluxDB.Address,
		Username:        cfg.Exporters.InfluxDB.Username,
		Password:        cfg.Exporters.InfluxDB.Password,
		Database:        cfg.Exporters.InfluxDB.Database,
		Timeout:         timeout,
		InstanceName:    cfg.Exporters.InfluxDB.InstanceName,
		ReportingPeriod: time.Duration(cfg.ReportingPeriod) * time.Second,
	})
}

var errDisabled = errors.New("opencensus influxdb exporter disabled")
