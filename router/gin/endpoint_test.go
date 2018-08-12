package gin

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
	"github.com/gin-gonic/gin"
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

func TestNew_post(t *testing.T) {

	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		t.Fatalf("Failed to register ochttp.DefaultServerViews error: %v", err)
	}

	hf := New(func(_ *config.EndpointConfig, _ proxy.Proxy) gin.HandlerFunc {
		return httpHandler(http.StatusOK, rand.Intn(512)+512)
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/post", hf(&config.EndpointConfig{}, proxy.NoopProxy))

	totalCount := 100000
	meanReqSize, meanRespSize := 512.0, 768.0

	for i := 0; i < totalCount; i++ {
		w := httptest.NewRecorder()
		data := make([]byte, rand.Intn(1024))
		req, err := http.NewRequest("POST", "/post", bytes.NewBuffer(data))
		if err != nil {
			t.Error(err.Error())
			return
		}
		engine.ServeHTTP(w, req)
	}

	checkViews(t, totalCount, meanReqSize, meanRespSize)
}

func TestNew_get(t *testing.T) {
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		t.Fatalf("Failed to register ochttp.DefaultServerViews error: %v", err)
	}

	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		t.Fatalf("Failed to register ochttp.DefaultServerViews error: %v", err)
	}

	hf := New(func(_ *config.EndpointConfig, _ proxy.Proxy) gin.HandlerFunc {
		return httpHandler(http.StatusOK, rand.Intn(512)+512)
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/get", hf(&config.EndpointConfig{}, proxy.NoopProxy))

	totalCount := 100000
	meanReqSize, meanRespSize := 0.0, 768.0

	for i := 0; i < totalCount; i++ {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/get", nil)
		if err != nil {
			t.Error(err.Error())
			return
		}
		engine.ServeHTTP(w, req)
	}

	checkViews(t, totalCount, meanReqSize, meanRespSize)
}

func checkViews(t *testing.T, totalCount int, meanReqSize, meanRespSize float64) {
	for _, serverView := range ochttp.DefaultServerViews {
		v := view.Find(serverView.Name)
		if v == nil {
			t.Errorf("view not found %q", serverView.Name)
			continue
		}
		rows, err := view.RetrieveData(serverView.Name)
		if err != nil {
			t.Error(err)
			continue
		}
		if got, want := len(rows), 1; got != want {
			t.Errorf("len(%q) = %d; want %d", serverView.Name, got, want)
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
			t.Fatalf("%s = %d; want %d", serverView.Name, got, want)
		}

		// We can only check sum for distribution views.
		switch serverView.Name {
		case "opencensus.io/http/server/request_bytes":
			if got, want := sum, meanReqSize*float64(totalCount); math.Sqrt(got*got-want*want) <= .01*want {
				t.Fatalf("%s = %g; want %g", serverView.Name, got, want)
			}
		case "opencensus.io/http/server/response_bytes":
			if got, want := sum, meanRespSize*float64(totalCount); math.Sqrt(got*got-want*want) <= .01*want {
				t.Fatalf("%s = %g; want %g", serverView.Name, got, want)
			}
		}
	}
}

func httpHandler(statusCode, respSize int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(statusCode)
		body := make([]byte, respSize)
		c.Writer.Write(body)
	}
}

func registerModule() error {
	if err := json.Unmarshal(extraConfig, &extraCfg); err != nil {
		return err
	}

	return opencensus.Register(context.Background(), config.ServiceConfig{ExtraConfig: extraCfg})
}
