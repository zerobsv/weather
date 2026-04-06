package weather

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	httpRequestsTotal metric.Float64Counter
	httpRequestDuration metric.Float64Histogram
	meter metric.Meter
)

func prometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Collect metrics
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		httpRequestsTotal.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.Key("method").String(c.Request.Method),
				attribute.Key("endpoint").String(c.FullPath()),
				attribute.Key("status").String(http.StatusText(status)),
			))
		httpRequestDuration.Record(context.Background(), duration,
			metric.WithAttributes(
				attribute.Key("method").String(c.Request.Method),
				attribute.Key("endpoint").String(c.FullPath()),
			))
	}
}

func WeatherServer() {

	registry := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		log.Fatal(err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	meter = provider.Meter("weather")

	// Create instruments
	httpRequestsTotal, err = meter.Float64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		log.Fatal(err)
	}
	httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Histogram of response time for handler in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Fatal(err)
	}

	initMetrics(meter)

	router := gin.Default()

	// Add Prometheus middleware
	router.Use(prometheusMiddleware())

	// Define routes
	router.GET("/", getHandleDefaultRoute)
	router.GET("/weather", instrumentedGetWeatherLocal)
	router.GET("/weather/:location", instrumentedGetWeatherInternational)

	router.GET("/weather/stress0", instrumentedGetWeatherStressTest0)
	router.GET("/weather/stress1", instrumentedGetWeatherStressTest1)
	router.GET("/weather/stress2", instrumentedGetWeatherStressTest2)
	router.GET("/weather/stress3", instrumentedGetWeatherStressTest3)

	// Add /metrics endpoint for Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))

	log.Println("Starting gin gonic...")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	<-ctx.Done()
	log.Println("timeout of 5 seconds.")
	log.Println("Server exiting")

}
