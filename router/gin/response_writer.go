package gin

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/gin-gonic/gin"
	"go.opencensus.io/plugin/ochttp"
)

type trackingResponseWriter struct {
	gin.ResponseWriter
	ctx     context.Context
	reqSize int64
	start   time.Time
	endOnce sync.Once
}

var _ gin.ResponseWriter = (*trackingResponseWriter)(nil)

func (t *trackingResponseWriter) end() {
	t.endOnce.Do(func() {
		m := []stats.Measurement{
			ochttp.ServerLatency.M(float64(time.Since(t.start)) / float64(time.Millisecond)),
			ochttp.ServerResponseBytes.M(int64(t.Size())),
		}
		if t.reqSize >= 0 {
			m = append(m, ochttp.ServerRequestBytes.M(t.reqSize))
		}
		status := t.Status()
		if status == 0 {
			status = http.StatusOK
		}
		ctx, _ := tag.New(t.ctx, tag.Upsert(ochttp.StatusCode, strconv.Itoa(status)))
		stats.Record(ctx, m...)
	})
}
