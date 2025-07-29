package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/postgres"
	"gorm.io/gorm"
)

// handleUsers handles /users endpoint
func (h *APIHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.listUsers(ctx, w, r)
	case http.MethodPost:
		h.createUser(ctx, w, r)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleUser handles /users/{userID} endpoint
func (h *APIHandler) handleUser(w http.ResponseWriter, r *http.Request) {
	// Extract userID from path
	userID := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")

	if userID == "" {
		h.respondError(w, http.StatusBadRequest, ErrUserIDRequired)
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.getUser(ctx, w, r, userID)
	case http.MethodPut:
		h.updateUser(ctx, w, r, userID)
	case http.MethodDelete:
		h.deleteUser(ctx, w, r, userID)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// listUsers lists all users
func (h *APIHandler) listUsers(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	h.logger.Debug("Listing all users")

	var users []postgres.Users
	if err := h.db.WithContext(ctx).Find(&users).Error; err != nil {
		h.logger.Error("Failed to list users", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	h.respondSuccess(w, http.StatusOK, users)
}

// createUser creates a new user
func (h *APIHandler) createUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Creating new user")

	var user postgres.Users
	if err := h.decodeJSON(r, &user); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Set default platform if not provided
	if user.Platform == "" {
		user.Platform = PlatformTelegram
	}

	// Validate input
	if errors := h.ValidateUserCreate(&user); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if family exists
	_, err := h.validateFamily(ctx, user.FamilyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to create user")
		}
		return
	}

	// Create user and tokens in a transaction
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create user
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Create tokens for child users
		if user.Role == RoleChild {
			tokens := postgres.Tokens{
				UserID: user.UserID,
				Tokens: 0,
			}
			if err := tx.Create(&tokens).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		h.logger.Error("Failed to create user", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	h.logger.Debug("User created successfully", "user_id", user.UserID, "role", user.Role)
	h.respondSuccess(w, http.StatusCreated, user)
}

// getUser retrieves a single user by ID
func (h *APIHandler) getUser(ctx context.Context, w http.ResponseWriter, _ *http.Request, userID string) {
	h.logger.Debug("Getting user", "user_id", userID)

	user, err := h.validateUser(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserNotFound)
		} else {
			h.logger.Error("Failed to get user", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user")
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, user)
}

// updateUser updates an existing user
func (h *APIHandler) updateUser(ctx context.Context, w http.ResponseWriter, r *http.Request, userID string) {
	h.logger.Debug("Updating user", "user_id", userID)

	var user postgres.Users
	if err := h.decodeJSON(r, &user); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateUserUpdate(&user); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if user exists
	existingUser, err := h.validateUser(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserNotFound)
		} else {
			h.logger.Error("Failed to find user for update", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update user")
		}
		return
	}

	// Check if family exists (if family is being updated)
	if user.FamilyUID != "" {
		_, err := h.validateFamily(ctx, user.FamilyUID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
			} else {
				h.logger.Error("Failed to validate family for user update", "error", err)
				h.respondError(w, http.StatusInternalServerError, "Failed to update user")
			}
			return
		}
	}

	// Update user
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).Updates(&user).Error; err != nil {
		h.logger.Error("Failed to update user", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Return updated user with preserved fields
	user.UserID = existingUser.UserID
	user.ID = existingUser.ID
	user.CreatedAt = existingUser.CreatedAt
	if user.FamilyUID == "" {
		user.FamilyUID = existingUser.FamilyUID
	}
	if user.Platform == "" {
		user.Platform = existingUser.Platform
	}
	if user.Role == "" {
		user.Role = existingUser.Role
	}
	if user.Name == "" {
		user.Name = existingUser.Name
	}

	h.respondSuccess(w, http.StatusOK, user)
}

// deleteUser deletes a user
func (h *APIHandler) deleteUser(ctx context.Context, w http.ResponseWriter, _ *http.Request, userID string) {
	h.logger.Debug("Deleting user", "user_id", userID)

	// Check if user exists
	_, err := h.validateUser(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserNotFound)
		} else {
			h.logger.Error("Failed to find user for deletion", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to delete user")
		}
		return
	}

	// Delete user
	if err := h.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&postgres.Users{}).Error; err != nil {
		h.logger.Error("Failed to delete user", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}