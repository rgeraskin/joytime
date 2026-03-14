package handlers

import (
	"net/http"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
)

// Token endpoints
func (h *APIHandler) handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			h.createTokenTransaction(w, r, authCtx)
		})(w, r)
	default:
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
	}
}

func (h *APIHandler) handleUserTokens(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)

		userID := strings.TrimPrefix(r.URL.Path, "/api/v1/tokens/users/")
		if userID == "" {
			h.respondError(w, http.StatusBadRequest, ErrUserIDRequired)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getUserTokens(w, r, authCtx, userID)
		case http.MethodPost:
			h.updateUserTokens(w, r, authCtx, userID)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handleTokenHistory(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)

		switch r.Method {
		case http.MethodGet:
			h.getTokenHistory(w, r, authCtx)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) handleUserTokenHistory(w http.ResponseWriter, r *http.Request) {
	h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)

		userID := strings.TrimPrefix(r.URL.Path, "/api/v1/token-history/users/")
		if userID == "" {
			h.respondError(w, http.StatusBadRequest, ErrUserIDRequired)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getUserTokenHistory(w, r, authCtx, userID)
		default:
			h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		}
	})(w, r)
}

func (h *APIHandler) createTokenTransaction(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	var request TokenAddRequest
	if err := h.decodeJSON(w, r, &request); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if validationErrors := h.ValidateTokenAddRequest(&request); len(validationErrors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(validationErrors))
		return
	}

	if request.UserID == "" {
		h.respondError(w, http.StatusBadRequest, ErrUserIDRequiredField)
		return
	}

	err := h.services.TokenService.AddTokensToUser(
		r.Context(),
		authCtx,
		request.UserID,
		request.Amount,
		request.Type,
		request.Description,
		request.TaskID,
		request.RewardID,
	)
	if err != nil {
		h.respondServiceError(w, err, "failed to create token transaction")
		return
	}

	h.respondSuccess(w, http.StatusCreated, map[string]string{"message": "token transaction completed"})
}

func (h *APIHandler) getUserTokens(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	tokens, err := h.services.TokenService.GetUserTokens(r.Context(), authCtx, userID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve user tokens")
		return
	}

	h.respondSuccess(w, http.StatusOK, tokens)
}

func (h *APIHandler) updateUserTokens(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	var update TokenAddRequest
	if err := h.decodeJSON(w, r, &update); err != nil {
		h.respondError(w, http.StatusBadRequest, ErrInvalidJSONFormat)
		return
	}

	if validationErrors := h.ValidateTokenAddRequest(&update); len(validationErrors) > 0 {
		h.respondError(w, http.StatusBadRequest, FormatValidationErrors(validationErrors))
		return
	}

	err := h.services.TokenService.AddTokensToUser(
		r.Context(),
		authCtx,
		userID,
		update.Amount,
		update.Type,
		update.Description,
		update.TaskID,
		update.RewardID,
	)
	if err != nil {
		h.respondServiceError(w, err, "failed to update user tokens")
		return
	}

	tokens, err := h.services.TokenService.GetUserTokens(r.Context(), authCtx, userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve updated tokens")
		return
	}

	h.respondSuccess(w, http.StatusOK, tokens)
}

func (h *APIHandler) getTokenHistory(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext) {
	history, err := h.services.TokenService.GetTokenHistory(r.Context(), authCtx, authCtx.UserID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve token history")
		return
	}

	h.respondSuccess(w, http.StatusOK, history)
}

func (h *APIHandler) getUserTokenHistory(w http.ResponseWriter, r *http.Request, authCtx *domain.AuthContext, userID string) {
	history, err := h.services.TokenService.GetTokenHistory(r.Context(), authCtx, userID)
	if err != nil {
		h.respondServiceError(w, err, "failed to retrieve user token history")
		return
	}

	h.respondSuccess(w, http.StatusOK, history)
}
