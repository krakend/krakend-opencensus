package opencensus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/devopsfaith/krakend/config"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func Register(cfg config.ServiceConfig, ve []view.Exporter, te []trace.Exporter, vs []*view.View) error {
	return register.Register(cfg, ve, te, vs)
}

type composableRegister struct {
	viewExporter       func(exporters ...view.Exporter)
	traceExporter      func(exporters ...trace.Exporter)
	registerViews      func(views ...*view.View) error
	setDefaultSampler  func(rate int)
	setReportingPeriod func(d time.Duration)
}

func (c composableRegister) Register(srvCfg config.ServiceConfig, ve []view.Exporter, te []trace.Exporter, vs []*view.View) error {
	cfg, err := parseCfg(srvCfg)
	if err != nil {
		return err
	}

	if len(vs) == 0 {
		vs = DefaultViews
	}

	c.viewExporter(ve...)
	c.traceExporter(te...)

	c.setDefaultSampler(cfg.SampleRate)
	c.setReportingPeriod(time.Duration(cfg.ReportingPeriod) * time.Second)

	return c.registerViews(vs...)
}

type Config struct {
	SampleRate      int `json:"sample_rate"`
	ReportingPeriod int `json:"reporting_period"`
}

const (
	ContextKey = "opencensus-request-span"
	Namespace  = "github_com/devopsfaith/krakend-opencensus"
)

var (
	ErrNoExtraConfig = errors.New("no extra config defined for the opencensus module")
	DefaultViews     = append(ochttp.DefaultServerViews, ochttp.DefaultClientViews...)
	register         = composableRegister{
		viewExporter:       registerViewExporter,
		traceExporter:      registerTraceExporter,
		setDefaultSampler:  setDefaultSampler,
		setReportingPeriod: setReportingPeriod,
		registerViews:      registerViews,
	}
)

func parseCfg(srvCfg config.ServiceConfig) (*Config, error) {
	cfg := new(Config)
	tmp, ok := srvCfg.ExtraConfig[Namespace]
	if !ok {
		return nil, ErrNoExtraConfig
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(tmp)
	if err := json.NewDecoder(buf).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
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
