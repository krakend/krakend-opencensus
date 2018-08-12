package opencensus

import (
	"context"
	"net/http"

	"github.com/devopsfaith/krakend/proxy"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

var defaultClient = &http.Client{Transport: &ochttp.Transport{}}

func NewHTTPClient(ctx context.Context) *http.Client {
	if !IsBackendEnabled() {
		return proxy.NewHTTPClient(ctx)
	}
	return defaultClient
}

func HTTPRequestExecutor(clientFactory proxy.HTTPClientFactory) proxy.HTTPRequestExecutor {
	if !IsBackendEnabled() {
		return proxy.DefaultHTTPRequestExecutor(clientFactory)
	}
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		client := clientFactory(ctx)
		if _, ok := client.Transport.(*ochttp.Transport); !ok {
			client.Transport = &ochttp.Transport{Base: client.Transport}
		}
		return client.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
	}
}
