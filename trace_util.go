package opencensus

import (
	"net/http"

	"go.opencensus.io/trace"
	"go.opencensus.io/plugin/ochttp"
)

func SpanNameFromURL(req *http.Request) string {
	return req.URL.Path
}

func RequestAttrs(r *http.Request) []trace.Attribute {
	userAgent := r.UserAgent()

	attrs := make([]trace.Attribute, 0, 5)
	attrs = append(attrs,
		trace.StringAttribute(ochttp.PathAttribute, r.URL.Path),
		trace.StringAttribute(ochttp.URLAttribute, r.URL.String()),
		trace.StringAttribute(ochttp.HostAttribute, r.Host),
		trace.StringAttribute(ochttp.MethodAttribute, r.Method),
	)

	if userAgent != "" {
		attrs = append(attrs, trace.StringAttribute(ochttp.UserAgentAttribute, userAgent))
	}

	return attrs
}

func ResponseAttrs(resp *http.Response) []trace.Attribute {
	return []trace.Attribute{
		trace.Int64Attribute(ochttp.StatusCodeAttribute, int64(resp.StatusCode)),
	}
}
