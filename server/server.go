package weather

import (
	"log"

	"github.com/gin-gonic/gin"
)

func WeatherServer() {

	router := gin.Default()

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "the weather is quite sad.",
		})
	})

	router.GET("/weather", GetWeatherLocal)

	log.Println("Starting gin gonic...")

	log.Fatal(router.RunTLS(":8080", "server.pem", "server.key"))

}
