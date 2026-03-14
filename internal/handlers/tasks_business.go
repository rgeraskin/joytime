package handlers

import (
	"net/http"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
)

// handleTasks handles /tasks endpoint using business logic layer
func (h *APIHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
		switch r.Method {
		case http.MethodPost:
			h.createTask(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

// handleTasksByFamily handles /tasks/{familyUID} endpoints using business logic
func (h *APIHandler) handleTasksByFamily(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
		familyUID, taskName, err := parseFamilyEntityPath(r.URL.Path, "/api/v1/tasks/")
		if err != nil {
			h.respondError(w, http.StatusBadRequest, ErrInvalidEntityEncoding)
			return
		}
		if familyUID == "" {
			h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
			return
		}

		if taskName != "" {
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
			h.getFamilyTasks(w, r, authCtx, familyUID)
		}
	})(w, r)
}

// createTask creates a new task
func (h *APIHandler) createTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	var task models.Tasks
	if err := h.decodeJSON(w, r, &task); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if err := validateTaskCreate(&task); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.services.TaskService.CreateTask(r.Context(), authCtx, &task)
	if err != nil {
		h.respondServiceError(w, err, "failed to create task")
		return
	}

	h.respondSuccess(w, http.StatusCreated, task)
}

// getFamilyTasks gets all tasks for a family
func (h *APIHandler) getFamilyTasks(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID string) {
	tasks, err := h.services.TaskService.GetTasksForFamily(r.Context(), authCtx, familyUID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve tasks")
		return
	}

	h.respondSuccess(w, http.StatusOK, tasks)
}

// getTask gets a single task by family and name
func (h *APIHandler) getTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	task, err := h.services.TaskService.GetTask(r.Context(), authCtx, familyUID, taskName)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve task")
		return
	}

	h.respondSuccess(w, http.StatusOK, task)
}

// updateTask updates a single task
func (h *APIHandler) updateTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	var updates domain.UpdateTaskRequest
	if err := h.decodeJSON(w, r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	task, err := h.services.TaskService.UpdateTask(r.Context(), authCtx, familyUID, taskName, &updates)
	if err != nil {
		h.respondServiceError(w, err, "failed to update task")
		return
	}

	h.respondSuccess(w, http.StatusOK, task)
}

// deleteTask deletes a single task
func (h *APIHandler) deleteTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	err := h.services.TaskService.DeleteTask(r.Context(), authCtx, familyUID, taskName)
	if err != nil {
		h.respondServiceError(w, err, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// completeTask marks a task as completed by a child
func (h *APIHandler) completeTask(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, taskName string) {
	task, err := h.services.TaskService.CompleteTask(r.Context(), authCtx, familyUID, taskName)
	if err != nil {
		h.respondServiceError(w, err, "failed to complete task")
		return
	}

	h.respondSuccess(w, http.StatusOK, task)
}
