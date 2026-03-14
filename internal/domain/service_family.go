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

// FamilyService handles family-related business logic
type FamilyService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
}

const (
	// Family UID generation constants
	familyUIDLength  = 6
	familyUIDCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Excluded confusing chars: 0,O,1,I
)

// NewFamilyService creates a new family service
func NewFamilyService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService) *FamilyService {
	return &FamilyService{
		db:     db,
		logger: logger,
		auth:   auth,
	}
}

// generateUniqueFamilyUID generates a unique family UID
func (s *FamilyService) generateUniqueFamilyUID(ctx context.Context) (string, error) {
	maxAttempts := 10
	for range maxAttempts {
		// Generate random UID using crypto/rand
		familyUIDBytes := make([]byte, familyUIDLength)
		for j := range familyUIDBytes {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(familyUIDCharset))))
			if err != nil {
				return "", fmt.Errorf("failed to generate random UID: %w", err)
			}
			familyUIDBytes[j] = familyUIDCharset[n.Int64()]
		}
		uid := string(familyUIDBytes)

		// Check if UID already exists
		var count int64
		if err := s.db.WithContext(ctx).Model(&models.Families{}).
			Where("uid = ?", uid).Count(&count).Error; err != nil {
			return "", err
		}

		if count == 0 {
			return uid, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique family UID after %d attempts", maxAttempts)
}

// FindFamily retrieves a family by UID without authorization checks.
// Used during Telegram registration to verify family existence before joining.
func (s *FamilyService) FindFamily(ctx context.Context, familyUID string) (*models.Families, error) {
	var family models.Families
	err := s.db.WithContext(ctx).Where("uid = ?", familyUID).First(&family).Error
	return &family, err
}

// GetFamily retrieves family information with business rule enforcement
func (s *FamilyService) GetFamily(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID string,
) (*models.Families, error) {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "family", "read", familyUID); err != nil {
		return nil, err
	}

	var family models.Families
	err := s.db.WithContext(ctx).Where("uid = ?", familyUID).First(&family).Error
	return &family, err
}

// CreateFamily creates a new family with auto-generated UID
func (s *FamilyService) CreateFamily(ctx context.Context, family *models.Families) error {
	// Generate unique UID if not provided
	if family.UID == "" {
		uid, err := s.generateUniqueFamilyUID(ctx)
		if err != nil {
			return err
		}
		family.UID = uid
	}

	return s.db.WithContext(ctx).Create(family).Error
}

// CreateFamilyWithAuth creates a family with authorization check.
// Uses authCtx.FamilyUID for both sides of the family-scoping check,
// so this effectively checks "does this role have family:create permission?"
// without scoping to a specific family (since we're creating a new one).
func (s *FamilyService) CreateFamilyWithAuth(ctx context.Context, authCtx *AuthContext, family *models.Families) error {
	if err := s.auth.RequirePermission(authCtx, "family", "create", authCtx.FamilyUID); err != nil {
		return err
	}
	return s.CreateFamily(ctx, family)
}

// UpdateFamily updates family information with business rule enforcement
func (s *FamilyService) UpdateFamily(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID string,
	updates *UpdateFamilyRequest,
) (*models.Families, error) {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "family", "update", familyUID); err != nil {
		return nil, err
	}

	if err := updates.Validate(); err != nil {
		return nil, err
	}

	var family models.Families
	err := s.db.WithContext(ctx).Where("uid = ?", familyUID).First(&family).Error
	if err != nil {
		return nil, err
	}

	// Use Select to specify exactly which fields can be updated
	err = s.db.WithContext(ctx).Model(&family).Select("name").Updates(map[string]any{
		"name": updates.Name,
	}).Error
	if err != nil {
		return nil, err
	}

	// Re-read to return current state
	if err := s.db.WithContext(ctx).First(&family, family.ID).Error; err != nil {
		return nil, err
	}

	return &family, nil
}
