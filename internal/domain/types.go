package domain

import (
	"errors"
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
)

// AuthContext represents the authentication context of who is making the request
type AuthContext struct {
	UserID    string
	UserRole  UserRole
	FamilyUID string
}

// Update DTOs - these define exactly which fields can be updated
type UpdateFamilyRequest struct {
	Name string `json:"name"`
}

type UpdateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type UpdateTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tokens      *int   `json:"tokens"`
	Status      string `json:"status"`
}

type UpdateRewardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tokens      *int   `json:"tokens"`
}
