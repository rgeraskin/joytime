package handlers

import (
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
)

// User endpoints
func (h *APIHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)

		switch r.Method {
		case http.MethodGet:
			h.listUsers(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handleUser(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)

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
		h.respondServiceError(w, err, "failed to retrieve users")
		return
	}

	h.respondSuccess(w, http.StatusOK, users)
}

func (h *APIHandler) getUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	user, err := h.services.UserService.GetUser(r.Context(), authCtx, userID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve user")
		return
	}

	h.respondSuccess(w, http.StatusOK, user)
}

func (h *APIHandler) updateUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	var updates domain.UpdateUserRequest
	if err := h.decodeJSON(w, r, &updates); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	user, err := h.services.UserService.UpdateUser(r.Context(), authCtx, userID, &updates)
	if err != nil {
		h.respondServiceError(w, err, "failed to update user")
		return
	}

	h.respondSuccess(w, http.StatusOK, user)
}

func (h *APIHandler) deleteUser(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	err := h.services.UserService.DeleteUser(r.Context(), authCtx, userID)
	if err != nil {
		h.respondServiceError(w, err, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
