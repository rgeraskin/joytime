package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// handleTokens handles /tokens endpoint
func (h *APIHandler) handleTokens(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.listTokens(ctx, w, r)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleUserTokens handles /tokens/{userID} endpoint
func (h *APIHandler) handleUserTokens(w http.ResponseWriter, r *http.Request) {
	// Extract userID from path
	userID := strings.TrimPrefix(r.URL.Path, "/api/v1/tokens/")

	if userID == "" {
		h.respondError(w, http.StatusBadRequest, ErrUserIDRequired)
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.getUserTokens(ctx, w, r, userID)
	case http.MethodPut:
		h.updateUserTokens(ctx, w, r, userID)
	case http.MethodPost:
		h.addTokensToUser(ctx, w, r, userID)
	case http.MethodDelete:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleTokenHistory handles /token-history endpoint
func (h *APIHandler) handleTokenHistory(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.listTokenHistory(ctx, w, r)
	case http.MethodPost:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	case http.MethodPut:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	case http.MethodDelete:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleUserTokenHistory handles /token-history/{userID} endpoint
func (h *APIHandler) handleUserTokenHistory(w http.ResponseWriter, r *http.Request) {
	// Extract userID from path
	userID := strings.TrimPrefix(r.URL.Path, "/api/v1/token-history/")

	if userID == "" {
		h.respondError(w, http.StatusBadRequest, ErrUserIDRequired)
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.getUserTokenHistory(ctx, w, r, userID)
	case http.MethodPost:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	case http.MethodPut:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	case http.MethodDelete:
		h.respondError(w, http.StatusNotImplemented, ErrNotImplemented)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// listTokens lists all tokens
func (h *APIHandler) listTokens(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	h.logger.Debug("Listing all tokens")

	var tokens []postgres.Tokens
	if err := h.db.WithContext(ctx).Find(&tokens).Error; err != nil {
		h.logger.Error("Failed to list tokens", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve tokens")
		return
	}

	h.respondSuccess(w, http.StatusOK, tokens)
}

// getUserTokens gets tokens for a specific user
func (h *APIHandler) getUserTokens(ctx context.Context, w http.ResponseWriter, _ *http.Request, userID string) {
	h.logger.Debug("Getting user tokens", "user_id", userID)

	var tokens postgres.Tokens
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).First(&tokens).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserTokensNotFound)
		} else {
			h.logger.Error("Failed to get user tokens", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user tokens")
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, tokens)
}

// updateUserTokens updates user tokens (direct update)
func (h *APIHandler) updateUserTokens(ctx context.Context, w http.ResponseWriter, r *http.Request, userID string) {
	h.logger.Debug("Updating user tokens", "user_id", userID)

	var tokensUpdate postgres.Tokens
	if err := h.decodeJSON(r, &tokensUpdate); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Check if user tokens exist
	var existingTokens postgres.Tokens
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).First(&existingTokens).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserTokensNotFound)
		} else {
			h.logger.Error("Failed to find tokens for update", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update user tokens")
		}
		return
	}

	// Update tokens
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).Updates(&tokensUpdate).Error; err != nil {
		h.logger.Error("Failed to update tokens", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to update user tokens")
		return
	}

	// Get updated tokens
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).First(&existingTokens).Error; err != nil {
		h.logger.Error("Failed to get updated tokens", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to update user tokens")
		return
	}

	h.respondSuccess(w, http.StatusOK, existingTokens)
}

// addTokensToUser adds or subtracts tokens from user balance
func (h *APIHandler) addTokensToUser(ctx context.Context, w http.ResponseWriter, r *http.Request, userID string) {
	h.logger.Debug("Adding tokens to user", "user_id", userID)

	var request TokenAddRequest
	if err := h.decodeJSON(r, &request); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate request
	if errors := h.ValidateTokenAddRequest(&request); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	var tokens postgres.Tokens

	// Use transaction to prevent race conditions
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current tokens with SELECT FOR UPDATE lock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&tokens).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new tokens record
				tokens = postgres.Tokens{
					UserID: userID,
					Tokens: 0,
				}
				if err := tx.Create(&tokens).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// Check for sufficient tokens if subtracting
		if request.Amount < 0 && tokens.Tokens < -request.Amount {
			return errors.New(ErrInsufficientTokens)
		}

		// Update tokens
		tokens.Tokens += request.Amount
		if err := tx.Save(&tokens).Error; err != nil {
			return err
		}

		// Create history record
		history := postgres.TokenHistory{
			UserID:      userID,
			Amount:      request.Amount,
			Type:        request.Type,
			Description: request.Description,
			TaskID:      request.TaskID,
			RewardID:    request.RewardID,
		}
		if err := tx.Create(&history).Error; err != nil {
			h.logger.Error("Failed to save token history", "error", err)
			// Don't fail the operation due to history error, just log it
		}

		return nil
	})

	if err != nil {
		if err.Error() == ErrInsufficientTokens {
			h.respondError(w, http.StatusBadRequest, ErrInsufficientTokens)
		} else {
			h.logger.Error("Failed to process token operation", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to process token operation")
		}
		return
	}

	h.logger.Debug("Token operation completed", "user_id", userID, "amount", request.Amount, "new_balance", tokens.Tokens)
	h.respondSuccess(w, http.StatusOK, tokens)
}

// listTokenHistory lists all token history
func (h *APIHandler) listTokenHistory(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	h.logger.Debug("Listing all token history")

	var history []postgres.TokenHistory
	if err := h.db.WithContext(ctx).Find(&history).Error; err != nil {
		h.logger.Error("Failed to list token history", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve token history")
		return
	}

	h.respondSuccess(w, http.StatusOK, history)
}

// getUserTokenHistory gets token history for a specific user
func (h *APIHandler) getUserTokenHistory(ctx context.Context, w http.ResponseWriter, r *http.Request, userID string) {
	h.logger.Debug("Getting user token history", "user_id", userID)

	var history []postgres.TokenHistory
	if err := h.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		h.logger.Error("Failed to get user token history", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve token history")
		return
	}

	h.respondSuccess(w, http.StatusOK, history)
}