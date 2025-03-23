package main

import (
	"fmt"
	"os"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/charmbracelet/log"
)

var logger *log.Logger

type Config struct {
	Token string
	DB    DBConfig
}

func (c *Config) Load() error {
	c.Token = os.Getenv("TOKEN")
	c.DB.User = os.Getenv("PGUSER")
	c.DB.Password = os.Getenv("PGPASSWORD")
	c.DB.Host = os.Getenv("PGHOST")
	c.DB.Port = os.Getenv("PGPORT")
	c.DB.Database = os.Getenv("PGDATABASE")
	if err := c.validate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validate() error {
	if c.Token == "" {
		return fmt.Errorf("TOKEN is required")
	}
	if c.DB.User == "" {
		return fmt.Errorf("PGUSER is required")
	}
	if c.DB.Password == "" {
		return fmt.Errorf("PGPASSWORD is required")
	}
	if c.DB.Host == "" {
		return fmt.Errorf("PGHOST is required")
	}
	if c.DB.Port == "" {
		return fmt.Errorf("PGPORT is required")
	}
	if c.DB.Database == "" {
		return fmt.Errorf("PGDATABASE is required")
	}
	return nil
}

func main() {
	// Initialize logger
	level := log.InfoLevel
	if os.Getenv("DEBUG") != "" {
		level = log.DebugLevel
	}
	logger = log.NewWithOptions(os.Stderr, log.Options{
		// ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
		// Formatter:       log.LogfmtFormatter,
	})

	// Load config
	logger.Info("Loading config...")
	config := &Config{}
	if err := config.Load(); err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Open database
	logger.Info("Opening database...")
	db := &DB{
		Config: &config.DB,
	}
	if err := db.Open(); err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	// db.Fill()

	logger.Info("Starting dispatcher...")
	dsp := echotron.NewDispatcher(config.Token, func(chatID int64) echotron.Bot {
		b := newBot(chatID, db, config.Token)
		if b == nil {
			logger.Error("Failed to create bot instance")
			return nil
		}
		return b
	})

	// Add error handling for the polling
	for {
		logger.Info("Polling...")
		if err := dsp.Poll(); err != nil {
			logger.Error("Polling error", "error", err)
		}
		logger.Info("Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
