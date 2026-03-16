package domain

import (
	"context"

	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TaskService handles task-related business logic
type TaskService struct {
	db     *gorm.DB
	auth   *CasbinAuthService
	tokens *TokenService
}

// NewTaskService creates a new task service
func NewTaskService(
	db *gorm.DB,
	auth *CasbinAuthService,
	tokens *TokenService,
) *TaskService {
	return &TaskService{
		db:     db,
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

	if err := ValidateEntityCreate(task.Name, task.Description, task.Tokens); err != nil {
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
	err := s.db.WithContext(ctx).
		Where("family_uid = ?", familyUID).
		Order("tokens DESC").
		Limit(maxListResults).
		Find(&tasks).
		Error
	return tasks, err
}

// GetTask retrieves a single task by family and name
func (s *TaskService) GetTask(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, taskName string,
) (*models.Tasks, error) {
	if err := s.auth.RequirePermission(authCtx, "tasks", "read", familyUID); err != nil {
		return nil, err
	}

	return findByFamilyAndName[models.Tasks](s.db, ctx, familyUID, taskName)
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

	var task *models.Tasks
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the task row to prevent concurrent status changes
		var t models.Tasks
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("family_uid = ? AND name = ?", familyUID, taskName).
			First(&t).Error; err != nil {
			return err
		}

		// Validate status transitions
		if t.Status == TaskStatusCompleted {
			return ErrTaskAlreadyCompleted
		}

		if authCtx.UserRole == RoleChild {
			// Child marks task as "check" — needs parent verification
			if t.Status != TaskStatusNew {
				return ErrTaskInvalidForReview
			}

			// If task is already assigned, only the assigned child can submit
			if t.AssignedToUserID != "" && t.AssignedToUserID != authCtx.UserID {
				return ErrTaskNotAssignedToUser
			}

			t.Status = TaskStatusCheck
			// Record which child submitted for review
			t.AssignedToUserID = authCtx.UserID

			if err := tx.Save(&t).Error; err != nil {
				return err
			}
			task = &t
			return nil
		}

		// Parent approves — mark as completed and award tokens to the assigned child
		if t.Status != TaskStatusCheck && t.Status != TaskStatusNew {
			return ErrTaskInvalidForApprove
		}

		if t.AssignedToUserID == "" {
			return ErrNoAssignedChild
		}

		// Award tokens, then reset to "new" so the task can be completed again
		if t.Tokens > 0 {
			taskID := t.ID
			if _, err := s.tokens.addTokensInTx(
				tx,
				t.AssignedToUserID,
				t.Tokens,
				TokenTypeTaskCompleted,
				HistoryDescTask+t.Name,
				&taskID,
				nil,
				nil,
			); err != nil {
				return err
			}
		}

		// Reset task to "new" so it's available for the next completion
		t.Status = TaskStatusNew
		t.AssignedToUserID = ""
		if err := tx.Select("status", "assigned_to_user_id").Save(&t).Error; err != nil {
			return err
		}
		task = &t
		return nil
	})
	if err != nil {
		return nil, err
	}

	return task, nil
}

// RejectTask resets a task from "check" back to "new", clearing the assignee.
// Used when a parent rejects a child's completed task.
func (s *TaskService) RejectTask(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, taskName string,
) (*models.Tasks, error) {
	if err := s.auth.RequirePermission(authCtx, "tasks", "complete", familyUID); err != nil {
		return nil, err
	}

	var task *models.Tasks
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var t models.Tasks
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("family_uid = ? AND name = ?", familyUID, taskName).
			First(&t).Error; err != nil {
			return err
		}

		if t.Status != TaskStatusCheck {
			return ErrTaskInvalidForReview
		}

		t.Status = TaskStatusNew
		t.AssignedToUserID = ""
		if err := tx.Select("status", "assigned_to_user_id").Save(&t).Error; err != nil {
			return err
		}
		task = &t
		return nil
	})
	if err != nil {
		return nil, err
	}

	return task, nil
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

	return deleteByFamilyAndName[models.Tasks](s.db, ctx, familyUID, taskName)
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

	if err := updates.Validate(); err != nil {
		return nil, err
	}

	task, err := findByFamilyAndName[models.Tasks](s.db, ctx, familyUID, taskName)
	if err != nil {
		return nil, err
	}

	// Validate status transition if status is being updated
	if updates.Status != "" && task.Status == TaskStatusCompleted {
		return nil, ErrTaskAlreadyCompleted
	}

	// Build selective update fields - only allow specific fields to be updated
	updateFields := make(UpdateFields)

	// Business rules for which fields can be updated by different roles
	switch authCtx.UserRole {
	case RoleParent:
		// Parents can update all task fields
		updateFields.AddStringIfNotEmpty("name", updates.Name)
		updateFields.AddStringIfNotEmpty("description", updates.Description)
		updateFields.AddIntIfSet("tokens", updates.Tokens)
		updateFields.AddStringIfNotEmpty("status", updates.Status)
	case RoleChild:
		// Children can only update status (for marking completion)
		updateFields.AddStringIfNotEmpty("status", updates.Status)
	}

	// Apply updates only if there are fields to update
	if len(updateFields) > 0 {
		if err := updateAndReload(s.db, ctx, task, task.ID, updateFields); err != nil {
			return nil, err
		}
	}

	return task, nil
}
