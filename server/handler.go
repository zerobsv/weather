package weather

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type WeatherData struct {
	Location struct {
		Name      string `json:"name"`
		Country   string `json:"country"`
		Region    string `json:"region"`
		Lat       string `json:"lat"`
		Lon       string `json:"lon"`
		Timezone  string `json:"timezone"`
		Localtime string `json:"localtime"`
		Epoch     int    `json:"epoch"`
		Offset    string `json:"offset"`
	} `json:"location"`
	Current struct {
		ObservationTime     string   `json:"observation_time"`
		Temperature         int      `json:"temperature"`
		WeatherCode         int      `json:"weather_code"`
		WeatherDescriptions []string `json:"weather_descriptions"`
		WindSpeed           int      `json:"wind_speed"`
		WindDegree          int      `json:"wind_degree"`
		WindDirection       string   `json:"wind_dir"`
		Pressure            int      `json:"pressure"`
		Precipitation       int      `json:"precip"`
		Humidity            int      `json:"humidity"`
		Cloudcover          int      `json:"cloudcover"`
		FeelsLike           int      `json:"feelslike"`
		UvIndex             int      `json:"uv_index"`
		Visibility          int      `json:"visibility"`
	} `json:"current"`
}

// sendWeatherStackRequest sends a GET request to the WeatherStack API to fetch the current weather data for a specified location.
//
// Parameters:
// location (string): The international location for which to fetch the weather data.
//
// Return:
// WeatherData: A struct containing the parsed weather data.
// error: An error if any occurred during the request or response processing.
func sendWeatherStackRequest(location string) (WeatherData, error) {
    var apiKey, err = parseApiKey()
    if err != nil {
        return WeatherData{}, fmt.Errorf("could not parse api key %v", err)
    }

    client := http.Client{Timeout: time.Duration(2) * time.Second}

    requestUrl := fmt.Sprintf("http://api.weatherstack.com/current?access_key=%s&query=%s", apiKey, location)

    log.Printf("Making a GET request to %s", requestUrl)

    resp, err := client.Get(requestUrl)

    if resp.StatusCode != http.StatusOK {
        return WeatherData{}, fmt.Errorf("weather API request failed to %s: %v", requestUrl, err)
    }

    log.Printf("response: %v", resp)

    if err != nil {
        return WeatherData{}, fmt.Errorf("failed to fetch weather data: %v", err)
    }

    defer resp.Body.Close()

    var weatherData WeatherData
    err = json.NewDecoder(resp.Body).Decode(&weatherData)
    if err != nil {
        return WeatherData{}, fmt.Errorf("error unmarshalling JSON response: %v", err)
    }

    return weatherData, nil
}

// getWeatherInternational retrieves the current weather data for a specified international location using the WeatherStack API.
//
// The function extracts the location from the request parameters, sends a GET request to the WeatherStack API with the specified access key and query parameters,
// handles potential errors during the request and response processing, and returns the weather data in the response body.
//
// Parameters:
// ctx (gin.Context): The Gin context containing request and response objects. The location is extracted from the "location" parameter.
//
// Return:
// None. The function responds with an HTTP status code and a JSON object containing the weather data for the specified location.
// If an error occurs during the request or response processing, an HTTP 500 status code is returned with an error message in the response body.
func getWeatherInternational(ctx *gin.Context) {

    city := ctx.Param("location")

    log.Printf("city param: %v", city)

    weatherData, err := sendWeatherStackRequest(city)

    if err != nil {
        log.Printf("Error fetching weather data: %v", err)
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weather data"})
        return
    }

    log.Println("Weather data: ", weatherData)

    ctx.JSON(http.StatusOK, gin.H{
        "city":        weatherData.Location.Name,
        "country":     weatherData.Location.Country,
        "temperature": fmt.Sprint(weatherData.Current.Temperature),
        "description": "who really cares anyway?",

    })

}

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
func getWeatherLocal(ctx *gin.Context) {

	city := "Bengaluru"

	log.Printf("city param: %v", city)

	weatherData, err := sendWeatherStackRequest(city)

	if err != nil {
		log.Printf("Error fetching weather data: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weather data"})
		return
	}

	log.Println("Weather data: ", weatherData)

	ctx.JSON(http.StatusOK, gin.H{
		"city":        weatherData.Location.Name,
		"country":     weatherData.Location.Country,
		"temperature": fmt.Sprint(weatherData.Current.Temperature),
		"description": "who really cares anyway?",
	})

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
func parseApiKey() (string, error) {
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
func getHandleDefaultRoute(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"message": "the weather is quite sad.",
	})
}
