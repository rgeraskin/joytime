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
	"time"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateCompletedTaskBlocked(t *testing.T) {
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

	// Create and fully complete a task
	task := &models.Tasks{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "Completed Task",
			Tokens:    10,
		},
		AssignedToUserID: child.UserID,
	}
	err := testHandler.services.TaskService.CreateTask(context.Background(), parentCtx, task)
	require.NoError(t, err)

	// Child submits for review
	_, err = testHandler.services.TaskService.CompleteTask(context.Background(), childCtx, family.UID, "Completed Task")
	require.NoError(t, err)

	// Parent approves — task resets to "new" (repeatable chores)
	approved, err := testHandler.services.TaskService.CompleteTask(context.Background(), parentCtx, family.UID, "Completed Task")
	require.NoError(t, err)

	t.Run("Task resets to new after parent approval", func(t *testing.T) {
		assert.Equal(t, domain.TaskStatusNew, approved.Status)
		assert.Empty(t, approved.AssignedToUserID)
	})

	t.Run("Task can be completed again after reset", func(t *testing.T) {
		// Child submits again
		resubmitted, err := testHandler.services.TaskService.CompleteTask(context.Background(), childCtx, family.UID, "Completed Task")
		require.NoError(t, err)
		assert.Equal(t, domain.TaskStatusCheck, resubmitted.Status)
	})
}

func TestNegativeBalanceBlocked(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	t.Run("Cannot deduct more tokens than balance", func(t *testing.T) {
		// Child has 50 tokens from setup
		err := testHandler.services.TokenService.AddTokensToUser(
			context.Background(), parentCtx, child.UserID,
			-51, domain.TokenTypeManualAdjustment, "Over-deduction", nil, nil,
		)
		assert.ErrorIs(t, err, domain.ErrInsufficientTokens)

		// Verify balance unchanged
		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(), parentCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, 50, tokens.Tokens)
	})

	t.Run("Can deduct exact balance", func(t *testing.T) {
		err := testHandler.services.TokenService.AddTokensToUser(
			context.Background(), parentCtx, child.UserID,
			-50, domain.TokenTypeManualAdjustment, "Exact deduction", nil, nil,
		)
		assert.NoError(t, err)

		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(), parentCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, 0, tokens.Tokens)
	})
}

func TestUpdateValidation(t *testing.T) {
	setupTestDB(t)
	family, parent, _, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	// Create a task and reward for testing updates
	task := &models.Tasks{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "Validation Task",
			Tokens:    10,
		},
	}
	err := testHandler.services.TaskService.CreateTask(context.Background(), parentCtx, task)
	require.NoError(t, err)

	reward := &models.Rewards{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "Validation Reward",
			Tokens:    10,
		},
	}
	err = testHandler.services.RewardService.CreateReward(context.Background(), parentCtx, reward)
	require.NoError(t, err)

	t.Run("Task update rejects name too long", func(t *testing.T) {
		longName := string(make([]byte, 101))
		updates := &domain.UpdateTaskRequest{Name: longName}
		_, err := testHandler.services.TaskService.UpdateTask(
			context.Background(), parentCtx, family.UID, "Validation Task", updates,
		)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("Task update rejects invalid status", func(t *testing.T) {
		updates := &domain.UpdateTaskRequest{Status: "invalid"}
		_, err := testHandler.services.TaskService.UpdateTask(
			context.Background(), parentCtx, family.UID, "Validation Task", updates,
		)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("Task update rejects negative tokens", func(t *testing.T) {
		neg := -1
		updates := &domain.UpdateTaskRequest{Tokens: &neg}
		_, err := testHandler.services.TaskService.UpdateTask(
			context.Background(), parentCtx, family.UID, "Validation Task", updates,
		)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("Reward update rejects tokens over max", func(t *testing.T) {
		over := 1001
		updates := &domain.UpdateRewardRequest{Tokens: &over}
		_, err := testHandler.services.RewardService.UpdateReward(
			context.Background(), parentCtx, family.UID, "Validation Reward", updates,
		)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("User update rejects invalid role", func(t *testing.T) {
		updates := &domain.UpdateUserRequest{Role: "admin"}
		_, err := testHandler.services.UserService.UpdateUser(
			context.Background(), parentCtx, parent.UserID, updates,
		)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("Family update rejects empty name", func(t *testing.T) {
		updates := &domain.UpdateFamilyRequest{Name: ""}
		_, err := testHandler.services.FamilyService.UpdateFamily(
			context.Background(), parentCtx, family.UID, updates,
		)
		assert.ErrorIs(t, err, domain.ErrValidation)
	})
}

func TestRewardClaimHTTP(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	// Create a reward
	reward := &models.Rewards{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "HTTP Claim Reward",
			Tokens:    20,
		},
	}
	err := testHandler.services.RewardService.CreateReward(context.Background(), parentCtx, reward)
	require.NoError(t, err)

	t.Run("Child can claim reward via HTTP", func(t *testing.T) {
		encodedName := url.PathEscape("HTTP Claim Reward")
		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/rewards/%s/%s", family.UID, encodedName),
			bytes.NewReader([]byte("{}")))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify tokens deducted
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}
		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(), childCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, 30, tokens.Tokens) // 50 - 20
	})

	t.Run("Claim with insufficient tokens returns 400", func(t *testing.T) {
		// Create an expensive reward
		expReward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID: family.UID,
				Name:      "Expensive HTTP Reward",
				Tokens:    1000,
			},
		}
		err := testHandler.services.RewardService.CreateReward(context.Background(), parentCtx, expReward)
		require.NoError(t, err)

		encodedName := url.PathEscape("Expensive HTTP Reward")
		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/rewards/%s/%s", family.UID, encodedName),
			bytes.NewReader([]byte("{}")))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp ErrorResponse
		err = json.NewDecoder(w.Body).Decode(&errResp)
		assert.NoError(t, err)
		assert.Contains(t, errResp.Error, "insufficient tokens")
	})

	t.Run("Claim nonexistent reward returns 404", func(t *testing.T) {
		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/rewards/%s/NonexistentReward", family.UID),
			bytes.NewReader([]byte("{}")))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handleRewardsByFamily(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTokenHistoryAccess(t *testing.T) {
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

	// Child already has 50 tokens from setup, but no history entry was created
	// Add tokens via service to create history
	err := testHandler.services.TokenService.AddTokensToUser(
		context.Background(), parentCtx, child.UserID,
		50, domain.TokenTypeManualAdjustment, "Initial bonus", nil, nil,
	)
	require.NoError(t, err)

	t.Run("Child can read own history", func(t *testing.T) {
		history, err := testHandler.services.TokenService.GetTokenHistory(context.Background(), childCtx, child.UserID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(history), 1)
		assert.Equal(t, 50, history[0].Amount)
		assert.Equal(t, domain.TokenTypeManualAdjustment, history[0].Type)
	})

	t.Run("Parent can read child history", func(t *testing.T) {
		history, err := testHandler.services.TokenService.GetTokenHistory(context.Background(), parentCtx, child.UserID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(history), 1)
	})

	t.Run("Child cannot read other user history", func(t *testing.T) {
		_, err := testHandler.services.TokenService.GetTokenHistory(context.Background(), childCtx, parent.UserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("History ordered by created_at DESC", func(t *testing.T) {
		// Add more tokens to create a second entry
		err := testHandler.services.TokenService.AddTokensToUser(
			context.Background(), parentCtx, child.UserID,
			10, domain.TokenTypeManualAdjustment, "Second bonus", nil, nil,
		)
		require.NoError(t, err)

		history, err := testHandler.services.TokenService.GetTokenHistory(context.Background(), parentCtx, child.UserID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(history), 2)
		assert.True(t, history[0].CreatedAt.After(history[1].CreatedAt) || history[0].CreatedAt.Equal(history[1].CreatedAt))
	})
}

func TestFamilyMemberManagement(t *testing.T) {
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

	t.Run("Parent can rename child", func(t *testing.T) {
		updates := &domain.UpdateUserRequest{Name: "New Name"}
		updated, err := testHandler.services.UserService.UpdateUser(context.Background(), parentCtx, child.UserID, updates)
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
	})

	t.Run("Parent cannot delete self", func(t *testing.T) {
		err := testHandler.services.UserService.DeleteUser(context.Background(), parentCtx, parent.UserID)
		assert.ErrorIs(t, err, domain.ErrCannotDeleteSelf)
	})

	t.Run("Parent can delete child", func(t *testing.T) {
		// Create a second child to delete
		secondChild := &models.Users{
			UserID:    fmt.Sprintf("child2_%s_%d", t.Name(), time.Now().UnixNano()),
			Name:      "Second Child",
			Role:      "child",
			FamilyUID: family.UID,
			Platform:  "telegram",
		}
		require.NoError(t, testDB.Create(secondChild).Error)

		err := testHandler.services.UserService.DeleteUser(context.Background(), parentCtx, secondChild.UserID)
		assert.NoError(t, err)

		// Verify user no longer found
		_, err = testHandler.services.UserService.FindUser(context.Background(), secondChild.UserID)
		assert.Error(t, err)
	})

	t.Run("Child cannot delete other users", func(t *testing.T) {
		err := testHandler.services.UserService.DeleteUser(context.Background(), childCtx, parent.UserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child cannot rename other users", func(t *testing.T) {
		updates := &domain.UpdateUserRequest{Name: "Hacked"}
		_, err := testHandler.services.UserService.UpdateUser(context.Background(), childCtx, parent.UserID, updates)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

func TestEntitiesSortedByTokensDescending(t *testing.T) {
	setupTestDB(t)
	family, parent, _, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	// Create tasks with different token values
	for _, tc := range []struct{ name string; tokens int }{
		{"Low Task", 5}, {"High Task", 20}, {"Mid Task", 10},
	} {
		task := &models.Tasks{Entities: models.Entities{FamilyUID: family.UID, Name: tc.name, Tokens: tc.tokens}}
		require.NoError(t, testHandler.services.TaskService.CreateTask(context.Background(), parentCtx, task))
	}

	// Create rewards with different token values
	for _, rc := range []struct{ name string; tokens int }{
		{"Cheap Reward", 3}, {"Expensive Reward", 15}, {"Mid Reward", 7},
	} {
		reward := &models.Rewards{Entities: models.Entities{FamilyUID: family.UID, Name: rc.name, Tokens: rc.tokens}}
		require.NoError(t, testHandler.services.RewardService.CreateReward(context.Background(), parentCtx, reward))
	}

	t.Run("Tasks sorted by tokens descending", func(t *testing.T) {
		tasks, err := testHandler.services.TaskService.GetTasksForFamily(context.Background(), parentCtx, family.UID)
		require.NoError(t, err)
		require.Len(t, tasks, 3)
		assert.Equal(t, 20, tasks[0].Tokens)
		assert.Equal(t, 10, tasks[1].Tokens)
		assert.Equal(t, 5, tasks[2].Tokens)
	})

	t.Run("Rewards sorted by tokens descending", func(t *testing.T) {
		rewards, err := testHandler.services.RewardService.GetRewardsForFamily(context.Background(), parentCtx, family.UID)
		require.NoError(t, err)
		require.Len(t, rewards, 3)
		assert.Equal(t, 15, rewards[0].Tokens)
		assert.Equal(t, 7, rewards[1].Tokens)
		assert.Equal(t, 3, rewards[2].Tokens)
	})
}

func TestParseFamilyEntityPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		prefix     string
		wantFamily string
		wantEntity string
		wantErr    bool
	}{
		{"family only", "/api/v1/tasks/FAM123", "/api/v1/tasks/", "FAM123", "", false},
		{"family and entity", "/api/v1/tasks/FAM123/Do+Dishes", "/api/v1/tasks/", "FAM123", "Do Dishes", false},
		{"empty after prefix", "/api/v1/tasks/", "/api/v1/tasks/", "", "", false},
		{"encoded entity", "/api/v1/rewards/FAM/My%20Reward", "/api/v1/rewards/", "FAM", "My Reward", false},
		{"trailing slash ignored", "/api/v1/tasks/FAM123/", "/api/v1/tasks/", "FAM123", "", false},
		{"bad encoding", "/api/v1/tasks/FAM/%zz", "/api/v1/tasks/", "FAM", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			family, entity, err := parseFamilyEntityPath(tt.path, tt.prefix)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantFamily, family)
			assert.Equal(t, tt.wantEntity, entity)
		})
	}
}

func TestRespondServiceError(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name         string
		err          error
		wantStatus   int
		wantContains string
	}{
		{"unauthorized", domain.ErrUnauthorized, http.StatusForbidden, "access denied"},
		{"not found", gorm.ErrRecordNotFound, http.StatusNotFound, ErrEntityNotFound},
		{"insufficient tokens", domain.ErrInsufficientTokens, http.StatusBadRequest, ErrInsufficientTokens},
		{"validation error", domain.ErrValidation, http.StatusBadRequest, "validation error"},
		{"cannot delete self", domain.ErrCannotDeleteSelf, http.StatusBadRequest, "cannot delete yourself"},
		{"task already completed", domain.ErrTaskAlreadyCompleted, http.StatusBadRequest, "already completed"},
		{"unknown error", fmt.Errorf("something broke"), http.StatusInternalServerError, "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testHandler.respondServiceError(w, tt.err, "fallback")
			assert.Equal(t, tt.wantStatus, w.Code)

			var resp ErrorResponse
			err := json.NewDecoder(w.Body).Decode(&resp)
			assert.NoError(t, err)
			assert.Contains(t, resp.Error, tt.wantContains)
		})
	}
}
