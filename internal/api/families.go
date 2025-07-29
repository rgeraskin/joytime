package api

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/postgres"
	"gorm.io/gorm"
)

// generateUniqueFamilyUID generates a unique family UID
func (h *APIHandler) generateUniqueFamilyUID(ctx context.Context) (string, error) {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		// Generate random UID
		familyUIDBytes := make([]byte, FAMILYUIDLENGTH)
		for j := range familyUIDBytes {
			familyUIDBytes[j] = FAMILYUIDCHARSET[rand.Intn(len(FAMILYUIDCHARSET))]
		}
		uid := string(familyUIDBytes)

		// Check if UID already exists
		var count int64
		if err := h.db.WithContext(ctx).Model(&postgres.Families{}).
			Where("uid = ?", uid).Count(&count).Error; err != nil {
			return "", err
		}

		if count == 0 {
			return uid, nil
		}
	}
	return "", errors.New("failed to generate unique family UID after multiple attempts")
}

// handleFamilies handles /families endpoint
func (h *APIHandler) handleFamilies(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.listFamilies(ctx, w, r)
	case http.MethodPost:
		h.createFamily(ctx, w, r)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// handleFamily handles /families/{uid} endpoint
func (h *APIHandler) handleFamily(w http.ResponseWriter, r *http.Request) {
	// Extract UID from path
	uid := strings.TrimPrefix(r.URL.Path, "/api/v1/families/")

	if uid == "" {
		h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		h.getFamily(ctx, w, r, uid)
	case http.MethodPut:
		h.updateFamily(ctx, w, r, uid)
	case http.MethodDelete:
		h.deleteFamily(ctx, w, r, uid)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

// listFamilies lists all families
func (h *APIHandler) listFamilies(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	h.logger.Debug("Listing all families")

	var families []postgres.Families
	if err := h.db.WithContext(ctx).Find(&families).Error; err != nil {
		h.logger.Error("Failed to list families", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve families")
		return
	}

	h.respondSuccess(w, http.StatusOK, families)
}

// createFamily creates a new family
func (h *APIHandler) createFamily(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Creating new family")

	var family postgres.Families
	if err := h.decodeJSON(r, &family); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateFamilyCreate(&family); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Generate unique family UID
	uid, err := h.generateUniqueFamilyUID(ctx)
	if err != nil {
		h.logger.Error("Failed to generate unique family UID", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to create family")
		return
	}
	family.UID = uid

	// Create family
	if err := h.db.WithContext(ctx).Create(&family).Error; err != nil {
		h.logger.Error("Failed to create family", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to create family")
		return
	}

	h.logger.Debug("Family created successfully", "uid", family.UID)
	h.respondSuccess(w, http.StatusCreated, family)
}

// getFamily retrieves a single family by UID
func (h *APIHandler) getFamily(
	ctx context.Context,
	w http.ResponseWriter,
	_ *http.Request,
	uid string,
) {
	h.logger.Debug("Getting family", "uid", uid)

	family, err := h.validateFamily(ctx, uid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to get family", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to retrieve family")
		}
		return
	}

	h.respondSuccess(w, http.StatusOK, family)
}

// updateFamily updates an existing family
func (h *APIHandler) updateFamily(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	h.logger.Debug("Updating family", "uid", uid)

	var family postgres.Families
	if err := h.decodeJSON(r, &family); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	// Validate input
	if errors := h.ValidateFamilyUpdate(&family); len(errors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(errors))
		return
	}

	// Check if family exists
	existingFamily, err := h.validateFamily(ctx, uid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to find family for update", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to update family")
		}
		return
	}

	// Update family
	if err := h.db.WithContext(ctx).Where("uid = ?", uid).Updates(&family).Error; err != nil {
		h.logger.Error("Failed to update family", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to update family")
		return
	}

	// Return updated family
	family.UID = existingFamily.UID
	family.ID = existingFamily.ID
	family.CreatedAt = existingFamily.CreatedAt
	h.respondSuccess(w, http.StatusOK, family)
}

// deleteFamily deletes a family
func (h *APIHandler) deleteFamily(
	ctx context.Context,
	w http.ResponseWriter,
	_ *http.Request,
	uid string,
) {
	h.logger.Debug("Deleting family", "uid", uid)

	// Check if family exists
	_, err := h.validateFamily(ctx, uid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrFamilyNotFound)
		} else {
			h.logger.Error("Failed to find family for deletion", "error", err)
			h.respondError(w, http.StatusInternalServerError, "Failed to delete family")
		}
		return
	}

	// Delete family
	if err := h.db.WithContext(ctx).Where("uid = ?", uid).Delete(&postgres.Families{}).Error; err != nil {
		h.logger.Error("Failed to delete family", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to delete family")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
