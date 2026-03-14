package domain

import (
	"errors"
	"fmt"
)

// Role-based permissions
type UserRole string

const (
	RoleParent UserRole = "parent"
	RoleChild  UserRole = "child"
)

// Task status constants
const (
	TaskStatusNew       = "new"
	TaskStatusCheck     = "check"
	TaskStatusCompleted = "completed"
)

// Token operation type constants
const (
	TokenTypeTaskCompleted    = "task_completed"
	TokenTypeRewardClaimed    = "reward_claimed"
	TokenTypeManualAdjustment = "manual_adjustment"
)

// Business logic errors
var (
	ErrUnauthorized          = errors.New("unauthorized operation")
	ErrCannotDeleteSelf      = errors.New("cannot delete yourself")
	ErrInsufficientTokens    = errors.New("insufficient tokens")
	ErrTaskAlreadyCompleted  = errors.New("task is already completed")
	ErrTaskInvalidForReview  = errors.New("task must be in 'new' status to submit for review")
	ErrTaskInvalidForApprove = errors.New("task must be in 'new' or 'check' status to complete")
	ErrNoAssignedChild       = errors.New("task has no assigned child")
	ErrTaskNotAssignedToUser = errors.New("task is not assigned to you")
	ErrValidation            = errors.New("validation error")
)

// Validation limits
const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
	MaxTokens            = 1000
)

// AuthContext represents the authentication context of who is making the request
type AuthContext struct {
	UserID    string
	UserRole  UserRole
	FamilyUID string
}

// validationErr returns a validation error with a descriptive message.
func validationErr(msg string) error {
	return fmt.Errorf("%w: %s", ErrValidation, msg)
}

// Update DTOs - these define exactly which fields can be updated
type UpdateFamilyRequest struct {
	Name string `json:"name"`
}

func (r *UpdateFamilyRequest) Validate() error {
	if r.Name == "" {
		return validationErr("family name cannot be empty")
	}
	if len(r.Name) > MaxNameLength {
		return validationErr("name too long (max 100 characters)")
	}
	return nil
}

type UpdateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (r *UpdateUserRequest) Validate() error {
	if len(r.Name) > MaxNameLength {
		return validationErr("name too long (max 100 characters)")
	}
	if r.Role != "" && r.Role != string(RoleParent) && r.Role != string(RoleChild) {
		return validationErr("invalid role: parent or child only")
	}
	return nil
}

type UpdateTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tokens      *int   `json:"tokens"`
	Status      string `json:"status"`
}

func (r *UpdateTaskRequest) Validate() error {
	if len(r.Name) > MaxNameLength {
		return validationErr("name too long (max 100 characters)")
	}
	if len(r.Description) > MaxDescriptionLength {
		return validationErr("description too long (max 500 characters)")
	}
	if r.Tokens != nil && (*r.Tokens < 0 || *r.Tokens > MaxTokens) {
		return validationErr("tokens must be between 0 and 1000")
	}
	if r.Status != "" && r.Status != TaskStatusNew && r.Status != TaskStatusCheck && r.Status != TaskStatusCompleted {
		return validationErr("invalid status: new, check, or completed only")
	}
	return nil
}

type UpdateRewardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tokens      *int   `json:"tokens"`
}

func (r *UpdateRewardRequest) Validate() error {
	if len(r.Name) > MaxNameLength {
		return validationErr("name too long (max 100 characters)")
	}
	if len(r.Description) > MaxDescriptionLength {
		return validationErr("description too long (max 500 characters)")
	}
	if r.Tokens != nil && (*r.Tokens < 0 || *r.Tokens > MaxTokens) {
		return validationErr("tokens must be between 0 and 1000")
	}
	return nil
}
