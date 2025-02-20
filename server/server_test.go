package weather

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAPIKey(t *testing.T) {
	key, err := ParseApiKey()
	assert.Nil(t, err)
	assert.NotEmpty(t, key)
}

func TestGetWeatherLocal(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/weather", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
