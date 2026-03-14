package handlers

import (
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
)

// Family endpoints
func (h *APIHandler) handleFamilies(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
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
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
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
	family, err := h.services.FamilyService.GetFamily(r.Context(), authCtx, authCtx.FamilyUID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve family")
		return
	}

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
		h.respondServiceError(w, err, "failed to create family")
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
		h.respondServiceError(w, err, "failed to retrieve family")
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
		h.respondServiceError(w, err, "failed to update family")
		return
	}

	h.respondSuccess(w, http.StatusOK, family)
}
