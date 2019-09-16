package prometheus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	opencensus "github.com/devopsfaith/krakend-opencensus"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*prometheus.Exporter, error) {
	if cfg.Exporters.Prometheus == nil {
		return nil, errDisabled
	}

	ns := "krakend"
	if cfg.Exporters.Prometheus.Namespace != "" {
		ns = cfg.Exporters.Prometheus.Namespace
	}

	exporter, err := prometheus.NewExporter(prometheus.Options{Namespace: ns})
	if err != nil {
		return exporter, err
	}

	router := http.NewServeMux()
	router.Handle("/metrics", exporter)
	server := http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%d", cfg.Exporters.Prometheus.Port),
	}

	go func() { log.Fatal(server.ListenAndServe()) }()

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		server.Shutdown(ctx)
		cancel()
	}()

	return exporter, nil
}

var errDisabled = errors.New("opencensus prometheus exporter disabled")
