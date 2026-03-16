package domain

import (
	"context"
	"errors"

	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TokenService handles token-related business logic
type TokenService struct {
	db   *gorm.DB
	auth *CasbinAuthService
}

// NewTokenService creates a new token service
func NewTokenService(db *gorm.DB, auth *CasbinAuthService) *TokenService {
	return &TokenService{
		db:   db,
		auth: auth,
	}
}

// AddTokensToUser adds tokens to a user with business rule enforcement.
// Returns the updated token balance.
func (s *TokenService) AddTokensToUser(
	ctx context.Context,
	authCtx *AuthContext,
	targetUserID string,
	amount int,
	tokenType, description string,
	taskID, rewardID, penaltyID *uint,
) (*models.Tokens, error) {
	// Get target user to check family and authorization
	var targetUser models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", targetUserID).First(&targetUser).Error
	if err != nil {
		return nil, err
	}

	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "tokens", "add", targetUser.FamilyUID); err != nil {
		return nil, err
	}

	return s.addTokens(ctx, targetUserID, amount, tokenType, description, taskID, rewardID, penaltyID)
}

// addTokens is an internal method that modifies tokens without permission checks.
// Used by authorized high-level operations (e.g. ClaimReward) that have already
// verified permissions at their own level.
func (s *TokenService) addTokens(
	ctx context.Context,
	targetUserID string,
	amount int,
	tokenType, description string,
	taskID, rewardID, penaltyID *uint,
) (*models.Tokens, error) {
	var result *models.Tokens
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tokens, err := s.addTokensInTx(tx, targetUserID, amount, tokenType, description, taskID, rewardID, penaltyID)
		if err != nil {
			return err
		}
		result = tokens
		return nil
	})
	return result, err
}

// addTokensInTx performs token modification within an existing transaction.
// Used when the caller already has a transaction (e.g. CompleteTask wraps
// task status update + token award in one tx).
func (s *TokenService) addTokensInTx(
	tx *gorm.DB,
	targetUserID string,
	amount int,
	tokenType, description string,
	taskID, rewardID, penaltyID *uint,
) (*models.Tokens, error) {
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
			return nil, err
		}
	}

	if tokens.Tokens+amount < 0 {
		return nil, ErrInsufficientTokens
	}
	tokens.Tokens += amount
	if err := tx.Save(&tokens).Error; err != nil {
		return nil, err
	}

	history := models.TokenHistory{
		UserID:      targetUserID,
		Amount:      amount,
		Type:        tokenType,
		Description: description,
		TaskID:      taskID,
		RewardID:    rewardID,
		PenaltyID:   penaltyID,
	}
	if err := tx.Create(&history).Error; err != nil {
		return nil, err
	}
	return &tokens, nil
}

// requireTokenReadPermission checks that authCtx may read the given user's tokens.
func (s *TokenService) requireTokenReadPermission(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) error {
	var targetUser models.Users
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&targetUser).Error; err != nil {
		return err
	}

	action := "read"
	if authCtx.UserID != userID {
		action = "read_others"
	}
	return s.auth.RequirePermission(authCtx, "tokens", action, targetUser.FamilyUID)
}

// GetUserTokens retrieves token balance for a user
func (s *TokenService) GetUserTokens(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) (*models.Tokens, error) {
	if err := s.requireTokenReadPermission(ctx, authCtx, userID); err != nil {
		return nil, err
	}

	var tokens models.Tokens
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&tokens).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.Tokens{
				UserID: userID,
				Tokens: 0,
			}, nil
		}
		return nil, err
	}

	return &tokens, nil
}

// maxHistoryRows is the maximum number of token history entries returned per query.
const maxHistoryRows = 100

// GetTokenHistory retrieves token history for a user (most recent first, capped).
func (s *TokenService) GetTokenHistory(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) ([]models.TokenHistory, error) {
	if err := s.requireTokenReadPermission(ctx, authCtx, userID); err != nil {
		return nil, err
	}

	var history []models.TokenHistory
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(maxHistoryRows).
		Find(&history).
		Error
	return history, err
}

// ClaimReward processes a reward claim with atomic balance check + deduction.
// Re-reads the reward inside the transaction to prevent TOCTOU races
// (e.g. a parent updating the reward cost between fetch and claim).
func (s *TokenService) ClaimReward(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID, rewardName string,
) error {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "rewards", "claim", familyUID); err != nil {
		return err
	}

	// Atomic re-read + balance check + deduction inside a single transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reward models.Rewards
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("family_uid = ? AND name = ?", familyUID, rewardName).
			First(&reward).Error; err != nil {
			return err
		}

		rewardID := reward.ID
		_, err := s.addTokensInTx(
			tx,
			authCtx.UserID,
			-reward.Tokens,
			TokenTypeRewardClaimed,
			HistoryDescReward+reward.Name,
			nil,
			&rewardID,
			nil,
		)
		return err
	})
}
