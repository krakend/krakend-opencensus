package gin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/proxy"
	krakendgin "github.com/luraproject/lura/v2/router/gin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"

	opencensus "github.com/krakend/krakend-opencensus/v2"
)

// New wraps a handler factory adding some simple instrumentation to the generated handlers
func New(hf krakendgin.HandlerFactory) krakendgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		return HandlerFunc(cfg, hf(cfg, p), nil)
	}
}

func HandlerFunc(cfg *config.EndpointConfig, next gin.HandlerFunc, prop propagation.HTTPFormat) gin.HandlerFunc {
	if !opencensus.IsRouterEnabled() {
		return next
	}
	if prop == nil {
		prop = &b3.HTTPFormat{}
	}
	pathExtractor := opencensus.GetAggregatedPathForMetrics(cfg)
	h := &handler{
		name:        cfg.Endpoint,
		propagation: prop,
		Handler:     next,
		StartOptions: trace.StartOptions{
			SpanKind: trace.SpanKindServer,
		},
		tags: []tagGenerator{
			func(_ *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyServerRoute, cfg.Endpoint) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Host, r.Host) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Method, r.Method) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Path, pathExtractor(r)) },
		},
	}
	return h.HandlerFunc
}

type handler struct {
	name             string
	propagation      propagation.HTTPFormat
	Handler          gin.HandlerFunc
	StartOptions     trace.StartOptions
	IsPublicEndpoint bool
	tags             []tagGenerator
}

type tagGenerator func(*http.Request) tag.Mutator

func (h *handler) HandlerFunc(c *gin.Context) {
	var traceEnd, statsEnd func()
	c.Request, traceEnd = h.startTrace(c.Writer, c.Request)
	c.Writer, statsEnd = h.startStats(c.Writer, c.Request)

	c.Set(opencensus.ContextKey, trace.FromContext(c.Request.Context()))
	h.Handler(c)

	statsEnd()
	traceEnd()
}

func (h *handler) startTrace(_ gin.ResponseWriter, r *http.Request) (*http.Request, func()) {
	ctx := r.Context()
	var span *trace.Span
	sc, ok := h.extractSpanContext(r)

	if ok && !h.IsPublicEndpoint {
		ctx, span = trace.StartSpanWithRemoteParent(
			ctx,
			h.name,
			sc,
			trace.WithSampler(h.StartOptions.Sampler),
			trace.WithSpanKind(h.StartOptions.SpanKind),
		)
	} else {
		ctx, span = trace.StartSpan(
			ctx,
			h.name,
			trace.WithSampler(h.StartOptions.Sampler),
			trace.WithSpanKind(h.StartOptions.SpanKind),
		)

		if ok {
			span.AddLink(trace.Link{
				TraceID:    sc.TraceID,
				SpanID:     sc.SpanID,
				Type:       trace.LinkTypeChild,
				Attributes: nil,
			})
		}
	}

	span.AddAttributes(opencensus.RequestAttrs(r)...)
	return r.WithContext(ctx), span.End
}

func (h *handler) extractSpanContext(r *http.Request) (trace.SpanContext, bool) {
	return h.propagation.SpanContextFromRequest(r)
}

func (h *handler) startStats(w gin.ResponseWriter, r *http.Request) (gin.ResponseWriter, func()) {
	tags := make([]tag.Mutator, len(h.tags))
	for i, t := range h.tags {
		tags[i] = t(r)
	}
	ctx, _ := tag.New(r.Context(), tags...)
	track := &trackingResponseWriter{
		start:          time.Now(),
		ctx:            ctx,
		ResponseWriter: w,
	}
	if r.Body == nil {
		// TODO: Handle cases where ContentLength is not set.
		track.reqSize = -1
	} else if r.ContentLength > 0 {
		track.reqSize = r.ContentLength
	}
	stats.Record(ctx, ochttp.ServerRequestCount.M(1))
	return track, track.end
}
