package domain

import (
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

// Services contains all business logic services
type Services struct {
	TaskService   *TaskService
	TokenService  *TokenService
	UserService   *UserService
	FamilyService *FamilyService
	RewardService *RewardService
	Auth          *CasbinAuthService
}

// NewServices creates a new services instance with Casbin authorization
func NewServices(db *gorm.DB, logger *log.Logger) (*Services, error) {
	// Initialize Casbin authorization service
	auth, err := NewCasbinAuthService(db, logger)
	if err != nil {
		return nil, err
	}

	return &Services{
		TaskService:   NewTaskService(db, logger, auth),
		TokenService:  NewTokenService(db, logger, auth),
		UserService:   NewUserService(db, logger, auth),
		FamilyService: NewFamilyService(db, logger, auth),
		RewardService: NewRewardService(db, logger, auth),
		Auth:          auth,
	}, nil
}