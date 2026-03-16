package domain

import (
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

// Services contains all business logic services
type Services struct {
	TaskService    *TaskService
	TokenService   *TokenService
	UserService    *UserService
	FamilyService  *FamilyService
	RewardService  *RewardService
	PenaltyService *PenaltyService
	InviteService  *InviteService
	Auth           *CasbinAuthService
}

// NewServices creates a new services instance with Casbin authorization
func NewServices(db *gorm.DB, logger *log.Logger) (*Services, error) {
	// Initialize Casbin authorization service
	auth, err := NewCasbinAuthService(logger)
	if err != nil {
		return nil, err
	}

	tokenService := NewTokenService(db, auth)

	return &Services{
		TaskService:    NewTaskService(db, auth, tokenService),
		TokenService:   tokenService,
		UserService:    NewUserService(db, auth),
		FamilyService:  NewFamilyService(db, auth),
		RewardService:  NewRewardService(db, auth),
		PenaltyService: NewPenaltyService(db, auth, tokenService),
		InviteService:  NewInviteService(db, auth),
		Auth:           auth,
	}, nil
}
