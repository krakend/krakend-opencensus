package opencensus

import (
	"context"
	"net/http"

	"github.com/devopsfaith/krakend/config"
	transport "github.com/devopsfaith/krakend/transport/http/client"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.opencensus.io/tag"
)

var defaultClient = &http.Client{Transport: &ochttp.Transport{}}

func NewHTTPClient(ctx context.Context) *http.Client {
	if !IsBackendEnabled() {
		return transport.NewHTTPClient(ctx)
	}
	return defaultClient
}

func HTTPRequestExecutor(clientFactory transport.HTTPClientFactory, cfg *config.Backend) transport.HTTPRequestExecutor {
	if !IsBackendEnabled() {
		return transport.DefaultHTTPRequestExecutor(clientFactory)
	}
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		// tags for stats
		tags := []tagGenerator{
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientHost, req.Host) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Host, req.Host) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientPath, GetAggregatedPathForBackendMetrics(cfg, req)) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Path, GetAggregatedPathForBackendMetrics(cfg, req)) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientMethod, req.Method) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Method, req.Method) },
		}
		client := clientFactory(ctx)
		client.Transport = &Transport{Base: client.Transport, tags: tags}
		return client.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
	}
}
