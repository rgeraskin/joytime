package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/postgres"
	"github.com/stretchr/testify/assert"
	psql "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var testHandler *APIHandler

// parseSuccessResponse parses a success response and extracts the data
func parseSuccessResponse(w *httptest.ResponseRecorder, target interface{}) error {
	var successResponse SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&successResponse); err != nil {
		return err
	}

	// Marshal and unmarshal to convert interface{} to target type
	dataBytes, err := json.Marshal(successResponse.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(dataBytes, target)
}

// parseErrorResponse parses an error response
func parseErrorResponse(w *httptest.ResponseRecorder) (*ErrorResponse, error) {
	var errorResponse ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	return &errorResponse, err
}

// assertSuccessResponse checks status code and parses success response
func assertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	assert.Equal(t, expectedStatus, w.Code)
	err := parseSuccessResponse(w, target)
	assert.NoError(t, err)
}

// assertErrorResponse checks status code and parses error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	assert.Equal(t, expectedStatus, w.Code)
	errorResp, err := parseErrorResponse(w)
	assert.NoError(t, err)
	assert.Contains(t, errorResp.Error, expectedErrorContains)
}

// cleanupTestData removes all test data from database in correct order to handle foreign key constraints
func cleanupTestData() {
	testHandler.db.Unscoped().Where("1 = 1").Delete(&postgres.TokenHistory{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&postgres.Tokens{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&postgres.Tasks{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&postgres.Rewards{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&postgres.Users{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&postgres.Families{})
}

// migrateTestSchema migrates all required models for testing
func migrateTestSchema(includeEntities bool) error {
	models := []interface{}{
		&postgres.Users{},
		&postgres.Families{},
	}

	if includeEntities {
		models = append(models, &postgres.Entities{})
	}

	models = append(models,
		&postgres.Tasks{},
		&postgres.Rewards{},
		&postgres.Tokens{},
		&postgres.TokenHistory{},
	)

	return testHandler.db.AutoMigrate(models...)
}

// setupTestDBConnection establishes database connection for testing
func setupTestDBConnection() error {
	level := log.InfoLevel
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
	})

	// Build DSN from environment variables like main app
	config := getTestDBConfig()
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.Host,
		config.User,
		config.Password,
		config.Database,
		config.Port,
	)

	var err error
	db, err := gorm.Open(psql.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			logger,
			gormlogger.Config{
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	if err != nil {
		return err
	}

	testHandler = NewAPIHandler(db, logger)
	return nil
}

func getTestDBConfig() *postgres.Config {
	getEnvOrDefault := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}

	return &postgres.Config{
		Host:     getEnvOrDefault("PGHOST", "localhost"),
		User:     getEnvOrDefault("PGUSER", "joytime"),
		Password: getEnvOrDefault("PGPASSWORD", "password"),
		Database: getEnvOrDefault("PGDATABASE", "joytime"),
		Port:     getEnvOrDefault("PGPORT", "5432"),
	}
}

func setupTestDB(t *testing.T) *postgres.Families {
	err := setupTestDBConnection()
	assert.NoError(t, err)

	// Clean existing data to ensure we start with a clean slate
	// Delete in correct order to handle foreign key constraints
	cleanupTestData()

	// Migrate the schema
	err = migrateTestSchema(false)
	assert.NoError(t, err)

	// Setup teardown cleanup function
	t.Cleanup(func() {
		// Clean test data after test completion
		cleanupTestData()
	})

	// create a family with unique UID
	uniqueUID := fmt.Sprintf("%s_%d", t.Name(), time.Now().UnixNano())
	family := postgres.Families{
		Name: t.Name(),
		UID:  uniqueUID,
	}
	result := testHandler.db.Create(&family)
	if result.Error != nil {
		t.Fatalf("Failed to create family: %v", result.Error)
	}

	return &family
}

// TestUserEndpoints tests user CRUD operations
func TestUserEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	t.Run("Create User", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_123",
			Name:      "Test User",
			Role:      "parent",
			FamilyUID: setupFamily.UID,
			Platform:  "telegram",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		var response postgres.Users
		assertSuccessResponse(t, w, http.StatusCreated, &response)
		assert.Equal(t, user.UserID, response.UserID)
		assert.Equal(t, user.Name, response.Name)
		assert.Equal(t, user.Role, response.Role)
		assert.Equal(t, user.FamilyUID, response.FamilyUID)
		assert.Equal(t, user.Platform, response.Platform)

		// Verify user is in the database
		var dbUser postgres.Users
		err := testHandler.db.Where("user_id = ?", "test_user_123").First(&dbUser).Error
		assert.NoError(t, err)
		assert.Equal(t, user.UserID, dbUser.UserID)
	})

	t.Run("Create User with wrong family uid", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_456",
			Name:      "Test User 2",
			Role:      "child",
			FamilyUID: "wrong_family_uid",
			Platform:  "telegram",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrFamilyNotFound)
	})

	t.Run("Create User with wrong role", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_789",
			Name:      "Test User 3",
			Role:      "admin",
			FamilyUID: setupFamily.UID,
			Platform:  "telegram",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrInvalidRole)
	})

	t.Run("Create User with missing required fields", func(t *testing.T) {
		// missing Role
		user := postgres.Users{
			UserID:    "test_user_no_role",
			Name:      "Test User No Role",
			FamilyUID: setupFamily.UID,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrRoleRequired)

		// missing Name
		user = postgres.Users{
			UserID:    "test_user_no_name",
			Role:      "child",
			FamilyUID: setupFamily.UID,
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrNameRequired)

		// missing UserID
		user = postgres.Users{
			Name:      "Test User No UserID",
			Role:      "child",
			FamilyUID: setupFamily.UID,
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrUserIDRequiredField)

		// missing FamilyUID
		user = postgres.Users{
			UserID: "test_user_no_family",
			Name:   "Test User No Family",
			Role:   "child",
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrFamilyUIDRequiredField)
	})

	t.Run("Create existing user", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_123", // Already exists from first test
			Name:      "Duplicate User",
			Role:      "child",
			FamilyUID: setupFamily.UID,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUsers(w, req)

		assertErrorResponse(t, w, http.StatusInternalServerError, "Failed to create user")
	})

	t.Run("Get User", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/test_user_123", nil)
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		var user postgres.Users
		assertSuccessResponse(t, w, http.StatusOK, &user)
		assert.Equal(t, "test_user_123", user.UserID)
	})

	t.Run("Get absent user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/absent_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrUserNotFound)
	})

	t.Run("Update User", func(t *testing.T) {
		user := postgres.Users{
			Name:     "Updated User",
			Platform: "web",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/api/v1/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		var updatedUser postgres.Users
		assertSuccessResponse(t, w, http.StatusOK, &updatedUser)
		assert.Equal(t, "Updated User", updatedUser.Name)

		// Verify user is updated in the database
		var dbUser postgres.Users
		err := testHandler.db.Where("user_id = ?", "test_user_123").First(&dbUser).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated User", dbUser.Name)
	})

	t.Run("Update User with wrong role", func(t *testing.T) {
		user := postgres.Users{
			Role: "admin",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/api/v1/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrInvalidRole)
	})

	t.Run("Update non-existent user", func(t *testing.T) {
		user := postgres.Users{
			Name: "Non-existent User",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/api/v1/users/nonexistent_user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrUserNotFound)
	})

	t.Run("Update User with wrong family uid (family not found)", func(t *testing.T) {
		user := postgres.Users{
			FamilyUID: "wrong_family_uid",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/api/v1/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrFamilyNotFound)

		// Verify user is not updated in the database
		var updatedUser postgres.Users
		err := testHandler.db.Where("user_id = ?", "test_user_123").First(&updatedUser).Error
		assert.NoError(t, err)
		assert.Equal(t, setupFamily.UID, updatedUser.FamilyUID)
	})

	t.Run("Update User to existing user id", func(t *testing.T) {
		user1 := postgres.Users{
			UserID:    "existing_user_123",
			Name:      "Existing User",
			Role:      "child",
			FamilyUID: setupFamily.UID,
		}
		testHandler.db.Create(&user1)

		user2 := postgres.Users{
			UserID: "existing_user_123", // Duplicate
		}
		body, _ := json.Marshal(user2)
		req := httptest.NewRequest("PUT", "/api/v1/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		assertErrorResponse(t, w, http.StatusInternalServerError, "Failed to update user")
	})

	t.Run("Delete User", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/test_user_123", nil)
		w := httptest.NewRecorder()
		testHandler.handleUser(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify user is deleted from the database
		var dbUser postgres.Users
		err := testHandler.db.Where("user_id = ?", "test_user_123").First(&dbUser).Error
		assert.Error(t, err)
	})
}

// TestFamilyEndpoints tests family CRUD operations
func TestFamilyEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	t.Run("List Families", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/families", nil)
		w := httptest.NewRecorder()
		testHandler.handleFamilies(w, req)

		var families []postgres.Families
		assertSuccessResponse(t, w, http.StatusOK, &families)
		assert.Len(t, families, 1)
		assert.Equal(t, "TestFamilyEndpoints", families[0].Name)

		// Verify families are in the database
		var dbFamilies []postgres.Families
		err := testHandler.db.Find(&dbFamilies).Error
		assert.NoError(t, err)
		assert.Equal(t, "TestFamilyEndpoints", dbFamilies[0].Name)
	})

	t.Run("Create Family", func(t *testing.T) {
		family := postgres.Families{
			Name: "New Test Family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/api/v1/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleFamilies(w, req)

		var response postgres.Families
		assertSuccessResponse(t, w, http.StatusCreated, &response)
		assert.Equal(t, family.Name, response.Name)
		assert.NotEmpty(t, response.UID)
		uid := response.UID

		// Verify family is in the database
		var dbFamily postgres.Families
		err := testHandler.db.Where("uid = ?", uid).First(&dbFamily).Error
		assert.NoError(t, err)
		assert.Equal(t, family.Name, dbFamily.Name)
	})

	t.Run("Create Family with restricted fields", func(t *testing.T) {
		family := postgres.Families{
			Name: "Test Family",
			UID:  "custom_uid",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/api/v1/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleFamilies(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "UID is auto-generated")

		family = postgres.Families{
			Name:            "Test Family",
			CreatedByUserID: "user123",
		}
		body, _ = json.Marshal(family)
		req = httptest.NewRequest("POST", "/api/v1/families", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		testHandler.handleFamilies(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "CreatedByUserID is auto-generated")
	})

	t.Run("Create Family without name", func(t *testing.T) {
		family := postgres.Families{}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/api/v1/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleFamilies(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrNameRequired)
	})

	t.Run("Get Family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/families/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		var family postgres.Families
		assertSuccessResponse(t, w, http.StatusOK, &family)
		assert.Equal(t, setupFamily.UID, family.UID)
	})

	t.Run("Update Family - validation error (name and uid both empty)", func(t *testing.T) {
		family := postgres.Families{}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("PUT", "/api/v1/families/"+setupFamily.UID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrNameOrUIDRequired)
	})

	t.Run("Update non-existent family", func(t *testing.T) {
		family := postgres.Families{
			Name: "Updated Family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("PUT", "/api/v1/families/non-existent-family", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrFamilyNotFound)
	})

	t.Run("Delete non-existent family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/families/non-existent-family", nil)
		w := httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrFamilyNotFound)
	})

	t.Run("Delete Family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/families/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify family is deleted from the database
		var dbFamily postgres.Families
		err := testHandler.db.Where("uid = ?", setupFamily.UID).First(&dbFamily).Error
		assert.Error(t, err)
	})

	t.Run("Create Family and then Get it (integration)", func(t *testing.T) {
		family := postgres.Families{
			Name: "Integration Test Family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/api/v1/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleFamilies(w, req)

		var createdFamily postgres.Families
		assertSuccessResponse(t, w, http.StatusCreated, &createdFamily)
		assert.Equal(t, family.Name, createdFamily.Name)
		assert.NotEmpty(t, createdFamily.UID)

		// Then get the created family
		req = httptest.NewRequest("GET", "/api/v1/families/"+createdFamily.UID, nil)
		w = httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		var retrievedFamily postgres.Families
		assertSuccessResponse(t, w, http.StatusOK, &retrievedFamily)
		assert.Equal(t, createdFamily.UID, retrievedFamily.UID)
		assert.Equal(t, createdFamily.Name, retrievedFamily.Name)
	})

	t.Run("Get non-existent family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/families/nonexistent-uid", nil)
		w := httptest.NewRecorder()
		testHandler.handleFamily(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrFamilyNotFound)
	})
}

// TestTaskEndpoints tests task CRUD operations
func TestTaskEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	t.Run("Create Task", func(t *testing.T) {
		task := postgres.Tasks{
			Entities: postgres.Entities{
				FamilyUID:   setupFamily.UID,
				Name:        "Test Task",
				Description: "Test Description",
				Tokens:      10,
			},
		}
		body, _ := json.Marshal(task)
		req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleTasks(w, req)

		var response postgres.Tasks
		assertSuccessResponse(t, w, http.StatusCreated, &response)
		assert.Equal(t, task.Name, response.Name)
		assert.Equal(t, task.Tokens, response.Tokens)

		// Verify task is in the database
		var dbTask postgres.Tasks
		err := testHandler.db.Where("name = ?", "Test Task").First(&dbTask).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Name, dbTask.Name)
	})

	t.Run("List Tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
		w := httptest.NewRecorder()
		testHandler.handleTasks(w, req)

		var tasks []postgres.Tasks
		assertSuccessResponse(t, w, http.StatusOK, &tasks)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Test Task", tasks[0].Name)

		// Verify tasks are in the database
		var dbTasks []postgres.Tasks
		err := testHandler.db.Find(&dbTasks).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Task", dbTasks[0].Name)
	})

	t.Run("Get Tasks by Family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		testHandler.handleTasksByFamily(w, req)

		var tasks []postgres.Tasks
		assertSuccessResponse(t, w, http.StatusOK, &tasks)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Test Task", tasks[0].Name)
	})

	t.Run("Delete Task", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/tasks/"+setupFamily.UID+"/Test%20Task", nil)
		w := httptest.NewRecorder()
		testHandler.handleTasksByFamily(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify task is deleted from the database
		var dbTask postgres.Tasks
		err := testHandler.db.Where("family_uid = ? AND name = ?", setupFamily.UID, "Test Task").
			First(&dbTask).
			Error
		assert.Error(t, err)
	})

	t.Run("Delete non-existent task", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/tasks/"+setupFamily.UID+"/Nonexistent%20Task", nil)
		w := httptest.NewRecorder()
		testHandler.handleTasksByFamily(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrEntityNotFound)
	})

	t.Run("Delete task for non-existent family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/tasks/nonexistent-family/Test%20Task", nil)
		w := httptest.NewRecorder()
		testHandler.handleTasksByFamily(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrFamilyNotFound)
	})
}

// TestRewardEndpoints tests reward CRUD operations
func TestRewardEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	t.Run("Create Reward", func(t *testing.T) {
		reward := postgres.Rewards{
			Entities: postgres.Entities{
				FamilyUID:   setupFamily.UID,
				Name:        "Test Reward",
				Description: "Test Description",
				Tokens:      5,
			},
		}
		body, _ := json.Marshal(reward)
		req := httptest.NewRequest("POST", "/api/v1/rewards", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleRewards(w, req)

		var response postgres.Rewards
		assertSuccessResponse(t, w, http.StatusCreated, &response)
		assert.Equal(t, reward.Name, response.Name)
		assert.Equal(t, reward.Tokens, response.Tokens)

		// Verify reward is in the database
		var dbReward postgres.Rewards
		err := testHandler.db.Where("name = ?", "Test Reward").First(&dbReward).Error
		assert.NoError(t, err)
		assert.Equal(t, reward.Name, dbReward.Name)
	})

	t.Run("List Rewards", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rewards", nil)
		w := httptest.NewRecorder()
		testHandler.handleRewards(w, req)

		var rewards []postgres.Rewards
		assertSuccessResponse(t, w, http.StatusOK, &rewards)
		assert.Len(t, rewards, 1)
		assert.Equal(t, "Test Reward", rewards[0].Name)

		// Verify rewards are in the database
		var dbRewards []postgres.Rewards
		err := testHandler.db.Find(&dbRewards).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Reward", dbRewards[0].Name)
	})

	t.Run("Get Rewards by Family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/rewards/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		testHandler.handleRewardsByFamily(w, req)

		var rewards []postgres.Rewards
		assertSuccessResponse(t, w, http.StatusOK, &rewards)
		assert.Len(t, rewards, 1)
		assert.Equal(t, "Test Reward", rewards[0].Name)
	})

	t.Run("Delete Reward", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/rewards/"+setupFamily.UID+"/Test%20Reward", nil)
		w := httptest.NewRecorder()
		testHandler.handleRewardsByFamily(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify reward is deleted from the database
		var dbReward postgres.Rewards
		err := testHandler.db.Where("family_uid = ? AND name = ?", setupFamily.UID, "Test Reward").
			First(&dbReward).
			Error
		assert.Error(t, err)
	})

	t.Run("Delete non-existent reward", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/rewards/"+setupFamily.UID+"/Nonexistent%20Reward", nil)
		w := httptest.NewRecorder()
		testHandler.handleRewardsByFamily(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrEntityNotFound)
	})

	t.Run("Delete reward for non-existent family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/rewards/nonexistent-family/Test%20Reward", nil)
		w := httptest.NewRecorder()
		testHandler.handleRewardsByFamily(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrFamilyNotFound)
	})
}

// TestTokenEndpoints tests token CRUD operations
func TestTokenEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Create a test user first
	user := postgres.Users{
		UserID:    "test_child_123",
		Name:      "Test Child",
		Role:      "child",
		FamilyUID: setupFamily.UID,
		Platform:  "telegram",
	}
	testHandler.db.Create(&user)

	// Create tokens for the user
	tokens := postgres.Tokens{
		UserID: "test_child_123",
		Tokens: 50,
	}
	testHandler.db.Create(&tokens)

	t.Run("Get User Tokens", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tokens/test_child_123", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		var tokenResponse postgres.Tokens
		assertSuccessResponse(t, w, http.StatusOK, &tokenResponse)
		assert.Equal(t, "test_child_123", tokenResponse.UserID)
		assert.Equal(t, 50, tokenResponse.Tokens)
	})

	t.Run("Update User Tokens", func(t *testing.T) {
		tokensUpdate := postgres.Tokens{
			Tokens: 75,
		}
		body, _ := json.Marshal(tokensUpdate)
		req := httptest.NewRequest("PUT", "/api/v1/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		var response postgres.Tokens
		assertSuccessResponse(t, w, http.StatusOK, &response)
		assert.Equal(t, 75, response.Tokens)

		// Verify tokens are updated in the database
		var dbTokens postgres.Tokens
		err := testHandler.db.Where("user_id = ?", "test_child_123").First(&dbTokens).Error
		assert.NoError(t, err)
		assert.Equal(t, 75, dbTokens.Tokens)
	})

	t.Run("Add Tokens to User", func(t *testing.T) {
		addTokensRequest := map[string]interface{}{
			"amount":      10,
			"type":        "task_completed",
			"description": "Completed chores",
		}
		body, _ := json.Marshal(addTokensRequest)
		req := httptest.NewRequest("POST", "/api/v1/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		var response postgres.Tokens
		assertSuccessResponse(t, w, http.StatusOK, &response)
		assert.Equal(t, 85, response.Tokens) // 75 + 10

		// Verify token history was created
		var history []postgres.TokenHistory
		err := testHandler.db.Where("user_id = ?", "test_child_123").Find(&history).Error
		assert.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, 10, history[0].Amount)
		assert.Equal(t, "task_completed", history[0].Type)
	})

	t.Run("Subtract Tokens from User", func(t *testing.T) {
		subtractTokensRequest := map[string]interface{}{
			"amount":      -15,
			"type":        "reward_claimed",
			"description": "Bought toy",
		}
		body, _ := json.Marshal(subtractTokensRequest)
		req := httptest.NewRequest("POST", "/api/v1/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		var response postgres.Tokens
		assertSuccessResponse(t, w, http.StatusOK, &response)
		assert.Equal(t, 70, response.Tokens) // 85 - 15
	})

	t.Run("Subtract More Tokens than Available", func(t *testing.T) {
		subtractTokensRequest := map[string]interface{}{
			"amount":      -100,
			"type":        "reward_claimed",
			"description": "Expensive toy",
		}
		body, _ := json.Marshal(subtractTokensRequest)
		req := httptest.NewRequest("POST", "/api/v1/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, ErrInsufficientTokens)
	})

	t.Run("Get Tokens for non-existent user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tokens/nonexistent_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		assertErrorResponse(t, w, http.StatusNotFound, ErrUserTokensNotFound)
	})

	t.Run("List All Tokens", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
		w := httptest.NewRecorder()
		testHandler.handleTokens(w, req)

		var allTokens []postgres.Tokens
		assertSuccessResponse(t, w, http.StatusOK, &allTokens)
		assert.Len(t, allTokens, 1)
		assert.Equal(t, "test_child_123", allTokens[0].UserID)
	})

	t.Run("Delete User Tokens (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/tokens/test_child_123", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokens(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})
}

// TestTokenHistoryEndpoints tests token history operations
func TestTokenHistoryEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Create a test user first
	user := postgres.Users{
		UserID:    "test_history_user",
		Name:      "Test History User",
		Role:      "child",
		FamilyUID: setupFamily.UID,
	}
	testHandler.db.Create(&user)

	// Create some token history
	history1 := postgres.TokenHistory{
		UserID:      "test_history_user",
		Amount:      10,
		Type:        "task_completed",
		Description: "Completed task",
	}
	history2 := postgres.TokenHistory{
		UserID:      "test_history_user",
		Amount:      -5,
		Type:        "reward_claimed",
		Description: "Первая награда",
	}
	testHandler.db.Create(&history1)
	testHandler.db.Create(&history2)

	t.Run("Get User Token History", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokenHistory(w, req)

		var history []postgres.TokenHistory
		assertSuccessResponse(t, w, http.StatusOK, &history)
		assert.Len(t, history, 2)
	})

	t.Run("Get User Token History (without pagination)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokenHistory(w, req)

		var history []postgres.TokenHistory
		assertSuccessResponse(t, w, http.StatusOK, &history)
		assert.Len(t, history, 2) // Should return all records since pagination is removed
	})

	t.Run("List All Token History", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/token-history", nil)
		w := httptest.NewRecorder()
		testHandler.handleTokenHistory(w, req)

		var allHistory []postgres.TokenHistory
		assertSuccessResponse(t, w, http.StatusOK, &allHistory)
		assert.Len(t, allHistory, 2)
	})

	t.Run("Create Token History (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/token-history", nil)
		w := httptest.NewRecorder()
		testHandler.handleTokenHistory(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})

	t.Run("Update Token History (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/token-history", nil)
		w := httptest.NewRecorder()
		testHandler.handleTokenHistory(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})

	t.Run("Delete Token History (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/token-history", nil)
		w := httptest.NewRecorder()
		testHandler.handleTokenHistory(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})

	t.Run("Create User Token History (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokenHistory(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})

	t.Run("Update User Token History (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokenHistory(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})

	t.Run("Delete User Token History (not implemented)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		testHandler.handleUserTokenHistory(w, req)

		assertErrorResponse(t, w, http.StatusNotImplemented, ErrNotImplemented)
	})
}
