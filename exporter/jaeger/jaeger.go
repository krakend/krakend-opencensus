package jaeger

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"contrib.go.opencensus.io/exporter/jaeger"
	opencensus "github.com/krakendio/krakend-opencensus/v2"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func getProcessTags(cfg *opencensus.Config) ([]jaeger.Tag, error) {
	if cfg.Exporters.Jaeger.ProcessTags == nil {
		return nil, nil
	}

	tags := make([]jaeger.Tag, 0, len(cfg.Exporters.Jaeger.ProcessTags))

	for key, value := range cfg.Exporters.Jaeger.ProcessTags {
		if value == nil {
			return nil, fmt.Errorf("jaeger process tag '%s' is nil", key)
		}
		switch v := value.(type) {
		case bool:
			tags = append(tags, jaeger.BoolTag(key, v))
		case float64:
			tags = append(tags, jaeger.Int64Tag(key, int64(v)))
		case string:
			tags = append(tags, jaeger.StringTag(key, v))
		default:
			return nil, fmt.Errorf("invalid type '%s' for jaeger process tag '%s'", reflect.TypeOf(value).String(), key)
		}
	}
	return tags, nil
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*jaeger.Exporter, error) {
	if cfg.Exporters.Jaeger == nil {
		return nil, errDisabled
	}

	processTags, err := getProcessTags(&cfg)
	if err != nil {
		return nil, err
	}

	e, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     cfg.Exporters.Jaeger.AgentEndpoint,
		CollectorEndpoint: cfg.Exporters.Jaeger.Endpoint,
		BufferMaxCount:    cfg.Exporters.Jaeger.BufferMaxCount,
		Process: jaeger.Process{
			ServiceName: cfg.Exporters.Jaeger.ServiceName,
			Tags:        processTags,
		},
	})
	if err != nil {
		return e, err
	}

	go func() {
		<-ctx.Done()
		e.Flush()
	}()

	return e, nil
}

var errDisabled = errors.New("opencensus jaeger exporter disabled")
