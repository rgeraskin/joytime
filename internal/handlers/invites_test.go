package handlers

import (
	"context"
	"testing"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteService(t *testing.T) {
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

	t.Run("Parent can create invite", func(t *testing.T) {
		invite, err := testHandler.services.InviteService.CreateInvite(
			context.Background(), parentCtx, family.UID, "child",
		)
		require.NoError(t, err)
		assert.Len(t, invite.Code, 8)
		assert.Equal(t, family.UID, invite.FamilyUID)
		assert.Equal(t, "child", invite.Role)
		assert.False(t, invite.Used)
	})

	t.Run("Child cannot create invite", func(t *testing.T) {
		_, err := testHandler.services.InviteService.CreateInvite(
			context.Background(), childCtx, family.UID, "child",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Use valid invite", func(t *testing.T) {
		invite, err := testHandler.services.InviteService.CreateInvite(
			context.Background(), parentCtx, family.UID, "child",
		)
		require.NoError(t, err)

		used, err := testHandler.services.InviteService.UseInvite(
			context.Background(), invite.Code,
		)
		require.NoError(t, err)
		assert.True(t, used.Used)
		assert.Equal(t, invite.FamilyUID, used.FamilyUID)
		assert.Equal(t, invite.Role, used.Role)
	})

	t.Run("Cannot reuse invite", func(t *testing.T) {
		invite, err := testHandler.services.InviteService.CreateInvite(
			context.Background(), parentCtx, family.UID, "child",
		)
		require.NoError(t, err)

		_, err = testHandler.services.InviteService.UseInvite(context.Background(), invite.Code)
		require.NoError(t, err)

		_, err = testHandler.services.InviteService.UseInvite(context.Background(), invite.Code)
		require.Error(t, err)
	})

	t.Run("Invalid code returns error", func(t *testing.T) {
		_, err := testHandler.services.InviteService.UseInvite(context.Background(), "INVALID1")
		require.Error(t, err)
	})

	t.Run("Invite codes are unique", func(t *testing.T) {
		codes := make(map[string]bool)
		for range 10 {
			invite, err := testHandler.services.InviteService.CreateInvite(
				context.Background(), parentCtx, family.UID, "child",
			)
			require.NoError(t, err)
			codes[invite.Code] = true
		}
		assert.Len(t, codes, 10, "All 10 invite codes should be unique")
	})
}
