package opencensus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/luraproject/lura/v2/config"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

type ExporterFactory func(context.Context, Config) (interface{}, error)

func RegisterExporterFactories(ef ExporterFactory) {
	mu.Lock()
	exporterFactories = append(exporterFactories, ef)
	mu.Unlock()
}

func Register(ctx context.Context, srvCfg config.ServiceConfig, vs ...*view.View) error {
	cfg, err := parseCfg(srvCfg)
	if err != nil {
		return err
	}

	err = errSingletonExporterFactoriesRegister
	registerOnce.Do(func() {
		register.ExporterFactories(ctx, *cfg, exporterFactories)

		err = register.Register(ctx, *cfg, vs)
		if err != nil {
			return
		}

		if cfg.EnabledLayers != nil {
			enabledLayers = *cfg.EnabledLayers
			return
		}

		enabledLayers = EnabledLayers{true, true, true}
	})

	return err
}

type composableRegister struct {
	viewExporter       func(exporters ...view.Exporter)
	traceExporter      func(exporters ...trace.Exporter)
	registerViews      func(views ...*view.View) error
	setDefaultSampler  func(rate int)
	setReportingPeriod func(d time.Duration)
}

func (c *composableRegister) ExporterFactories(ctx context.Context, cfg Config, fs []ExporterFactory) {
	viewExporters := []view.Exporter{}
	traceExporters := []trace.Exporter{}

	for _, f := range fs {
		e, err := f(ctx, cfg)
		if err != nil {
			continue
		}
		if ve, ok := e.(view.Exporter); ok {
			viewExporters = append(viewExporters, ve)
		}
		if te, ok := e.(trace.Exporter); ok {
			traceExporters = append(traceExporters, te)
		}
	}

	c.viewExporter(viewExporters...)
	c.traceExporter(traceExporters...)
}

func (c composableRegister) Register(_ context.Context, cfg Config, vs []*view.View) error {
	if len(vs) == 0 {
		vs = DefaultViews
	}

	c.setDefaultSampler(cfg.SampleRate)
	c.setReportingPeriod(time.Duration(cfg.ReportingPeriod) * time.Second)

	// modify metric tags
	// ref: https://godoc.org/go.opencensus.io/plugin/ochttp#pkg-variables
	if cfg.Exporters.Prometheus != nil {
		for _, view := range vs {
			// client metrics (method + statuscode tags are enabled by default)
			if strings.Contains(view.Name, "http/client") {
				// Host
				if cfg.Exporters.Prometheus.HostTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.KeyClientHost)
				}

				// Path
				if cfg.Exporters.Prometheus.PathTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.KeyClientPath)
				}

				// Method
				if cfg.Exporters.Prometheus.MethodTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.KeyClientMethod)
				}

				// StatusCode
				if cfg.Exporters.Prometheus.StatusCodeTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.KeyClientStatus)
				}
			}

			// server metrics
			if strings.Contains(view.Name, "http/server") {
				// Host
				if cfg.Exporters.Prometheus.HostTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.Host)
				}

				// Path
				if cfg.Exporters.Prometheus.PathTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.Path)
				}

				// Method
				if cfg.Exporters.Prometheus.MethodTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.Method)
				}

				// StatusCode
				if cfg.Exporters.Prometheus.StatusCodeTag {
					view.TagKeys = appendIfMissing(view.TagKeys, ochttp.StatusCode)
				}
			}
		}
	}

	return c.registerViews(vs...)
}

type Config struct {
	SampleRate      int            `json:"sample_rate"`
	ReportingPeriod int            `json:"reporting_period"`
	EnabledLayers   *EnabledLayers `json:"enabled_layers"`
	Exporters       Exporters      `json:"exporters"`
}

type EndpointExtraConfig struct {
	PathAggregation string `json:"path_aggregation"`
}

type Exporters struct {
	InfluxDB    *InfluxDBConfig    `json:"influxdb"`
	Zipkin      *ZipkinConfig      `json:"zipkin"`
	Jaeger      *JaegerConfig      `json:"jaeger"`
	Prometheus  *PrometheusConfig  `json:"prometheus"`
	Logger      *struct{}          `json:"logger"`
	Xray        *XrayConfig        `json:"xray"`
	Stackdriver *StackdriverConfig `json:"stackdriver"`
	Ocagent     *OcagentConfig     `json:"ocagent"`
	DataDog     *DataDogConfig     `json:"datadog"`
	ExtraConfig config.ExtraConfig `json:"extra_config"`
}

type InfluxDBConfig struct {
	Address      string `json:"address"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Timeout      string `json:"timeout"`
	PingEnabled  bool   `json:"ping"`
	Database     string `json:"db"`
	InstanceName string `json:"service_name"`
	BufferSize   int    `json:"buffer_size"`
}

type ZipkinConfig struct {
	CollectorURL string `json:"collector_url"`
	ServiceName  string `json:"service_name"`
	IP           string `json:"ip"`
	Port         int    `json:"port"`
}

type JaegerConfig struct {
	AgentEndpoint  string                 `json:"agent_endpoint"`
	Endpoint       string                 `json:"endpoint"`
	ServiceName    string                 `json:"service_name"`
	BufferMaxCount int                    `json:"buffer_max_count"`
	ProcessTags    map[string]interface{} `json:"process_tags"`
}

type PrometheusConfig struct {
	Namespace     string `json:"namespace"`
	Port          int    `json:"port"`
	HostTag       bool   `json:"tag_host"`
	PathTag       bool   `json:"tag_path"`
	MethodTag     bool   `json:"tag_method"`
	StatusCodeTag bool   `json:"tag_statuscode"`
}

type XrayConfig struct {
	UseEnv    bool   `json:"use_env"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key_id"`
	SecretKey string `json:"secret_access_key"`
	Version   string `json:"version"`
}

type StackdriverConfig struct {
	ProjectID     string            `json:"project_id"`
	MetricPrefix  string            `json:"metric_prefix"`
	DefaultLabels map[string]string `json:"default_labels"`
}

type OcagentConfig struct {
	Address            string            `json:"address"`
	ServiceName        string            `json:"service_name"`
	Headers            map[string]string `json:"headers"`
	Insecure           bool              `json:"insecure"`
	Reconnection       string            `json:"reconnection"`
	EnaableCompression bool              `json:"enable_compression"`
}

type DataDogConfig struct {
	Namespace              string                 `json:"namespace"`
	Service                string                 `json:"service"`
	TraceAddr              string                 `json:"trace_address"`
	StatsAddr              string                 `json:"stats_address"`
	Tags                   []string               `json:"tags"`
	GlobalTags             map[string]interface{} `json:"global_tags"`
	DisableCountPerBuckets bool                   `json:"disable_count_per_buckets"`
}

const (
	ContextKey = "opencensus-request-span"
	Namespace  = "github_com/devopsfaith/krakend-opencensus"
)

var (
	DefaultViews = []*view.View{
		ochttp.ClientSentBytesDistribution,
		ochttp.ClientReceivedBytesDistribution,
		ochttp.ClientRoundtripLatencyDistribution,
		ochttp.ClientCompletedCount,

		ochttp.ServerRequestCountView,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ochttp.ServerLatencyView,
		ochttp.ServerRequestCountByMethod,
		ochttp.ServerResponseCountByStatusCode,
	}

	exporterFactories                     = []ExporterFactory{}
	ErrNoConfig                           = errors.New("no extra config defined for the opencensus module")
	errSingletonExporterFactoriesRegister = errors.New("expecting only one exporter factory registration per instance")
	mu                                    = new(sync.RWMutex)
	register                              = composableRegister{
		viewExporter:       registerViewExporter,
		traceExporter:      registerTraceExporter,
		setDefaultSampler:  setDefaultSampler,
		setReportingPeriod: setReportingPeriod,
		registerViews:      registerViews,
	}
	registerOnce  = new(sync.Once)
	enabledLayers EnabledLayers
)

type EnabledLayers struct {
	Router  bool `json:"router"`
	Pipe    bool `json:"pipe"`
	Backend bool `json:"backend"`
}

func IsRouterEnabled() bool {
	return enabledLayers.Router
}

func IsPipeEnabled() bool {
	return enabledLayers.Pipe
}

func IsBackendEnabled() bool {
	return enabledLayers.Backend
}

func parseCfg(srvCfg config.ServiceConfig) (*Config, error) {
	cfg := new(Config)
	tmp, ok := srvCfg.ExtraConfig[Namespace]
	if !ok {
		return nil, ErrNoConfig
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(tmp)
	if err := json.NewDecoder(buf).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseEndpointConfig(endpointCfg *config.EndpointConfig) (*EndpointExtraConfig, error) {
	cfg := new(EndpointExtraConfig)
	if endpointCfg == nil || endpointCfg.ExtraConfig == nil {
		return nil, ErrNoConfig
	}
	tmp, ok := endpointCfg.ExtraConfig[Namespace]
	if !ok {
		return nil, ErrNoConfig
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(tmp)
	if err := json.NewDecoder(buf).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseBackendConfig(backendCfg *config.Backend) (*EndpointExtraConfig, error) {
	cfg := new(EndpointExtraConfig)
	if backendCfg == nil || backendCfg.ExtraConfig == nil {
		return nil, ErrNoConfig
	}
	tmp, ok := backendCfg.ExtraConfig[Namespace]
	if !ok {
		return nil, ErrNoConfig
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(tmp)
	if err := json.NewDecoder(buf).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

var (
	replaceMetricPath        = regexp.MustCompile(`:([^\/]+)`)
	replaceMetricBackendPath = regexp.MustCompile(`{{.(.*?)}}`)
)

// GetAggregatedPathForMetrics returns a path aggregator function ready to reduce path cardinality in the metrics
func GetAggregatedPathForMetrics(cfg *config.EndpointConfig) func(r *http.Request) string {
	if cfg == nil {
		return simplePathExtractor
	}

	aggregationMode := aggregationModePattern
	endpointExtraCfg, endpointExtraCfgErr := parseEndpointConfig(cfg)
	if endpointExtraCfgErr == nil {
		aggregationMode = endpointExtraCfg.PathAggregation
	}

	if aggregationMode == aggregationModeLastParam {
		// only aggregates the last section of the path if it is a parameter,
		// will default to pattern mode if the last part of the url is not a parameter (misconfiguration)
		lastArgument := cfg.Endpoint[strings.LastIndex(cfg.Endpoint, "/")+1:]
		if len(lastArgument) > 0 && lastArgument[0] == endpointPrefix {
			return func(r *http.Request) string {
				// lastArgument is a parameter, aggregate and overwrite path
				path := r.URL.Path[:strings.LastIndex(r.URL.Path, "/")+1] + lastArgument
				return strings.ToLower(replaceMetricPath.ReplaceAllString(path, `{$1}`))
			}
		}
	}

	if aggregationMode == aggregationModePOff {
		// no aggregration (use with caution!)
		return simplePathExtractor
	}

	// normalize path
	return fixedPathExtractor(replaceMetricPath.ReplaceAllString(cfg.Endpoint, `{$1}`))
}

// GetAggregatedPathForBackendMetrics returns a path aggregator function ready to reduce path cardinality in the metrics
func GetAggregatedPathForBackendMetrics(cfg *config.Backend) func(r *http.Request) string {
	if cfg == nil {
		return simplePathExtractor
	}
	aggregationMode := aggregationModePattern
	endpointExtraCfg, endpointExtraCfgErr := parseBackendConfig(cfg)
	if endpointExtraCfgErr == nil {
		aggregationMode = endpointExtraCfg.PathAggregation
	}

	if aggregationMode == aggregationModeLastParam {
		// only aggregates the last section of the path if it is a parameter,
		// will default to pattern mode if the last part of the url is not a parameter (misconfiguration)
		lastArgument := cfg.URLPattern[strings.LastIndex(cfg.URLPattern, "/")+1:]
		prefixSize := len(backendPrefix)
		if len(lastArgument) > prefixSize && lastArgument[:prefixSize] == backendPrefix {
			return func(r *http.Request) string {
				// lastArgument is a parameter, aggregate and overwrite path
				path := r.URL.Path[:strings.LastIndex(r.URL.Path, "/")+1] + lastArgument
				return strings.ToLower(replaceMetricBackendPath.ReplaceAllString(path, `{$1}`))
			}
		}
	}

	if aggregationMode == aggregationModePOff {
		// no aggregration (use with caution!)
		return simplePathExtractor
	}

	// normalize path
	return fixedPathExtractor(replaceMetricBackendPath.ReplaceAllString(cfg.URLPattern, `{$1}`))
}

func simplePathExtractor(r *http.Request) string { return r.URL.Path }

func fixedPathExtractor(path string) func(r *http.Request) string {
	path = strings.ToLower(path)
	return func(_ *http.Request) string { return path }
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

const (
	aggregationModePattern   = "pattern"
	aggregationModeLastParam = "lastparam"
	aggregationModePOff      = "off"

	endpointPrefix = ':'
	backendPrefix  = "{{."
)
