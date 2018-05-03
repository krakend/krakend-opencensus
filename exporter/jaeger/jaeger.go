package jaeger

import (
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

func Exporter(endpoint, serviceName string) (trace.Exporter, error) {
	return jaeger.NewExporter(jaeger.Options{
		Endpoint:    endpoint,
		ServiceName: serviceName,
	})
}
