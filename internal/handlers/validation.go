package handlers

import (
	"fmt"
	"strings"

	"github.com/rgeraskin/joytime/internal/models"
)

// sanitizeInput trims whitespace from user input
func sanitizeInput(input string) string {
	return strings.TrimSpace(input)
}

func sanitizeFamily(family *models.Families) {
	family.Name = sanitizeInput(family.Name)
	family.UID = sanitizeInput(family.UID)
	family.CreatedByUserID = sanitizeInput(family.CreatedByUserID)
}

func sanitizeTask(task *models.Tasks) {
	task.Name = sanitizeInput(task.Name)
	task.Description = sanitizeInput(task.Description)
	task.FamilyUID = sanitizeInput(task.FamilyUID)
	task.Status = sanitizeInput(task.Status)
}

func sanitizeReward(reward *models.Rewards) {
	reward.Name = sanitizeInput(reward.Name)
	reward.Description = sanitizeInput(reward.Description)
	reward.FamilyUID = sanitizeInput(reward.FamilyUID)
}

func sanitizeTokenAddRequest(req *TokenAddRequest) {
	req.UserID = sanitizeInput(req.UserID)
	req.Type = sanitizeInput(req.Type)
	req.Description = sanitizeInput(req.Description)
}

// ValidateFamilyCreate validates family creation request
func (h *APIHandler) ValidateFamilyCreate(family *models.Families) []ValidationError {
	// Sanitize inputs first
	sanitizeFamily(family)

	var errors []ValidationError

	// Check restricted fields
	if family.UID != "" {
		errors = append(errors, ValidationError{Field: "uid", Message: "UID is auto-generated"})
	}
	if family.CreatedByUserID != "" {
		errors = append(errors, ValidationError{Field: "created_by_user_id", Message: "CreatedByUserID is auto-generated"})
	}

	// Required fields
	if family.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}

	// Validate name length
	if len(family.Name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	return errors
}

// ValidateTaskCreate validates task creation request
func (h *APIHandler) ValidateTaskCreate(task *models.Tasks) []ValidationError {
	// Sanitize inputs first
	sanitizeTask(task)

	var errors []ValidationError

	// Required fields
	if task.FamilyUID == "" {
		errors = append(errors, ValidationError{Field: "family_uid", Message: ErrFamilyUIDRequiredField})
	}
	if task.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if task.Tokens <= 0 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens must be greater than 0"})
	}

	// Validate tokens range
	if task.Tokens > 1000 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens too high (max 1000)"})
	}

	// Validate name length
	if len(task.Name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	// Validate description length
	if len(task.Description) > 500 {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// ValidateRewardCreate validates reward creation request
func (h *APIHandler) ValidateRewardCreate(reward *models.Rewards) []ValidationError {
	// Sanitize inputs first
	sanitizeReward(reward)

	var errors []ValidationError

	// Required fields
	if reward.FamilyUID == "" {
		errors = append(errors, ValidationError{Field: "family_uid", Message: ErrFamilyUIDRequiredField})
	}
	if reward.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if reward.Tokens <= 0 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens must be greater than 0"})
	}

	// Validate tokens range
	if reward.Tokens > 1000 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens too high (max 1000)"})
	}

	// Validate name length
	if len(reward.Name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	// Validate description length
	if len(reward.Description) > 500 {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// ValidateTokenAddRequest validates token add/subtract request
func (h *APIHandler) ValidateTokenAddRequest(req *TokenAddRequest) []ValidationError {
	// Sanitize inputs first
	sanitizeTokenAddRequest(req)

	var errors []ValidationError

	// Amount cannot be zero
	if req.Amount == 0 {
		errors = append(errors, ValidationError{Field: "amount", Message: "Amount cannot be zero"})
	}

	// Type is required
	if req.Type == "" {
		errors = append(errors, ValidationError{Field: "type", Message: "Type is required"})
	}

	// Validate type
	if req.Type != "" && !h.validateTokenType(req.Type) {
		errors = append(errors, ValidationError{Field: "type", Message: ErrInvalidTokenType})
	}

	// Validate amount range
	if req.Amount < -1000 || req.Amount > 1000 {
		errors = append(errors, ValidationError{Field: "amount", Message: "Amount must be between -1000 and 1000"})
	}

	// Validate description length
	if len(req.Description) > 500 {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// FormatValidationErrors formats validation errors into a single string
func FormatValidationErrors(errors []ValidationError) string {
	if len(errors) == 0 {
		return ""
	}

	var messages []string
	for _, err := range errors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}

	return strings.Join(messages, "; ")
}