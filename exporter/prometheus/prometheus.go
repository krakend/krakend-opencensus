package prometheus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	opencensus "github.com/devopsfaith/krakend-opencensus"
	"go.opencensus.io/exporter/prometheus"
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
	exporter, err := prometheus.NewExporter(prometheus.Options{})
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
