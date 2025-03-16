package weather

import (
	"log"

	"github.com/gin-gonic/gin"
)

func WeatherServer() {

	router := gin.Default()

	router.GET("/", getHandleDefaultRoute)
	router.GET("/weather", getWeatherLocal)
	router.GET("/weather/:location", getWeatherInternational)

	router.GET("/weather/stress", getWeatherStressTest)

	log.Println("Starting gin gonic...")

	log.Fatal(router.RunTLS(":8080", "server.pem", "server.key"))

}
