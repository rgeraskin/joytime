package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"gorm.io/gorm"
)

// User endpoints
func (h *APIHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, "Service context not found")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.listUsers(w, r, authCtx)
		case http.MethodPost:
			h.createUser(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handleUser(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, "Service context not found")
			return
		}

		userID := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
		if userID == "" {
			h.respondError(w, http.StatusBadRequest, ErrUserIDRequired)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getUser(w, r, authCtx, userID)
		case http.MethodPut:
			h.updateUser(w, r, authCtx, userID)
		case http.MethodDelete:
			h.deleteUser(w, r, authCtx, userID)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) listUsers(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	users, err := h.services.UserService.GetFamilyUsers(r.Context(), authCtx, authCtx.FamilyUID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	h.respondSuccess(w, http.StatusOK, users)
}

func (h *APIHandler) createUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	// Note: User creation might need special handling during family registration
	h.respondError(w, http.StatusNotImplemented, "User creation through this endpoint not implemented")
}

func (h *APIHandler) getUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	user, err := h.services.UserService.GetUser(r.Context(), authCtx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve user")
		return
	}

	h.respondSuccess(w, http.StatusOK, user)
}

func (h *APIHandler) updateUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	var updates domain.UpdateUserRequest
	if err := h.decodeJSON(r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	user, err := h.services.UserService.UpdateUser(r.Context(), authCtx, userID, &updates)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	h.respondSuccess(w, http.StatusOK, user)
}

func (h *APIHandler) deleteUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	err := h.services.UserService.DeleteUser(r.Context(), authCtx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			h.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		if err == gorm.ErrRecordNotFound {
			h.respondError(w, http.StatusNotFound, ErrUserNotFound)
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}