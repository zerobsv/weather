package weather

import (
	"context"
	stdlog "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	httpRequestsTotal   metric.Float64Counter
	httpRequestDuration metric.Float64Histogram
	meter               metric.Meter
	logger              log.Logger
	slogLogger          *slog.Logger
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
		stdlog.Fatal(err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	meter = provider.Meter("weather")

	loggerProvider := sdklog.NewLoggerProvider()
	global.SetLoggerProvider(loggerProvider)
	logger = global.Logger("weather")

	slogLogger = otelslog.NewLogger("weather", otelslog.WithLoggerProvider(loggerProvider))

	// Create instruments
	httpRequestsTotal, err = meter.Float64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		stdlog.Fatal(err)
	}
	httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Histogram of response time for handler in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		stdlog.Fatal(err)
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

	slogLogger.Info("Starting gin gonic on :8080")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			stdlog.Fatalf("listen: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slogLogger.Info("Shutdown Server ...")

	if err := srv.Shutdown(ctx); err != nil {
		stdlog.Fatal("Server Shutdown:", err)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	<-ctx.Done()

	slogLogger.Info("timeout of 5 seconds.")
	slogLogger.Info("Server exiting")

}
