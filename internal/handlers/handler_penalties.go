package handlers

import (
	"net/http"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
)

// Penalty endpoints
func (h *APIHandler) handlePenalties(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
		switch r.Method {
		case http.MethodPost:
			h.createPenalty(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handlePenaltiesByFamily(w http.ResponseWriter, r *http.Request) {
	h.authed(func(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
		familyUID, penaltyName, err := parseFamilyEntityPath(r.URL.Path, "/api/v1/penalties/")
		if err != nil {
			h.respondError(w, http.StatusBadRequest, ErrInvalidEntityEncoding)
			return
		}
		if familyUID == "" {
			h.respondError(w, http.StatusBadRequest, ErrFamilyUIDRequired)
			return
		}

		if penaltyName != "" {
			switch r.Method {
			case http.MethodGet:
				h.getPenalty(w, r, authCtx, familyUID, penaltyName)
			case http.MethodPut:
				h.updatePenalty(w, r, authCtx, familyUID, penaltyName)
			case http.MethodDelete:
				h.deletePenalty(w, r, authCtx, familyUID, penaltyName)
			case http.MethodPost:
				h.applyPenalty(w, r, authCtx, familyUID, penaltyName)
			default:
				h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
			}
		} else {
			h.getFamilyPenalties(w, r, authCtx, familyUID)
		}
	})(w, r)
}

func (h *APIHandler) createPenalty(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	var penalty models.Penalties
	if err := h.decodeJSON(w, r, &penalty); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if err := validatePenaltyCreate(&penalty); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.services.PenaltyService.CreatePenalty(r.Context(), authCtx, &penalty)
	if err != nil {
		h.respondServiceError(w, err, "failed to create penalty")
		return
	}

	h.respondSuccess(w, http.StatusCreated, penalty)
}

func (h *APIHandler) getFamilyPenalties(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID string) {
	penalties, err := h.services.PenaltyService.GetPenaltiesForFamily(r.Context(), authCtx, familyUID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve penalties")
		return
	}

	h.respondSuccess(w, http.StatusOK, penalties)
}

func (h *APIHandler) getPenalty(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, penaltyName string) {
	penalty, err := h.services.PenaltyService.GetPenalty(r.Context(), authCtx, familyUID, penaltyName)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve penalty")
		return
	}

	h.respondSuccess(w, http.StatusOK, penalty)
}

func (h *APIHandler) updatePenalty(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, penaltyName string) {
	var updates domain.UpdatePenaltyRequest
	if err := h.decodeJSON(w, r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	penalty, err := h.services.PenaltyService.UpdatePenalty(r.Context(), authCtx, familyUID, penaltyName, &updates)
	if err != nil {
		h.respondServiceError(w, err, "failed to update penalty")
		return
	}

	h.respondSuccess(w, http.StatusOK, penalty)
}

func (h *APIHandler) deletePenalty(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, penaltyName string) {
	err := h.services.PenaltyService.DeletePenalty(r.Context(), authCtx, familyUID, penaltyName)
	if err != nil {
		h.respondServiceError(w, err, "failed to delete penalty")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) applyPenalty(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, familyUID, penaltyName string) {
	var req struct {
		ChildUserID string `json:"child_user_id"`
	}
	if err := h.decodeJSON(w, r, &req); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}
	req.ChildUserID = sanitizeInput(req.ChildUserID)
	if req.ChildUserID == "" {
		h.respondError(w, http.StatusBadRequest, "child_user_id is required")
		return
	}

	penalty, err := h.services.PenaltyService.ApplyPenalty(r.Context(), authCtx, familyUID, penaltyName, req.ChildUserID)
	if err != nil {
		h.respondServiceError(w, err, "failed to apply penalty")
		return
	}

	h.respondSuccess(w, http.StatusOK, penalty)
}
