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
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	httpRequestsTotal   metric.Float64Counter
	httpRequestDuration metric.Float64Histogram
	meter               metric.Meter
	logger              *slog.Logger
	traceProvider       *sdktrace.TracerProvider
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

	// Initialize trace exporter for otel-collector sidecar
	traceExporter, err := otlptracegrpc.New(context.Background())
	if err != nil {
		stdlog.Fatal("Failed to create trace exporter: ", err)
	}
	traceProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(traceProvider)

	// Initialize log exporter for otel-collector sidecar
	logExporter, err := otlploggrpc.New(context.Background())
	if err != nil {
		stdlog.Fatal("Failed to create log exporter: ", err)
	}
	loggerProvider := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(logExporter)))
	global.SetLoggerProvider(loggerProvider)

	logger = otelslog.NewLogger("weather", otelslog.WithLoggerProvider(loggerProvider))

	// Create instruments
	httpRequestsTotal, err = meter.Float64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		logger.Error("Failed to create http_requests_total counter", "error", err)
		stdlog.Fatal(err)
	}
	httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Histogram of response time for handler in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		logger.Error("Failed to create http_request_duration_seconds histogram", "error", err)
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

	logger.Info("Starting gin gonic on :8081")

	// Setup metrics server on separate port
	metricsRouter := gin.New()
	metricsRouter.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	metricsServer := &http.Server{
		Addr:    ":8082",
		Handler: metricsRouter,
	}

	logger.Info("Starting metrics server on :8082")

	srv := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			stdlog.Fatalf("listen: %v\n", err)
		}
	}()

	go func() {
		// metrics server connections
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start metrics server", "error", err)
			stdlog.Fatalf("listen: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown Server ...")

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server Shutdown Failed", "error", err)
		stdlog.Fatal("Server Shutdown:", err)
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		logger.Error("Metrics Server Shutdown Failed", "error", err)
		stdlog.Fatal("Metrics Server Shutdown:", err)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	<-ctx.Done()

	logger.Info("timeout of 5 seconds.")
	logger.Info("Server exiting")

	// Shutdown trace provider to flush remaining spans
	if err := traceProvider.Shutdown(context.Background()); err != nil {
		logger.Error("Failed to shutdown trace provider", "error", err)
	}

}
