package handlers

import (
	"net/http"

	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

const (
	ADDRESS          = ":8080"
	FAMILYUIDCHARSET = "abcdefghjkmnpqrstuvwxyz23456789"
	FAMILYUIDLENGTH  = 6
)

// SetupAPI configures and returns the HTTP server for the API with RBAC enforcement
func SetupAPI(database *gorm.DB, _logger *log.Logger) *http.Server {
	handler := NewAPIHandler(database, _logger)
	mux := http.NewServeMux()

	_logger.Debug("Setting up API with business logic and RBAC enforcement")

	// Apply authentication middleware to all business endpoints
	// Note: In production, you might want more granular middleware application

	// API v1 routes with business logic and RBAC
	mux.HandleFunc("/api/v1/families", handler.handleFamilies)
	mux.HandleFunc("/api/v1/families/", handler.handleFamily)
	mux.HandleFunc("/api/v1/users", handler.handleUsers)
	mux.HandleFunc("/api/v1/users/", handler.handleUser)
	mux.HandleFunc("/api/v1/tasks", handler.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", handler.handleTasksByFamily)
	mux.HandleFunc("/api/v1/rewards", handler.handleRewards)
	mux.HandleFunc("/api/v1/rewards/", handler.handleRewardsByFamily)
	mux.HandleFunc("/api/v1/tokens", handler.handleTokens)
	mux.HandleFunc("/api/v1/tokens/", handler.handleUserTokens)
	mux.HandleFunc("/api/v1/token-history", handler.handleTokenHistory)
	mux.HandleFunc("/api/v1/token-history/", handler.handleUserTokenHistory)

	// Special endpoints that might not need auth (like health checks, family registration)
	mux.HandleFunc("/api/v1/health", handler.handleHealth)


	return &http.Server{
		Addr:    ADDRESS,
		Handler: mux,
	}
}

// handleHealth provides a simple health check endpoint
func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "joytime-api",
	})
}


