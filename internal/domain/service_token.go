package domain

import (
	"context"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		return s.addTokensInTx(tx, targetUserID, amount, tokenType, description, taskID, rewardID)
	})
}

// addTokensInTx performs token modification within an existing transaction.
// Used when the caller already has a transaction (e.g. CompleteTask wraps
// task status update + token award in one tx).
func (s *TokenService) addTokensInTx(
	tx *gorm.DB,
	targetUserID string,
	amount int,
	tokenType, description string,
	taskID, rewardID *uint,
) error {
	var tokens models.Tokens
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", targetUserID).First(&tokens).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tokens = models.Tokens{
				UserID: targetUserID,
				Tokens: 0,
			}
		} else {
			return err
		}
	}

	tokens.Tokens += amount
	if tokens.Tokens < 0 {
		return ErrInsufficientTokens
	}
	if err := tx.Save(&tokens).Error; err != nil {
		return err
	}

	history := models.TokenHistory{
		UserID:      targetUserID,
		Amount:      amount,
		Type:        tokenType,
		Description: description,
		TaskID:      taskID,
		RewardID:    rewardID,
	}
	return tx.Create(&history).Error
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

// ClaimReward processes a reward claim with atomic balance check + deduction
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

	// Atomic balance check + deduction inside a single transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tokens models.Tokens
		// Lock the row to prevent concurrent claims
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", authCtx.UserID).First(&tokens).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInsufficientTokens
			}
			return err
		}

		if tokens.Tokens < reward.Tokens {
			return ErrInsufficientTokens
		}

		tokens.Tokens -= reward.Tokens
		if err := tx.Save(&tokens).Error; err != nil {
			return err
		}

		history := models.TokenHistory{
			UserID:      authCtx.UserID,
			Amount:      -reward.Tokens,
			Type:        TokenTypeRewardClaimed,
			Description: "Claimed reward: " + reward.Name,
			RewardID:    &rewardID,
		}
		return tx.Create(&history).Error
	})
}
