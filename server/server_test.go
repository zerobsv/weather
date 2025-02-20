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
	GetWeatherLocal(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetWeatherLocalResponseLocation(t *testing.T) {

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	GetWeatherLocal(ctx)

	assert.Equal(t, http.StatusOK, w.Code)

	var data map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &data)
	if err != nil {
		t.Errorf("Error unmarshalling JSON response: %v", err)
	}

	log.Printf("JSON response: %v", data)

	location, ok := data["location"].(map[string]interface{})
	if !ok {
		t.Error("Invalid 'location' data type in response")
		return
	}

	var locationCity string = location["city"].(string)
	assert.Equal(t, "Bengaluru", locationCity)

	var locationCountry string = location["country"].(string)
	assert.Equal(t, "India", locationCountry)
}

// TestGetHandleDefaultRouteResponse tests the HandleDefaultRoute function to ensure it handles the request correctly.
func TestGetHandleDefaultRouteResponse(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	GetHandleDefaultRoute(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
}
