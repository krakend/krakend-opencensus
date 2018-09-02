package opencensus

import (
	"context"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	"go.opencensus.io/trace"
)

const errCtxCanceledMsg = "context canceled"

func Middleware(name string) proxy.Middleware {
	if !IsPipeEnabled() {
		return proxy.EmptyMiddleware
	}
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		if len(next) < 1 {
			panic(proxy.ErrNotEnoughProxies)
		}
		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			var span *trace.Span
			ctx, span = trace.StartSpan(trace.NewContext(ctx, fromContext(ctx)), name)
			resp, err := next[0](ctx, req)

			if err != nil {
				if err.Error() != errCtxCanceledMsg {
					span.AddAttributes(trace.StringAttribute("error", err.Error()))
				} else {
					span.AddAttributes(trace.BoolAttribute("canceled", true))
				}
			}
			span.AddAttributes(trace.BoolAttribute("complete", resp != nil && resp.IsComplete))
			span.End()

			return resp, err
		}
	}
}

func ProxyFactory(pf proxy.Factory) proxy.FactoryFunc {
	if !IsPipeEnabled() {
		return pf.New
	}
	return func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		next, err := pf.New(cfg)
		if err != nil {
			return next, err
		}
		return Middleware("pipe-" + cfg.Endpoint)(next), nil
	}
}

func BackendFactory(bf proxy.BackendFactory) proxy.BackendFactory {
	if !IsBackendEnabled() {
		return bf
	}
	return func(cfg *config.Backend) proxy.Proxy {
		return Middleware("backend-" + cfg.URLPattern)(bf(cfg))
	}
}
