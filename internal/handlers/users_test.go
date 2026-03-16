package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestUserHTTPHandlers(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	t.Run("GET /users lists family users", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUsers(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var users []models.Users
		assertSuccessResponse(t, w, http.StatusOK, &users)
		assert.Len(t, users, 2) // parent + child
	})

	t.Run("GET /users/{id} gets a specific user", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/users/%s", child.UserID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUser(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var user models.Users
		assertSuccessResponse(t, w, http.StatusOK, &user)
		assert.Equal(t, child.UserID, user.UserID)
	})

	t.Run("PUT /users/{id} updates a user", func(t *testing.T) {
		body := map[string]any{
			"name": "Renamed Child",
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT",
			fmt.Sprintf("/api/v1/users/%s", child.UserID),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUser(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var user models.Users
		assertSuccessResponse(t, w, http.StatusOK, &user)
		assert.Equal(t, "Renamed Child", user.Name)
	})

	t.Run("DELETE /users/{id} deletes a user", func(t *testing.T) {
		req := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/v1/users/%s", child.UserID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUser(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify user is gone
		reqGet := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/users/%s", child.UserID), nil)
		reqGet.Header.Set("X-User-ID", parent.UserID)
		wGet := httptest.NewRecorder()
		testHandler.handleUser(wGet, reqGet)
		assert.Equal(t, http.StatusNotFound, wGet.Code)
	})

	t.Run("Missing user ID returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/", nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUser(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Method not allowed on /users", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/users", nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUsers(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()

		testHandler.handleUsers(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Child cannot list all users", func(t *testing.T) {
		// Re-create child since we deleted it above
		newChild := &models.Users{
			UserID:    fmt.Sprintf("child_reread_%s_%d", t.Name(), time.Now().UnixNano()),
			Name:      "New Child",
			Role:      "child",
			FamilyUID: family.UID,
			Platform:  "telegram",
		}
		err := testDB.Create(newChild).Error
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		req.Header.Set("X-User-ID", newChild.UserID)
		w := httptest.NewRecorder()

		testHandler.handleUsers(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
