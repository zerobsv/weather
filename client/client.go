package main

import (
	"encoding/json"
	"log"
	"net/http"
	"io"
	"time"
)

func main() {
	// Send a request to the weather service for today
	log.Println("Sending a request to the weather service for today's weather...")

	client := http.Client{Timeout: time.Duration(1) * time.Second}
	response, err := client.Get("https://localhost:8080/weather")

	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatalf("Error fetching weather data: status code %d", response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var jsonResponse []byte
	err = json.Unmarshal(bodyBytes, &jsonResponse)

	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	log.Println("Today's weather is: ", string(jsonResponse))

}
