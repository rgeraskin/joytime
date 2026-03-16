package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/database"
)

type Config struct {
	Token string
	DB    database.Config
}

func NewConfig(logger *log.Logger) (*Config, error) {
	config := &Config{}

	logger.Debug("Loading config...")
	config.load()

	logger.Debug("Validating config...")
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	logger.Debug("Config loaded successfully")
	return config, nil
}

func (c *Config) load() {
	c.Token = os.Getenv("TOKEN")

	c.DB.Type = os.Getenv("DB_TYPE")
	if c.DB.Type == "" {
		c.DB.Type = "sqlite"
	}

	c.DB.Path = os.Getenv("DB_PATH")
	if c.DB.Path == "" {
		c.DB.Path = "joytime.db"
	}

	c.DB.User = os.Getenv("PGUSER")
	c.DB.Password = os.Getenv("PGPASSWORD")
	c.DB.Host = os.Getenv("PGHOST")
	c.DB.Port = os.Getenv("PGPORT")
	c.DB.Database = os.Getenv("PGDATABASE")
}

func (c *Config) validate() error {
	if c.Token == "" {
		return fmt.Errorf("TOKEN environment variable is required")
	}

	if c.DB.Type == "postgres" {
		if c.DB.User == "" {
			return fmt.Errorf("PGUSER environment variable is required for postgres")
		}
		if c.DB.Password == "" {
			return fmt.Errorf("PGPASSWORD environment variable is required for postgres")
		}
		if c.DB.Host == "" {
			return fmt.Errorf("PGHOST environment variable is required for postgres")
		}
		if c.DB.Port == "" {
			return fmt.Errorf("PGPORT environment variable is required for postgres")
		}
		if _, err := strconv.Atoi(c.DB.Port); err != nil {
			return fmt.Errorf("PGPORT must be a valid port number")
		}
		if c.DB.Database == "" {
			return fmt.Errorf("PGDATABASE environment variable is required for postgres")
		}
	}

	return nil
}
