package database

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/glebarez/sqlite"
	"github.com/rgeraskin/joytime/internal/models"
	postgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	Type     string // "sqlite" (default) or "postgres"
	Path     string // SQLite file path (default: "joytime.db")
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

func NewDB(config *Config, fillOnly bool, logger *log.Logger) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch config.Type {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			config.Host,
			config.User,
			config.Password,
			config.Database,
			config.Port,
		)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(config.Path)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
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

	if config.Type == "postgres" {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		if err := sqlDB.Ping(); err != nil {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
	}

	logger.Info("Migrating database...", "type", config.Type)
	err = db.AutoMigrate(
		&models.Users{},
		&models.Families{},
		&models.Tokens{},
		&models.Tasks{},
		&models.Rewards{},
		&models.Penalties{},
		&models.Invites{},
		&models.TokenHistory{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create partial unique indexes that exclude soft-deleted rows.
	// Both PostgreSQL and SQLite support partial indexes with WHERE clause.
	for _, table := range []string{"tasks", "rewards", "penalties"} {
		// Drop the old GORM-generated non-partial unique index (table-specific name)
		oldIdx := fmt.Sprintf("idx_%s_name", table)
		if err := db.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %s`, oldIdx)).Error; err != nil {
			return nil, fmt.Errorf("failed to drop old index %s: %w", oldIdx, err)
		}

		idx := fmt.Sprintf("idx_%s_family_name_active", table)
		sql := fmt.Sprintf(
			`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (family_uid, name) WHERE deleted_at IS NULL`,
			idx, table,
		)
		if err := db.Exec(sql).Error; err != nil {
			return nil, fmt.Errorf("failed to create partial unique index on %s: %w", table, err)
		}
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
