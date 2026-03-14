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

// Business logic errors
var (
	ErrUnauthorized          = errors.New("unauthorized operation")
	ErrInsufficientTokens    = errors.New("insufficient tokens")
	ErrTaskAlreadyCompleted  = errors.New("task is already completed")
	ErrTaskInvalidForReview  = errors.New("task must be in 'new' status to submit for review")
	ErrTaskInvalidForApprove = errors.New("task must be in 'new' or 'check' status to complete")
	ErrNoAssignedChild       = errors.New("task has no assigned child")
)

// AuthContext represents the authentication context of who is making the request
type AuthContext struct {
	UserID    string
	UserRole  UserRole
	FamilyUID string
}

// Update DTOs - these define exactly which fields can be updated
type UpdateFamilyRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type UpdateUserRequest struct {
	Name string `json:"name" validate:"omitempty,min=1,max=100"`
	Role string `json:"role" validate:"omitempty,oneof=parent child"`
}

type UpdateTaskRequest struct {
	Name        string `json:"name"        validate:"omitempty,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Tokens      *int   `json:"tokens"      validate:"omitempty,min=0,max=1000"`
	Status      string `json:"status"      validate:"omitempty,oneof=new check completed"`
}

type UpdateRewardRequest struct {
	Name        string `json:"name"        validate:"omitempty,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Tokens      *int   `json:"tokens"      validate:"omitempty,min=0,max=1000"`
}
