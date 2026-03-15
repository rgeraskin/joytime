package domain

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

const (
	inviteCodeLength  = 8
	inviteCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// InviteService handles invite code business logic
type InviteService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
}

// NewInviteService creates a new invite service
func NewInviteService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService) *InviteService {
	return &InviteService{
		db:     db,
		logger: logger,
		auth:   auth,
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

// UseInvite finds an unused invite by code, marks it as used, and returns it.
// No auth check — used during registration before a user has an auth context.
func (s *InviteService) UseInvite(ctx context.Context, code string) (*models.Invites, error) {
	var invite models.Invites
	err := s.db.WithContext(ctx).
		Where("code = ? AND used = ?", code, false).
		First(&invite).Error
	if err != nil {
		return nil, err
	}

	invite.Used = true
	if err := s.db.WithContext(ctx).Save(&invite).Error; err != nil {
		return nil, err
	}

	return &invite, nil
}

func generateInviteCode() (string, error) {
	code := make([]byte, inviteCodeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeCharset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate invite code: %w", err)
		}
		code[i] = inviteCodeCharset[n.Int64()]
	}
	return string(code), nil
}
