package domain

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// PenaltyService handles penalty-related business logic
type PenaltyService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
	tokens *TokenService
}

// NewPenaltyService creates a new penalty service
func NewPenaltyService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService, tokens *TokenService) *PenaltyService {
	return &PenaltyService{
		db:     db,
		logger: logger,
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
	err := s.db.WithContext(ctx).Where("family_uid = ?", familyUID).Find(&penalties).Error
	return penalties, err
}

// UpdatePenalty updates a penalty
func (s *PenaltyService) UpdatePenalty(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, penaltyName string,
	updates *UpdateRewardRequest,
) (*models.Penalties, error) {
	if err := s.auth.RequirePermission(authCtx, "penalties", "update", familyUID); err != nil {
		return nil, err
	}

	if err := updates.Validate(); err != nil {
		return nil, err
	}

	var penalty models.Penalties
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, penaltyName).
		First(&penalty).Error
	if err != nil {
		return nil, err
	}

	updateFields := make(UpdateFields)
	updateFields.AddStringIfNotEmpty("name", updates.Name)
	updateFields.AddStringIfNotEmpty("description", updates.Description)
	updateFields.AddIntIfSet("tokens", updates.Tokens)

	if len(updateFields) > 0 {
		err = s.db.WithContext(ctx).
			Model(&penalty).
			Select(updateFields.Keys()).
			Updates(updateFields.ToMap()).
			Error
		if err != nil {
			return nil, err
		}

		if err := s.db.WithContext(ctx).First(&penalty, penalty.ID).Error; err != nil {
			return nil, err
		}
	}

	return &penalty, nil
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

	result := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, penaltyName).
		Delete(&models.Penalties{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
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

	var penalty models.Penalties
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, penaltyName).
		First(&penalty).Error
	if err != nil {
		return nil, err
	}

	if err := s.tokens.addTokens(
		ctx,
		childUserID,
		-penalty.Tokens,
		TokenTypePenalty,
		"Penalty: "+penalty.Name,
		nil,
		nil,
	); err != nil {
		return nil, err
	}

	return &penalty, nil
}
