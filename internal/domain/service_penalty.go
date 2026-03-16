package domain

import (
	"context"

	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// PenaltyService handles penalty-related business logic
type PenaltyService struct {
	db     *gorm.DB
	auth   *CasbinAuthService
	tokens *TokenService
}

// NewPenaltyService creates a new penalty service
func NewPenaltyService(
	db *gorm.DB,
	auth *CasbinAuthService,
	tokens *TokenService,
) *PenaltyService {
	return &PenaltyService{
		db:     db,
		auth:   auth,
		tokens: tokens,
	}
}

// CreatePenalty creates a new penalty
func (s *PenaltyService) CreatePenalty(
	ctx context.Context,
	authCtx *AuthContext,
	penalty *models.Penalties,
) error {
	if err := s.auth.RequirePermission(authCtx, "penalties", "create", penalty.FamilyUID); err != nil {
		return err
	}

	if err := ValidateEntityCreate(penalty.Name, penalty.Description, penalty.Tokens); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(penalty).Error
}

// GetPenaltiesForFamily retrieves penalties for a family
func (s *PenaltyService) GetPenaltiesForFamily(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID string,
) ([]models.Penalties, error) {
	if err := s.auth.RequirePermission(authCtx, "penalties", "read", familyUID); err != nil {
		return nil, err
	}

	var penalties []models.Penalties
	err := s.db.WithContext(ctx).
		Where("family_uid = ?", familyUID).
		Order("tokens DESC").
		Find(&penalties).
		Error
	return penalties, err
}

// GetPenalty retrieves a single penalty by family and name
func (s *PenaltyService) GetPenalty(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, penaltyName string,
) (*models.Penalties, error) {
	if err := s.auth.RequirePermission(authCtx, "penalties", "read", familyUID); err != nil {
		return nil, err
	}

	return findByFamilyAndName[models.Penalties](s.db, ctx, familyUID, penaltyName)
}

// UpdatePenalty updates a penalty
func (s *PenaltyService) UpdatePenalty(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, penaltyName string,
	updates *UpdatePenaltyRequest,
) (*models.Penalties, error) {
	if err := s.auth.RequirePermission(authCtx, "penalties", "update", familyUID); err != nil {
		return nil, err
	}

	if err := updates.Validate(); err != nil {
		return nil, err
	}

	penalty, err := findByFamilyAndName[models.Penalties](s.db, ctx, familyUID, penaltyName)
	if err != nil {
		return nil, err
	}

	updateFields := make(UpdateFields)
	updateFields.AddStringIfNotEmpty("name", updates.Name)
	updateFields.AddStringIfNotEmpty("description", updates.Description)
	updateFields.AddIntIfSet("tokens", updates.Tokens)

	if len(updateFields) > 0 {
		if err := updateAndReload(s.db, ctx, penalty, penalty.ID, updateFields); err != nil {
			return nil, err
		}
	}

	return penalty, nil
}

// DeletePenalty deletes a penalty
func (s *PenaltyService) DeletePenalty(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, penaltyName string,
) error {
	if err := s.auth.RequirePermission(authCtx, "penalties", "delete", familyUID); err != nil {
		return err
	}

	return deleteByFamilyAndName[models.Penalties](s.db, ctx, familyUID, penaltyName)
}

// ApplyPenalty deducts tokens from a child as a penalty
func (s *PenaltyService) ApplyPenalty(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, penaltyName, childUserID string,
) (*models.Penalties, error) {
	if err := s.auth.RequirePermission(authCtx, "penalties", "apply", familyUID); err != nil {
		return nil, err
	}

	// Verify the child belongs to the same family
	var child models.Users
	if err := s.db.WithContext(ctx).Where("user_id = ?", childUserID).First(&child).Error; err != nil {
		return nil, err
	}
	if child.FamilyUID != familyUID {
		return nil, ErrUnauthorized
	}

	penalty, err := findByFamilyAndName[models.Penalties](s.db, ctx, familyUID, penaltyName)
	if err != nil {
		return nil, err
	}

	penaltyID := penalty.ID
	if _, err := s.tokens.addTokens(
		ctx,
		childUserID,
		-penalty.Tokens,
		TokenTypePenalty,
		HistoryDescPenalty+penalty.Name,
		nil,
		nil,
		&penaltyID,
	); err != nil {
		return nil, err
	}

	return penalty, nil
}
