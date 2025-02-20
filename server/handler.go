package weather

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// GetWeatherLocal retrieves the current weather data for Bengaluru using the WeatherStack API.
//
// The function sends a GET request to the WeatherStack API with the specified access key and query parameters.
// It handles potential errors during the request and response processing.
// If an error occurs, it logs the error and returns an HTTP 500 status code with an error message in the response body.
// If the request is successful, it decodes the JSON response and returns the weather data in the response body.
//
// Parameters:
// ctx (gin.Context): The Gin context containing request and response objects.
//
// Return: weather data for the current location as a JSON string
// None
func GetWeatherLocal(ctx *gin.Context) {

	var apiKey, err = ParseApiKey()
	if err != nil {
		log.Fatalf("Error reading API key: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read API key"})
		return
	}

	log.Println("Invoked function to get today's weather...")

	client := http.Client{Timeout: time.Duration(2) * time.Second}

	var requestUrl = "http://api.weatherstack.com/current?access_key=" + string(apiKey) + "&query=Bengaluru"
	log.Printf("Making a GET request to %s", requestUrl)

	resp, err := client.Get(requestUrl)

	if err != nil {
		log.Fatalf("Error fetching weather data: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weather data"})
		return
	}

	defer resp.Body.Close()

	ctx.JSON(http.StatusOK, resp.Body)

}

// ParseApiKey reads the API key from a file and returns it.
//
// The function opens the file "./api.key" and reads its contents.
// If the file cannot be opened or read, an error is returned.
//
// Parameters:
// None
//
// Return: the api key as a string
func ParseApiKey() (string, error) {
	// Parse API key from file and return it
	file, err := os.ReadFile("./api.key")
	if err != nil {
		return "", err
	}
	return string(file), nil
}

// HandleDefaultRoute handles the default route of the application.
// It responds with a JSON object containing a message.
//
// Parameters:
// ctx (gin.Context): The Gin context containing request and response objects.
//
// Return:
// None
func GetHandleDefaultRoute(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"message": "the weather is quite sad.",
	})
}
