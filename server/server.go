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
)


func WeatherServer() {

	router := gin.Default()

	router.GET("/", getHandleDefaultRoute)
	router.GET("/weather", getWeatherLocal)
	router.GET("/weather/:location", getWeatherInternational)

	router.GET("/weather/stress0", getWeatherStressTest0)
	router.GET("/weather/stress1", getWeatherStressTest1)
	router.GET("/weather/stress2", getWeatherStressTest2)
	router.GET("/weather/stress3", getWeatherStressTest3)

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
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")



}
