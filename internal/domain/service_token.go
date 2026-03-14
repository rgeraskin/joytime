package domain

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// TokenService handles token-related business logic
type TokenService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
}

// NewTokenService creates a new token service
func NewTokenService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService) *TokenService {
	return &TokenService{
		db:     db,
		logger: logger,
		auth:   auth,
	}
}

// AddTokensToUser adds tokens to a user with business rule enforcement
func (s *TokenService) AddTokensToUser(
	ctx context.Context,
	authCtx *AuthContext,
	targetUserID string,
	amount int,
	tokenType, description string,
	taskID, rewardID *uint,
) error {
	// Get target user to check family and authorization
	var targetUser models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", targetUserID).First(&targetUser).Error
	if err != nil {
		return err
	}

	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tokens", "add", targetUser.FamilyUID); err != nil {
		return err
	}

	return s.addTokens(ctx, targetUserID, amount, tokenType, description, taskID, rewardID)
}

// addTokens is an internal method that modifies tokens without permission checks.
// Used by authorized high-level operations (e.g. ClaimReward) that have already
// verified permissions at their own level.
func (s *TokenService) addTokens(
	ctx context.Context,
	targetUserID string,
	amount int,
	tokenType, description string,
	taskID, rewardID *uint,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current tokens
		var tokens models.Tokens
		err := tx.Where("user_id = ?", targetUserID).First(&tokens).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new token record
				tokens = models.Tokens{
					UserID: targetUserID,
					Tokens: 0,
				}
			} else {
				return err
			}
		}

		// Update tokens
		tokens.Tokens += amount
		if err := tx.Save(&tokens).Error; err != nil {
			return err
		}

		// Create history record
		history := models.TokenHistory{
			UserID:      targetUserID,
			Amount:      amount,
			Type:        tokenType,
			Description: description,
			TaskID:      taskID,
			RewardID:    rewardID,
		}
		return tx.Create(&history).Error
	})
}

// GetUserTokens retrieves token balance for a user
func (s *TokenService) GetUserTokens(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) (*models.Tokens, error) {
	// Get target user to check family and authorization
	var targetUser models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&targetUser).Error
	if err != nil {
		return nil, err
	}

	// Check permission and family access using Casbin
	if authCtx.UserID == userID {
		// User accessing their own tokens
		if err := s.auth.RequirePermission(authCtx, "tokens", "read", targetUser.FamilyUID); err != nil {
			return nil, err
		}
	} else {
		// Parent accessing other user's tokens
		if err := s.auth.RequirePermission(authCtx, "tokens", "read_others", targetUser.FamilyUID); err != nil {
			return nil, err
		}
	}

	var tokens models.Tokens
	err = s.db.WithContext(ctx).Where("user_id = ?", userID).First(&tokens).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return zero tokens if no record exists
			return &models.Tokens{
				UserID: userID,
				Tokens: 0,
			}, nil
		}
		return nil, err
	}

	return &tokens, nil
}

// GetTokenHistory retrieves token history for a user
func (s *TokenService) GetTokenHistory(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) ([]models.TokenHistory, error) {
	// Get target user to check family and authorization
	var targetUser models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&targetUser).Error
	if err != nil {
		return nil, err
	}

	// Check permission and family access using Casbin
	if authCtx.UserID == userID {
		// User accessing their own history
		if err := s.auth.RequirePermission(authCtx, "tokens", "read", targetUser.FamilyUID); err != nil {
			return nil, err
		}
	} else {
		// Parent accessing other user's history
		if err := s.auth.RequirePermission(authCtx, "tokens", "read_others", targetUser.FamilyUID); err != nil {
			return nil, err
		}
	}

	var history []models.TokenHistory
	err = s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&history).
		Error
	return history, err
}

// ClaimReward processes a reward claim
func (s *TokenService) ClaimReward(
	ctx context.Context,
	authCtx *AuthContext,
	rewardID uint,
) error {
	// Get reward details to check family and authorization
	var reward models.Rewards
	err := s.db.WithContext(ctx).Where("id = ?", rewardID).First(&reward).Error
	if err != nil {
		return err
	}

	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "rewards", "claim", reward.FamilyUID); err != nil {
		return err
	}

	// Business Rule: User must have enough tokens
	userTokens, err := s.GetUserTokens(ctx, authCtx, authCtx.UserID)
	if err != nil {
		return err
	}

	if userTokens.Tokens < reward.Tokens {
		return ErrInsufficientTokens
	}

	// Deduct tokens (using internal method — permission already checked via rewards:claim)
	return s.addTokens(
		ctx,
		authCtx.UserID,
		-reward.Tokens,
		"reward_claimed",
		"Claimed reward: "+reward.Name,
		nil,
		&rewardID,
	)
}
