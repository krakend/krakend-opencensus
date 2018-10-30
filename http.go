package opencensus

import (
	"context"
	"net/http"

	transport "github.com/devopsfaith/krakend/transport/http/client"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

var defaultClient = &http.Client{Transport: &ochttp.Transport{}}

func NewHTTPClient(ctx context.Context) *http.Client {
	if !IsBackendEnabled() {
		return transport.NewHTTPClient(ctx)
	}
	return defaultClient
}

func HTTPRequestExecutor(clientFactory transport.HTTPClientFactory) transport.HTTPRequestExecutor {
	if !IsBackendEnabled() {
		return transport.DefaultHTTPRequestExecutor(clientFactory)
	}
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		client := clientFactory(ctx)
		if _, ok := client.Transport.(*ochttp.Transport); !ok {
			client.Transport = &ochttp.Transport{Base: client.Transport}
		}
		return client.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
	}
}
