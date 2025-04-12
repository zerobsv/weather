package weather

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Coordinates struct {
	Longitude float64 `json:"lon"`
	Latitude  float64 `json:"lat"`
}

type Weather struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Main struct {
	Temp      float64 `json:"temp"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	FeelsLike float64 `json:"feels_like"`
	Pressure  float64 `json:"pressure"`
	SeaLevel  float64 `json:"sea_level"`
	GrndLevel float64 `json:"grnd_level"`
	Humidity  int     `json:"humidity"`
}

type Wind struct {
	Speed float64 `json:"speed"`
	Deg   float64 `json:"deg"`
}

type Clouds struct {
	All int `json:"all"`
}

type Rain struct {
	OneH   float64 `json:"1h,omitempty"`
	ThreeH float64 `json:"3h,omitempty"`
}

type Snow struct {
	OneH   float64 `json:"1h,omitempty"`
	ThreeH float64 `json:"3h,omitempty"`
}

type Sys struct {
	Type    int     `json:"type"`
	ID      int     `json:"id"`
	Message float64 `json:"message"`
	Country string  `json:"country"`
	Sunrise int     `json:"sunrise"`
	Sunset  int     `json:"sunset"`
}

type WeatherData struct {
	GeoPos     Coordinates `json:"coord"`
	Sys        Sys         `json:"sys"`
	Base       string      `json:"base"`
	Weather    []Weather   `json:"weather"`
	Main       Main        `json:"main"`
	Visibility int         `json:"visibility"`
	Wind       Wind        `json:"wind"`
	Clouds     Clouds      `json:"clouds"`
	Rain       Rain        `json:"rain"`
	Snow       Snow        `json:"snow"`
	Dt         int         `json:"dt"`
	ID         int         `json:"id"`
	Name       string      `json:"name"`
	Cod        int         `json:"cod"`
	Timezone   int         `json:"timezone"`
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

	requestUrl := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s", location, apiKey)

	log.Printf("Making a GET request to %s", requestUrl)

	resp, err := client.Get(requestUrl)

	log.Printf("response: %v", resp)

	if err != nil {
		return WeatherData{}, fmt.Errorf("failed to fetch weather data: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("weather API request failed to %s: %v", requestUrl, err)
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
		"city":        weatherData.Name,
		"country":     weatherData.Sys.Country,
		"temperature": fmt.Sprint(weatherData.Main.Temp),
		"description": weatherData.Weather[0].Description,
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
		"city":        weatherData.Name,
		"country":     weatherData.Sys.Country,
		"temperature": fmt.Sprint(weatherData.Main.Temp),
		"description": weatherData.Weather[0].Description,
	})

}

type SharedQueue struct {
	mutex sync.RWMutex
	data  []WeatherData
}

func (q *SharedQueue) Push(data WeatherData) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.data = append(q.data, data)
}

func (q *SharedQueue) GetAll() []WeatherData {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	results := make([]WeatherData, 0, len(q.data))
	results = append(results, q.data...)

	return results
}

func stressTestHelper0(location string, sq *SharedQueue) error {

	weatherData, err := sendWeatherStackRequest(location)

	if err != nil {
		log.Printf("Error fetching weather data for %s: %v", location, err)
		return err
	}

	sq.Push(weatherData)

	return nil

}

/*
getWeatherStressTest0 performs a stress test by concurrently fetching weather data
for a list of cities. It uses goroutines to handle each city request in parallel,
collects the results in a shared queue, and returns a JSON response with the weather
information for each city.

Parameters:
- ctx: The Gin context used to handle the HTTP request and response.

The function logs the weather data for each city and sends a JSON response with
the city name, country, temperature, and weather description.
*/
func getWeatherStressTest0(ctx *gin.Context) {
	var wg sync.WaitGroup

	cities := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Sydney", "Berlin", "Moscow", "Cairo", "Rio%20de%20Janeiro", "Miami", "Sao%20Paulo", "Madrid", "Barcelona", "Lisbon", "Vienna", "Buenos%20Aires", "Bangkok", "Singapore", "San%20Francisco", "Shanghai", "Mumbai", "Hong%20Kong"}

	// repetitions := 10
	// result := make([]string, len(cities)*repetitions)

	// for i := 0; i < repetitions; i++ {
	// 	result = append(result, cities...)
	// }

	sq := &SharedQueue{}

	for _, city := range cities {
		wg.Add(1)
		go func(city string) {
			defer wg.Done()
			err := stressTestHelper0(city, sq)
			if err != nil {
				log.Printf("Weather fetch failed for city: %s", city)
			}
		}(city)
	}

	// Barrier: Block until all goroutines are done, then continue, will block on long running goroutines
	wg.Wait()

	var stressResponse []gin.H

	log.Println("All the results: ")
	for _, data := range sq.GetAll() {

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			"description": data.Weather[0].Description,
		})

		log.Println("City: ", data.Name, " Country: ", data.Sys.Country, " Temperature: ", fmt.Sprint(data.Main.Temp), " Description: ", data.Weather[0].Description)
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func stressTestHelper1(location string, c chan WeatherData) error {

	weatherData, err := sendWeatherStackRequest(location)

	if err != nil {
		log.Printf("Error fetching weather data for %s: %v", location, err)
		return err
	}

	c <- weatherData

	return nil
}

func getWeatherStressTest1(ctx *gin.Context) {

	cities := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Sydney", "Berlin", "Moscow", "Cairo", "Rio%20de%20Janeiro", "Miami", "Sao%20Paulo", "Madrid", "Barcelona", "Lisbon", "Vienna", "Buenos%20Aires", "Bangkok", "Singapore", "San%20Francisco", "Shanghai", "Mumbai", "Hong%20Kong"}

	// repetitions := 10
	// result := make([]string, len(cities)*repetitions)

	// for i := 0; i < repetitions; i++ {
	// 	result = append(result, cities...)
	// }

	channel := make(chan WeatherData, len(cities))

	for _, city := range cities {
		go func(city string) {
			err := stressTestHelper1(city, channel)
			if err != nil {
				log.Printf("Weather fetch failed for city: %s", city)
			}
		}(city)
	}

	defer close(channel)

	var stressResponse []gin.H

	log.Println("All the results: ")
	for i := 0; i < len(cities); i++ {

		// CSP Advanatage: No barrier, all the channel slots are polled for data and all
		// the goroutines which are done are processed immediately and other long running
		// goroutines don't block while fetching the results
		data := <-channel

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			"description": data.Weather[0].Description,
		})

		log.Println("City: ", data.Name, " Country: ", data.Sys.Country, " Temperature: ", fmt.Sprint(data.Main.Temp), " Description: ", data.Weather[0].Description)
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func (q *SharedQueue) GetLength() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	return len(q.data)
}

func (q *SharedQueue) GetAllBlocking(count int) []WeatherData {
	results := make([]WeatherData, 0, count)

	// Barrier: Wait for queue to be populated
	for q.GetLength() < count {
		time.Sleep(1 * time.Nanosecond)
	}

	q.mutex.RLock()
	defer q.mutex.RUnlock()

	// Collect all the results
	results = append(results, q.data...)

	return results
}

func stressTestHelper2(location string, sq *SharedQueue) error {

	weatherData, err := sendWeatherStackRequest(location)

	if err != nil {
		log.Printf("Error fetching weather data for %s: %v", location, err)
		return err
	}

	sq.Push(weatherData)

	return nil

}

func getWeatherStressTest2(ctx *gin.Context) {

	cities := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Sydney", "Berlin", "Moscow", "Cairo", "Rio%20de%20Janeiro", "Miami", "Sao%20Paulo", "Madrid", "Barcelona", "Lisbon", "Vienna", "Buenos%20Aires", "Bangkok", "Singapore", "San%20Francisco", "Shanghai", "Mumbai", "Hong%20Kong"}

	// repetitions := 10
	// result := make([]string, len(cities)*repetitions)

	// for i := 0; i < repetitions; i++ {
	// 	result = append(result, cities...)
	// }

	sq := &SharedQueue{}

	for _, city := range cities {
		go func(city string) {
			err := stressTestHelper2(city, sq)
			if err != nil {
				log.Printf("Weather fetch failed for city: %s", city)
			}
		}(city)
	}

	results := sq.GetAllBlocking(len(cities))

	var stressResponse []gin.H

	log.Println("All the results: ")
	for _, data := range results {

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			"description": data.Weather[0].Description,
		})

		log.Println("City: ", data.Name, " Country: ", data.Sys.Country, " Temperature: ", fmt.Sprint(data.Main.Temp), " Description: ", data.Weather[0].Description)
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func (q *SharedQueue) Pop() WeatherData {
	// SENSITIVE LOCKING: This read lock has to be done strictly BEFORE.
	// Yield Barrier: Wait for at least one element to be present in the queue
	for q.GetLength() < 1 {
		time.Sleep(1 * time.Nanosecond)
	}

	// PANIC: Two goros have passed this barrier! :O

	// SENSITIVE LOCKING: This write lock has to be done strictly AFTER.
	q.mutex.Lock()
	defer q.mutex.Unlock()

	tmp := q.data[0]
	q.data = q.data[1:]

	return tmp
}

func (q *SharedQueue) GetAllYielding(count int, ch chan WeatherData) {

	// Yield Barrier: Wait for at least one element to be present in the queue
	for count > 0 {
		go func() {
			// Collect the result and pop
			ch <- q.Pop()
		}()
		count--
	}

}

func stressTestHelper3(location string, sq *SharedQueue) error {

	weatherData, err := sendWeatherStackRequest(location)

	if err != nil {
		log.Printf("Error fetching weather data for %s: %v", location, err)
		return err
	}

	sq.Push(weatherData)

	return nil

}

func getWeatherStressTest3(ctx *gin.Context) {

	cities := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Sydney", "Berlin", "Moscow", "Cairo", "Rio%20de%20Janeiro", "Miami", "Sao%20Paulo", "Madrid", "Barcelona", "Lisbon", "Vienna", "Buenos%20Aires", "Bangkok", "Singapore", "San%20Francisco", "Shanghai", "Mumbai", "Hong%20Kong"}

	// repetitions := 10
	// result := make([]string, len(cities)*repetitions)

	// for i := 0; i < repetitions; i++ {
	// 	result = append(result, cities...)
	// }

	sq := &SharedQueue{}

	for _, city := range cities {
		go func(city string) {
			err := stressTestHelper3(city, sq)
			if err != nil {
				log.Printf("Weather fetch failed for city: %s", city)
			}
		}(city)
	}

	channel := make(chan WeatherData)
	defer close(channel)

	sq.GetAllYielding(len(cities), channel)

	var stressResponse []gin.H

	log.Println("All the results: ")
	for i := 0; i < len(cities); i++ {

		data := <-channel

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			"description": data.Weather[0].Description,
		})

		log.Println("City: ", data.Name, " Country: ", data.Sys.Country, " Temperature: ", fmt.Sprint(data.Main.Temp), " Description: ", data.Weather[0].Description)
	}

	ctx.JSON(http.StatusOK, stressResponse)

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
	return strings.TrimSpace(string(file)), nil
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
