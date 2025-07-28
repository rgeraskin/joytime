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

// setupIntegrationDB настраивает тестовую базу данных для интеграционных тестов
func setupIntegrationDB(t *testing.T) {
	level := log.InfoLevel
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           level,
	})

	var err error

	// Используем тестовую базу данных
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

	// Очищаем базу данных перед тестами
	db.Exec("DELETE FROM token_histories")
	db.Exec("DELETE FROM tokens")
	db.Exec("DELETE FROM tasks")
	db.Exec("DELETE FROM rewards")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM families")

	// Мигрируем схему
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

// TestFullAPIWorkflow тестирует полный рабочий процесс API
// Заменяет test_api.sh скрипт
func TestFullAPIWorkflow(t *testing.T) {
	setupIntegrationDB(t)

	t.Run("Complete API Workflow", func(t *testing.T) {
		// 1. Создание семьи
		t.Log("1. 📋 Создание семьи...")
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

		// 2. Создание родителя
		t.Log("2. 👤 Создание родителя...")
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

		// 3. Создание ребенка
		t.Log("3. 👶 Создание ребенка...")
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

		// 4. Проверка токенов ребенка (должны быть автоматически созданы)
		t.Log("4. ⚡ Проверка токенов ребенка...")
		req = httptest.NewRequest(http.MethodGet, "/tokens/user_child_456", nil)
		w = httptest.NewRecorder()
		handleUserTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tokens postgres.Tokens
		err = json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)
		assert.Equal(t, "user_child_456", tokens.UserID)
		assert.Equal(t, 0, tokens.Tokens) // Начальное значение

		// 5. Создание задания
		t.Log("5. 📝 Создание задания...")
		taskData := map[string]interface{}{
			"family_uid":  familyUID,
			"name":        "Убраться в комнате",
			"tokens":      10,
			"description": "Навести порядок и пропылесосить",
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
		assert.Equal(t, "Убраться в комнате", task.Name)
		assert.Equal(t, 10, task.Tokens)
		taskID := task.ID

		// 6. Создание награды
		t.Log("6. 🎁 Создание награды...")
		rewardData := map[string]interface{}{
			"family_uid":  familyUID,
			"name":        "Посмотреть мультики",
			"tokens":      5,
			"description": "15 минут мультиков",
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
		assert.Equal(t, "Посмотреть мультики", reward.Name)
		assert.Equal(t, 5, reward.Tokens)
		rewardID := reward.ID

		// 7. Получение всех заданий семьи
		t.Log("7. 📊 Получение всех заданий семьи...")
		req = httptest.NewRequest(http.MethodGet, "/tasks/"+familyUID, nil)
		w = httptest.NewRecorder()
		handleEntities(w, req, "tasks")

		assert.Equal(t, http.StatusOK, w.Code)
		var tasks []postgres.Tasks
		err = json.Unmarshal(w.Body.Bytes(), &tasks)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "Убраться в комнате", tasks[0].Name)

		// 8. Получение всех наград семьи
		t.Log("8. 🏆 Получение всех наград семьи...")
		req = httptest.NewRequest(http.MethodGet, "/rewards/"+familyUID, nil)
		w = httptest.NewRecorder()
		handleEntities(w, req, "rewards")

		assert.Equal(t, http.StatusOK, w.Code)
		var rewards []postgres.Rewards
		err = json.Unmarshal(w.Body.Bytes(), &rewards)
		require.NoError(t, err)
		assert.Len(t, rewards, 1)
		assert.Equal(t, "Посмотреть мультики", rewards[0].Name)

		// 9. Начисление токенов ребенку за выполнение задания (+10)
		t.Log("9. 💰 Начисление токенов ребенку за выполнение задания (+10)...")
		addTokensData := map[string]interface{}{
			"amount":      10,
			"type":        "task_completed",
			"description": "Выполнил задание: Убраться в комнате",
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

		// 10. Списание токенов за награду (-5)
		t.Log("10. 🎯 Списание токенов за награду (-5)...")
		subtractTokensData := map[string]interface{}{
			"amount":      -5,
			"type":        "reward_claimed",
			"description": "Получил награду: Посмотреть мультики",
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

		// 11. Получение истории токенов ребенка
		t.Log("11. 📜 Получение истории токенов ребенка...")
		req = httptest.NewRequest(http.MethodGet, "/token-history/user_child_456", nil)
		w = httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var history []postgres.TokenHistory
		err = json.Unmarshal(w.Body.Bytes(), &history)
		require.NoError(t, err)
		assert.Len(t, history, 2) // Два события: +10 и -5

		// Проверяем события в истории
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
		assert.True(t, foundTaskComplete, "История должна содержать выполнение задания")
		assert.True(t, foundRewardClaim, "История должна содержать получение награды")

		// 12. Получение истории токенов с пагинацией (limit=1)
		t.Log("12. 📈 Получение истории токенов с пагинацией (limit=1)...")
		req = httptest.NewRequest(http.MethodGet, "/token-history/user_child_456?limit=1&offset=0", nil)
		w = httptest.NewRecorder()
		handleUserTokenHistory(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var paginatedHistory []postgres.TokenHistory
		err = json.Unmarshal(w.Body.Bytes(), &paginatedHistory)
		require.NoError(t, err)
		assert.Len(t, paginatedHistory, 1)

		// 13. Список всех пользователей
		t.Log("13. 👥 Список всех пользователей...")
		req = httptest.NewRequest(http.MethodGet, "/users", nil)
		w = httptest.NewRecorder()
		handleUsers(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var allUsers []postgres.Users
		err = json.Unmarshal(w.Body.Bytes(), &allUsers)
		require.NoError(t, err)
		assert.Len(t, allUsers, 2) // Родитель и ребенок

		// 14. Список всех семей
		t.Log("14. 🏠 Список всех семей...")
		req = httptest.NewRequest(http.MethodGet, "/families", nil)
		w = httptest.NewRecorder()
		handleFamilies(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var allFamilies []postgres.Families
		err = json.Unmarshal(w.Body.Bytes(), &allFamilies)
		require.NoError(t, err)
		assert.Len(t, allFamilies, 1)
		assert.Equal(t, "Test Family", allFamilies[0].Name)

		// 15. Список всех токенов
		t.Log("15. 💎 Список всех токенов...")
		req = httptest.NewRequest(http.MethodGet, "/tokens", nil)
		w = httptest.NewRecorder()
		handleTokens(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var allTokens []postgres.Tokens
		err = json.Unmarshal(w.Body.Bytes(), &allTokens)
		require.NoError(t, err)
		assert.Len(t, allTokens, 1) // Только у ребенка
		assert.Equal(t, 5, allTokens[0].Tokens)

		// 16. Обновление пользователя (смена платформы)
		t.Log("16. 📝 Обновление пользователя (смена платформы)...")
		updateData := map[string]interface{}{
			"platform":     "web",
			"input_state":  "waiting_for_task_name",
			"user_id":      "user_child_456",
			"name":         "Test Child",
			"role":         "child",
			"family_uid":   familyUID,
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

		t.Log("✅ Все интеграционные тесты прошли успешно!")
		t.Logf("🔍 Результаты:")
		t.Logf("- Семья создана с UID: %s", familyUID)
		t.Logf("- Создано 2 пользователя (родитель и ребенок)")
		t.Logf("- Создано 1 задание и 1 награда")
		t.Logf("- Выполнены операции с токенами (начисление/списание)")
		t.Logf("- Создана история операций с токенами")
		t.Logf("- API поддерживает разные платформы (telegram, web)")
	})
}

// TestTokenOperations тестирует различные операции с токенами
func TestTokenOperations(t *testing.T) {
	setupIntegrationDB(t)

	// Создаем тестовые данные
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
			"description": "Бонус от родителей",
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
			"description": "Потратил на игрушку",
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
			"description": "Попытка потратить больше чем есть",
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