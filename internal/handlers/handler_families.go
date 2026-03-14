package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// Family endpoints
func (h *APIHandler) handleFamilies(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, ErrAuthContextNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.listFamilies(w, r, authCtx)
		case http.MethodPost:
			h.createFamily(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handleFamily(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, ErrAuthContextNotFound)
			return
		}

		familyUID := strings.TrimPrefix(r.URL.Path, "/api/v1/families/")
		if familyUID == "" {
			h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getFamily(w, r, authCtx, familyUID)
		case http.MethodPut:
			h.updateFamily(w, r, authCtx, familyUID)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) listFamilies(
	w http.ResponseWriter,
	r *http.Request,
	authCtx *domain.AuthContext,
) {
	// For families, user can only see their own family
	family, err := h.services.FamilyService.GetFamily(r.Context(), authCtx, authCtx.FamilyUID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve family")
		return
	}

	// Return as array for consistency with list endpoints
	h.respondSuccess(w, http.StatusOK, []models.Families{*family})
}

func (h *APIHandler) createFamily(
	w http.ResponseWriter,
	r *http.Request,
	authCtx *domain.AuthContext,
) {
	var family models.Families
	if err := h.decodeJSON(w, r, &family); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if validationErrors := h.ValidateFamilyCreate(&family); len(validationErrors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(validationErrors))
		return
	}

	err := h.services.FamilyService.CreateFamilyWithAuth(r.Context(), authCtx, &family)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to create family")
		return
	}

	h.respondSuccess(w, http.StatusCreated, family)
}

func (h *APIHandler) getFamily(
	w http.ResponseWriter,
	r *http.Request,
	authCtx *domain.AuthContext,
	familyUID string,
) {
	family, err := h.services.FamilyService.GetFamily(r.Context(), authCtx, familyUID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondError(w, http.StatusNotFound, ErrFamilyNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve family")
		return
	}

	h.respondSuccess(w, http.StatusOK, family)
}

func (h *APIHandler) updateFamily(
	w http.ResponseWriter,
	r *http.Request,
	authCtx *domain.AuthContext,
	familyUID string,
) {
	var updates domain.UpdateFamilyRequest
	if err := h.decodeJSON(w, r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	family, err := h.services.FamilyService.UpdateFamily(r.Context(), authCtx, familyUID, &updates)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Only parents can update family information")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondError(w, http.StatusNotFound, ErrFamilyNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to update family")
		return
	}

	h.respondSuccess(w, http.StatusOK, family)
}
