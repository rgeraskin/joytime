package api

import (
	"fmt"
	"html"
	"strings"

	"github.com/rgeraskin/joytime/internal/postgres"
)

// sanitizeInput sanitizes user input by escaping HTML and trimming whitespace
func sanitizeInput(input string) string {
	return html.EscapeString(strings.TrimSpace(input))
}

// sanitizeInputs sanitizes all string fields in the provided data
func sanitizeUser(user *postgres.Users) {
	user.Name = sanitizeInput(user.Name)
	user.UserID = sanitizeInput(user.UserID)
	user.FamilyUID = sanitizeInput(user.FamilyUID)
	user.Platform = sanitizeInput(user.Platform)
	user.Role = sanitizeInput(user.Role)
	user.InputState = sanitizeInput(user.InputState)
	user.InputContext = sanitizeInput(user.InputContext)
}

func sanitizeFamily(family *postgres.Families) {
	family.Name = sanitizeInput(family.Name)
	family.UID = sanitizeInput(family.UID)
	family.CreatedByUserID = sanitizeInput(family.CreatedByUserID)
}

func sanitizeTask(task *postgres.Tasks) {
	task.Name = sanitizeInput(task.Name)
	task.Description = sanitizeInput(task.Description)
	task.FamilyUID = sanitizeInput(task.FamilyUID)
	task.Status = sanitizeInput(task.Status)
}

func sanitizeReward(reward *postgres.Rewards) {
	reward.Name = sanitizeInput(reward.Name)
	reward.Description = sanitizeInput(reward.Description)
	reward.FamilyUID = sanitizeInput(reward.FamilyUID)
}

func sanitizeEntity(entity *postgres.Entities) {
	entity.Name = sanitizeInput(entity.Name)
	entity.Description = sanitizeInput(entity.Description)
	entity.FamilyUID = sanitizeInput(entity.FamilyUID)
}

func sanitizeTokenAddRequest(req *TokenAddRequest) {
	req.Type = sanitizeInput(req.Type)
	req.Description = sanitizeInput(req.Description)
}

// ValidateUserCreate validates user creation request
func (h *APIHandler) ValidateUserCreate(user *postgres.Users) []ValidationError {
	// Sanitize inputs first
	sanitizeUser(user)

	var errors []ValidationError

	// Required fields
	if user.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if user.FamilyUID == "" {
		errors = append(errors, ValidationError{Field: "family_uid", Message: ErrFamilyUIDRequiredField})
	}
	if user.UserID == "" {
		errors = append(errors, ValidationError{Field: "user_id", Message: ErrUserIDRequiredField})
	}
	if user.Role == "" {
		errors = append(errors, ValidationError{Field: "role", Message: ErrRoleRequired})
	}

	// Validate role
	if user.Role != "" && !h.validateRole(user.Role) {
		errors = append(errors, ValidationError{Field: "role", Message: ErrInvalidRole})
	}

	// Validate platform if provided
	if user.Platform != "" && !h.validatePlatform(user.Platform) {
		errors = append(errors, ValidationError{Field: "platform", Message: ErrInvalidPlatform})
	}

	// Validate name length
	if len(user.Name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	// Validate UserID format (basic validation)
	if user.UserID != "" && len(user.UserID) < 3 {
		errors = append(errors, ValidationError{Field: "user_id", Message: "UserID too short (min 3 characters)"})
	}

	return errors
}

// ValidateUserUpdate validates user update request
func (h *APIHandler) ValidateUserUpdate(user *postgres.Users) []ValidationError {
	// Sanitize inputs first
	sanitizeUser(user)

	var errors []ValidationError

	// Validate role if provided
	if user.Role != "" && !h.validateRole(user.Role) {
		errors = append(errors, ValidationError{Field: "role", Message: ErrInvalidRole})
	}

	// Validate platform if provided
	if user.Platform != "" && !h.validatePlatform(user.Platform) {
		errors = append(errors, ValidationError{Field: "platform", Message: ErrInvalidPlatform})
	}

	// Validate name length if provided
	if user.Name != "" && len(user.Name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	return errors
}

// ValidateFamilyCreate validates family creation request
func (h *APIHandler) ValidateFamilyCreate(family *postgres.Families) []ValidationError {
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

// ValidateFamilyUpdate validates family update request
func (h *APIHandler) ValidateFamilyUpdate(family *postgres.Families) []ValidationError {
	// Sanitize inputs first
	sanitizeFamily(family)

	var errors []ValidationError

	// Check required fields
	if family.Name == "" && family.UID == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameOrUIDRequired})
	}

	// Validate name length if provided
	if family.Name != "" && len(family.Name) > 100 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 100 characters)"})
	}

	return errors
}

// ValidateTaskCreate validates task creation request
func (h *APIHandler) ValidateTaskCreate(task *postgres.Tasks) []ValidationError {
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
	if len(task.Name) > 200 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 200 characters)"})
	}

	// Validate description length
	if len(task.Description) > 500 {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// ValidateRewardCreate validates reward creation request
func (h *APIHandler) ValidateRewardCreate(reward *postgres.Rewards) []ValidationError {
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
	if len(reward.Name) > 200 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 200 characters)"})
	}

	// Validate description length
	if len(reward.Description) > 500 {
		errors = append(errors, ValidationError{Field: "description", Message: "Description too long (max 500 characters)"})
	}

	return errors
}

// ValidateEntityUpdate validates entity update request
func (h *APIHandler) ValidateEntityUpdate(entity *postgres.Entities) []ValidationError {
	// Sanitize inputs first
	sanitizeEntity(entity)

	var errors []ValidationError

	// Required fields
	if entity.FamilyUID == "" {
		errors = append(errors, ValidationError{Field: "family_uid", Message: ErrFamilyUIDRequiredField})
	}
	if entity.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: ErrNameRequired})
	}
	if entity.Tokens == 0 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens is required"})
	}

	// Validate tokens range
	if entity.Tokens < 0 || entity.Tokens > 1000 {
		errors = append(errors, ValidationError{Field: "tokens", Message: "Tokens must be between 1 and 1000"})
	}

	// Validate name length
	if len(entity.Name) > 200 {
		errors = append(errors, ValidationError{Field: "name", Message: "Name too long (max 200 characters)"})
	}

	// Validate description length
	if len(entity.Description) > 500 {
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