package handlers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAuthTestData creates a family with parent and child users for authorization testing
func setupAuthTestData(
	t *testing.T,
) (family *models.Families, parent *models.Users, child *models.Users) {
	// Create family
	family = &models.Families{
		Name: fmt.Sprintf("AuthTestFamily_%s", t.Name()),
		UID:  fmt.Sprintf("family_uid_%s_%d", t.Name(), time.Now().UnixNano()),
	}
	result := testDB.Create(family)
	require.NoError(t, result.Error)

	// Create parent user
	parent = &models.Users{
		UserID:    fmt.Sprintf("parent_%s", t.Name()),
		Name:      "Test Parent",
		Role:      "parent",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	result = testDB.Create(parent)
	require.NoError(t, result.Error)

	// Create child user
	child = &models.Users{
		UserID:    fmt.Sprintf("child_%s", t.Name()),
		Name:      "Test Child",
		Role:      "child",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	result = testDB.Create(child)
	require.NoError(t, result.Error)

	// Create tokens for child
	tokens := &models.Tokens{
		UserID: child.UserID,
		Tokens: 50,
	}
	result = testDB.Create(tokens)
	require.NoError(t, result.Error)

	return family, parent, child
}

// setupSecondFamily creates a second family for testing family isolation
func setupSecondFamily(
	t *testing.T,
) (family *models.Families, parent *models.Users, child *models.Users) {
	// Create second family
	family = &models.Families{
		Name: fmt.Sprintf("SecondFamily_%s", t.Name()),
		UID:  fmt.Sprintf("family2_uid_%s_%d", t.Name(), time.Now().UnixNano()),
	}
	result := testDB.Create(family)
	require.NoError(t, result.Error)

	// Create parent in second family
	parent = &models.Users{
		UserID:    fmt.Sprintf("parent2_%s", t.Name()),
		Name:      "Second Parent",
		Role:      "parent",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	result = testDB.Create(parent)
	require.NoError(t, result.Error)

	// Create child in second family
	child = &models.Users{
		UserID:    fmt.Sprintf("child2_%s", t.Name()),
		Name:      "Second Child",
		Role:      "child",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	result = testDB.Create(child)
	require.NoError(t, result.Error)

	return family, parent, child
}

// TestCasbinParentPermissions tests that parents can do parent-only actions
func TestCasbinParentPermissions(t *testing.T) {
	setupTestDB(t)
	family, parent, child := setupAuthTestData(t)

	t.Run("Parent can create tasks", func(t *testing.T) {
		// Test direct service call with parent context
		authCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}

		taskData := &models.Tasks{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "Parent Created Task",
				Description: "Task created by parent",
				Tokens:      15,
			},
		}

		err := testHandler.services.TaskService.CreateTask(
			context.Background(),
			authCtx,
			taskData,
		)
		assert.NoError(t, err)

		// Verify task was created
		var createdTask models.Tasks
		err = testDB.Where("name = ?", "Parent Created Task").First(&createdTask).Error
		assert.NoError(t, err)
		assert.Equal(t, "Parent Created Task", createdTask.Name)
	})

	t.Run("Parent can create rewards via service", func(t *testing.T) {
		authCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}

		rewardData := &models.Rewards{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "Parent Created Reward",
				Description: "Reward created by parent",
				Tokens:      10,
			},
		}

		err := testHandler.services.RewardService.CreateReward(
			context.Background(), authCtx, rewardData,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Parent Created Reward", rewardData.Name)
		assert.NotZero(t, rewardData.ID)
	})

	t.Run("Parent can update other users in family", func(t *testing.T) {
		authCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}

		updates := &domain.UpdateUserRequest{
			Name: "Updated Child Name",
		}

		updatedUser, err := testHandler.services.UserService.UpdateUser(
			context.Background(),
			authCtx,
			child.UserID,
			updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Child Name", updatedUser.Name)
	})

	t.Run("Parent can manually adjust tokens", func(t *testing.T) {
		authCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}

		err := testHandler.services.TokenService.AddTokensToUser(
			context.Background(),
			authCtx,
			child.UserID,
			25,
			domain.TokenTypeManualAdjustment,
			"Parent bonus",
			nil,
			nil,
		)
		assert.NoError(t, err)

		// Verify tokens were added
		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(),
			authCtx,
			child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, 75, tokens.Tokens) // 50 + 25
	})

	t.Run("Parent can read other users' tokens", func(t *testing.T) {
		authCtx := &domain.AuthContext{
			UserID:    parent.UserID,
			UserRole:  domain.RoleParent,
			FamilyUID: family.UID,
		}

		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(),
			authCtx,
			child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, child.UserID, tokens.UserID)
	})
}

// TestCasbinChildPermissions tests that children can do child-only actions and are blocked from others
func TestCasbinChildPermissions(t *testing.T) {
	setupTestDB(t)
	family, parent, child := setupAuthTestData(t)

	// Create a task that child can complete
	parentCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	taskData := &models.Tasks{
		Entities: models.Entities{
			FamilyUID:   family.UID,
			Name:        "Child Test Task",
			Description: "Task for child to complete",
			Tokens:      20,
		},
	}
	err := testHandler.services.TaskService.CreateTask(context.Background(), parentCtx, taskData)
	require.NoError(t, err)

	t.Run("Child can read tasks", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		tasks, err := testHandler.services.TaskService.GetTasksForFamily(
			context.Background(),
			childCtx,
			family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Child Test Task", tasks[0].Name)
	})

	t.Run("Child can complete tasks", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		completedTask, err := testHandler.services.TaskService.CompleteTask(
			context.Background(),
			childCtx,
			family.UID,
			"Child Test Task",
		)
		assert.NoError(t, err)
		assert.Equal(t, "check", completedTask.Status) // Child completion creates "check" status
	})

	t.Run("Child can read own tokens", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		tokens, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(),
			childCtx,
			child.UserID,
		)
		assert.NoError(t, err)
		assert.Equal(t, child.UserID, tokens.UserID)
	})

	t.Run("Child can update own profile", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		updates := &domain.UpdateUserRequest{
			Name: "Updated Child Name",
		}

		updatedUser, err := testHandler.services.UserService.UpdateUser(
			context.Background(),
			childCtx,
			child.UserID,
			updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Child Name", updatedUser.Name)
	})

	t.Run("Child CANNOT create tasks", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		unauthorizedTask := &models.Tasks{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "Unauthorized Task",
				Description: "Child should not be able to create this",
				Tokens:      10,
			},
		}

		err := testHandler.services.TaskService.CreateTask(
			context.Background(),
			childCtx,
			unauthorizedTask,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT delete tasks", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		err := testHandler.services.TaskService.DeleteTask(
			context.Background(),
			childCtx,
			family.UID,
			"Child Test Task",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})



	t.Run("Child CANNOT manually adjust tokens", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		err := testHandler.services.TokenService.AddTokensToUser(
			context.Background(),
			childCtx,
			child.UserID,
			100,
			domain.TokenTypeManualAdjustment,
			"Child trying to cheat",
			nil,
			nil,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "child cannot add tokens")
	})

	t.Run("Child CANNOT read other users' tokens", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		_, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(),
			childCtx,
			parent.UserID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child CANNOT update other users", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		updates := &domain.UpdateUserRequest{
			Name: "Child trying to update parent",
		}

		_, err := testHandler.services.UserService.UpdateUser(
			context.Background(),
			childCtx,
			parent.UserID,
			updates,
		)
		assert.Error(t, err)
		// ABAC blocks children from update_others permission
		assert.Contains(t, err.Error(), "child cannot update_others users")
	})

	t.Run("Child CANNOT delete users", func(t *testing.T) {
		childCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		err := testHandler.services.UserService.DeleteUser(
			context.Background(),
			childCtx,
			parent.UserID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// TestCasbinFamilyIsolation tests that users from different families cannot access each other's data
func TestCasbinFamilyIsolation(t *testing.T) {
	setupTestDB(t)

	// Setup two families
	family1, parent1, child1 := setupAuthTestData(t)
	family2, parent2, child2 := setupSecondFamily(t)

	// Create tasks in each family
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

	// Create task in family1
	task1Data := &models.Tasks{
		Entities: models.Entities{
			FamilyUID:   family1.UID,
			Name:        "Family1 Task",
			Description: "Task for family 1",
			Tokens:      10,
		},
	}
	err := testHandler.services.TaskService.CreateTask(context.Background(), parent1Ctx, task1Data)
	require.NoError(t, err)

	// Create task in family2
	task2Data := &models.Tasks{
		Entities: models.Entities{
			FamilyUID:   family2.UID,
			Name:        "Family2 Task",
			Description: "Task for family 2",
			Tokens:      15,
		},
	}
	err = testHandler.services.TaskService.CreateTask(context.Background(), parent2Ctx, task2Data)
	require.NoError(t, err)

	t.Run("Parent from family1 cannot access family2 tasks", func(t *testing.T) {
		_, err := testHandler.services.TaskService.GetTasksForFamily(
			context.Background(),
			parent1Ctx,
			family2.UID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Parent from family1 cannot access family2 users", func(t *testing.T) {
		_, err := testHandler.services.UserService.GetUser(
			context.Background(),
			parent1Ctx,
			child2.UserID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child from family1 cannot access family2 tasks", func(t *testing.T) {
		child1Ctx := &domain.AuthContext{
			UserID:    child1.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family1.UID,
		}

		_, err := testHandler.services.TaskService.GetTasksForFamily(
			context.Background(),
			child1Ctx,
			family2.UID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Child from family1 cannot access family2 user tokens", func(t *testing.T) {
		child1Ctx := &domain.AuthContext{
			UserID:    child1.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family1.UID,
		}

		_, err := testHandler.services.TokenService.GetUserTokens(
			context.Background(),
			child1Ctx,
			child2.UserID,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Parent from family1 cannot delete tasks from family2", func(t *testing.T) {
		err := testHandler.services.TaskService.DeleteTask(
			context.Background(),
			parent1Ctx,
			family2.UID,
			"Family2 Task",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Each family can only access their own data", func(t *testing.T) {
		// Family1 can access their own tasks
		tasks1, err := testHandler.services.TaskService.GetTasksForFamily(
			context.Background(),
			parent1Ctx,
			family1.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, tasks1, 1)
		assert.Equal(t, "Family1 Task", tasks1[0].Name)

		// Family2 can access their own tasks
		tasks2, err := testHandler.services.TaskService.GetTasksForFamily(
			context.Background(),
			parent2Ctx,
			family2.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, tasks2, 1)
		assert.Equal(t, "Family2 Task", tasks2[0].Name)
	})
}


