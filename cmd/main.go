package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/database"
	"github.com/rgeraskin/joytime/internal/handlers"
)

func main() {
	// Initialize flags
	var fill bool
	flag.BoolVar(
		&fill,
		"fill",
		false,
		"If set, the database will be filled with data, then exit.",
	)
	flag.Parse()

	// Initialize logger
	level := log.InfoLevel
	if os.Getenv("DEBUG") != "" {
		level = log.DebugLevel
	}
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
	})

	// Get config
	logger.Info("Getting config...")
	config, err := NewConfig(logger)
	if err != nil {
		logger.Fatal("Failed to get config", "error", err)
	}

	// Get database
	logger.Info("Getting database...")
	db, err := database.NewDB(&config.DB, fill, logger)
	if err != nil {
		logger.Fatal("Failed to get database", "error", err)
	}

	// If only fill is requested, exit
	if fill {
		logger.Info("Database filled successfully, exiting...")
		return
	}

	// Start HTTP API server with graceful shutdown
	apiServer := handlers.SetupAPI(db, logger)

	go func() {
		logger.Info("Starting HTTP API server", "address", handlers.ADDRESS)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP API server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	// Close database connection
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get sql.DB for cleanup", "error", err)
	} else {
		_ = sqlDB.Close()
	}

	logger.Info("Server stopped")
}
