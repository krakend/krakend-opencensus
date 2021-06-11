package opencensus

import (
	"net/http"
	"testing"

	"github.com/luraproject/lura/config"
)

func TestGetAggregatedPathForMetrics(t *testing.T) {
	for i, tc := range []struct {
		cfg      *config.EndpointConfig
		expected string
	}{
		{
			cfg:      &config.EndpointConfig{Endpoint: "/api/:foo/:bar"},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.EndpointConfig{
				Endpoint: "/api/:foo/:bar",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "pattern"},
				},
			},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.EndpointConfig{
				Endpoint: "/api/:foo/:bar",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "lastparam"},
				},
			},
			expected: "/api/foo/{bar}",
		},
		{
			cfg: &config.EndpointConfig{
				Endpoint: "/api/:foo/:bar",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "off"},
				},
			},
			expected: "/api/foo/bar",
		},
		{
			expected: "/api/foo/bar",
		},
	} {
		extractor := GetAggregatedPathForMetrics(tc.cfg)
		r, _ := http.NewRequest("GET", "http://example.tld/api/foo/bar", nil)
		if tag := extractor(r); tag != tc.expected {
			t.Errorf("tc-%d: unexpected result: %s", i, tag)
		}
	}
}

func TestGetAggregatedPathForBackendMetrics(t *testing.T) {
	for i, tc := range []struct {
		cfg      *config.Backend
		expected string
	}{
		{
			cfg:      &config.Backend{URLPattern: "/api/{{.Foo}}/{{.Bar}}"},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.Backend{
				URLPattern: "/api/{{.Foo}}/{{.Bar}}",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "pattern"},
				},
			},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.Backend{
				URLPattern: "/api/{{.Foo}}/{{.Bar}}",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "lastparam"},
				},
			},
			expected: "/api/foo/{bar}",
		},
		{
			cfg: &config.Backend{
				URLPattern: "/api/{{.Foo}}/{{.Bar}}",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "off"},
				},
			},
			expected: "/api/foo/bar",
		},
		{
			expected: "/api/foo/bar",
		},
	} {
		extractor := GetAggregatedPathForBackendMetrics(tc.cfg)
		r, _ := http.NewRequest("GET", "http://example.tld/api/foo/bar", nil)
		if tag := extractor(r); tag != tc.expected {
			t.Errorf("tc-%d: unexpected result: %s", i, tag)
		}
	}
}
