package opencensus

import (
	"context"
	"net/http"

	"github.com/luraproject/lura/v2/config"
	transport "github.com/luraproject/lura/v2/transport/http/client"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/tag"
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
	return HTTPRequestExecutorFromConfig(clientFactory, nil)
}

func HTTPRequestExecutorFromConfig(clientFactory transport.HTTPClientFactory, cfg *config.Backend) transport.HTTPRequestExecutor {
	if !IsBackendEnabled() {
		return transport.DefaultHTTPRequestExecutor(clientFactory)
	}

	pathExtractor := GetAggregatedPathForBackendMetrics(cfg)

	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		httpClient := clientFactory(ctx)

		if _, ok := httpClient.Transport.(*Transport); ok {
			return httpClient.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
		}

		c := &http.Client{
			Transport: &Transport{
				Base: httpClient.Transport,
				tags: []tagGenerator{
					func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientHost, req.Host) },
					func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientPath, pathExtractor(r)) },
					func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientMethod, req.Method) },
				},
			},
			CheckRedirect: httpClient.CheckRedirect,
			Jar:           httpClient.Jar,
			Timeout:       httpClient.Timeout,
		}
		return c.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
	}
}
