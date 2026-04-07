// TESTED: ERRORS FIXED

package weather

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	weatherRequestDuration metric.Float64Histogram
	weatherRequestCounter  metric.Float64Counter
	tracer                 trace.Tracer
)

func initMetrics(m metric.Meter) {
	var err error
	weatherRequestDuration, err = m.Float64Histogram(
		"weather_request_duration_seconds",
		metric.WithDescription("Histogram of response time for weather requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		stdlog.Fatal(err)
	}
	weatherRequestCounter, err = m.Float64Counter(
		"weather_requests_total",
		metric.WithDescription("Total number of weather requests"),
	)
	if err != nil {
		stdlog.Fatal(err)
	}

	// Initialize tracer from global provider
	tracer = otel.Tracer("weather-service")
}

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

// sendWeatherRequest sends a GET request to the WeatherStack API to fetch the current weather data for a specified location.
//
// Parameters:
// location (string): The international location for which to fetch the weather data.
//
// Return:
// WeatherData: A struct containing the parsed weather data.
// error: An error if any occurred during the request or response processing.
func sendWeatherRequest(location string) (WeatherData, error) {
	var apiKey, err = parseApiKey()
	if err != nil {
		return WeatherData{}, fmt.Errorf("could not parse api key %v", err)
	}

	client := http.Client{Timeout: time.Duration(200) * time.Millisecond}

	requestUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s", location, apiKey)

	slogLogger.Info("Making a GET request", "url", requestUrl)

	resp, err := client.Get(requestUrl)

	slogLogger.Info("API response received", "status", resp)

	if err != nil {
		if os.IsTimeout(err) {
			return WeatherData{}, fmt.Errorf("failed to fetch weather data: %v", err)
		}
		return WeatherData{}, fmt.Errorf("failed to fetch weather data: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return WeatherData{}, fmt.Errorf("weather API request failed to %s: %v", requestUrl, err)
	}

	defer resp.Body.Close()

	weatherData := WeatherData{}
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

	slogLogger.Info("Processing city parameter", "city", city)

	weatherData, err := instrumentedSendWeatherRequest(city)

	if err != nil {
		slogLogger.Error("Error fetching weather data", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weather data"})
		return
	}

	slogLogger.Info("Weather data retrieved", "city", weatherData.Name)

	ctx.JSON(http.StatusOK, gin.H{
		"city":        weatherData.Name,
		"country":     weatherData.Sys.Country,
		"temperature": fmt.Sprint(weatherData.Main.Temp),
		// "description": weatherData.Weather[0].Description,
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

	slogLogger.Info("Fetching local weather", "city", city)

	weatherData, err := instrumentedSendWeatherRequest(city)

	if err != nil {
		slogLogger.Error("Error fetching weather data", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weather data"})
		return
	}

	slogLogger.Info("Weather data retrieved", "city", weatherData.Name)

	ctx.JSON(http.StatusOK, gin.H{
		"city":        weatherData.Name,
		"country":     weatherData.Sys.Country,
		"temperature": fmt.Sprint(weatherData.Main.Temp),
		// "description": weatherData.Weather[0].Description,
	})

}

func stressTestHelper0(location string, sq *SharedQueue) error {

	weatherData, err := instrumentedSendWeatherRequest(location)

	if err != nil {
		slogLogger.Info("Pushing empty data due to error", "location", location)
		sq.Push(weatherData)
		slogLogger.Error("Error fetching weather data", "location", location, "error", err)
		return err
	}

	slogLogger.Info("Pushing weather data", "location", location)
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
				slogLogger.Error("Weather fetch failed", "city", city)
			}
		}(city)
	}

	// Barrier: Block until all goroutines are done, then continue, will block on long running goroutines
	wg.Wait()

	var stressResponse []gin.H

	slogLogger.Info("Processing stress test 0 results")
	for _, data := range sq.GetAll() {

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			// "description": data.Weather[0].Description,
		})

		slogLogger.Info("Result", "city", data.Name, "country", data.Sys.Country, "temperature", fmt.Sprint(data.Main.Temp))
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func stressTestHelper1(location string, c chan WeatherData) error {

	weatherData, err := instrumentedSendWeatherRequest(location)

	if err != nil {
		c <- weatherData
		slogLogger.Info("Pushing empty data due to error", "location", location)
		slogLogger.Error("Error fetching weather data", "location", location, "error", err)
		return err
	}

	slogLogger.Info("Pushing weather data", "location", location)
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
	defer close(channel)

	for _, city := range cities {
		go func(city string) {
			err := stressTestHelper1(city, channel)
			if err != nil {
				slogLogger.Error("Weather fetch failed", "city", city)
			}
		}(city)
	}

	var stressResponse []gin.H

	slogLogger.Info("Processing stress test 1 results")
	for i := 0; i < len(cities); i++ {

		// CSP Advanatage: No barrier, all the channel slots are polled for data and all
		// the goroutines which are done are processed immediately and other long running
		// goroutines don't block while fetching the results
		data := <-channel

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			// "description": data.Weather[0].Description,
		})

		slogLogger.Info("Result", "city", data.Name, "country", data.Sys.Country, "temperature", fmt.Sprint(data.Main.Temp))
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func stressTestHelper2(location string, sq *SharedQueue) error {

	weatherData, err := instrumentedSendWeatherRequest(location)

	if err != nil {
		slogLogger.Info("Pushing empty data due to error", "location", location)
		sq.Push(weatherData)
		slogLogger.Error("Error fetching weather data", "location", location, "error", err)
		return err
	}

	slogLogger.Info("Pushing weather data", "location", location)
	sq.Push(weatherData)

	return nil

}

// Barrier till buffer is full, and then drain.
// Excellent work, works at scale!
func getWeatherStressTest2(ctx *gin.Context) {

	// cities := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Sydney", "Berlin", "Moscow", "Cairo", "Rio%20de%20Janeiro", "Miami", "Sao%20Paulo", "Madrid", "Barcelona", "Lisbon", "Vienna", "Buenos%20Aires", "Bangkok", "Singapore", "San%20Francisco", "Shanghai", "Mumbai", "Hong%20Kong"}

	temp := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris"}

	repetitions := 1
	result := make([]string, len(temp)*repetitions)

	for i := 0; i < repetitions; i++ {
		result = append(result, temp...)
	}

	cities := result
	sq := &SharedQueue{}

	for _, city := range cities {
		go func(city string) {
			err := stressTestHelper2(city, sq)
			if err != nil {
				slogLogger.Error("Weather fetch failed", "city", city)
			}
		}(city)
	}

	results := sq.GetAllBlocking(len(cities))

	var stressResponse []gin.H

	slogLogger.Info("Processing stress test 2 results")
	for _, data := range results {

		// description produces a BoundsError which is not in the scope of what I'm trying to do here
		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			// "description": data.Weather[0].Description,
		})

		// slogLogger.Info("City: ", data.Name, " Country: ", data.Sys.Country, " Temperature: ", fmt.Sprint(data.Main.Temp), " Description: ", data.Weather[0].Description)
		slogLogger.Info("Result", "city", data.Name, "country", data.Sys.Country, "temperature", fmt.Sprint(data.Main.Temp))
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func stressTestHelper3(location string, sq *SharedQueue) error {

	weatherData, err := instrumentedSendWeatherRequest(location)

	if err != nil {
		slogLogger.Info("Pushing empty data due to error", "location", location)
		sq.FastPush(weatherData)
		slogLogger.Error("Error fetching weather data", "location", location, "error", err)
		return err
	}

	slogLogger.Info("Pushing weather data", "location", location)
	sq.FastPush(weatherData)

	return nil

}

// Barrier till the first element is present, keep draining the queue while producer is pushing data.
// Excellent work, works at scale!
func getWeatherStressTest3(ctx *gin.Context) {

	// cities := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Sydney", "Berlin", "Moscow", "Cairo", "Rio%20de%20Janeiro", "Miami", "Sao%20Paulo", "Madrid", "Barcelona", "Lisbon", "Vienna", "Buenos%20Aires", "Bangkok", "Singapore", "San%20Francisco", "Shanghai", "Mumbai", "Hong%20Kong"}

	temp := []string{"Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris", "Bengaluru", "New%20York", "Tokyo", "London", "Paris"}

	repetitions := 1
	result := make([]string, len(temp)*repetitions)

	for i := 0; i < repetitions; i++ {
		result = append(result, temp...)
	}

	cities := result

	// cities := []string{"Lisbon", "Vienna", "Tokyo", "London", "Paris"}

	sq := &SharedQueue{notify: true}

	for _, city := range cities {
		go func(city string) {
			err := stressTestHelper3(city, sq)
			if err != nil {
				slogLogger.Error("Weather fetch failed", "city", city)
			}
		}(city)
	}

	channel := make(chan WeatherData, 1)
	defer close(channel)

	// Handle panic for consumer goroutine
	defer func() {
		if err := recover(); err != nil {
			slogLogger.Error("Consumer goroutine panicked", "error", err)
		}
	}()

	go sq.GetAllYielding(len(cities), channel)

	var stressResponse []gin.H

	slogLogger.Info("Processing stress test 3 results")
	for i := 0; i < len(cities); i++ {

		slogLogger.Debug("Queue iteration", "iteration", i, "queueSize", len(sq.data))

		data := <-channel

		stressResponse = append(stressResponse, gin.H{
			"city":        data.Name,
			"country":     data.Sys.Country,
			"temperature": fmt.Sprint(data.Main.Temp),
			// "description": fmt.Sprint(data.Weather[0].Description),
		})

		slogLogger.Info("Result", "city", data.Name, "country", data.Sys.Country, "temperature", fmt.Sprint(data.Main.Temp))

		slogLogger.Debug("Queue post-iteration", "iteration", i, "queueSize", len(sq.data))
	}

	ctx.JSON(http.StatusOK, stressResponse)

}

func instrumentedSendWeatherRequest(location string) (WeatherData, error) {
	ctx, span := tracer.Start(context.Background(), "sendWeatherRequest")
	defer span.End()

	span.SetAttributes(
		attribute.String("location", location),
	)

	start := time.Now()
	weatherRequestCounter.Add(ctx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("sendWeatherRequest")))
	data, err := sendWeatherRequest(location)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(ctx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("sendWeatherRequest")))

	if err != nil {
		span.RecordError(err)
	}

	return data, err
}

func instrumentedGetWeatherInternational(ctx *gin.Context) {
	traceCtx, span := tracer.Start(ctx.Request.Context(), "getWeatherInternational")
	defer span.End()

	location := ctx.Param("location")
	span.SetAttributes(
		attribute.String("location", location),
		attribute.String("method", ctx.Request.Method),
		attribute.String("path", ctx.Request.URL.Path),
	)

	start := time.Now()
	weatherRequestCounter.Add(traceCtx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherInternational")))
	getWeatherInternational(ctx)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(traceCtx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherInternational")))

	span.SetAttributes(attribute.Int("http.status_code", ctx.Writer.Status()))
}

func instrumentedGetWeatherLocal(ctx *gin.Context) {
	traceCtx, span := tracer.Start(ctx.Request.Context(), "getWeatherLocal")
	defer span.End()

	span.SetAttributes(
		attribute.String("location", "Bengaluru"),
		attribute.String("method", ctx.Request.Method),
		attribute.String("path", ctx.Request.URL.Path),
	)

	start := time.Now()
	weatherRequestCounter.Add(traceCtx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherLocal")))
	getWeatherLocal(ctx)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(traceCtx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherLocal")))

	span.SetAttributes(attribute.Int("http.status_code", ctx.Writer.Status()))
}

func instrumentedGetWeatherStressTest0(ctx *gin.Context) {
	traceCtx, span := tracer.Start(ctx.Request.Context(), "getWeatherStressTest0")
	defer span.End()

	span.SetAttributes(
		attribute.String("method", ctx.Request.Method),
		attribute.String("path", ctx.Request.URL.Path),
	)

	start := time.Now()
	weatherRequestCounter.Add(traceCtx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest0")))
	getWeatherStressTest0(ctx)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(traceCtx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest0")))

	span.SetAttributes(attribute.Int("http.status_code", ctx.Writer.Status()))
}

func instrumentedGetWeatherStressTest1(ctx *gin.Context) {
	traceCtx, span := tracer.Start(ctx.Request.Context(), "getWeatherStressTest1")
	defer span.End()

	span.SetAttributes(
		attribute.String("method", ctx.Request.Method),
		attribute.String("path", ctx.Request.URL.Path),
	)

	start := time.Now()
	weatherRequestCounter.Add(traceCtx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest1")))
	getWeatherStressTest1(ctx)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(traceCtx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest1")))

	span.SetAttributes(attribute.Int("http.status_code", ctx.Writer.Status()))
}

func instrumentedGetWeatherStressTest2(ctx *gin.Context) {
	traceCtx, span := tracer.Start(ctx.Request.Context(), "getWeatherStressTest2")
	defer span.End()

	span.SetAttributes(
		attribute.String("method", ctx.Request.Method),
		attribute.String("path", ctx.Request.URL.Path),
	)

	start := time.Now()
	weatherRequestCounter.Add(traceCtx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest2")))
	getWeatherStressTest2(ctx)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(traceCtx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest2")))

	span.SetAttributes(attribute.Int("http.status_code", ctx.Writer.Status()))
}

func instrumentedGetWeatherStressTest3(ctx *gin.Context) {
	traceCtx, span := tracer.Start(ctx.Request.Context(), "getWeatherStressTest3")
	defer span.End()

	span.SetAttributes(
		attribute.String("method", ctx.Request.Method),
		attribute.String("path", ctx.Request.URL.Path),
	)

	start := time.Now()
	weatherRequestCounter.Add(traceCtx, 1,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest3")))
	getWeatherStressTest3(ctx)
	duration := time.Since(start).Seconds()
	weatherRequestDuration.Record(traceCtx, duration,
		metric.WithAttributes(attribute.Key("endpoint").String("getWeatherStressTest3")))

	span.SetAttributes(attribute.Int("http.status_code", ctx.Writer.Status()))
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

/*
ERRORS FOR ADVANCED YIELDING MAP REDUCE:

CI Failed: Client Timeout issues causing consumer to suspend execution.....
2025/04/13 15:52:26 Error fetching weather data for : failed to fetch weather data: Get "https://api.openweathermap.org/data/2.5/weather?q=&appid=7c8c4670fac07e8aa7c50d45c295bf3a": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2025/04/13 15:52:26 Weather fetch failed for city:
2025/04/13 15:52:26 Error fetching weather data for New%20York: failed to fetch weather data: Get "https://api.openweathermap.org/data/2.5/weather?q=New%20York&appid=7c8c4670fac07e8aa7c50d45c295bf3a": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2025/04/13 15:52:26 Weather fetch failed for city: New%20York
2025/04/13 15:52:26 Error fetching weather data for Tokyo: failed to fetch weather data: Get "https://api.openweathermap.org/data/2.5/weather?q=Tokyo&appid=7c8c4670fac07e8aa7c50d45c295bf3a": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2025/04/13 15:52:26 Weather fetch failed for city: Tokyo
2025/04/13 15:52:26 $$$$$$$$$$$$ ITER 7 $$$$$$$$$$$$$$$$$$$ QUEUE CONTENTS POST: [{{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0} {{0 0} {0 0 0

2025/04/12 12:51:01 response: <nil>
2025/04/12 12:51:01 pushing data:  {{0 0} {0 0 0  0 0}  [] {0 0 0 0 0 0 0 0} 0 {0 0} {0} {0 0} {0 0} 0 0  0 0}
2025/04/12 12:51:01 Error fetching weather data for Tokyo: failed to fetch weather data: Get "https://api.openweathermap.org/data/2.5/weather?q=Tokyo&appid=7c8c4670fac07e8aa7c50d45c295bf3a": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2025/04/12 12:51:01 Weather fetch failed for city: Tokyo


2025/04/12 12:51:01 [Recovery] 2025/04/12 - 12:51:01 panic recovered:
GET /weather/stress3 HTTP/2.0
Host: localhost:8080
User-Agent: curl/8.5.0


runtime error: index out of range [0] with length 0
/usr/lib/go-1.22/src/runtime/panic.go:114 (0x43809b)
        goPanicIndex: panic(boundsError{x: int64(x), signed: true, y: y, code: boundsIndex})
/mnt/c/Users/munis/Desktop/github_stuff/weather/server/handler.go:631 (0x7ac8f7)
        getWeatherStressTest3: "description": data.Weather[0].Description,
/home/neobsv/go/pkg/mod/github.com/gin-gonic/gin@v1.10.0/context.go:185 (0x7a1199)
        (*Context).Next: c.handlers[c.index](c)

SOLUTION: Increase timeout to 5 seconds, API side error, channel buffer increased

*/
