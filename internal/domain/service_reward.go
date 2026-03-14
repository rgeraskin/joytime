package domain

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// RewardService handles reward-related business logic
type RewardService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
}

// NewRewardService creates a new reward service
func NewRewardService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService) *RewardService {
	return &RewardService{
		db:     db,
		logger: logger,
		auth:   auth,
	}
}

// CreateReward creates a new reward with business rule enforcement
func (s *RewardService) CreateReward(
	ctx context.Context,
	authCtx *AuthContext,
	reward *models.Rewards,
) error {
	if err := s.auth.RequirePermission(authCtx, "rewards", "create", reward.FamilyUID); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Create(reward).Error
}

// GetRewardsForFamily retrieves rewards for a family
func (s *RewardService) GetRewardsForFamily(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID string,
) ([]models.Rewards, error) {
	if err := s.auth.RequirePermission(authCtx, "rewards", "read", familyUID); err != nil {
		return nil, err
	}

	var rewards []models.Rewards
	err := s.db.WithContext(ctx).Where("family_uid = ?", familyUID).Find(&rewards).Error
	return rewards, err
}

// GetReward retrieves a single reward by family and name
func (s *RewardService) GetReward(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, rewardName string,
) (*models.Rewards, error) {
	if err := s.auth.RequirePermission(authCtx, "rewards", "read", familyUID); err != nil {
		return nil, err
	}

	var reward models.Rewards
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		First(&reward).
		Error
	if err != nil {
		return nil, err
	}

	return &reward, nil
}

// UpdateReward updates a reward with business rule enforcement
func (s *RewardService) UpdateReward(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, rewardName string,
	updates *UpdateRewardRequest,
) (*models.Rewards, error) {
	if err := s.auth.RequirePermission(authCtx, "rewards", "update", familyUID); err != nil {
		return nil, err
	}

	var reward models.Rewards
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		First(&reward).
		Error
	if err != nil {
		return nil, err
	}

	updateFields := make(UpdateFields)
	updateFields.AddStringIfNotEmpty("name", updates.Name)
	updateFields.AddStringIfNotEmpty("description", updates.Description)
	updateFields.AddIntIfSet("tokens", updates.Tokens)

	if len(updateFields) > 0 {
		err = s.db.WithContext(ctx).
			Model(&reward).
			Select([]string{"name", "description", "tokens"}).
			Updates(updateFields.ToMap()).
			Error
		if err != nil {
			return nil, err
		}
	}

	return &reward, nil
}

// DeleteReward deletes a reward with business rule enforcement
func (s *RewardService) DeleteReward(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, rewardName string,
) error {
	if err := s.auth.RequirePermission(authCtx, "rewards", "delete", familyUID); err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, rewardName).
		Delete(&models.Rewards{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
