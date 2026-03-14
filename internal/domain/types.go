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

// Common field validators

func validateName(name string, required bool) error {
	if required && name == "" {
		return validationErr("name is required")
	}
	if len(name) > MaxNameLength {
		return validationErr("name too long (max 100 characters)")
	}
	return nil
}

func validateDescription(desc string) error {
	if len(desc) > MaxDescriptionLength {
		return validationErr("description too long (max 500 characters)")
	}
	return nil
}

func validateTokensRequired(tokens int) error {
	if tokens <= 0 {
		return validationErr("tokens must be greater than 0")
	}
	if tokens > MaxTokens {
		return validationErr("tokens too high (max 1000)")
	}
	return nil
}

func validateTokensOptional(tokens *int) error {
	if tokens != nil && (*tokens < 0 || *tokens > MaxTokens) {
		return validationErr("tokens must be between 0 and 1000")
	}
	return nil
}

func validateRole(role string) error {
	if role != "" && role != string(RoleParent) && role != string(RoleChild) {
		return validationErr("invalid role: parent or child only")
	}
	return nil
}

func validateStatus(status string) error {
	if status != "" && status != TaskStatusNew && status != TaskStatusCheck && status != TaskStatusCompleted {
		return validationErr("invalid status: new, check, or completed only")
	}
	return nil
}

// ValidateFamilyCreate validates a family creation request.
func ValidateFamilyCreate(name, uid, createdByUserID string) error {
	if uid != "" {
		return validationErr("uid is auto-generated")
	}
	if createdByUserID != "" {
		return validationErr("created_by_user_id is auto-generated")
	}
	return validateName(name, true)
}

// ValidateEntityCreate validates shared entity fields for task/reward creation.
func ValidateEntityCreate(name, description string, tokens int) error {
	if err := validateName(name, true); err != nil {
		return err
	}
	if err := validateDescription(description); err != nil {
		return err
	}
	return validateTokensRequired(tokens)
}

// ValidateTokenTransaction validates a token add/subtract request.
func ValidateTokenTransaction(amount int, tokenType, description string) error {
	if amount == 0 {
		return validationErr("amount cannot be zero")
	}
	if amount < -MaxTokens || amount > MaxTokens {
		return validationErr("amount must be between -1000 and 1000")
	}
	if tokenType == "" {
		return validationErr("type is required")
	}
	if tokenType != TokenTypeTaskCompleted && tokenType != TokenTypeRewardClaimed && tokenType != TokenTypeManualAdjustment {
		return validationErr("invalid token type")
	}
	return validateDescription(description)
}

// Update DTOs - these define exactly which fields can be updated
type UpdateFamilyRequest struct {
	Name string `json:"name"`
}

func (r *UpdateFamilyRequest) Validate() error {
	return validateName(r.Name, true)
}

type UpdateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (r *UpdateUserRequest) Validate() error {
	if err := validateName(r.Name, false); err != nil {
		return err
	}
	return validateRole(r.Role)
}

type UpdateTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tokens      *int   `json:"tokens"`
	Status      string `json:"status"`
}

func (r *UpdateTaskRequest) Validate() error {
	if err := validateName(r.Name, false); err != nil {
		return err
	}
	if err := validateDescription(r.Description); err != nil {
		return err
	}
	if err := validateTokensOptional(r.Tokens); err != nil {
		return err
	}
	return validateStatus(r.Status)
}

type UpdateRewardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tokens      *int   `json:"tokens"`
}

func (r *UpdateRewardRequest) Validate() error {
	if err := validateName(r.Name, false); err != nil {
		return err
	}
	if err := validateDescription(r.Description); err != nil {
		return err
	}
	return validateTokensOptional(r.Tokens)
}
