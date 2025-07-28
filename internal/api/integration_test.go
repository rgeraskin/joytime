package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	psql "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupIntegrationDB sets up test database for integration tests
func setupIntegrationDB(t *testing.T) {
	level := log.InfoLevel
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
	})

	var err error

	// Use test database
	testDSN := "host=localhost user=joytime password=password dbname=joytime port=5432 sslmode=disable"
	db, err = gorm.Open(psql.Open(testDSN), &gorm.Config{
		Logger: gormlogger.New(
			logger,
			gormlogger.Config{
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	require.NoError(t, err)

	// Clean database before tests
	db.Exec("DELETE FROM token_histories")
	db.Exec("DELETE FROM tokens")
	db.Exec("DELETE FROM tasks")
	db.Exec("DELETE FROM rewards")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM families")

	// Migrate schema
	err = db.AutoMigrate(
		&postgres.Users{},
		&postgres.Families{},
		&postgres.Entities{},
		&postgres.Tasks{},
		&postgres.Rewards{},
		&postgres.Tokens{},
		&postgres.TokenHistory{},
	)
	require.NoError(t, err)
}

// TestFullAPIWorkflow tests complete API workflow
// Replaces test_api.sh script
func TestFullAPIWorkflow(t *testing.T) {
	setupIntegrationDB(t)

	t.Run("Complete API Workflow", func(t *testing.T) {
		// 1. Create family
		t.Log("1. 📋 Creating family...")
		familyData := map[string]interface{}{
			"name": "Test Family",
		}
		familyJSON, _ := json.Marshal(familyData)
		req := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(familyJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var family postgres.Families
		err := json.Unmarshal(w.Body.Bytes(), &family)
		require.NoError(t, err)
		require.NotEmpty(t, family.UID)
		familyUID := family.UID
		t.Logf("Family UID: %s", familyUID)

		// 2. Create parent
		t.Log("2. 👤 Creating parent...")
		parentData := map[string]interface{}{
			"user_id":    "user_parent_123",
			"name":       "Test Parent",
			"role":       "parent",
			"family_uid": familyUID,
			"platform":   "telegram",
		}
		parentJSON, _ := json.Marshal(parentData)
		req = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(parentJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var parent postgres.Users
		err = json.Unmarshal(w.Body.Bytes(), &parent)
		require.NoError(t, err)
		assert.Equal(t, "user_parent_123", parent.UserID)
		assert.Equal(t, "Test Parent", parent.Name)
		assert.Equal(t, "parent", parent.Role)

		// 3. Create child
		t.Log("3. 👶 Creating child...")
		childData := map[string]interface{}{
			"user_id":    "user_child_456",
			"name":       "Test Child",
			"role":       "child",
			"family_uid": familyUID,
			"platform":   "telegram",
		}
		childJSON, _ := json.Marshal(childData)
		req = httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(childJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var child postgres.Users
		err = json.Unmarshal(w.Body.Bytes(), &child)
		require.NoError(t, err)
		assert.Equal(t, "user_child_456", child.UserID)
		assert.Equal(t, "Test Child", child.Name)
		assert.Equal(t, "child", child.Role)

		// 4. Check child tokens (should be automatically created)
		t.Log("4. ⚡ Checking child tokens...")
		req = httptest.NewRequest(http.MethodGet, "/tokens/user_child_456", nil)
		w = httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokens postgres.Tokens
		err = json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)
		assert.Equal(t, "user_child_456", tokens.UserID)
		assert.Equal(t, 0, tokens.Tokens) // Initial value

		// 5. Create task
		t.Log("5. 📝 Creating task...")
		taskData := map[string]interface{}{
			"family_uid":  familyUID,
			"name":        "Clean the room",
			"tokens":      10,
			"description": "Tidy up and vacuum",
		}
		taskJSON, _ := json.Marshal(taskData)
		req = httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(taskJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleEntities(w, req, "tasks")

		assert.Equal(t, http.StatusCreated, w.Code)
		var task postgres.Tasks
		err = json.Unmarshal(w.Body.Bytes(), &task)
		require.NoError(t, err)
		assert.Equal(t, "Clean the room", task.Name)
		assert.Equal(t, 10, task.Tokens)
		taskID := task.ID

		// 6. Create reward
		t.Log("6. 🎁 Creating reward...")
		rewardData := map[string]interface{}{
			"family_uid":  familyUID,
			"name":        "Watch cartoons",
			"tokens":      5,
			"description": "15 minutes of cartoons",
		}
		rewardJSON, _ := json.Marshal(rewardData)
		req = httptest.NewRequest(http.MethodPost, "/rewards", bytes.NewBuffer(rewardJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleEntities(w, req, "rewards")

		assert.Equal(t, http.StatusCreated, w.Code)
		var reward postgres.Rewards
		err = json.Unmarshal(w.Body.Bytes(), &reward)
		require.NoError(t, err)
		assert.Equal(t, "Watch cartoons", reward.Name)
		assert.Equal(t, 5, reward.Tokens)
		rewardID := reward.ID

		// 7. Get all family tasks
		t.Log("7. 📊 Getting all family tasks...")
		req = httptest.NewRequest(http.MethodGet, "/tasks/"+familyUID, nil)
		w = httptest.NewRecorder()
		handleEntities(w, req, "tasks")

		assert.Equal(t, http.StatusOK, w.Code)
		var tasks []postgres.Tasks
		err = json.Unmarshal(w.Body.Bytes(), &tasks)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Clean the room", tasks[0].Name)

		// 8. Get all family rewards
		t.Log("8. 🏆 Getting all family rewards...")
		req = httptest.NewRequest(http.MethodGet, "/rewards/"+familyUID, nil)
		w = httptest.NewRecorder()
		handleEntities(w, req, "rewards")

		assert.Equal(t, http.StatusOK, w.Code)
		var rewards []postgres.Rewards
		err = json.Unmarshal(w.Body.Bytes(), &rewards)
		require.NoError(t, err)
		assert.Len(t, rewards, 1)
		assert.Equal(t, "Watch cartoons", rewards[0].Name)

		// 9. Award tokens to child for completing task (+10)
		t.Log("9. 💰 Awarding tokens to child for completing task (+10)...")
		addTokensData := map[string]interface{}{
			"amount":      10,
			"type":        "task_completed",
			"description": "Completed task: Clean the room",
			"task_id":     taskID,
		}
		addTokensJSON, _ := json.Marshal(addTokensData)
		req = httptest.NewRequest(http.MethodPost, "/tokens/user_child_456", bytes.NewBuffer(addTokensJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		err = json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)
		assert.Equal(t, 10, tokens.Tokens)

		// 10. Deduct tokens for reward (-5)
		t.Log("10. 🎯 Deducting tokens for reward (-5)...")
		subtractTokensData := map[string]interface{}{
			"amount":      -5,
			"type":        "reward_claimed",
			"description": "Claimed reward: Watch cartoons",
			"reward_id":   rewardID,
		}
		subtractTokensJSON, _ := json.Marshal(subtractTokensData)
		req = httptest.NewRequest(http.MethodPost, "/tokens/user_child_456", bytes.NewBuffer(subtractTokensJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		err = json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)
		assert.Equal(t, 5, tokens.Tokens) // 10 - 5 = 5

		// 11. Get child token history
		t.Log("11. 📜 Getting child token history...")
		req = httptest.NewRequest(http.MethodGet, "/token-history/user_child_456", nil)
		w = httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var history []postgres.TokenHistory
		err = json.Unmarshal(w.Body.Bytes(), &history)
		require.NoError(t, err)
		assert.Len(t, history, 2) // Two events: +10 and -5

		// Check events in history
		foundTaskComplete := false
		foundRewardClaim := false
		for _, h := range history {
			if h.Type == "task_completed" && h.Amount == 10 {
				foundTaskComplete = true
				assert.Equal(t, taskID, *h.TaskID)
			}
			if h.Type == "reward_claimed" && h.Amount == -5 {
				foundRewardClaim = true
				assert.Equal(t, rewardID, *h.RewardID)
			}
		}
		assert.True(t, foundTaskComplete, "History should contain task completion")
		assert.True(t, foundRewardClaim, "History should contain reward claim")

		// 12. Get token history with pagination (limit=1)
		t.Log("12. 📈 Getting token history with pagination (limit=1)...")
		req = httptest.NewRequest(http.MethodGet, "/token-history/user_child_456?limit=1&offset=0", nil)
		w = httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var paginatedHistory []postgres.TokenHistory
		err = json.Unmarshal(w.Body.Bytes(), &paginatedHistory)
		require.NoError(t, err)
		assert.Len(t, paginatedHistory, 1)

		// 13. List all users
		t.Log("13. 👥 Listing all users...")
		req = httptest.NewRequest(http.MethodGet, "/users", nil)
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var allUsers []postgres.Users
		err = json.Unmarshal(w.Body.Bytes(), &allUsers)
		require.NoError(t, err)
		assert.Len(t, allUsers, 2) // Parent and child

		// 14. List all families
		t.Log("14. 🏠 Listing all families...")
		req = httptest.NewRequest(http.MethodGet, "/families", nil)
		w = httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var allFamilies []postgres.Families
		err = json.Unmarshal(w.Body.Bytes(), &allFamilies)
		require.NoError(t, err)
		assert.Len(t, allFamilies, 1)
		assert.Equal(t, "Test Family", allFamilies[0].Name)

		// 15. List all tokens
		t.Log("15. 💎 Listing all tokens...")
		req = httptest.NewRequest(http.MethodGet, "/tokens", nil)
		w = httptest.NewRecorder()
		handleTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var allTokens []postgres.Tokens
		err = json.Unmarshal(w.Body.Bytes(), &allTokens)
		require.NoError(t, err)
		assert.Len(t, allTokens, 1) // Only child has tokens
		assert.Equal(t, 5, allTokens[0].Tokens)

		// 16. Update user (change platform)
		t.Log("16. 📝 Updating user (changing platform)...")
		updateData := map[string]interface{}{
			"platform":    "web",
			"input_state": "waiting_for_task_name",
			"user_id":     "user_child_456",
			"name":        "Test Child",
			"role":        "child",
			"family_uid":  familyUID,
		}
		updateJSON, _ := json.Marshal(updateData)
		req = httptest.NewRequest(http.MethodPut, "/users/user_child_456", bytes.NewBuffer(updateJSON))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var updatedUser postgres.Users
		err = json.Unmarshal(w.Body.Bytes(), &updatedUser)
		require.NoError(t, err)
		assert.Equal(t, "web", updatedUser.Platform)
		assert.Equal(t, "waiting_for_task_name", updatedUser.InputState)

		t.Log("✅ All integration tests passed successfully!")
		t.Logf("🔍 Results:")
		t.Logf("- Family created with UID: %s", familyUID)
		t.Logf("- Created 2 users (parent and child)")
		t.Logf("- Created 1 task and 1 reward")
		t.Logf("- Performed token operations (add/subtract)")
		t.Logf("- Created token operation history")
		t.Logf("- API supports different platforms (telegram, web)")
	})
}

// TestTokenOperations tests various token operations
func TestTokenOperations(t *testing.T) {
	setupIntegrationDB(t)

	// Create test data
	family := &postgres.Families{
		Name:            "Token Test Family",
		UID:             "token_test_family",
		CreatedByUserID: "user_token_parent",
	}
	db.Create(family)

	user := &postgres.Users{
		UserID:    "user_token_child",
		Name:      "Token Test Child",
		Role:      "child",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	db.Create(user)

	tokens := &postgres.Tokens{
		UserID: user.UserID,
		Tokens: 0,
	}
	db.Create(tokens)

	t.Run("Add Tokens", func(t *testing.T) {
		addData := map[string]interface{}{
			"amount":      15,
			"type":        "manual_adjustment",
			"description": "Bonus from parents",
		}
		addJSON, _ := json.Marshal(addData)
		req := httptest.NewRequest(http.MethodPost, "/tokens/user_token_child", bytes.NewBuffer(addJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result postgres.Tokens
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, 15, result.Tokens)
	})

	t.Run("Subtract Tokens", func(t *testing.T) {
		subtractData := map[string]interface{}{
			"amount":      -8,
			"type":        "reward_claimed",
			"description": "Spent on toy",
		}
		subtractJSON, _ := json.Marshal(subtractData)
		req := httptest.NewRequest(http.MethodPost, "/tokens/user_token_child", bytes.NewBuffer(subtractJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result postgres.Tokens
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, 7, result.Tokens) // 15 - 8 = 7
	})

	t.Run("Insufficient Tokens", func(t *testing.T) {
		subtractData := map[string]interface{}{
			"amount":      -10,
			"type":        "reward_claimed",
			"description": "Attempt to spend more than available",
		}
		subtractJSON, _ := json.Marshal(subtractData)
		req := httptest.NewRequest(http.MethodPost, "/tokens/user_token_child", bytes.NewBuffer(subtractJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Insufficient tokens")
	})
}
