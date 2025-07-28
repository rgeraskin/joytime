package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/api"
	"github.com/rgeraskin/joytime/internal/postgres"
)

var logger *log.Logger

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
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
	})

	// Get config
	logger.Info("Getting config...")
	config, err := NewConfig()
	if err != nil {
		logger.Fatal("Failed to get config", "error", err)
	}

	// Get database
	logger.Info("Getting database...")
	db, err := postgres.NewDB(&config.DB, fill, logger)
	if err != nil {
		logger.Fatal("Failed to get database", "error", err)
	}

	// If only fill is requested, exit
	if fill {
		logger.Info("Database filled successfully, exiting...")
		return
	}

	// Start HTTP API server
	apiServer := api.SetupAPI(db, logger)
	// go func() {
	// 	logger.Info("Starting HTTP API server", "port", api.PORT)
	// 	if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 		logger.Fatal("HTTP API server error", "error", err)
	// 	}
	// }()
	logger.Info("Starting HTTP API server", "address", api.ADDRESS)
	if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("HTTP API server error", "error", err)
	}

	// logger.Info("Starting dispatcher...")
	// dsp := echotron.NewDispatcher(config.Token, func(chatID int64) echotron.Bot {
	// 	b := newBot(chatID, db, config.Token)
	// 	if b == nil {
	// 		logger.Fatal("Failed to create bot instance")
	// 	}
	// 	return b
	// })

	// // Add error handling for the polling
	// for {
	// 	logger.Info("Polling...")
	// 	if err := dsp.Poll(); err != nil {
	// 		logger.Error("Polling error", "error", err)
	// 	}
	// 	logger.Info("Reconnecting in 5 seconds...")
	// 	time.Sleep(5 * time.Second)
	// }
}
