package mux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devopsfaith/krakend-opencensus"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

var (
	extraConfig = []byte(`{
		"github_com/devopsfaith/krakend-opencensus": {
			"enabled_layers": {
				"router": true
			}
		}}`)
	extraCfg map[string]interface{}
)

func init() {
	if err := registerModule(); err != nil {
		fmt.Printf("Problem registering opencensus module: %s", err.Error())
	}
}

func TestNew(t *testing.T) {
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		t.Fatalf("Failed to register ochttp.DefaultServerViews error: %v", err)
	}

	hf := New(func(_ *config.EndpointConfig, _ proxy.Proxy) http.HandlerFunc {
		return httpHandler(http.StatusOK, rand.Intn(512)+512).ServeHTTP
	})
	h := hf(&config.EndpointConfig{}, proxy.NoopProxy)

	totalCount := 100000
	meanReqSize, meanRespSize := 512.0, 768.0

	for i := 0; i < totalCount; i++ {
		w := httptest.NewRecorder()
		data := make([]byte, rand.Intn(1024))
		req, err := http.NewRequest("POST", "/some", bytes.NewBuffer(data))
		if err != nil {
			t.Error(err.Error())
			return
		}
		h(w, req)
	}

	views := []string{
		"opencensus.io/http/server/request_count",
		"opencensus.io/http/server/latency",
		"opencensus.io/http/server/request_bytes",
		"opencensus.io/http/server/response_bytes",
	}
	for _, viewName := range views {
		v := view.Find(viewName)
		if v == nil {
			t.Errorf("view not found %q", viewName)
			continue
		}
		rows, err := view.RetrieveData(viewName)
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := len(rows), 1; got != want {
			t.Errorf("len(%q) = %d; want %d", viewName, got, want)
			continue
		}
		data := rows[0].Data

		var count int
		var sum float64
		switch data := data.(type) {
		case *view.CountData:
			count = int(data.Value)
		case *view.DistributionData:
			count = int(data.Count)
			sum = data.Sum()
		default:
			t.Errorf("Unkown data type: %v", data)
			continue
		}

		if got, want := count, totalCount; got != want {
			t.Fatalf("%s = %d; want %d", viewName, got, want)
		}

		// We can only check sum for distribution views.
		switch viewName {
		case "opencensus.io/http/server/request_bytes":
			if got, want := sum, meanReqSize*float64(totalCount); math.Sqrt(got*got-want*want) <= .01*want {
				t.Fatalf("%s = %g; want %g", viewName, got, want)
			}
		case "opencensus.io/http/server/response_bytes":
			if got, want := sum, meanRespSize*float64(totalCount); math.Sqrt(got*got-want*want) <= .01*want {
				t.Fatalf("%s = %g; want %g", viewName, got, want)
			}
		}
	}
}

func httpHandler(statusCode, respSize int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		body := make([]byte, respSize)
		w.Write(body)
	})
}

func registerModule() error {
	if err := json.Unmarshal(extraConfig, &extraCfg); err != nil {
		return err
	}

	return opencensus.Register(context.Background(), config.ServiceConfig{ExtraConfig: extraCfg})
}
