package opencensus

import (
	"context"
	"net/http"

	"github.com/devopsfaith/krakend/config"
	transport "github.com/devopsfaith/krakend/transport/http/client"
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
		client := clientFactory(ctx)
		if _, ok := client.Transport.(*Transport); !ok {
			tags := []tagGenerator{
				func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientHost, req.Host) },
				func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientPath, pathExtractor(r)) },
				func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientMethod, req.Method) },
			}
			client.Transport = &Transport{Base: client.Transport, tags: tags}
		}

		return client.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
	}
}
