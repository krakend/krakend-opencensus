package mux

import (
	"net/http"

	opencensus "github.com/devopsfaith/krakend-opencensus"
	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/proxy"
	"github.com/luraproject/lura/router/mux"
	"go.opencensus.io/plugin/ochttp"
)

func New(hf mux.HandlerFactory) mux.HandlerFactory {
	if !opencensus.IsRouterEnabled() {
		return hf
	}
	return func(cfg *config.EndpointConfig, p proxy.Proxy) http.HandlerFunc {
		handler := ochttp.Handler{Handler: tagAggregationMiddleware(hf(cfg, p), cfg)}
		return handler.ServeHTTP
	}
}

func tagAggregationMiddleware(next http.Handler, cfg *config.EndpointConfig) http.Handler {
	pathExtractor := opencensus.GetAggregatedPathForMetrics(cfg)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ochttp.SetRoute(r.Context(), pathExtractor(r))
		next.ServeHTTP(w, r)
	})
}
