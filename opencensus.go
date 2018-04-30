package opencensus

import (
	"context"
	"net/http"
	"time"

	"github.com/devopsfaith/krakend/proxy"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func Register(cfg Config) error {
	registerViewExporter(cfg.ViewExporters...)
	registerTraceExporter(cfg.TraceExporters...)
	setDefaultSampler(cfg.SampleRate)
	setReportingPeriod(cfg.ReportingPeriod)
	return registerViews(cfg.Views...)
}

type Config struct {
	ViewExporters   []view.Exporter
	TraceExporters  []trace.Exporter
	SampleRate      int
	ReportingPeriod time.Duration
	Views           []*view.View
}

const ContextKey = "opencensus-request-span"

var (
	DefaultViews  = append(ochttp.DefaultServerViews, ochttp.DefaultClientViews...)
	defaultClient = &http.Client{Transport: &ochttp.Transport{}}
)

func NewHTTPClient(_ context.Context) *http.Client {
	return defaultClient
}

func HTTPRequestExecutor(clientFactory proxy.HTTPClientFactory) proxy.HTTPRequestExecutor {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		client := clientFactory(ctx)
		if _, ok := client.Transport.(*ochttp.Transport); !ok {
			client.Transport = &ochttp.Transport{Base: client.Transport}
		}
		return client.Do(req.WithContext(trace.WithSpan(ctx, fromContext(ctx))))
	}
}

func fromContext(ctx context.Context) *trace.Span {
	span := trace.FromContext(ctx)
	if span == nil {
		span, _ = ctx.Value(ContextKey).(*trace.Span)
	}
	return span
}

func registerViewExporter(exporters ...view.Exporter) {
	for _, e := range exporters {
		view.RegisterExporter(e)
	}
}

func registerTraceExporter(exporters ...trace.Exporter) {
	for _, e := range exporters {
		trace.RegisterExporter(e)
	}
}

func setDefaultSampler(rate int) {
	var sampler trace.Sampler
	switch {
	case rate <= 0:
		sampler = trace.NeverSample()
	case rate >= 100:
		sampler = trace.AlwaysSample()
	default:
		sampler = trace.ProbabilitySampler(float64(rate) / 100.0)
	}
	trace.ApplyConfig(trace.Config{DefaultSampler: sampler})
}

func setReportingPeriod(d time.Duration) {
	view.SetReportingPeriod(d)
}

func registerViews(views ...*view.View) error {
	return view.Register(views...)
}
