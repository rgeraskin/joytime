package domain

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// TaskService handles task-related business logic
type TaskService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
}

// NewTaskService creates a new task service
func NewTaskService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService) *TaskService {
	return &TaskService{
		db:     db,
		logger: logger,
		auth:   auth,
	}
}

// CreateTask creates a new task with business rule enforcement
func (s *TaskService) CreateTask(
	ctx context.Context,
	authCtx *AuthContext,
	task *models.Tasks,
) error {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tasks", "create", task.FamilyUID); err != nil {
		return err
	}

	// Set default status if not provided
	if task.Status == "" {
		task.Status = "new"
	}

	return s.db.WithContext(ctx).Create(task).Error
}

// GetTasksForFamily retrieves tasks for a family with role-based filtering
func (s *TaskService) GetTasksForFamily(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID string,
) ([]models.Tasks, error) {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tasks", "read", familyUID); err != nil {
		return nil, err
	}

	var tasks []models.Tasks
	err := s.db.WithContext(ctx).Where("family_uid = ?", familyUID).Find(&tasks).Error
	return tasks, err
}

// CompleteTask marks a task as completed
func (s *TaskService) CompleteTask(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, taskName string,
) (*models.Tasks, error) {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tasks", "complete", familyUID); err != nil {
		return nil, err
	}

	var task models.Tasks
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		First(&task).
		Error
	if err != nil {
		return nil, err
	}

	// Business Rule: Children can complete tasks, parents can mark them as completed
	// Both roles can perform this action, but with different implications

	if authCtx.UserRole == RoleChild {
		// Child marks task as "check" - needs parent verification
		task.Status = "check"
	} else {
		// Parent can directly mark as completed
		task.Status = "completed"
	}

	err = s.db.WithContext(ctx).Save(&task).Error
	return &task, err
}

// DeleteTask deletes a task with business rule enforcement
func (s *TaskService) DeleteTask(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, taskName string,
) error {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tasks", "delete", familyUID); err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		Delete(&models.Tasks{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// UpdateTask updates a task with business rule enforcement
func (s *TaskService) UpdateTask(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, taskName string,
	updates *UpdateTaskRequest,
) (*models.Tasks, error) {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tasks", "update", familyUID); err != nil {
		return nil, err
	}

	var task models.Tasks
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, taskName).
		First(&task).
		Error
	if err != nil {
		return nil, err
	}

	// Build selective update fields - only allow specific fields to be updated
	updateFields := make(UpdateFields)
	allowedFields := []string{}

	// Business rules for which fields can be updated by different roles
	switch authCtx.UserRole {
	case RoleParent:
		// Parents can update all task fields
		updateFields.AddFieldIfNotEmpty("name", updates.Name)
		updateFields.AddFieldIfNotEmpty("description", updates.Description)
		updateFields.AddFieldIfNotEmpty("tokens", updates.Tokens)
		updateFields.AddFieldIfNotEmpty("status", updates.Status)
		allowedFields = []string{"name", "description", "tokens", "status"}
	case RoleChild:
		// Children can only update status (for marking completion)
		updateFields.AddFieldIfNotEmpty("status", updates.Status)
		allowedFields = []string{"status"}
	}

	// Apply updates only if there are fields to update
	if len(updateFields) > 0 {
		err = s.db.WithContext(ctx).
			Model(&task).
			Select(allowedFields).
			Updates(updateFields.ToMap()).
			Error
		if err != nil {
			return nil, err
		}
	}

	return &task, nil
}
