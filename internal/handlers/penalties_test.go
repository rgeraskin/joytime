package handlers

import (
	"context"
	"testing"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Create a penalty in family1
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
		// child1 was created in family1 by setupAuthTestData
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
