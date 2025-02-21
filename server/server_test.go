package weather

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	getWeatherLocal(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
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
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/weather?location=Bengaluru", nil)

	getWeatherLocal(ctx)

	assert.Equal(t, http.StatusOK, w.Code)

	var data map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

	assert.Equal(t, "Bengaluru", data["city"])
	assert.Equal(t, "India", data["country"])
	assert.NotEmpty(t, data["temperature"])
}

// TestGetHandleDefaultRouteResponse tests the HandleDefaultRoute function to ensure it handles the request correctly.
func TestGetHandleDefaultRouteResponse(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	getHandleDefaultRoute(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
}
