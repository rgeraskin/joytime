package handlers

import (
	"fmt"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
)

// sanitizeInput trims whitespace from user input
func sanitizeInput(input string) string {
	return strings.TrimSpace(input)
}

func sanitizeFamily(family *models.Families) {
	family.Name = sanitizeInput(family.Name)
	family.UID = sanitizeInput(family.UID)
	family.CreatedByUserID = sanitizeInput(family.CreatedByUserID)
}

func sanitizeTask(task *models.Tasks) {
	task.Name = sanitizeInput(task.Name)
	task.Description = sanitizeInput(task.Description)
	task.FamilyUID = sanitizeInput(task.FamilyUID)
	task.Status = sanitizeInput(task.Status)
}

func sanitizeReward(reward *models.Rewards) {
	reward.Name = sanitizeInput(reward.Name)
	reward.Description = sanitizeInput(reward.Description)
	reward.FamilyUID = sanitizeInput(reward.FamilyUID)
}

func sanitizePenalty(penalty *models.Penalties) {
	penalty.Name = sanitizeInput(penalty.Name)
	penalty.Description = sanitizeInput(penalty.Description)
	penalty.FamilyUID = sanitizeInput(penalty.FamilyUID)
}

func sanitizeTokenAddRequest(req *TokenAddRequest) {
	req.UserID = sanitizeInput(req.UserID)
	req.Type = sanitizeInput(req.Type)
	req.Description = sanitizeInput(req.Description)
}

// validateFamilyCreate sanitizes and validates a family creation request.
func validateFamilyCreate(family *models.Families) error {
	sanitizeFamily(family)
	return domain.ValidateFamilyCreate(family.Name, family.UID, family.CreatedByUserID)
}

// validateTaskCreate sanitizes and validates a task creation request.
func validateTaskCreate(task *models.Tasks) error {
	sanitizeTask(task)
	if task.FamilyUID == "" {
		return fmt.Errorf("%w: family_uid is required", domain.ErrValidation)
	}
	return domain.ValidateEntityCreate(task.Name, task.Description, task.Tokens)
}

// validateRewardCreate sanitizes and validates a reward creation request.
func validateRewardCreate(reward *models.Rewards) error {
	sanitizeReward(reward)
	if reward.FamilyUID == "" {
		return fmt.Errorf("%w: family_uid is required", domain.ErrValidation)
	}
	return domain.ValidateEntityCreate(reward.Name, reward.Description, reward.Tokens)
}

// validatePenaltyCreate sanitizes and validates a penalty creation request.
func validatePenaltyCreate(penalty *models.Penalties) error {
	sanitizePenalty(penalty)
	if penalty.FamilyUID == "" {
		return fmt.Errorf("%w: family_uid is required", domain.ErrValidation)
	}
	return domain.ValidateEntityCreate(penalty.Name, penalty.Description, penalty.Tokens)
}

// validateTokenAddRequest sanitizes and validates a token transaction request.
func validateTokenAddRequest(req *TokenAddRequest) error {
	sanitizeTokenAddRequest(req)
	return domain.ValidateTokenTransaction(req.Amount, req.Type, req.Description)
}
