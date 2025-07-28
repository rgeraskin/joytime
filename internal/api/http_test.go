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

// cleanupTestData removes all test data from database in correct order to handle foreign key constraints
func cleanupTestData() {
	db.Unscoped().Where("1 = 1").Delete(&postgres.TokenHistory{})
	db.Unscoped().Where("1 = 1").Delete(&postgres.Tokens{})
	db.Unscoped().Where("1 = 1").Delete(&postgres.Tasks{})
	db.Unscoped().Where("1 = 1").Delete(&postgres.Rewards{})
	db.Unscoped().Where("1 = 1").Delete(&postgres.Users{})
	db.Unscoped().Where("1 = 1").Delete(&postgres.Families{})
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

	return db.AutoMigrate(models...)
}

// setupTestDBConnection establishes database connection for testing
func setupTestDBConnection() error {
	level := log.InfoLevel
	logger = log.NewWithOptions(os.Stderr, log.Options{
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
	db, err = gorm.Open(psql.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			logger,
			gormlogger.Config{
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	return err
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
	result := db.Create(&family)
	if result.Error != nil {
		t.Fatalf("Failed to create family: %v", result.Error)
	}

	return &family
}

func TestUserEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Test creating a user
	t.Run("Create User", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_123",
			Name:      "Test User",
			Role:      "parent",
			FamilyUID: setupFamily.UID,
			Platform:  "telegram",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		fmt.Println(req)
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response postgres.Users
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, user.UserID, response.UserID)
		assert.Equal(t, user.Name, response.Name)
		assert.Equal(t, user.Role, response.Role)
		assert.Equal(t, user.FamilyUID, response.FamilyUID)
		assert.Equal(t, user.Platform, response.Platform)

		// Verify user is in the database
		var dbUser postgres.Users
		err = db.Where("user_id = ?", "test_user_123").First(&dbUser).Error
		assert.NoError(t, err)
		assert.Equal(t, user.UserID, dbUser.UserID)
	})

	// Test creating a user with wrong family uid (family not found)
	t.Run("Create User with wrong family uid", func(t *testing.T) {
		// family not found
		user := postgres.Users{
			UserID:    "test_user_456",
			Name:      "Test User",
			Role:      "parent",
			FamilyUID: "nonexistent",
			Platform:  "web",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Family not found")
	})

	// Test creating a user with wrong role
	t.Run("Create User with wrong role", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_789",
			Name:      "Test User",
			Role:      "stranger",
			FamilyUID: setupFamily.UID,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid role: parent or child only")
	})

	// Test creating a user with missing required fields
	t.Run("Create User with missing required fields", func(t *testing.T) {
		// missing Role
		user := postgres.Users{
			UserID:    "test_user_missing_role",
			Name:      "Test User",
			FamilyUID: setupFamily.UID,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: Role")

		// missing Name
		user = postgres.Users{
			UserID:    "test_user_missing_name",
			Role:      "parent",
			FamilyUID: setupFamily.UID,
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: Name")

		// missing UserID
		user = postgres.Users{
			Name:      "Test User",
			Role:      "parent",
			FamilyUID: setupFamily.UID,
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: UserID")

		// missing FamilyUID
		user = postgres.Users{
			UserID: "test_user_missing_family",
			Name:   "Test User",
			Role:   "parent",
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: FamilyUID")
	})

	// Test creating an existing user
	t.Run("Create existing user", func(t *testing.T) {
		user := postgres.Users{
			UserID:    "test_user_123",
			Name:      "Test User",
			Role:      "parent",
			FamilyUID: setupFamily.UID,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(
			t,
			w.Body.String(),
			"ERROR: duplicate key value violates unique constraint \"uni_users_user_id\" (SQLSTATE 23505)",
		)
	})

	// Test getting a user
	t.Run("Get User", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/test_user_123", nil)
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Users
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "test_user_123", response.UserID)
	})

	// Test getting an absent user
	t.Run("Get absent user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/absent_user", nil)
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not found")
	})

	// Test updating a user
	t.Run("Update User", func(t *testing.T) {
		user := postgres.Users{
			UserID: "test_user_123",
			Name:   "Updated User",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Users
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated User", response.Name)

		// Verify user is updated in the database
		var updatedUser postgres.Users
		err = db.Where("user_id = ?", "test_user_123").First(&updatedUser).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated User", updatedUser.Name)
	})

	// Test updating a user with wrong role
	t.Run("Update User with wrong role", func(t *testing.T) {
		user := postgres.Users{
			UserID: "test_user_123",
			Role:   "stranger",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid role: parent or child only")
	})

	// Test updating a non-existent user
	t.Run("Update non-existent user", func(t *testing.T) {
		user := postgres.Users{
			UserID: "nonexistent_user",
			Role:   "parent",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/nonexistent_user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test updating a user with wrong family uid
	t.Run("Update User with wrong family uid (family not found)", func(t *testing.T) {
		// family not found
		user := postgres.Users{
			UserID:    "test_user_123",
			Name:      "Updated User",
			Role:      "parent",
			FamilyUID: "nonexistent",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Family not found")

		// Verify user is not updated in the database
		var updatedUser postgres.Users
		err := db.Where("user_id = ?", "test_user_123").First(&updatedUser).Error
		assert.NoError(t, err)
		assert.Equal(t, setupFamily.UID, updatedUser.FamilyUID)
	})

	// Test updating a user to existing user_id
	t.Run("Update User to existing user_id", func(t *testing.T) {
		user1 := postgres.Users{
			UserID:    "existing_user_123",
			Name:      "Another User",
			Role:      "parent",
			FamilyUID: setupFamily.UID,
		}
		user2 := postgres.Users{
			UserID: "existing_user_123",
		}
		db.Create(&user1)

		body, _ := json.Marshal(user2)
		req := httptest.NewRequest("PUT", "/users/test_user_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(
			t,
			w.Body.String(),
			"ERROR: duplicate key value violates unique constraint \"uni_users_user_id\" (SQLSTATE 23505)",
		)
	})

	// Test deleting a user
	t.Run("Delete User", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/test_user_123", nil)
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify user is deleted from the database
		var dbUser postgres.Users
		err := db.Where("user_id = ?", "test_user_123").First(&dbUser).Error
		assert.Error(t, err)
	})
}

func TestFamilyEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Test listing families (run first before any families are created)
	t.Run("List Families", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/families", nil)
		w := httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.Families
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "TestFamilyEndpoints", response[0].Name)

		// Verify families are in the database
		var dbFamilies []postgres.Families
		err = db.Find(&dbFamilies).Error
		assert.NoError(t, err)
		assert.Equal(t, "TestFamilyEndpoints", dbFamilies[0].Name)
	})

	// Test creating a family
	t.Run("Create Family", func(t *testing.T) {
		family := postgres.Families{
			Name: "Test Family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response postgres.Families
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, family.Name, response.Name)
		uid := response.UID

		// Verify family is in the database
		var dbFamily postgres.Families
		err = db.Where("uid = ?", uid).First(&dbFamily).Error
		assert.NoError(t, err)
		assert.Equal(t, family.Name, dbFamily.Name)

		// Note: Family cleanup handled by teardown function
	})

	// Test creating a family with restricted fields
	t.Run("Create Family with restricted fields", func(t *testing.T) {
		// uid
		family := postgres.Families{
			Name: "Test Family Restricted Fields 1",
			UID:  "test-family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Restricted fields: UID")

		// created by user id
		family = postgres.Families{
			Name:            "Test Family Restricted Fields 2",
			CreatedByUserID: "user_123",
		}
		body, _ = json.Marshal(family)
		req = httptest.NewRequest("POST", "/families", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Restricted fields: CreatedByUserID")
	})

	// Test creating a family with missing required fields
	t.Run("Create Family with missing required fields", func(t *testing.T) {
		// missing name
		family := postgres.Families{}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: Name")
	})

	// Test updating a family
	t.Run("Update Family", func(t *testing.T) {
		family := postgres.Families{
			Name: "Updated Family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("PUT", "/families/"+setupFamily.UID, bytes.NewBuffer(body))
		fmt.Println(req.URL.Path)
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Families
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Family", response.Name)
	})

	// Test update family without required fields
	t.Run("Update family without required fields", func(t *testing.T) {
		family := postgres.Families{}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("PUT", "/families/"+setupFamily.UID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: Name or UID")
	})

	// Test update non-existent family
	t.Run("Update non-existent family", func(t *testing.T) {
		family := postgres.Families{
			Name: "Updated Family",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("PUT", "/families/non-existent-family", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test delete non-existent family
	t.Run("Delete non-existent family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/families/non-existent-family", nil)
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test updating a family to existing uid
	t.Run("Update Family to existing uid", func(t *testing.T) {
		family1 := postgres.Families{
			Name: "Existing Family",
			UID:  "existing-uid",
		}
		family2 := postgres.Families{
			UID: "existing-uid",
		}
		db.Create(&family1)

		body, _ := json.Marshal(family2)
		req := httptest.NewRequest("PUT", "/families/"+setupFamily.UID, bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(
			t,
			w.Body.String(),
			"ERROR: duplicate key value violates unique constraint \"uni_families_uid\" (SQLSTATE 23505)",
		)
	})

	// Test deleting a family
	t.Run("Delete Family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/families/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify family is deleted from the database
		var dbFamily postgres.Families
		err := db.Where("uid = ?", setupFamily.UID).First(&dbFamily).Error
		assert.Error(t, err)
	})

	// Test getting a single family
	t.Run("Get Single Family", func(t *testing.T) {
		// Create a family for this test
		family := postgres.Families{
			Name: "Single Family Test",
		}
		body, _ := json.Marshal(family)
		req := httptest.NewRequest("POST", "/families", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var createdFamily postgres.Families
		err := json.NewDecoder(w.Body).Decode(&createdFamily)
		assert.NoError(t, err)

		// Now get the single family
		req = httptest.NewRequest("GET", "/families/"+createdFamily.UID, nil)
		w = httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Families
		err = json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, createdFamily.UID, response.UID)
		assert.Equal(t, "Single Family Test", response.Name)
	})

	// Test getting a non-existent single family
	t.Run("Get Non-existent Single Family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/families/nonexistent-uid", nil)
		w := httptest.NewRecorder()
		handleFamily(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Family not found")
	})
}

func TestTaskEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Test creating a task
	t.Run("Create Task", func(t *testing.T) {
		task := postgres.Tasks{
			Entities: postgres.Entities{
				Name:        "Test Task",
				FamilyUID:   setupFamily.UID,
				Description: "Test Description",
				Tokens:      10,
			},
			OneOff: false,
		}
		body, _ := json.Marshal(task)
		req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleTasks(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response postgres.Tasks
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, task.Name, response.Name)
		assert.Equal(t, task.FamilyUID, response.FamilyUID)
		assert.Equal(t, task.Description, response.Description)
		assert.Equal(t, task.Tokens, response.Tokens)
		assert.Equal(t, task.OneOff, response.OneOff)

		// Verify task is in the database
		var dbTask postgres.Tasks
		err = db.Where("name = ?", "Test Task").First(&dbTask).Error
		assert.NoError(t, err)
		assert.Equal(t, task.Name, dbTask.Name)
		assert.Equal(t, task.FamilyUID, dbTask.FamilyUID)
		assert.Equal(t, task.Description, dbTask.Description)
		assert.Equal(t, task.Tokens, dbTask.Tokens)
		assert.Equal(t, task.OneOff, dbTask.OneOff)
	})

	// Test listing tasks
	t.Run("List Tasks", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tasks", nil)
		w := httptest.NewRecorder()
		handleTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.Tasks
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "Test Task", response[0].Name)
		assert.Equal(t, "Test Description", response[0].Description)
		assert.Equal(t, 10, response[0].Tokens)
		assert.Equal(t, false, response[0].OneOff)

		// Verify tasks are in the database
		var dbTasks []postgres.Tasks
		err = db.Find(&dbTasks).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Task", dbTasks[0].Name)
		assert.Equal(t, setupFamily.UID, dbTasks[0].FamilyUID)
		assert.Equal(t, "Test Description", dbTasks[0].Description)
		assert.Equal(t, 10, dbTasks[0].Tokens)
		assert.Equal(t, false, dbTasks[0].OneOff)
	})

	// Test getting tasks by family
	t.Run("Get Tasks by Family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tasks/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		handleTask(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.Tasks
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "Test Task", response[0].Name)
	})

	// Test deleting a task by family and name
	t.Run("Delete Task", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/task/"+setupFamily.UID+"/Test%20Task", nil)
		w := httptest.NewRecorder()
		handleSingleTask(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify task is deleted from the database
		var dbTask postgres.Tasks
		err := db.Where("family_uid = ? AND name = ?", setupFamily.UID, "Test Task").
			First(&dbTask).
			Error
		assert.Error(t, err)
	})

	// Test deleting a non-existent task
	t.Run("Delete Non-existent Task", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/task/"+setupFamily.UID+"/Nonexistent%20Task", nil)
		w := httptest.NewRecorder()
		handleSingleTask(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Entity not found")
	})

	// Test deleting a task with wrong family
	t.Run("Delete Task with Wrong Family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/task/nonexistent-family/Test%20Task", nil)
		w := httptest.NewRecorder()
		handleSingleTask(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Family not found")
	})
}

func TestRewardEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Test creating a reward
	t.Run("Create Reward", func(t *testing.T) {
		reward := postgres.Rewards{
			Entities: postgres.Entities{
				Name:        "Test Reward",
				FamilyUID:   setupFamily.UID,
				Description: "Test Description",
				Tokens:      10,
			},
		}
		body, _ := json.Marshal(reward)
		req := httptest.NewRequest("POST", "/rewards", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleRewards(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response postgres.Rewards
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, reward.Name, response.Name)
		assert.Equal(t, reward.FamilyUID, response.FamilyUID)
		assert.Equal(t, reward.Description, response.Description)
		assert.Equal(t, reward.Tokens, response.Tokens)

		// Verify reward is in the database
		var dbReward postgres.Rewards
		err = db.Where("name = ?", "Test Reward").First(&dbReward).Error
		assert.NoError(t, err)
		assert.Equal(t, reward.Name, dbReward.Name)
		assert.Equal(t, reward.FamilyUID, dbReward.FamilyUID)
		assert.Equal(t, reward.Description, dbReward.Description)
		assert.Equal(t, reward.Tokens, dbReward.Tokens)
	})

	// Test listing rewards
	t.Run("List Rewards", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rewards", nil)
		w := httptest.NewRecorder()
		handleRewards(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.Rewards
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "Test Reward", response[0].Name)
		assert.Equal(t, "Test Description", response[0].Description)
		assert.Equal(t, 10, response[0].Tokens)

		// Verify rewards are in the database
		var dbRewards []postgres.Rewards
		err = db.Find(&dbRewards).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Reward", dbRewards[0].Name)
		assert.Equal(t, setupFamily.UID, dbRewards[0].FamilyUID)
		assert.Equal(t, "Test Description", dbRewards[0].Description)
		assert.Equal(t, 10, dbRewards[0].Tokens)
	})

	// Test getting rewards by family
	t.Run("Get Rewards by Family", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rewards/"+setupFamily.UID, nil)
		w := httptest.NewRecorder()
		handleReward(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.Rewards
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "Test Reward", response[0].Name)
	})

	// Test deleting a reward by family and name
	t.Run("Delete Reward", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/reward/"+setupFamily.UID+"/Test%20Reward", nil)
		w := httptest.NewRecorder()
		handleSingleReward(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify reward is deleted from the database
		var dbReward postgres.Rewards
		err := db.Where("family_uid = ? AND name = ?", setupFamily.UID, "Test Reward").
			First(&dbReward).
			Error
		assert.Error(t, err)
	})

	// Test deleting a non-existent reward
	t.Run("Delete Non-existent Reward", func(t *testing.T) {
		req := httptest.NewRequest(
			"DELETE",
			"/reward/"+setupFamily.UID+"/Nonexistent%20Reward",
			nil,
		)
		w := httptest.NewRecorder()
		handleSingleReward(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Entity not found")
	})

	// Test deleting a reward with wrong family
	t.Run("Delete Reward with Wrong Family", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/reward/nonexistent-family/Test%20Reward", nil)
		w := httptest.NewRecorder()
		handleSingleReward(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Family not found")
	})
}

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
	db.Create(&user)

	// Create tokens for the user
	tokens := postgres.Tokens{
		UserID: "test_child_123",
		Tokens: 50,
	}
	db.Create(&tokens)

	// Test getting user tokens
	t.Run("Get User Tokens", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tokens/test_child_123", nil)
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Tokens
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "test_child_123", response.UserID)
		assert.Equal(t, 50, response.Tokens)
	})

	// Test updating user tokens
	t.Run("Update User Tokens", func(t *testing.T) {
		updateTokens := postgres.Tokens{
			Tokens: 75,
		}
		body, _ := json.Marshal(updateTokens)
		req := httptest.NewRequest("PUT", "/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Tokens
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, 75, response.Tokens)

		// Verify tokens are updated in the database
		var dbTokens postgres.Tokens
		err = db.Where("user_id = ?", "test_child_123").First(&dbTokens).Error
		assert.NoError(t, err)
		assert.Equal(t, 75, dbTokens.Tokens)
	})

	// Test adding tokens to user
	t.Run("Add Tokens to User", func(t *testing.T) {
		addRequest := struct {
			Amount      int    `json:"amount"`
			Type        string `json:"type"`
			Description string `json:"description"`
		}{
			Amount:      10,
			Type:        "task_completed",
			Description: "Completed task",
		}
		body, _ := json.Marshal(addRequest)
		req := httptest.NewRequest("POST", "/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Tokens
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, 85, response.Tokens) // 75 + 10

		// Verify token history was created
		var history []postgres.TokenHistory
		err = db.Where("user_id = ?", "test_child_123").Find(&history).Error
		assert.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, 10, history[0].Amount)
		assert.Equal(t, "task_completed", history[0].Type)
	})

	// Test subtracting tokens from user
	t.Run("Subtract Tokens from User", func(t *testing.T) {
		subtractRequest := struct {
			Amount      int    `json:"amount"`
			Type        string `json:"type"`
			Description string `json:"description"`
		}{
			Amount:      -5,
			Type:        "reward_claimed",
			Description: "Получил награду",
		}
		body, _ := json.Marshal(subtractRequest)
		req := httptest.NewRequest("POST", "/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Tokens
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, 80, response.Tokens) // 85 - 5
	})

	// Test insufficient tokens
	t.Run("Insufficient Tokens", func(t *testing.T) {
		subtractRequest := struct {
			Amount      int    `json:"amount"`
			Type        string `json:"type"`
			Description string `json:"description"`
		}{
			Amount:      -100,
			Type:        "reward_claimed",
			Description: "Попытка получить дорогую награду",
		}
		body, _ := json.Marshal(subtractRequest)
		req := httptest.NewRequest("POST", "/tokens/test_child_123", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Insufficient tokens")
	})

	// Test getting tokens for non-existent user
	t.Run("Get Tokens for non-existent user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tokens/nonexistent_user", nil)
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User tokens not found")
	})

	// Test listing all tokens
	t.Run("List All Tokens", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tokens", nil)
		w := httptest.NewRecorder()
		handleTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.Tokens
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "test_child_123", response[0].UserID)
		assert.Equal(t, 80, response[0].Tokens)
	})

	// Test deleting tokens (not implemented)
	t.Run("Delete Tokens - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/tokens/test_child_123", nil)
		w := httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})
}

func TestTokenHistoryEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)

	// Create a test user first
	user := postgres.Users{
		UserID:    "test_history_user",
		Name:      "Test History User",
		Role:      "child",
		FamilyUID: setupFamily.UID,
	}
	db.Create(&user)

	// Create some token history
	history1 := postgres.TokenHistory{
		UserID:      "test_history_user",
		Amount:      10,
		Type:        "task_completed",
		Description: "Первое задание",
	}
	history2 := postgres.TokenHistory{
		UserID:      "test_history_user",
		Amount:      -5,
		Type:        "reward_claimed",
		Description: "Первая награда",
	}
	db.Create(&history1)
	db.Create(&history2)

	// Test getting user token history
	t.Run("Get User Token History", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.TokenHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)

		// History should be ordered by created_at DESC
		assert.Equal(t, "test_history_user", response[0].UserID)
		assert.Equal(t, "test_history_user", response[1].UserID)
	})

	// Test getting user token history with pagination
	t.Run("Get User Token History with Pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/token-history/test_history_user?limit=1&offset=0", nil)
		w := httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.TokenHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "test_history_user", response[0].UserID)
	})

	// Test listing all token history
	t.Run("List All Token History", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/token-history", nil)
		w := httptest.NewRecorder()
		handleTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []postgres.TokenHistory
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)
	})

	// Test creating token history (not implemented)
	t.Run("Create Token History - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/token-history", nil)
		w := httptest.NewRecorder()
		handleTokenHistory(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})

	// Test updating all token history (not implemented)
	t.Run("Update All Token History - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/token-history", nil)
		w := httptest.NewRecorder()
		handleTokenHistory(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})

	// Test deleting all token history (not implemented)
	t.Run("Delete All Token History - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/token-history", nil)
		w := httptest.NewRecorder()
		handleTokenHistory(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})

	// Test creating user token history (not implemented)
	t.Run("Create User Token History - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})

	// Test updating user token history (not implemented)
	t.Run("Update User Token History - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})

	// Test deleting user token history (not implemented)
	t.Run("Delete User Token History - Not Implemented", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/token-history/test_history_user", nil)
		w := httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "Not implemented")
	})
}
