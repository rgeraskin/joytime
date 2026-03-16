package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskHTTPHandlers(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	t.Run("POST /tasks creates a task", func(t *testing.T) {
		body := map[string]any{
			"family_uid":  family.UID,
			"name":        "HTTP Test Task",
			"description": "Created via HTTP",
			"tokens":      15,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasks(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var task models.Tasks
		assertSuccessResponse(t, w, http.StatusCreated, &task)
		assert.Equal(t, "HTTP Test Task", task.Name)
		assert.Equal(t, 15, task.Tokens)
	})

	t.Run("GET /tasks/{familyUID} lists family tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tasks/%s", family.UID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasksByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var tasks []models.Tasks
		assertSuccessResponse(t, w, http.StatusOK, &tasks)
		assert.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("GET /tasks/{familyUID}/{taskName} gets single task", func(t *testing.T) {
		encodedName := url.PathEscape("HTTP Test Task")
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/tasks/%s/%s", family.UID, encodedName), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasksByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var task models.Tasks
		assertSuccessResponse(t, w, http.StatusOK, &task)
		assert.Equal(t, "HTTP Test Task", task.Name)
	})

	t.Run("PUT /tasks/{familyUID}/{taskName} updates task", func(t *testing.T) {
		newTokens := 25
		body := map[string]any{
			"tokens": newTokens,
		}
		bodyJSON, _ := json.Marshal(body)
		encodedName := url.PathEscape("HTTP Test Task")

		req := httptest.NewRequest("PUT",
			fmt.Sprintf("/api/v1/tasks/%s/%s", family.UID, encodedName),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasksByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /tasks/{familyUID}/{taskName} completes task (child submits)", func(t *testing.T) {
		encodedName := url.PathEscape("HTTP Test Task")
		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/tasks/%s/%s", family.UID, encodedName),
			bytes.NewReader([]byte("{}")))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasksByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var task models.Tasks
		assertSuccessResponse(t, w, http.StatusOK, &task)
		assert.Equal(t, domain.TaskStatusCheck, task.Status)
	})

	t.Run("DELETE /tasks/{familyUID}/{taskName} deletes task", func(t *testing.T) {
		// Create a task to delete
		parentCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}
		delTask := &models.Tasks{
			Entities: models.Entities{
				FamilyUID: family.UID,
				Name:      "Task To Delete",
				Tokens:    5,
			},
		}
		require.NoError(t, testHandler.services.TaskService.CreateTask(context.Background(), parentCtx, delTask))

		encodedName := url.PathEscape("Task To Delete")
		req := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/v1/tasks/%s/%s", family.UID, encodedName), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasksByFamily(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Child cannot create tasks via HTTP", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
			"name":       "Cheat Task",
			"tokens":     1,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasks(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tasks", nil)
		w := httptest.NewRecorder()

		testHandler.handleTasks(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET nonexistent task returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/tasks/%s/NonexistentTask", family.UID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasksByFamily(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Missing required fields returns bad request", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
			// missing name and tokens
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleTasks(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRejectTaskFlow(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}
	childCtx := &domain.AuthContext{
		UserID:    child.UserID,
		UserRole:  domain.RoleChild,
		FamilyUID: family.UID,
	}

	// Create a task
	task := &models.Tasks{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "Reject Test Task",
			Tokens:    10,
		},
	}
	err := testHandler.services.TaskService.CreateTask(context.Background(), parentCtx, task)
	require.NoError(t, err)

	// Child submits for review
	submitted, err := testHandler.services.TaskService.CompleteTask(
		context.Background(), childCtx, family.UID, "Reject Test Task",
	)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusCheck, submitted.Status)
	assert.Equal(t, child.UserID, submitted.AssignedToUserID)

	t.Run("Parent rejects task", func(t *testing.T) {
		rejected, err := testHandler.services.TaskService.RejectTask(
			context.Background(), parentCtx, family.UID, "Reject Test Task",
		)
		require.NoError(t, err)
		assert.Equal(t, domain.TaskStatusNew, rejected.Status)
		assert.Empty(t, rejected.AssignedToUserID)
	})

	t.Run("Rejecting non-check task fails", func(t *testing.T) {
		// Task is now in "new" status after rejection
		_, err := testHandler.services.TaskService.RejectTask(
			context.Background(), parentCtx, family.UID, "Reject Test Task",
		)
		assert.ErrorIs(t, err, domain.ErrTaskInvalidForReview)
	})
}
