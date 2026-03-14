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
	tokens *TokenService
}

// NewTaskService creates a new task service
func NewTaskService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService, tokens *TokenService) *TaskService {
	return &TaskService{
		db:     db,
		logger: logger,
		auth:   auth,
		tokens: tokens,
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
		task.Status = TaskStatusNew
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

// CompleteTask marks a task as completed and awards tokens when parent approves.
// Child submits for review (new → check), parent approves (check/new → completed + tokens).
// Task status update and token award are wrapped in a single transaction.
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

	// Validate status transitions
	if task.Status == TaskStatusCompleted {
		return nil, ErrTaskAlreadyCompleted
	}

	if authCtx.UserRole == RoleChild {
		// Child marks task as "check" — needs parent verification
		if task.Status != TaskStatusNew {
			return nil, ErrTaskInvalidForReview
		}

		// If task is already assigned, only the assigned child can submit
		if task.AssignedToUserID != "" && task.AssignedToUserID != authCtx.UserID {
			return nil, ErrTaskNotAssignedToUser
		}

		task.Status = TaskStatusCheck
		// Record which child submitted for review
		task.AssignedToUserID = authCtx.UserID

		if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
			return nil, err
		}
		return &task, nil
	}

	// Parent approves — mark as completed and award tokens to the assigned child
	if task.Status != TaskStatusCheck && task.Status != TaskStatusNew {
		return nil, ErrTaskInvalidForApprove
	}

	if task.AssignedToUserID == "" {
		return nil, ErrNoAssignedChild
	}

	// Wrap status update + token award in a single transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task.Status = TaskStatusCompleted
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		if task.Tokens > 0 {
			taskID := task.ID
			if err := s.tokens.addTokensInTx(
				tx,
				task.AssignedToUserID,
				task.Tokens,
				"task_completed",
				"Completed task: "+task.Name,
				&taskID,
				nil,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &task, nil
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
		updateFields.AddStringIfNotEmpty("name", updates.Name)
		updateFields.AddStringIfNotEmpty("description", updates.Description)
		updateFields.AddIntIfSet("tokens", updates.Tokens)
		updateFields.AddStringIfNotEmpty("status", updates.Status)
		allowedFields = []string{"name", "description", "tokens", "status"}
	case RoleChild:
		// Children can only update status (for marking completion)
		updateFields.AddStringIfNotEmpty("status", updates.Status)
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
