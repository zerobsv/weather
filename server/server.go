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
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time for handler in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// func prometheusMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		start := time.Now()

// 		// Process request
// 		c.Next()

// 		// Collect metrics
// 		duration := time.Since(start).Seconds()
// 		status := c.Writer.Status()
// 		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), http.StatusText(status)).Inc()
// 		httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
// 	}
// }

func WeatherServer() {

	router := gin.Default()

	// Add Prometheus middleware
	// router.Use(prometheusMiddleware())

	// Define routes
	router.GET("/", getHandleDefaultRoute)
	router.GET("/weather", instrumentedGetWeatherLocal)
	router.GET("/weather/:location", instrumentedGetWeatherInternational)

	router.GET("/weather/stress0", instrumentedGetWeatherStressTest0)
	router.GET("/weather/stress1", instrumentedGetWeatherStressTest1)
	router.GET("/weather/stress2", instrumentedGetWeatherStressTest2)
	router.GET("/weather/stress3", instrumentedGetWeatherStressTest3)

	// Add /metrics endpoint for Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	log.Println("Starting gin gonic...")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		// service connections
		if err := srv.ListenAndServeTLS("server.pem", "server.key"); err != nil && err != http.ErrServerClosed {
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
