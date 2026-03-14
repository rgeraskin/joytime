package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// handleTasks handles /tasks endpoint using business logic layer
func (h *APIHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, ErrAuthContextNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.listTasks(w, r, authCtx)
		case http.MethodPost:
			h.createTask(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

// handleTasksByFamily handles /tasks/{familyUID} endpoints using business logic
func (h *APIHandler) handleTasksByFamily(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, ErrAuthContextNotFound)
			return
		}

		// Extract familyUID from path
		familyUID := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
		if familyUID == "" {
			h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
			return
		}

		// Split path to check for task name (for individual task operations)
		parts := strings.SplitN(familyUID, "/", 2)
		familyUID = parts[0]

		if len(parts) == 2 && parts[1] != "" {
			// Individual task operation: /tasks/{familyUID}/{taskName}
			taskName, err := url.QueryUnescape(parts[1])
			if err != nil {
				h.respondError(w, http.StatusBadRequest, ErrInvalidEntityEncoding)
				return
			}

			switch r.Method {
			case http.MethodGet:
				h.getTask(w, r, authCtx, familyUID, taskName)
			case http.MethodPut:
				h.updateTask(w, r, authCtx, familyUID, taskName)
			case http.MethodDelete:
				h.deleteTask(w, r, authCtx, familyUID, taskName)
			case http.MethodPost:
				h.completeTask(w, r, authCtx, familyUID, taskName)
			default:
				h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
			}
		} else {
			// Family tasks operation: /tasks/{familyUID}
			h.getFamilyTasks(w, r, authCtx, familyUID)
		}
	})(w, r)
}

// listTasks gets all tasks (admin-level, might not be needed for most users)
func (h *APIHandler) listTasks(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	// Most users should use getFamilyTasks instead
	// This might be used for admin purposes
	h.respondError(w, http.StatusNotImplemented, "Global task listing not implemented - use family-specific endpoint")
}

// createTask creates a new task
func (h *APIHandler) createTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	var task models.Tasks
	if err := h.decodeJSON(r, &task); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if validationErrors := h.ValidateTaskCreate(&task); len(validationErrors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(validationErrors))
		return
	}

	err := h.services.TaskService.CreateTask(r.Context(), authCtx, &task)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Only parents can create tasks")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	h.respondSuccess(w, http.StatusCreated, task)
}

// getFamilyTasks gets all tasks for a family
func (h *APIHandler) getFamilyTasks(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID string) {
	tasks, err := h.services.TaskService.GetTasksForFamily(r.Context(), authCtx, familyUID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	h.respondSuccess(w, http.StatusOK, tasks)
}

// getTask gets a single task by family and name
func (h *APIHandler) getTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	// Get all family tasks and find the specific one
	tasks, err := h.services.TaskService.GetTasksForFamily(r.Context(), authCtx, familyUID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	// Find the specific task
	for _, task := range tasks {
		if task.Name == taskName {
			h.respondSuccess(w, http.StatusOK, task)
			return
		}
	}

	h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
}

// updateTask updates a single task
func (h *APIHandler) updateTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	var updates domain.UpdateTaskRequest
	if err := h.decodeJSON(r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	task, err := h.services.TaskService.UpdateTask(r.Context(), authCtx, familyUID, taskName, &updates)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Only parents can update tasks")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	h.respondSuccess(w, http.StatusOK, task)
}

// deleteTask deletes a single task
func (h *APIHandler) deleteTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	err := h.services.TaskService.DeleteTask(r.Context(), authCtx, familyUID, taskName)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Only parents can delete tasks")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// completeTask marks a task as completed by a child
func (h *APIHandler) completeTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	task, err := h.services.TaskService.CompleteTask(r.Context(), authCtx, familyUID, taskName)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
			return
		}
		if errors.Is(err, domain.ErrTaskAlreadyCompleted) ||
			errors.Is(err, domain.ErrTaskInvalidForReview) ||
			errors.Is(err, domain.ErrTaskInvalidForApprove) ||
			errors.Is(err, domain.ErrNoAssignedChild) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to complete task")
		return
	}

	h.respondSuccess(w, http.StatusOK, task)
}