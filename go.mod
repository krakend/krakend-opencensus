module github.com/devopsfaith/krakend-opencensus

go 1.13

require (
	contrib.go.opencensus.io/exporter/aws v0.0.0-20190807220307-c50fb1bd7f21
	contrib.go.opencensus.io/exporter/jaeger v0.1.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	contrib.go.opencensus.io/exporter/stackdriver v0.12.7
	contrib.go.opencensus.io/exporter/zipkin v0.1.1
	github.com/aws/aws-sdk-go v1.25.10
	github.com/devopsfaith/krakend v0.0.0-20190930092458-9e6fc3784eca
	github.com/gin-gonic/gin v1.4.0
	github.com/influxdata/influxdb v1.7.8 // indirect
	github.com/kpacha/opencensus-influxdb v0.0.0-20181102202715-663e2683a27c
	github.com/openzipkin/zipkin-go v0.2.2
	go.opencensus.io v0.22.1
)
