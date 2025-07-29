package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/rgeraskin/joytime/internal/postgres"
	"gorm.io/gorm"
)

// handleTasks handles /tasks endpoint
func (h *APIHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.listTasks(ctx, w, r)
	case http.MethodPost:
		h.createTask(ctx, w, r)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleTasksByFamily handles /tasks/{familyUID} and /tasks/{familyUID}/{taskName} endpoints
func (h *APIHandler) handleTasksByFamily(w http.ResponseWriter, r *http.Request) {
	// Extract path parts
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
		return
	}

	familyUID := parts[0]
	ctx := context.Background()

	// Check if we have a task name (individual task operation)
	if len(parts) == 2 && parts[1] != "" {
		taskName, err := url.QueryUnescape(parts[1])
		if err != nil {
			h.respondError(w, http.StatusBadRequest, ErrInvalidEntityEncoding)
			return
		}

		// Handle individual task operations
		switch r.Method {
		case http.MethodGet:
			h.getSingleTask(ctx, w, r, familyUID, taskName)
		case http.MethodPut:
			h.updateSingleTask(ctx, w, r, familyUID, taskName)
		case http.MethodDelete:
			h.deleteSingleTask(ctx, w, r, familyUID, taskName)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	} else {
		// Handle family task operations
		switch r.Method {
		case http.MethodGet:
			h.getTasksByFamily(ctx, w, r, familyUID)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	}
}



// listTasks lists all tasks
func (h *APIHandler) listTasks(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	h.logger.Debug("Listing all tasks")

	var tasks []postgres.Tasks
	if err := h.db.WithContext(ctx).Find(&tasks).Error; err != nil {
		h.logger.Error("Failed to list tasks", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	h.respondSuccess(w, http.StatusOK, tasks)
}

// createTask creates a new task
func (h *APIHandler) createTask(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Creating new task")

	var task postgres.Tasks
	if err := h.decodeJSON(r, &task); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateTaskCreate(&task); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if family exists
	_, err := h.validateFamily(ctx, task.FamilyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to create task")
		}
		return
	}

	// Create task
	if err := h.db.WithContext(ctx).Create(&task).Error; err != nil {
		h.logger.Error("Failed to create task", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	h.logger.Debug("Task created successfully", "task_id", task.ID, "name", task.Name)
	h.respondSuccess(w, http.StatusCreated, task)
}

// getTasksByFamily gets all tasks for a family
func (h *APIHandler) getTasksByFamily(ctx context.Context, w http.ResponseWriter, _ *http.Request, familyUID string) {
	h.logger.Debug("Getting tasks by family", "family_uid", familyUID)

	// Check if family exists
	_, err := h.validateFamily(ctx, familyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		}
		return
	}

	var tasks []postgres.Tasks
	if err := h.db.WithContext(ctx).Where("family_uid = ?", familyUID).Find(&tasks).Error; err != nil {
		h.logger.Error("Failed to get tasks by family", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	h.respondSuccess(w, http.StatusOK, tasks)
}

// getSingleTask gets a single task by family and name
func (h *APIHandler) getSingleTask(ctx context.Context, w http.ResponseWriter, _ *http.Request, familyUID, taskName string) {
	h.logger.Debug("Getting single task", "family_uid", familyUID, "task_name", taskName)

	// Check if family exists
	_, err := h.validateFamily(ctx, familyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve task")
		}
		return
	}

	var task postgres.Tasks
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
		} else {
			h.logger.Error("Failed to get single task", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve task")
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, task)
}

// updateSingleTask updates a single task
func (h *APIHandler) updateSingleTask(ctx context.Context, w http.ResponseWriter, r *http.Request, familyUID, taskName string) {
	h.logger.Debug("Updating single task", "family_uid", familyUID, "task_name", taskName)

	var updateData postgres.Entities
	if err := h.decodeJSON(r, &updateData); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateEntityUpdate(&updateData); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if family exists
	_, err := h.validateFamily(ctx, updateData.FamilyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update task")
		}
		return
	}

	// Check if task exists
	var existingTask postgres.Tasks
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		First(&existingTask).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
		} else {
			h.logger.Error("Failed to find task for update", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update task")
		}
		return
	}

	// Update task
	existingTask.Tokens = updateData.Tokens
	existingTask.Description = updateData.Description
	if err := h.db.WithContext(ctx).Save(&existingTask).Error; err != nil {
		h.logger.Error("Failed to update task", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	h.respondSuccess(w, http.StatusOK, existingTask)
}

// deleteSingleTask deletes a single task
func (h *APIHandler) deleteSingleTask(ctx context.Context, w http.ResponseWriter, _ *http.Request, familyUID, taskName string) {
	h.logger.Debug("Deleting single task", "family_uid", familyUID, "task_name", taskName)

	// Check if family exists
	_, err := h.validateFamily(ctx, familyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to delete task")
		}
		return
	}

	// Check if task exists
	var existingTask postgres.Tasks
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		First(&existingTask).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
		} else {
			h.logger.Error("Failed to find task for deletion", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to delete task")
		}
		return
	}

	// Delete task
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		Delete(&postgres.Tasks{}).Error; err != nil {
		h.logger.Error("Failed to delete task", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}