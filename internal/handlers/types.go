package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/domain"
	"gorm.io/gorm"
)

// APIHandler contains the dependencies for API handlers
type APIHandler struct {
	logger   *log.Logger
	services *domain.Services
}

// NewAPIHandler creates a new API handler with dependencies
func NewAPIHandler(services *domain.Services, logger *log.Logger) *APIHandler {
	return &APIHandler{
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
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// TokenAddRequest represents request for adding/subtracting tokens
type TokenAddRequest struct {
	UserID      string `json:"user_id,omitempty"`
	Amount      int    `json:"amount"`
	Type        string `json:"type"`
	Description string `json:"description"`
	TaskID      *uint  `json:"task_id,omitempty"`
	RewardID    *uint  `json:"reward_id,omitempty"`
}

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

// respondServiceError maps domain/database errors to HTTP responses.
func (h *APIHandler) respondServiceError(w http.ResponseWriter, err error, fallbackMsg string) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		h.respondError(w, http.StatusForbidden, "access denied")
	case errors.Is(err, gorm.ErrRecordNotFound):
		h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
	case errors.Is(err, domain.ErrInsufficientTokens):
		h.respondError(w, http.StatusBadRequest, ErrInsufficientTokens)
	case errors.Is(err, domain.ErrValidation):
		h.respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrCannotDeleteSelf):
		h.respondError(w, http.StatusBadRequest, "cannot delete yourself")
	case errors.Is(err, domain.ErrTaskAlreadyCompleted),
		errors.Is(err, domain.ErrTaskInvalidForReview),
		errors.Is(err, domain.ErrTaskInvalidForApprove),
		errors.Is(err, domain.ErrNoAssignedChild),
		errors.Is(err, domain.ErrTaskNotAssignedToUser):
		h.respondError(w, http.StatusBadRequest, err.Error())
	default:
		h.respondError(w, http.StatusInternalServerError, fallbackMsg)
	}
}

// maxRequestBodySize is the maximum allowed request body size (1MB)
const maxRequestBodySize = 1 << 20

// decodeJSON decodes JSON request body with size limit
func (h *APIHandler) decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	return json.NewDecoder(r.Body).Decode(v)
}

// parseFamilyEntityPath extracts familyUID and optional entity name from a URL path
// with the given prefix. Returns (familyUID, entityName, error).
// entityName is empty if the path only contains a familyUID.
func parseFamilyEntityPath(path, prefix string) (familyUID, entityName string, err error) {
	familyUID = strings.TrimPrefix(path, prefix)
	if familyUID == "" {
		return "", "", nil
	}

	parts := strings.SplitN(familyUID, "/", 2)
	familyUID = parts[0]

	if len(parts) == 2 && parts[1] != "" {
		entityName, err = url.QueryUnescape(parts[1])
		if err != nil {
			return familyUID, "", err
		}
	}

	return familyUID, entityName, nil
}
