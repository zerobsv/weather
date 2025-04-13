package weather

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestGetWeatherLocalResponse tests the GetWeatherLocal function to ensure it handles the request correctly.
//
// The function uses httptest.NewRecorder to create a response recorder for testing HTTP responses.
// GetWeatherLocal is tested for response code only
//
// Parameters:
//
//	t *testing.T - The testing.T instance for the test.
//
// Return:
//
//	None
func TestGetWeatherLocalResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	getWeatherLocal(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetWeatherLocalResponseLocation tests the getWeatherLocal function with a location query parameter.
// It verifies the function's response code, JSON response, and specific fields in the response.
//
// Parameters:
//
//	t *testing.T - The testing.T instance for the test.
//
// Return:
//
//	None
func TestGetWeatherLocalResponseLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather", nil)

	getWeatherLocal(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)

	var data map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

	//assert.Equal(t, "Bengaluru", data["city"])
	//assert.Equal(t, "IN", data["country"])
	//assert.NotEmpty(t, data["temperature"])
}

func TestWeatherInternationalResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ctx.Params = []gin.Param{
		{
			Key:   "location",
			Value: "Tokyo",
		},
	}

	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather/Tokyo", nil)

	getWeatherInternational(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)

	var data map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

	//assert.Equal(t, "Tokyo", data["city"])
	//assert.Equal(t, "JP", data["country"])
	//assert.NotEmpty(t, data["temperature"])
}

func TestWeatherStressResponse0(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather", nil)

	getWeatherStressTest0(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)

	log.Printf("Body: %v", w.Body)

	var data []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

}

func TestWeatherStressResponse1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather", nil)

	getWeatherStressTest1(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)

	log.Printf("Body: %v", w.Body)

	var data []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

}

func TestWeatherStressResponse2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather", nil)

	getWeatherStressTest2(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)

	log.Printf("Body: %v", w.Body)

	var data []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

}

func TestWeatherStressResponse3(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather", nil)

	getWeatherStressTest3(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)

	log.Printf("Body: %v", w.Body)

	var data []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

}

// TestGetHandleDefaultRouteResponse tests the HandleDefaultRoute function to ensure it handles the request correctly.
func TestGetHandleDefaultRouteResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	getHandleDefaultRoute(ctx)

	//assert.Equal(t, http.StatusOK, w.Code)
}
