package database

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	postgres "gorm.io/driver/postgres"
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

func NewDB(config *Config, fillOnly bool, logger *log.Logger) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.Host,
		config.User,
		config.Password,
		config.Database,
		config.Port,
	)

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

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Migrating database...")
	err = db.AutoMigrate(
		&models.Users{},
		&models.Families{},
		&models.Tokens{},
		&models.Tasks{},
		&models.Rewards{},
		&models.TokenHistory{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	if fillOnly {
		logger.Info("Filling database...")
		err = Fill(db)
		if err != nil {
			return nil, fmt.Errorf("failed to fill database: %w", err)
		}
	}

	return db, nil
}
