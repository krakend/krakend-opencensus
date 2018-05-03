package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"github.com/devopsfaith/krakend-opencensus"
	"github.com/devopsfaith/krakend-opencensus/exporter/jaeger"
	"github.com/devopsfaith/krakend-opencensus/exporter/prometheus"
	"github.com/devopsfaith/krakend-opencensus/exporter/zipkin"
	opencensusgin "github.com/devopsfaith/krakend-opencensus/router/gin"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	prometheusPort := flag.Int("s", 9091, "Port of the prometheus service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	configFile := flag.String("c", "/etc/krakend/configuration.json", "Path to the configuration filename")
	zipkinURL := flag.String("zipkin", "http://192.168.99.100:9411/api/v2/spans", "url of the zipkin endpoint")
	jaegerURL := flag.String("jaeger", "http://192.168.99.100:14268", "url of the jaeger thrift endpoint")
	serviceName := flag.String("name", "krakend", "name of the service")
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case sig := <-sigs:
			log.Println("Signal intercepted:", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	parser := config.NewParser()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	logger, _ := logging.NewLogger(*logLevel, os.Stdout, "[KRAKEND]")

	// Register stats and trace exporters to export the collected data.
	{
		exporter, err := prometheus.Exporter(ctx, *prometheusPort)
		if err != nil {
			logger.Fatal("creating the prometheus exporter:", err.Error())
		}

		zipkinExporter := zipkin.Exporter(*zipkinURL, *serviceName, "127.0.0.1", serviceConfig.Port)
		jaegerExporter, err := jaeger.Exporter(*jaegerURL, *serviceName)
		if err != nil {
			log.Fatal("creating the jaeger exporter:", err.Error())
		}

		if err := opencensus.Register(serviceConfig, []view.Exporter{exporter}, []trace.Exporter{zipkinExporter, jaegerExporter}, []*view.View{}); err != nil {
			log.Fatal(err)
		}
	}

	bf := func(cfg *config.Backend) proxy.Proxy {
		return proxy.NewHTTPProxyWithHTTPExecutor(cfg, opencensus.HTTPRequestExecutor(proxy.NewHTTPClient), cfg.Decoder)
	}

	// setup the krakend router
	routerFactory := krakendgin.NewFactory(krakendgin.Config{
		Engine:         gin.Default(),
		ProxyFactory:   opencensus.ProxyFactory(proxy.NewDefaultFactory(opencensus.BackendFactory(bf), logger)),
		Middlewares:    []gin.HandlerFunc{},
		Logger:         logger,
		HandlerFactory: opencensusgin.New(krakendgin.EndpointHandler),
	})

	// start the engine
	routerFactory.NewWithContext(ctx).Run(serviceConfig)
}
