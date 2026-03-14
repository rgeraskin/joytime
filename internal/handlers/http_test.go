package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/database"
	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	psql "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var testHandler *APIHandler

// parseSuccessResponse parses a success response and extracts the data
func parseSuccessResponse(w *httptest.ResponseRecorder, target any) error {
	var successResponse SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&successResponse); err != nil {
		return err
	}

	// Marshal and unmarshal to convert any to target type
	dataBytes, err := json.Marshal(successResponse.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(dataBytes, target)
}

// assertSuccessResponse checks status code and parses success response
func assertSuccessResponse(
	t *testing.T,
	w *httptest.ResponseRecorder,
	expectedStatus int,
	target any,
) {
	assert.Equal(t, expectedStatus, w.Code)
	err := parseSuccessResponse(w, target)
	assert.NoError(t, err)
}

// cleanupTestData removes all test data from database in correct order to handle foreign key constraints
func cleanupTestData() {
	testHandler.db.Unscoped().Where("1 = 1").Delete(&models.TokenHistory{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&models.Tokens{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&models.Tasks{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&models.Rewards{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&models.Users{})
	testHandler.db.Unscoped().Where("1 = 1").Delete(&models.Families{})
}

// migrateTestSchema migrates all required models for testing
func migrateTestSchema(includeEntities bool) error {
	modelsList := []any{
		&models.Users{},
		&models.Families{},
	}

	if includeEntities {
		modelsList = append(modelsList, &models.Entities{})
	}

	modelsList = append(modelsList,
		&models.Tasks{},
		&models.Rewards{},
		&models.Tokens{},
		&models.TokenHistory{},
	)

	return testHandler.db.AutoMigrate(modelsList...)
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

func getTestDBConfig() *database.Config {
	getEnvOrDefault := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}

	return &database.Config{
		Host:     getEnvOrDefault("PGHOST", "localhost"),
		User:     getEnvOrDefault("PGUSER", "joytime"),
		Password: getEnvOrDefault("PGPASSWORD", "password"),
		Database: getEnvOrDefault("PGDATABASE", "joytime"),
		Port:     getEnvOrDefault("PGPORT", "5432"),
	}
}

func setupTestDB(t *testing.T) *models.Families {
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
	family := models.Families{
		Name: t.Name(),
		UID:  uniqueUID,
	}
	result := testHandler.db.Create(&family)
	if result.Error != nil {
		t.Fatalf("Failed to create family: %v", result.Error)
	}

	return &family
}

// setupServiceTestData creates test data and returns service context for testing
func setupServiceTestData(
	t *testing.T,
) (*models.Families, *models.Users, *models.Users, *domain.AuthContext) {
	// Create family
	family := &models.Families{
		Name: fmt.Sprintf("TestFamily_%s_%d", t.Name(), time.Now().UnixNano()),
	}

	// Service layer automatically generates UID
	err := testHandler.services.FamilyService.CreateFamily(context.Background(), family)
	require.NoError(t, err)

	// Create parent user
	parent := &models.Users{
		UserID:    fmt.Sprintf("parent_%s_%d", t.Name(), time.Now().UnixNano()),
		Name:      "Test Parent",
		Role:      "parent",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	err = testHandler.db.Create(parent).Error
	require.NoError(t, err)

	// Create child user
	child := &models.Users{
		UserID:    fmt.Sprintf("child_%s_%d", t.Name(), time.Now().UnixNano()),
		Name:      "Test Child",
		Role:      "child",
		FamilyUID: family.UID,
		Platform:  "telegram",
	}
	err = testHandler.db.Create(child).Error
	require.NoError(t, err)

	// Create tokens for child
	tokens := &models.Tokens{
		UserID: child.UserID,
		Tokens: 50,
	}
	err = testHandler.db.Create(tokens).Error
	require.NoError(t, err)

	// Create service context for parent (has most permissions)
	authCtx := &domain.AuthContext{
		UserID:    parent.UserID,
		UserRole:  domain.RoleParent,
		FamilyUID: family.UID,
	}

	return family, parent, child, authCtx
}

// SetAuthContext adds auth context to a request context
func SetAuthContext(ctx context.Context, authCtx *domain.AuthContext) context.Context {
	return context.WithValue(ctx, ContextKeyAuthContext, authCtx)
}

// TestGenerateUniqueFamilyUID tests that family UID generation works and is unique
func TestGenerateUniqueFamilyUID(t *testing.T) {
	setupTestDB(t)

	t.Run("Generate Family UID via Service", func(t *testing.T) {
		// Test UID generation through service layer
		family := &models.Families{Name: "Test Family"}
		err := testHandler.services.FamilyService.CreateFamily(context.Background(), family)
		assert.NoError(t, err)
		assert.NotEmpty(t, family.UID)
		assert.Equal(t, 6, len(family.UID), "UID should be 6 characters")

		// Check it only contains valid characters
		validChars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		for _, char := range family.UID {
			assert.Contains(t, validChars, string(char))
		}
	})

	t.Run("Generated UIDs are unique", func(t *testing.T) {
		family1 := &models.Families{Name: "Test Family 1"}
		err := testHandler.services.FamilyService.CreateFamily(context.Background(), family1)
		assert.NoError(t, err)

		family2 := &models.Families{Name: "Test Family 2"}
		err = testHandler.services.FamilyService.CreateFamily(context.Background(), family2)
		assert.NoError(t, err)

		assert.NotEqual(t, family1.UID, family2.UID, "Generated UIDs should be different")
	})

	t.Run("Family creation automatically generates UID", func(t *testing.T) {
		family := &models.Families{Name: "Auto UID Family"}
		// Don't set UID - service should generate it

		err := testHandler.services.FamilyService.CreateFamily(context.Background(), family)
		assert.NoError(t, err)
		assert.NotEmpty(t, family.UID, "Service should automatically generate UID")
		assert.Equal(t, 6, len(family.UID))

		// Verify it's actually in the database
		var dbFamily models.Families
		err = testHandler.db.Where("uid = ?", family.UID).First(&dbFamily).Error
		assert.NoError(t, err)
		assert.Equal(t, family.Name, dbFamily.Name)
	})
}

// TestNewHandlerNames tests that the renamed handlers (without WithBusiness suffix) work
func TestNewHandlerNames(t *testing.T) {
	setupTestDB(t)

	t.Run("Handler functions exist and are callable", func(t *testing.T) {
		// Test that all renamed handlers exist and can be referenced
		assert.NotNil(t, testHandler.handleFamilies)
		assert.NotNil(t, testHandler.handleFamily)
		assert.NotNil(t, testHandler.handleUsers)
		assert.NotNil(t, testHandler.handleUser)
		assert.NotNil(t, testHandler.handleTasks)
		assert.NotNil(t, testHandler.handleTasksByFamily)
		assert.NotNil(t, testHandler.handleRewards)
		assert.NotNil(t, testHandler.handleRewardsByFamily)
		assert.NotNil(t, testHandler.handleTokens)
		assert.NotNil(t, testHandler.handleUserTokens)
		assert.NotNil(t, testHandler.handleTokenHistory)
		assert.NotNil(t, testHandler.handleUserTokenHistory)
	})

	t.Run("Rewards require authentication", func(t *testing.T) {
		// Rewards are now implemented and require auth
		req := httptest.NewRequest("POST", "/api/v1/rewards", nil)
		w := httptest.NewRecorder()
		testHandler.handleRewards(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Tokens endpoint rejects non-POST methods", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
		w := httptest.NewRecorder()
		testHandler.handleTokens(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// TestServiceLayerIntegration tests that business logic and RBAC work correctly
func TestServiceLayerIntegration(t *testing.T) {
	setupTestDB(t)
	family, parent, child, authCtx := setupServiceTestData(t)

	t.Run("Family Creation with UID Generation", func(t *testing.T) {
		newFamily := &models.Families{
			Name: "Service Test Family",
		}

		// Test that service layer generates UID automatically
		err := testHandler.services.FamilyService.CreateFamily(context.Background(), newFamily)
		assert.NoError(t, err)
		assert.NotEmpty(t, newFamily.UID)
		assert.Equal(t, 6, len(newFamily.UID))
	})

	t.Run("RBAC Enforcement via Service Layer", func(t *testing.T) {
		// Create child service context
		childServiceCtx := &domain.AuthContext{
			UserID:    child.UserID,
			UserRole:  domain.RoleChild,
			FamilyUID: family.UID,
		}

		// Test that child cannot create tasks
		testTask := &models.Tasks{
			Entities: models.Entities{
				FamilyUID:   family.UID,
				Name:        "Child Task Attempt",
				Description: "This should fail",
				Tokens:      10,
			},
		}

		err := testHandler.services.TaskService.CreateTask(
			context.Background(),
			childServiceCtx,
			testTask,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Family Operations Work via Service Layer", func(t *testing.T) {
		// Test family operations work through service layer
		retrievedFamily, err := testHandler.services.FamilyService.GetFamily(
			context.Background(),
			authCtx,
			family.UID,
		)
		assert.NoError(t, err)
		assert.Equal(t, family.Name, retrievedFamily.Name)

		// Test family update
		updates := &domain.UpdateFamilyRequest{
			Name: "Updated Family via Service",
		}
		updatedFamily, err := testHandler.services.FamilyService.UpdateFamily(
			context.Background(),
			authCtx,
			family.UID,
			updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Family via Service", updatedFamily.Name)
	})

	t.Run("User Operations Work via Service Layer", func(t *testing.T) {
		// Test user operations work through service layer
		users, err := testHandler.services.UserService.GetFamilyUsers(
			context.Background(),
			authCtx,
			family.UID,
		)
		assert.NoError(t, err)
		assert.Len(t, users, 2) // parent and child

		// Test user update
		updates := &domain.UpdateUserRequest{
			Name: "Updated Parent via Service",
		}
		updatedUser, err := testHandler.services.UserService.UpdateUser(
			context.Background(),
			authCtx,
			parent.UserID,
			updates,
		)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Parent via Service", updatedUser.Name)
	})
}
