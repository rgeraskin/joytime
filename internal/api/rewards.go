package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/rgeraskin/joytime/internal/postgres"
	"gorm.io/gorm"
)

// handleRewards handles /rewards endpoint
func (h *APIHandler) handleRewards(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.listRewards(ctx, w, r)
	case http.MethodPost:
		h.createReward(ctx, w, r)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleRewardsByFamily handles /rewards/{familyUID} and /rewards/{familyUID}/{rewardName} endpoints
func (h *APIHandler) handleRewardsByFamily(w http.ResponseWriter, r *http.Request) {
	// Extract path parts
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/rewards/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
		return
	}

	familyUID := parts[0]
	ctx := context.Background()

	// Check if we have a reward name (individual reward operation)
	if len(parts) == 2 && parts[1] != "" {
		rewardName, err := url.QueryUnescape(parts[1])
		if err != nil {
			h.respondError(w, http.StatusBadRequest, ErrInvalidEntityEncoding)
			return
		}

		// Handle individual reward operations
		switch r.Method {
		case http.MethodGet:
			h.getSingleReward(ctx, w, r, familyUID, rewardName)
		case http.MethodPut:
			h.updateSingleReward(ctx, w, r, familyUID, rewardName)
		case http.MethodDelete:
			h.deleteSingleReward(ctx, w, r, familyUID, rewardName)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	} else {
		// Handle family reward operations
		switch r.Method {
		case http.MethodGet:
			h.getRewardsByFamily(ctx, w, r, familyUID)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	}
}



// listRewards lists all rewards
func (h *APIHandler) listRewards(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	h.logger.Debug("Listing all rewards")

	var rewards []postgres.Rewards
	if err := h.db.WithContext(ctx).Find(&rewards).Error; err != nil {
		h.logger.Error("Failed to list rewards", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve rewards")
		return
	}

	h.respondSuccess(w, http.StatusOK, rewards)
}

// createReward creates a new reward
func (h *APIHandler) createReward(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Creating new reward")

	var reward postgres.Rewards
	if err := h.decodeJSON(r, &reward); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateRewardCreate(&reward); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if family exists
	_, err := h.validateFamily(ctx, reward.FamilyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to create reward")
		}
		return
	}

	// Create reward
	if err := h.db.WithContext(ctx).Create(&reward).Error; err != nil {
		h.logger.Error("Failed to create reward", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to create reward")
		return
	}

	h.logger.Debug("Reward created successfully", "reward_id", reward.ID, "name", reward.Name)
	h.respondSuccess(w, http.StatusCreated, reward)
}

// getRewardsByFamily gets all rewards for a family
func (h *APIHandler) getRewardsByFamily(ctx context.Context, w http.ResponseWriter, _ *http.Request, familyUID string) {
	h.logger.Debug("Getting rewards by family", "family_uid", familyUID)

	// Check if family exists
	_, err := h.validateFamily(ctx, familyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve rewards")
		}
		return
	}

	var rewards []postgres.Rewards
	if err := h.db.WithContext(ctx).Where("family_uid = ?", familyUID).Find(&rewards).Error; err != nil {
		h.logger.Error("Failed to get rewards by family", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve rewards")
		return
	}

	h.respondSuccess(w, http.StatusOK, rewards)
}

// getSingleReward gets a single reward by family and name
func (h *APIHandler) getSingleReward(ctx context.Context, w http.ResponseWriter, _ *http.Request, familyUID, rewardName string) {
	h.logger.Debug("Getting single reward", "family_uid", familyUID, "reward_name", rewardName)

	// Check if family exists
	_, err := h.validateFamily(ctx, familyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve reward")
		}
		return
	}

	var reward postgres.Rewards
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		First(&reward).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
		} else {
			h.logger.Error("Failed to get single reward", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve reward")
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, reward)
}

// updateSingleReward updates a single reward
func (h *APIHandler) updateSingleReward(ctx context.Context, w http.ResponseWriter, r *http.Request, familyUID, rewardName string) {
	h.logger.Debug("Updating single reward", "family_uid", familyUID, "reward_name", rewardName)

	var updateData postgres.Entities
	if err := h.decodeJSON(r, &updateData); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateEntityUpdate(&updateData); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if family exists
	_, err := h.validateFamily(ctx, updateData.FamilyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update reward")
		}
		return
	}

	// Check if reward exists
	var existingReward postgres.Rewards
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		First(&existingReward).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
		} else {
			h.logger.Error("Failed to find reward for update", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update reward")
		}
		return
	}

	// Update reward
	existingReward.Tokens = updateData.Tokens
	existingReward.Description = updateData.Description
	if err := h.db.WithContext(ctx).Save(&existingReward).Error; err != nil {
		h.logger.Error("Failed to update reward", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to update reward")
		return
	}

	h.respondSuccess(w, http.StatusOK, existingReward)
}

// deleteSingleReward deletes a single reward
func (h *APIHandler) deleteSingleReward(ctx context.Context, w http.ResponseWriter, _ *http.Request, familyUID, rewardName string) {
	h.logger.Debug("Deleting single reward", "family_uid", familyUID, "reward_name", rewardName)

	// Check if family exists
	_, err := h.validateFamily(ctx, familyUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusBadRequest, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to validate family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to delete reward")
		}
		return
	}

	// Check if reward exists
	var existingReward postgres.Rewards
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		First(&existingReward).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrEntityNotFound)
		} else {
			h.logger.Error("Failed to find reward for deletion", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to delete reward")
		}
		return
	}

	// Delete reward
	if err := h.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		Delete(&postgres.Rewards{}).Error; err != nil {
		h.logger.Error("Failed to delete reward", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to delete reward")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}