package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenHTTPHandlers(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	t.Run("POST /tokens/users/{id} with manual_adjustment succeeds", func(t *testing.T) {
		body := map[string]any{
			"amount":      5,
			"type":        "manual_adjustment",
			"description": "bonus",
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/tokens/users/%s", child.UserID),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUserTokens(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /tokens/users/{id} with task_completed is rejected", func(t *testing.T) {
		body := map[string]any{
			"amount":      10,
			"type":        "task_completed",
			"description": "trying to fake task completion",
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/tokens/users/%s", child.UserID),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUserTokens(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /tokens/users/{id} with reward_claimed is rejected", func(t *testing.T) {
		body := map[string]any{
			"amount":      -10,
			"type":        "reward_claimed",
			"description": "trying to fake reward claim",
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/tokens/users/%s", child.UserID),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUserTokens(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET /tokens/users/{id} returns token balance", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/tokens/users/%s", child.UserID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUserTokens(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	_ = family // used in setup
}
