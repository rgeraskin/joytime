package handlers

import (
	"context"
	"net/http"

	"github.com/rgeraskin/joytime/internal/domain"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	ContextKeyAuthContext contextKey = "auth_context"
)

// AuthMiddleware extracts user information and creates service context
// In a real application, this would extract user info from JWT token or session
func (h *APIHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For now, we'll extract user_id from a header (in production, use proper auth)
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			h.respondError(w, http.StatusUnauthorized, "Missing user authentication")
			return
		}

		// Create auth context
		authCtx, err := h.services.UserService.CreateAuthContext(r.Context(), userID)
		if err != nil {
			h.logger.Error("Failed to create auth context", "error", err, "user_id", userID)
			h.respondError(w, http.StatusUnauthorized, "Invalid user")
			return
		}

		// Add auth context to request context
		ctx := context.WithValue(r.Context(), ContextKeyAuthContext, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// authed wraps a handler that requires authentication, extracting the auth context automatically.
func (h *APIHandler) authed(fn func(http.ResponseWriter, *http.Request, *domain.AuthContext)) http.HandlerFunc {
	return h.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		authCtx, ok := r.Context().Value(ContextKeyAuthContext).(*domain.AuthContext)
		if !ok || authCtx == nil {
			h.respondError(w, http.StatusInternalServerError, "auth context not found")
			return
		}
		fn(w, r, authCtx)
	})
}

