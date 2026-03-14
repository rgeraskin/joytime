package domain

import (
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

// Services contains all business logic services
type Services struct {
	db            *gorm.DB
	TaskService   *TaskService
	TokenService  *TokenService
	UserService   *UserService
	FamilyService *FamilyService
	RewardService *RewardService
	Auth          *CasbinAuthService
}

// DB returns the underlying database connection
func (s *Services) DB() *gorm.DB {
	return s.db
}

// NewServices creates a new services instance with Casbin authorization
func NewServices(db *gorm.DB, logger *log.Logger) (*Services, error) {
	// Initialize Casbin authorization service
	auth, err := NewCasbinAuthService(db, logger)
	if err != nil {
		return nil, err
	}

	tokenService := NewTokenService(db, logger, auth)

	return &Services{
		db:          db,
		TaskService:   NewTaskService(db, logger, auth, tokenService),
		TokenService:  tokenService,
		UserService:   NewUserService(db, logger, auth),
		FamilyService: NewFamilyService(db, logger, auth),
		RewardService: NewRewardService(db, logger, auth),
		Auth:          auth,
	}, nil
}