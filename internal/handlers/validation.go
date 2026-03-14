package handlers

import (
	"fmt"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
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
	sanitizeFamily(family)

	var errors []ValidationError

	if family.UID != "" {
		errors = append(errors, ValidationError{Field: "uid", Message: "UID is auto-generated"})
	}
	if family.CreatedByUserID != "" {
		errors = append(errors, ValidationError{Field: "created_by_user_id", Message: "CreatedByUserID is auto-generated"})
	}
	if family.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if len(family.Name) > domain.MaxNameLength {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	return errors
}

// ValidateTaskCreate validates task creation request
func (h *APIHandler) ValidateTaskCreate(task *models.Tasks) []ValidationError {
	sanitizeTask(task)

	var errors []ValidationError

	if task.FamilyUID == "" {
		errors = append(errors, ValidationError{Field: "family_uid", Message: ErrFamilyUIDRequired})
	}
	if task.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if task.Tokens <= 0 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens must be greater than 0"})
	}
	if task.Tokens > domain.MaxTokens {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens too high (max 1000)"})
	}
	if len(task.Name) > domain.MaxNameLength {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}
	if len(task.Description) > domain.MaxDescriptionLength {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// ValidateRewardCreate validates reward creation request
func (h *APIHandler) ValidateRewardCreate(reward *models.Rewards) []ValidationError {
	sanitizeReward(reward)

	var errors []ValidationError

	if reward.FamilyUID == "" {
		errors = append(errors, ValidationError{Field: "family_uid", Message: ErrFamilyUIDRequired})
	}
	if reward.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if reward.Tokens <= 0 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens must be greater than 0"})
	}
	if reward.Tokens > domain.MaxTokens {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens too high (max 1000)"})
	}
	if len(reward.Name) > domain.MaxNameLength {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}
	if len(reward.Description) > domain.MaxDescriptionLength {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// ValidateTokenAddRequest validates token add/subtract request
func (h *APIHandler) ValidateTokenAddRequest(req *TokenAddRequest) []ValidationError {
	sanitizeTokenAddRequest(req)

	var errors []ValidationError

	if req.Amount == 0 {
		errors = append(errors, ValidationError{Field: "amount", Message: "Amount cannot be zero"})
	}
	if req.Type == "" {
		errors = append(errors, ValidationError{Field: "type", Message: "Type is required"})
	}
	if req.Type != "" && !h.validateTokenType(req.Type) {
		errors = append(errors, ValidationError{Field: "type", Message: ErrInvalidTokenType})
	}
	if req.Amount < -domain.MaxTokens || req.Amount > domain.MaxTokens {
		errors = append(errors, ValidationError{Field: "amount", Message: "Amount must be between -1000 and 1000"})
	}
	if len(req.Description) > domain.MaxDescriptionLength {
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
