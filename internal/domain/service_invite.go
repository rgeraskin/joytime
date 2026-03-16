package domain

import (
	"context"

	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

const (
	inviteCodeLength = 8
)

// InviteService handles invite code business logic
type InviteService struct {
	db   *gorm.DB
	auth *CasbinAuthService
}

// NewInviteService creates a new invite service
func NewInviteService(db *gorm.DB, auth *CasbinAuthService) *InviteService {
	return &InviteService{
		db:   db,
		auth: auth,
	}
}

// CreateInvite generates a one-time invite code for a family
func (s *InviteService) CreateInvite(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, role string,
) (*models.Invites, error) {
	if err := s.auth.RequirePermission(authCtx, "invites", "create", familyUID); err != nil {
		return nil, err
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	invite := &models.Invites{
		Code:            code,
		FamilyUID:       familyUID,
		Role:            role,
		Used:            false,
		CreatedByUserID: authCtx.UserID,
	}

	if err := s.db.WithContext(ctx).Create(invite).Error; err != nil {
		return nil, err
	}

	return invite, nil
}

// UseInvite atomically marks an unused invite as used and returns it.
// No auth check — used during registration before a user has an auth context.
// Uses a single UPDATE with WHERE used=false to prevent race conditions.
func (s *InviteService) UseInvite(ctx context.Context, code string) (*models.Invites, error) {
	var invite models.Invites

	// Atomically claim the invite: only one concurrent caller can succeed.
	result := s.db.WithContext(ctx).
		Model(&invite).
		Where("code = ? AND used = ?", code, false).
		Update("used", true)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// Read back the full invite record
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&invite).Error; err != nil {
		return nil, err
	}

	return &invite, nil
}

func generateInviteCode() (string, error) {
	return generateRandomCode(inviteCodeLength)
}
