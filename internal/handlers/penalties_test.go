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

// TestPenaltyServiceCRUD tests the full penalty CRUD lifecycle through the service layer
func TestPenaltyServiceCRUD(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	t.Run("Parent can create penalty", func(t *testing.T) {
		penalty := &models.Penalties{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "No Screen Time",
				Description: "Lost screen time privileges",
				Tokens:      10,
			},
		}

		err := testHandler.services.PenaltyService.CreatePenalty(
			context.Background(), parentCtx, penalty,
		)
		assert.NoError(t, err)
		assert.NotZero(t, penalty.ID)
	})

	t.Run("Parent can list family penalties", func(t *testing.T) {
		penalties, err := testHandler.services.PenaltyService.GetPenaltiesForFamily(
			context.Background(), parentCtx, family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, penalties, 1)
		assert.Equal(t, "No Screen Time", penalties[0].Name)
		assert.Equal(t, 10, penalties[0].Tokens)
	})

	t.Run("Parent can get single penalty", func(t *testing.T) {
		penalty, err := testHandler.services.PenaltyService.GetPenalty(
			context.Background(), parentCtx, family.UID, "No Screen Time",
		)
		assert.NoError(t, err)
		assert.Equal(t, "No Screen Time", penalty.Name)
	})

	t.Run("Parent can update penalty", func(t *testing.T) {
		newTokens := 15
		updates := &domain.UpdatePenaltyRequest{
			Name:   "Extended No Screen Time",
			Tokens: &newTokens,
		}

		updated, err := testHandler.services.PenaltyService.UpdatePenalty(
			context.Background(), parentCtx, family.UID, "No Screen Time", updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Extended No Screen Time", updated.Name)
		assert.Equal(t, 15, updated.Tokens)
	})

	t.Run("Child can read penalties", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		penalties, err := testHandler.services.PenaltyService.GetPenaltiesForFamily(
			context.Background(), childCtx, family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, penalties, 1)
	})

	t.Run("Parent can delete penalty", func(t *testing.T) {
		err := testHandler.services.PenaltyService.DeletePenalty(
			context.Background(), parentCtx, family.UID, "Extended No Screen Time",
		)
		assert.NoError(t, err)

		penalties, err := testHandler.services.PenaltyService.GetPenaltiesForFamily(
			context.Background(), parentCtx, family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, penalties, 0)
	})

	t.Run("Delete nonexistent penalty returns not found", func(t *testing.T) {
		err := testHandler.services.PenaltyService.DeletePenalty(
			context.Background(), parentCtx, family.UID, "Nonexistent",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
	})
}

// TestPenaltyRBAC tests that RBAC rules are enforced for penalty operations
func TestPenaltyRBAC(t *testing.T) {
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

	// Create a penalty for testing child restrictions
	penalty := &models.Penalties{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "Bad Behavior",
			Tokens:    20,
		},
	}
	err := testHandler.services.PenaltyService.CreatePenalty(
		context.Background(), parentCtx, penalty,
	)
	require.NoError(t, err)

	t.Run("Child CANNOT create penalties", func(t *testing.T) {
		childPenalty := &models.Penalties{
			Entities: models.Entities{
				FamilyUID: family.UID,
				Name:      "Self Penalty",
				Tokens:    1,
			},
		}

		err := testHandler.services.PenaltyService.CreatePenalty(
			context.Background(), childCtx, childPenalty,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT update penalties", func(t *testing.T) {
		one := 1
		updates := &domain.UpdatePenaltyRequest{Tokens: &one}
		_, err := testHandler.services.PenaltyService.UpdatePenalty(
			context.Background(), childCtx, family.UID, "Bad Behavior", updates,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT delete penalties", func(t *testing.T) {
		err := testHandler.services.PenaltyService.DeletePenalty(
			context.Background(), childCtx, family.UID, "Bad Behavior",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT apply penalties", func(t *testing.T) {
		_, err := testHandler.services.PenaltyService.ApplyPenalty(
			context.Background(), childCtx, family.UID, "Bad Behavior", child.UserID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// TestApplyPenaltyIntegration tests the penalty application flow with token deduction
func TestApplyPenaltyIntegration(t *testing.T) {
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

	penalty := &models.Penalties{
		Entities: models.Entities{
			FamilyUID: family.UID,
			Name:      "Messy Room",
			Tokens:    15,
		},
	}
	err := testHandler.services.PenaltyService.CreatePenalty(
		context.Background(), parentCtx, penalty,
	)
	require.NoError(t, err)

	t.Run("Applying penalty deducts tokens from child", func(t *testing.T) {
		// Child starts with 50 tokens (from setupServiceTestData)
		result, err := testHandler.services.PenaltyService.ApplyPenalty(
			context.Background(), parentCtx, family.UID, "Messy Room", child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Messy Room", result.Name)

		// Verify tokens were deducted
		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(), childCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, 35, tokens.Tokens) // 50 - 15
	})

	t.Run("Token history records the penalty", func(t *testing.T) {
		history, err := testHandler.services.TokenService.GetTokenHistory(
			context.Background(), childCtx, child.UserID,
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, history)

		var found bool
		for _, h := range history {
			if h.Type == domain.TokenTypePenalty {
				found = true
				assert.Equal(t, -15, h.Amount)
				assert.Contains(t, h.Description, "Messy Room")
				assert.NotNil(t, h.PenaltyID)
				break
			}
		}
		assert.True(t, found, "Expected to find penalty entry in history")
	})

	t.Run("Penalty fails with insufficient tokens", func(t *testing.T) {
		bigPenalty := &models.Penalties{
			Entities: models.Entities{
				FamilyUID: family.UID,
				Name:      "Huge Penalty",
				Tokens:    1000,
			},
		}
		err := testHandler.services.PenaltyService.CreatePenalty(
			context.Background(), parentCtx, bigPenalty,
		)
		require.NoError(t, err)

		_, err = testHandler.services.PenaltyService.ApplyPenalty(
			context.Background(), parentCtx, family.UID, "Huge Penalty", child.UserID,
		)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInsufficientTokens)
	})
}

// TestApplyPenaltyCrossFamilyIsolation tests that a parent cannot apply a penalty
// to a child from a different family.
func TestApplyPenaltyCrossFamilyIsolation(t *testing.T) {
	setupTestDB(t)
	family1, parent1, _ := setupAuthTestData(t)
	_, _, child2 := setupSecondFamily(t)

	parent1Ctx := &domain.AuthContext{
		UserID:    parent1.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family1.UID,
	}

	penalty := &models.Penalties{
		Entities: models.Entities{
			FamilyUID:   family1.UID,
			Name:        "No Screen Time",
			Description: "Lost screen time privileges",
			Tokens:      10,
		},
	}
	err := testHandler.services.PenaltyService.CreatePenalty(
		context.Background(), parent1Ctx, penalty,
	)
	require.NoError(t, err)

	t.Run("Parent cannot apply penalty to child from different family", func(t *testing.T) {
		_, err := testHandler.services.PenaltyService.ApplyPenalty(
			context.Background(),
			parent1Ctx,
			family1.UID,
			"No Screen Time",
			child2.UserID,
		)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUnauthorized)
	})

	t.Run("Parent can apply penalty to child in same family", func(t *testing.T) {
		_, child1 := getAuthTestUsers(t, family1.UID)

		result, err := testHandler.services.PenaltyService.ApplyPenalty(
			context.Background(),
			parent1Ctx,
			family1.UID,
			"No Screen Time",
			child1.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, "No Screen Time", result.Name)
	})
}

// TestPenaltyHTTPHandlers tests the penalty HTTP endpoints end-to-end
func TestPenaltyHTTPHandlers(t *testing.T) {
	setupTestDB(t)
	family, parent, child, _ := setupServiceTestData(t)

	t.Run("POST /penalties creates a penalty", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
			"name":       "Late Bedtime",
			"tokens":     10,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenalties(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var penalty models.Penalties
		assertSuccessResponse(t, w, http.StatusCreated, &penalty)
		assert.Equal(t, "Late Bedtime", penalty.Name)
		assert.Equal(t, 10, penalty.Tokens)
	})

	t.Run("GET /penalties/{familyUID} lists family penalties", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/penalties/%s", family.UID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var penalties []models.Penalties
		assertSuccessResponse(t, w, http.StatusOK, &penalties)
		assert.Len(t, penalties, 1)
		assert.Equal(t, "Late Bedtime", penalties[0].Name)
	})

	t.Run("GET /penalties/{familyUID}/{name} gets single penalty", func(t *testing.T) {
		encodedName := url.PathEscape("Late Bedtime")
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/penalties/%s/%s", family.UID, encodedName), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var penalty models.Penalties
		assertSuccessResponse(t, w, http.StatusOK, &penalty)
		assert.Equal(t, "Late Bedtime", penalty.Name)
	})

	t.Run("PUT /penalties/{familyUID}/{name} updates penalty", func(t *testing.T) {
		body := map[string]any{
			"name":   "Very Late Bedtime",
			"tokens": 20,
		}
		bodyJSON, _ := json.Marshal(body)
		encodedName := url.PathEscape("Late Bedtime")

		req := httptest.NewRequest("PUT",
			fmt.Sprintf("/api/v1/penalties/%s/%s", family.UID, encodedName),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var penalty models.Penalties
		assertSuccessResponse(t, w, http.StatusOK, &penalty)
		assert.Equal(t, "Very Late Bedtime", penalty.Name)
		assert.Equal(t, 20, penalty.Tokens)
	})

	t.Run("POST /penalties/{familyUID}/{name} applies penalty", func(t *testing.T) {
		body := map[string]any{
			"child_user_id": child.UserID,
		}
		bodyJSON, _ := json.Marshal(body)
		encodedName := url.PathEscape("Very Late Bedtime")

		req := httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/penalties/%s/%s", family.UID, encodedName),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("DELETE /penalties/{familyUID}/{name} deletes penalty", func(t *testing.T) {
		encodedName := url.PathEscape("Very Late Bedtime")
		req := httptest.NewRequest("DELETE",
			fmt.Sprintf("/api/v1/penalties/%s/%s", family.UID, encodedName), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Child cannot create penalties via HTTP", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
			"name":       "Cheat Penalty",
			"tokens":     1,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", child.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenalties(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/penalties", nil)
		w := httptest.NewRecorder()

		testHandler.handlePenalties(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid JSON returns bad request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/penalties",
			bytes.NewReader([]byte("not json")))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenalties(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Missing required fields returns bad request", func(t *testing.T) {
		body := map[string]any{
			"family_uid": family.UID,
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenalties(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET nonexistent penalty returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/penalties/%s/Nonexistent", family.UID), nil)
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Apply penalty missing child_user_id returns bad request", func(t *testing.T) {
		// Create a penalty to apply
		penaltyBody := map[string]any{
			"family_uid": family.UID,
			"name":       "Apply Test",
			"tokens":     5,
		}
		penaltyJSON, _ := json.Marshal(penaltyBody)
		req := httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(penaltyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w := httptest.NewRecorder()
		testHandler.handlePenalties(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Try to apply without child_user_id
		body := map[string]any{}
		bodyJSON, _ := json.Marshal(body)
		encodedName := url.PathEscape("Apply Test")
		req = httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/penalties/%s/%s", family.UID, encodedName),
			bytes.NewReader(bodyJSON))
		req.Header.Set("X-User-ID", parent.UserID)
		w = httptest.NewRecorder()

		testHandler.handlePenaltiesByFamily(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// getAuthTestUsers retrieves parent and child users for a family from the database.
func getAuthTestUsers(t *testing.T, familyUID string) (parent, child *models.Users) {
	t.Helper()
	var users []models.Users
	err := testDB.Where("family_uid = ?", familyUID).Find(&users).Error
	require.NoError(t, err)
	for i := range users {
		switch users[i].Role {
		case "parent":
			parent = &users[i]
		case "child":
			child = &users[i]
		}
	}
	require.NotNil(t, parent, "no parent found for family %s", familyUID)
	require.NotNil(t, child, "no child found for family %s", familyUID)
	return parent, child
}
