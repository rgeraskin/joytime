package api

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

// SetupAPI configures and returns the HTTP server for the API
func SetupAPI(database *gorm.DB, _logger *log.Logger) *http.Server {
	handler := NewAPIHandler(database, _logger)
	mux := http.NewServeMux()

	_logger.Debug("Setting up API with versioned endpoints")

	// API v1 routes
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

	return &http.Server{
		Addr:    ADDRESS,
		Handler: mux,
	}
}
