package opencensus

import (
	"context"
	"net/http"

	"github.com/devopsfaith/krakend/proxy"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

var defaultClient = &http.Client{Transport: &ochttp.Transport{}}

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
