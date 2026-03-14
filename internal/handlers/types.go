package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// APIHandler contains the dependencies for API handlers
type APIHandler struct {
	db       *gorm.DB
	logger   *log.Logger
	services *domain.Services
}

// NewAPIHandler creates a new API handler with dependencies
func NewAPIHandler(database *gorm.DB, logger *log.Logger) *APIHandler {
	services, err := domain.NewServices(database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize services with Casbin", "error", err)
	}

	return &APIHandler{
		db:       database,
		logger:   logger,
		services: services,
	}
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Data    any `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// TokenAddRequest represents request for adding/subtracting tokens
type TokenAddRequest struct {
	Amount      int    `json:"amount"`
	Type        string `json:"type"`
	Description string `json:"description"`
	TaskID      *uint  `json:"task_id,omitempty"`
	RewardID    *uint  `json:"reward_id,omitempty"`
}



// ValidationError represents validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// EntityType represents the type of entity (tasks or rewards)
type EntityType string

const (
	EntityTypeTasks   EntityType = "tasks"
	EntityTypeRewards EntityType = "rewards"
)

// respondJSON sends a JSON response
func (h *APIHandler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// respondError sends a standardized error response
func (h *APIHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{Error: message})
}

// respondSuccess sends a standardized success response
func (h *APIHandler) respondSuccess(w http.ResponseWriter, status int, data any) {
	h.respondJSON(w, status, SuccessResponse{Data: data})
}

// decodeJSON decodes JSON request body
func (h *APIHandler) decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// validateFamily checks if a family exists
func (h *APIHandler) validateFamily(ctx context.Context, familyUID string) (*models.Families, error) {
	var family models.Families
	if err := h.db.WithContext(ctx).Where("uid = ?", familyUID).First(&family).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, err
	}
	return &family, nil
}

// validateUser checks if a user exists
func (h *APIHandler) validateUser(ctx context.Context, userID string) (*models.Users, error) {
	var user models.Users
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, err
	}
	return &user, nil
}



// validateRole checks if role is valid
func (h *APIHandler) validateRole(role string) bool {
	return role == RoleParent || role == RoleChild
}

// validateTokenType checks if token operation type is valid
func (h *APIHandler) validateTokenType(tokenType string) bool {
	return tokenType == TokenTypeTaskCompleted ||
		tokenType == TokenTypeRewardClaimed ||
		tokenType == TokenTypeManualAdjustment
}



// validatePlatform checks if platform is valid
func (h *APIHandler) validatePlatform(platform string) bool {
	return platform == PlatformTelegram ||
		platform == PlatformWeb ||
		platform == PlatformMobile
}