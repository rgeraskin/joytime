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

	// Parent approves
	_, err = testHandler.services.TaskService.CompleteTask(context.Background(), parentCtx, family.UID, "Completed Task")
	require.NoError(t, err)

	t.Run("Cannot update status of completed task", func(t *testing.T) {
		updates := &domain.UpdateTaskRequest{Status: "new"}
		_, err := testHandler.services.TaskService.UpdateTask(
			context.Background(), parentCtx, family.UID, "Completed Task", updates,
		)
		assert.ErrorIs(t, err, domain.ErrTaskAlreadyCompleted)
	})

	t.Run("Can still update non-status fields of completed task", func(t *testing.T) {
		updates := &domain.UpdateTaskRequest{Description: "Updated description"}
		updated, err := testHandler.services.TaskService.UpdateTask(
			context.Background(), parentCtx, family.UID, "Completed Task", updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Updated description", updated.Description)
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
