package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/postgres"
	"github.com/stretchr/testify/assert"
	psql "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const dsn = "host=localhost user=postgres password=password dbname=joytime port=55667 sslmode=disable"

func setupTestDB(t *testing.T) *postgres.Families {
	level := log.InfoLevel
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
	})

	var err error

	// db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db, err = gorm.Open(psql.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			logger,
			gormlogger.Config{
				// SlowThreshold:             time.Second,       // Slow SQL threshold
				// LogLevel:                  gormlogger.Silent, // Log level
				IgnoreRecordNotFoundError: true, // Ignore ErrRecordNotFound error for logger
				// ParameterizedQueries:      true,              // Don't include params in the SQL log
				// Colorful:                  false,             // Disable color
			},
		),
	})
	assert.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(
		&postgres.Users{},
		&postgres.Families{},
		&postgres.Tasks{},
		&postgres.Rewards{},
		&postgres.Tokens{},
	)
	assert.NoError(t, err)

	// create a family
	family := postgres.Families{
		Name: t.Name(),
		UID:  t.Name(),
	}
	db.Create(&family)
	return &family
}

func TestUserEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)
	defer db.Delete(setupFamily)

	// Test creating a user
	t.Run("Create User", func(t *testing.T) {
		user := postgres.Users{
			UID:      "test-user",
			Name:     "Test User",
			Role:     "parent",
			FamilyID: 1,
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
		assert.Equal(t, user.UID, response.UID)
		assert.Equal(t, user.Name, response.Name)
		assert.Equal(t, user.Role, response.Role)
		assert.Equal(t, user.FamilyID, response.FamilyID)

		// Verify user is in the database
		var dbUser postgres.Users
		err = db.Where("uid = ?", "test-user").First(&dbUser).Error
		assert.NoError(t, err)
		assert.Equal(t, user.UID, dbUser.UID)
	})

	// Test creating a user with wrong family id (family not found)
	t.Run("Create User with wrong family id", func(t *testing.T) {
		// family not found
		user := postgres.Users{
			UID:      "test-user",
			Name:     "Test User",
			Role:     "parent",
			FamilyID: 2,
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
			UID:      "test-user",
			Name:     "Test User",
			Role:     "stranger",
			FamilyID: 1,
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
			UID:      "test-user",
			Name:     "Test User",
			FamilyID: 1,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer([]byte(body)))
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: Role")

		// missing Name
		user = postgres.Users{
			UID:      "test-user",
			Role:     "parent",
			FamilyID: 1,
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: Name")

		// missing UID
		user = postgres.Users{
			Name:     "Test User",
			Role:     "parent",
			FamilyID: 1,
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: UID")

		// missing FamilyID
		user = postgres.Users{
			UID:  "test-user",
			Name: "Test User",
			Role: "parent",
		}
		body, _ = json.Marshal(user)
		req = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing required fields: FamilyID")
	})

	// Test creating an existing user
	t.Run("Create existing user", func(t *testing.T) {
		user := postgres.Users{
			UID:      "test-user",
			Name:     "Test User",
			Role:     "parent",
			FamilyID: 1,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(
			t,
			w.Body.String(),
			"ERROR: duplicate key value violates unique constraint \"uni_users_uid\" (SQLSTATE 23505)",
		)
	})

	// Test getting a user
	t.Run("Get User", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/test-user", nil)
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Users
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "test-user", response.UID)
	})

	// Test getting an absent user
	t.Run("Get absent user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/absent-user", nil)
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not found")
	})

	// Test updating a user
	t.Run("Update User", func(t *testing.T) {
		user := postgres.Users{
			UID:  "test-user",
			Name: "Updated User",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/test-user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response postgres.Users
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated User", response.Name)

		// Verify user is updated in the database
		var updatedUser postgres.Users
		err = db.Where("uid = ?", "test-user").First(&updatedUser).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated User", updatedUser.Name)
	})

	// Test updating a user with wrong role
	t.Run("Update User with wrong role", func(t *testing.T) {
		user := postgres.Users{
			UID:  "test-user",
			Role: "stranger",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/test-user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid role: parent or child only")
	})

	// Test updating a non-existent user
	t.Run("Update non-existent user", func(t *testing.T) {
		user := postgres.Users{
			UID:  "non-existent-user",
			Role: "parent",
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/non-existent-user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// Test updating a user with wrong family id
	t.Run("Update User with wrong family id (family not found)", func(t *testing.T) {
		// family not found
		user := postgres.Users{
			UID:      "test-user",
			Name:     "Updated User",
			Role:     "parent",
			FamilyID: 2,
		}
		body, _ := json.Marshal(user)
		req := httptest.NewRequest("PUT", "/users/test-user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Family not found")

		// Verify user is not updated in the database
		var updatedUser postgres.Users
		err := db.Where("uid = ?", "test-user").First(&updatedUser).Error
		assert.NoError(t, err)
		assert.Equal(t, 1, updatedUser.FamilyID)
	})

	// Test updating a user to existing uid
	t.Run("Update User to existing uid", func(t *testing.T) {
		user1 := postgres.Users{
			UID: "existing-uid",
		}
		user2 := postgres.Users{
			UID: "existing-uid",
		}
		db.Create(&user1)
		defer db.Delete(&user1)

		body, _ := json.Marshal(user2)
		req := httptest.NewRequest("PUT", "/users/test-user", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(
			t,
			w.Body.String(),
			"ERROR: duplicate key value violates unique constraint \"uni_users_uid\" (SQLSTATE 23505)",
		)
	})

	// Test deleting a user
	t.Run("Delete User", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/test-user", nil)
		w := httptest.NewRecorder()
		handleUser(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify user is deleted from the database
		var dbUser postgres.Users
		err := db.Where("uid = ?", "test-user").First(&dbUser).Error
		assert.Error(t, err)
	})
}

func TestFamilyEndpoints(t *testing.T) {
	setupFamily := setupTestDB(t)
	// defer db.Delete(setupFamily)

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

		// Delete family
		db.Delete(&dbFamily)
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
			CreatedByUserID: 1,
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

	// Test listing families
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
			UID: "existing-uid",
		}
		family2 := postgres.Families{
			UID: "existing-uid",
		}
		db.Create(&family1)
		defer db.Delete(&family1)

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
}

// func TestTaskEndpoints(t *testing.T) {
// 	setupTestDB(t)

// 	// Test creating a task
// 	t.Run("Create Task", func(t *testing.T) {
// 		task := postgres.Entities{
// 			Name:        "Test Task",
// 			FamilyID:    1,
// 			Description: "Test Description",
// 			Tokens:      10,
// 			OneOff:      false,
// 		}
// 		body, _ := json.Marshal(task)
// 		req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
// 		w := httptest.NewRecorder()
// 		handleTasks(w, req)

// 		assert.Equal(t, http.StatusCreated, w.Code)

// 		var response postgres.Tasks
// 		err := json.NewDecoder(w.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, task.Name, response.Name)
// 		assert.Equal(t, task.FamilyID, response.FamilyID)
// 		assert.Equal(t, task.Description, response.Description)
// 		assert.Equal(t, task.Tokens, response.Tokens)
// 		assert.Equal(t, task.OneOff, response.OneOff)

// 		// Verify task is in the database
// 		var dbTask postgres.Tasks
// 		err = db.Where("name = ?", "Test Task").First(&dbTask).Error
// 		assert.NoError(t, err)
// 		assert.Equal(t, task.Name, dbTask.Name)
// 		assert.Equal(t, task.FamilyID, dbTask.FamilyID)
// 		assert.Equal(t, task.Description, dbTask.Description)
// 		assert.Equal(t, task.Tokens, dbTask.Tokens)
// 		assert.Equal(t, task.OneOff, dbTask.OneOff)
// 	})

// 	// Test listing tasks
// 	t.Run("List Tasks", func(t *testing.T) {
// 		req := httptest.NewRequest("GET", "/tasks", nil)
// 		w := httptest.NewRecorder()
// 		handleTasks(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response []postgres.Tasks
// 		err := json.NewDecoder(w.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		assert.Len(t, response, 1)
// 		assert.Equal(t, "Test Task", response[0].Name)
// 		assert.Equal(t, "Test Description", response[0].Description)
// 		assert.Equal(t, 10, response[0].Tokens)
// 		assert.Equal(t, false, response[0].OneOff)

// 		// Verify tasks are in the database
// 		var dbTasks []postgres.Tasks
// 		err = db.Find(&dbTasks).Error
// 		assert.NoError(t, err)
// 		assert.Equal(t, "Test Task", dbTasks[0].Name)
// 		assert.Equal(t, 1, dbTasks[0].FamilyID)
// 		assert.Equal(t, "Test Description", dbTasks[0].Description)
// 		assert.Equal(t, 10, dbTasks[0].Tokens)
// 		assert.Equal(t, false, dbTasks[0].OneOff)
// 	})
// }

// func TestRewardEndpoints(t *testing.T) {
// 	setupTestDB(t)

// 	// Test creating a reward
// 	t.Run("Create Reward", func(t *testing.T) {
// 		reward := postgres.Entities{
// 			Name:        "Test Reward",
// 			FamilyID:    1,
// 			Description: "Test Description",
// 			Tokens:      10,
// 			OneOff:      false,
// 		}
// 		body, _ := json.Marshal(reward)
// 		req := httptest.NewRequest("POST", "/rewards", bytes.NewBuffer(body))
// 		w := httptest.NewRecorder()
// 		handleRewards(w, req)

// 		assert.Equal(t, http.StatusCreated, w.Code)

// 		var response postgres.Rewards
// 		err := json.NewDecoder(w.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, reward.Name, response.Name)
// 		assert.Equal(t, reward.FamilyID, response.FamilyID)
// 		assert.Equal(t, reward.Description, response.Description)
// 		assert.Equal(t, reward.Tokens, response.Tokens)
// 		assert.Equal(t, reward.OneOff, response.OneOff)

// 		// Verify reward is in the database
// 		var dbReward postgres.Rewards
// 		err = db.Where("name = ?", "Test Reward").First(&dbReward).Error
// 		assert.NoError(t, err)
// 		assert.Equal(t, reward.Name, dbReward.Name)
// 		assert.Equal(t, reward.FamilyID, dbReward.FamilyID)
// 		assert.Equal(t, reward.Description, dbReward.Description)
// 		assert.Equal(t, reward.Tokens, dbReward.Tokens)
// 		assert.Equal(t, reward.OneOff, dbReward.OneOff)
// 	})
// 	// Test listing rewards
// 	t.Run("List Rewards", func(t *testing.T) {
// 		req := httptest.NewRequest("GET", "/rewards", nil)
// 		w := httptest.NewRecorder()
// 		handleRewards(w, req)

// 		assert.Equal(t, http.StatusOK, w.Code)

// 		var response []postgres.Rewards
// 		err := json.NewDecoder(w.Body).Decode(&response)
// 		assert.NoError(t, err)
// 		assert.Len(t, response, 1)
// 		assert.Equal(t, "Test Reward", response[0].Name)
// 		assert.Equal(t, "Test Description", response[0].Description)
// 		assert.Equal(t, 10, response[0].Tokens)
// 		assert.Equal(t, false, response[0].OneOff)

// 		// Verify rewards are in the database
// 		var dbRewards []postgres.Rewards
// 		err = db.Find(&dbRewards).Error
// 		assert.NoError(t, err)
// 		assert.Equal(t, "Test Reward", dbRewards[0].Name)
// 		assert.Equal(t, 1, dbRewards[0].FamilyID)
// 		assert.Equal(t, "Test Description", dbRewards[0].Description)
// 		assert.Equal(t, 10, dbRewards[0].Tokens)
// 		assert.Equal(t, false, dbRewards[0].OneOff)
// 	})
// }
