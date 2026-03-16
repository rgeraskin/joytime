package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	setupTestDB(t)

	t.Run("GET /health returns healthy status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		testHandler.handleHealth(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var data map[string]string
		assertSuccessResponse(t, w, http.StatusOK, &data)
		assert.Equal(t, "healthy", data["status"])
		assert.Equal(t, "joytime-api", data["service"])
	})

	t.Run("Non-GET method returns 405", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		testHandler.handleHealth(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}
