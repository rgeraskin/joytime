package postgres

import (
	"fmt"

	"github.com/charmbracelet/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

func NewDB(config *Config, fill_only bool, logger *log.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.Host,
		config.User,
		config.Password,
		config.Database,
		config.Port,
	)

	var err error
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			logger,
			gormlogger.Config{
				IgnoreRecordNotFoundError: true,
			},
		),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	logger.Info("Migrating database...")
	err = db.AutoMigrate(
		&Users{},
		&Families{},
		&Tokens{},
		&Tasks{},
		&Rewards{},
		&TokenHistory{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	if fill_only {
		logger.Info("Filling database...")
		err = fill(db)
		if err != nil {
			return nil, fmt.Errorf("failed to fill database: %w", err)
		}
	}

	return db, nil
}
