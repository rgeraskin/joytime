package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRewardServiceCRUD tests the full reward CRUD lifecycle through the service layer
func TestRewardServiceCRUD(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	t.Run("Parent can create reward", func(t *testing.T) {
		reward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "Extra Screen Time",
				Description: "30 minutes of extra screen time",
				Tokens:      20,
			},
		}

		err := testHandler.services.RewardService.CreateReward(
			context.Background(), parentCtx, reward,
		)
		assert.NoError(t, err)
		assert.NotZero(t, reward.ID)
	})

	t.Run("Parent can list family rewards", func(t *testing.T) {
		rewards, err := testHandler.services.RewardService.GetRewardsForFamily(
			context.Background(), parentCtx, family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, rewards, 1)
		assert.Equal(t, "Extra Screen Time", rewards[0].Name)
		assert.Equal(t, 20, rewards[0].Tokens)
	})

	t.Run("Parent can update reward", func(t *testing.T) {
		newTokens := 25
		updates := &domain.UpdateRewardRequest{
			Name:   "Extended Screen Time",
			Tokens: &newTokens,
		}

		updated, err := testHandler.services.RewardService.UpdateReward(
			context.Background(), parentCtx, family.UID, "Extra Screen Time", updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Extended Screen Time", updated.Name)
	})

	t.Run("Child can read rewards", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		rewards, err := testHandler.services.RewardService.GetRewardsForFamily(
			context.Background(), childCtx, family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, rewards, 1)
	})

	t.Run("Parent can delete reward", func(t *testing.T) {
		err := testHandler.services.RewardService.DeleteReward(
			context.Background(), parentCtx, family.UID, "Extended Screen Time",
		)
		assert.NoError(t, err)

		// Verify it's gone
		rewards, err := testHandler.services.RewardService.GetRewardsForFamily(
			context.Background(), parentCtx, family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, rewards, 0)
	})

	t.Run("Delete nonexistent reward returns not found", func(t *testing.T) {
		err := testHandler.services.RewardService.DeleteReward(
			context.Background(), parentCtx, family.UID, "Nonexistent Reward",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
	})

	t.Run("Update nonexistent reward returns not found", func(t *testing.T) {
		updates := &domain.UpdateRewardRequest{Name: "Whatever"}
		_, err := testHandler.services.RewardService.UpdateReward(
			context.Background(), parentCtx, family.UID, "Nonexistent Reward", updates,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
	})
}

// TestRewardRBAC tests that RBAC rules are enforced for reward operations
func TestRewardRBAC(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	childCtx := &domain.AuthContext{
		UserID:    child.UserID,
		UserRole:  domain.RoleChild,
		FamilyUID: family.UID,
	}

	// Create a reward for testing child restrictions
	reward := &models.Rewards{
		Entities: models.Entities{
			FamilyUID:   family.UID,
			Name:        "Game Time",
			Description: "1 hour of gaming",
			Tokens:      30,
		},
	}
	err := testHandler.services.RewardService.CreateReward(
		context.Background(), parentCtx, reward,
	)
	require.NoError(t, err)

	t.Run("Child CANNOT create rewards", func(t *testing.T) {
		childReward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "Free Candy",
				Description: "Unlimited candy",
				Tokens:      1,
			},
		}

		err := testHandler.services.RewardService.CreateReward(
			context.Background(), childCtx, childReward,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT update rewards", func(t *testing.T) {
		one := 1
		updates := &domain.UpdateRewardRequest{Tokens: &one}
		_, err := testHandler.services.RewardService.UpdateReward(
			context.Background(), childCtx, family.UID, "Game Time", updates,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT delete rewards", func(t *testing.T) {
		err := testHandler.services.RewardService.DeleteReward(
			context.Background(), childCtx, family.UID, "Game Time",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// TestRewardFamilyIsolation tests that rewards are isolated between families
func TestRewardFamilyIsolation(t *testing.T) {
	setupTestDB(t)
	family1, parent1, _ := setupAuthTestData(t)
	family2, parent2, _ := setupSecondFamily(t)

	parent1Ctx := &domain.AuthContext{
		UserID:    parent1.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family1.UID,
	}

	parent2Ctx := &domain.AuthContext{
		UserID:    parent2.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family2.UID,
	}

	// Create rewards in each family
	reward1 := &models.Rewards{
		Entities: models.Entities{
			FamilyUID: family1.UID,
			Name:      "Family1 Reward",
			Tokens:    10,
		},
	}
	err := testHandler.services.RewardService.CreateReward(
		context.Background(), parent1Ctx, reward1,
	)
	require.NoError(t, err)

	reward2 := &models.Rewards{
		Entities: models.Entities{
			FamilyUID: family2.UID,
			Name:      "Family2 Reward",
			Tokens:    20,
		},
	}
	err = testHandler.services.RewardService.CreateReward(
		context.Background(), parent2Ctx, reward2,
	)
	require.NoError(t, err)

	t.Run("Parent from family1 cannot read family2 rewards", func(t *testing.T) {
		_, err := testHandler.services.RewardService.GetRewardsForFamily(
			context.Background(), parent1Ctx, family2.UID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Parent from family1 cannot update family2 rewards", func(t *testing.T) {
		updates := &domain.UpdateRewardRequest{Name: "Hijacked"}
		_, err := testHandler.services.RewardService.UpdateReward(
			context.Background(), parent1Ctx, family2.UID, "Family2 Reward", updates,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Parent from family1 cannot delete family2 rewards", func(t *testing.T) {
		err := testHandler.services.RewardService.DeleteReward(
			context.Background(), parent1Ctx, family2.UID, "Family2 Reward",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Each family sees only their own rewards", func(t *testing.T) {
		rewards1, err := testHandler.services.RewardService.GetRewardsForFamily(
			context.Background(), parent1Ctx, family1.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, rewards1, 1)
		assert.Equal(t, "Family1 Reward", rewards1[0].Name)

		rewards2, err := testHandler.services.RewardService.GetRewardsForFamily(
			context.Background(), parent2Ctx, family2.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, rewards2, 1)
		assert.Equal(t, "Family2 Reward", rewards2[0].Name)
	})
}

// TestRewardClaimIntegration tests the reward claiming flow with token deduction
func TestRewardClaimIntegration(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	childCtx := &domain.AuthContext{
		UserID:    child.UserID,
		UserRole:  domain.RoleChild,
		FamilyUID: family.UID,
	}

	// Create a reward
	reward := &models.Rewards{
		Entities: models.Entities{
			FamilyUID:   family.UID,
			Name:        "Movie Night",
			Description: "Pick a movie for family night",
			Tokens:      30,
		},
	}
	err := testHandler.services.RewardService.CreateReward(
		context.Background(), parentCtx, reward,
	)
	require.NoError(t, err)

	t.Run("Child can claim reward with sufficient tokens", func(t *testing.T) {
		// Child starts with 50 tokens (from setupServiceTestData)
		err := testHandler.services.TokenService.ClaimReward(
			context.Background(), childCtx, reward.ID,
		)
		assert.NoError(t, err)

		// Verify tokens were deducted
		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(), childCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, 20, tokens.Tokens) // 50 - 30
	})

	t.Run("Token history records the claim", func(t *testing.T) {
		history, err := testHandler.services.TokenService.GetTokenHistory(
			context.Background(), childCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, history)

		// Find the reward claim entry
		var found bool
		for _, h := range history {
			if h.Type == "reward_claimed" {
				found = true
				assert.Equal(t, -30, h.Amount)
				assert.Contains(t, h.Description, "Movie Night")
				assert.NotNil(t, h.RewardID)
				break
			}
		}
		assert.True(t, found, "Expected to find reward_claimed entry in history")
	})

	t.Run("Child cannot claim reward with insufficient tokens", func(t *testing.T) {
		// Create an expensive reward
		expensiveReward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID: family.UID,
				Name:      "New Bicycle",
				Tokens:    1000,
			},
		}
		err := testHandler.services.RewardService.CreateReward(
			context.Background(), parentCtx, expensiveReward,
		)
		require.NoError(t, err)

		err = testHandler.services.TokenService.ClaimReward(
			context.Background(), childCtx, expensiveReward.ID,
		)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInsufficientTokens)
	})
}

// TestRewardHTTPHandlers tests the reward HTTP endpoints end-to-end
func TestRewardHTTPHandlers(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	t.Run("POST /rewards creates a reward", func(t *testing.T) {
		body := map[string]any{
			"family_uid":  family.UID,
			"name":        "Ice Cream Trip",
			"description": "Trip to the ice cream shop",
			"tokens":      15,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/rewards", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var reward models.Rewards
		assertSuccessResponse(t, w, http.StatusCreated, &reward)
		assert.Equal(t, "Ice Cream Trip", reward.Name)
		assert.Equal(t, 15, reward.Tokens)
	})

	t.Run("GET /rewards/{familyUID} lists family rewards", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/rewards/%s", family.UID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var rewards []models.Rewards
		assertSuccessResponse(t, w, http.StatusOK, &rewards)
		assert.Len(t, rewards, 1)
		assert.Equal(t, "Ice Cream Trip", rewards[0].Name)
	})

	t.Run("GET /rewards/{familyUID}/{rewardName} gets single reward", func(t *testing.T) {
		encodedName := url.PathEscape("Ice Cream Trip")
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/rewards/%s/%s", family.UID, encodedName), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var reward models.Rewards
		assertSuccessResponse(t, w, http.StatusOK, &reward)
		assert.Equal(t, "Ice Cream Trip", reward.Name)
	})

	t.Run("PUT /rewards/{familyUID}/{rewardName} updates reward", func(t *testing.T) {
		body := map[string]any{
			"name":   "Gelato Trip",
			"tokens": 20,
		}
		bodyJSON, _ := json.Marshal(body)
		encodedName := url.PathEscape("Ice Cream Trip")

		req := httptest.NewRequest("PUT",
			fmt.Sprintf("/api/v1/rewards/%s/%s", family.UID, encodedName),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("DELETE /rewards/{familyUID}/{rewardName} deletes reward", func(t *testing.T) {
		encodedName := url.PathEscape("Gelato Trip")
		req := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/v1/rewards/%s/%s", family.UID, encodedName), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Child can read rewards via HTTP", func(t *testing.T) {
		// Create a reward first
		reward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID: family.UID,
				Name:      "Sticker",
				Tokens:    5,
			},
		}
		parentCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}
		err := testHandler.services.RewardService.CreateReward(
			context.Background(), parentCtx, reward,
		)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/rewards/%s", family.UID), nil)
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Child cannot create rewards via HTTP", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
			"name":       "Cheat Reward",
			"tokens":     1,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/rewards", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/rewards", nil)
		w := httptest.NewRecorder()

		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid JSON returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/rewards",
			bytes.NewReader([]byte("not json")))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing required fields returns bad request", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
			// missing name and tokens
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/rewards", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET nonexistent reward returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/rewards/%s/NonexistentReward", family.UID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rewards", nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}
