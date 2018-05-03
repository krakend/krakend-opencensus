package zipkin

import (
	"net"

	"github.com/openzipkin/zipkin-go/model"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/trace"
)

func Exporter(collectorURL, serviceName, IP string, port int) trace.Exporter {
	return zipkin.NewExporter(
		httpreporter.NewReporter(collectorURL),
		&model.Endpoint{
			ServiceName: serviceName,
			IPv4:        net.ParseIP(IP),
			Port:        uint16(port),
		})
}
