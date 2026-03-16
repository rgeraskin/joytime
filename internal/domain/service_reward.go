package domain

import (
	"context"

	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// RewardService handles reward-related business logic
type RewardService struct {
	db   *gorm.DB
	auth *CasbinAuthService
}

// NewRewardService creates a new reward service
func NewRewardService(db *gorm.DB, auth *CasbinAuthService) *RewardService {
	return &RewardService{
		db:   db,
		auth: auth,
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

	if err := ValidateEntityCreate(reward.Name, reward.Description, reward.Tokens); err != nil {
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
	err := s.db.WithContext(ctx).
		Where("family_uid = ?", familyUID).
		Order("tokens DESC").
		Limit(maxListResults).
		Find(&rewards).Error
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

	return findByFamilyAndName[models.Rewards](s.db, ctx, familyUID, rewardName)
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

	if err := updates.Validate(); err != nil {
		return nil, err
	}

	reward, err := findByFamilyAndName[models.Rewards](s.db, ctx, familyUID, rewardName)
	if err != nil {
		return nil, err
	}

	updateFields := make(UpdateFields)
	updateFields.AddStringIfNotEmpty("name", updates.Name)
	updateFields.AddStringIfNotEmpty("description", updates.Description)
	updateFields.AddIntIfSet("tokens", updates.Tokens)

	if len(updateFields) > 0 {
		if err := updateAndReload(s.db, ctx, reward, reward.ID, updateFields); err != nil {
			return nil, err
		}
	}

	return reward, nil
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

	return deleteByFamilyAndName[models.Rewards](s.db, ctx, familyUID, rewardName)
}
