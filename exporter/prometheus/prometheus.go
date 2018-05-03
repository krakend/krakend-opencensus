package prometheus

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

func Exporter(ctx context.Context, port int) (view.Exporter, error) {
	exporter, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		return exporter, err
	}

	router := http.NewServeMux()
	router.Handle("/metrics", exporter)
	server := http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%d", port),
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
