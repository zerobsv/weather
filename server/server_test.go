package weather

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestParseAPIKey tests the ParseApiKey function to ensure it correctly retrieves the API key.
//
// Assert:
//
//	True - Passes the test, file parsed successfully
func TestParseAPIKey(t *testing.T) {
	key, err := ParseApiKey()
	assert.Nil(t, err)
	assert.NotEmpty(t, key)
}

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

// TestGetHandleDefaultRouteResponse tests the HandleDefaultRoute function to ensure it handles the request correctly.
func TestGetHandleDefaultRouteResponse(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	GetHandleDefaultRoute(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
}
