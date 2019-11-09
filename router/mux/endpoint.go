package mux

import (
	"net/http"

	opencensus "github.com/devopsfaith/krakend-opencensus"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/router/mux"
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
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ochttp.SetRoute(r.Context(), opencensus.GetAggregatedPathForMetrics(cfg, r))
		next.ServeHTTP(w, r)
    })
}
