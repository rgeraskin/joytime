package api

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/postgres"
	"gorm.io/gorm"
)

const (
	ADDRESS          = ":8080"
	FAMILYUIDCHARSET = "abcdefghjkmnpqrstuvwxyz23456789"
	FAMILYUIDLENGTH  = 6
)

var (
	db     *gorm.DB
	logger *log.Logger
)

// SetupAPI configures and returns the HTTP server for the API
func SetupAPI(database *gorm.DB, _logger *log.Logger) *http.Server {
	db = database
	logger = _logger
	mux := http.NewServeMux()

	logger.Debug("Setting up API")

	// Define routes
	mux.HandleFunc("/tasks", handleTasks)
	mux.HandleFunc("/tasks/", handleTask)
	mux.HandleFunc("/task/", handleSingleTask) // New route for individual task operations
	mux.HandleFunc("/rewards", handleRewards)
	mux.HandleFunc("/rewards/", handleReward)
	mux.HandleFunc("/reward/", handleSingleReward) // New route for individual reward operations
	mux.HandleFunc("/families", handleFamilies)
	mux.HandleFunc("/families/", handleFamily)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/", handleUser)
	mux.HandleFunc("/tokens", handleTokens)
	mux.HandleFunc("/tokens/", handleUserTokens)
	mux.HandleFunc("/token-history", handleTokenHistory)
	mux.HandleFunc("/token-history/", handleUserTokenHistory)

	return &http.Server{
		Addr:    ADDRESS,
		Handler: mux,
	}
}

func handleFamily(w http.ResponseWriter, r *http.Request) {
	UID := r.URL.Path[len("/families/"):]

	switch r.Method {
	case http.MethodGet:
		// Get single family
		logger.Debug("Getting family", "UID", UID)
		var family postgres.Families
		if err := db.Where("uid = ?", UID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(family); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPut:
		// decode family
		var family postgres.Families
		if err := json.NewDecoder(r.Body).Decode(&family); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check if name and uid are provided
		if family.Name == "" && family.UID == "" {
			http.Error(w, "Missing required fields: Name or UID", http.StatusBadRequest)
			return
		}

		// check if family exists
		var existingFamily postgres.Families
		if err := db.Where("uid = ?", UID).First(&existingFamily).Error; err != nil {
			http.Error(w, "Family not found", http.StatusNotFound)
			return
		}

		// Update family
		logger.Debug("Updating family", "UID", UID)
		if err := db.Where("uid = ?", UID).Updates(&family).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(family); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		// check if family exists
		var existingFamily postgres.Families
		if err := db.Where("uid = ?", UID).First(&existingFamily).Error; err != nil {
			http.Error(w, "Family not found", http.StatusNotFound)
			return
		}

		// Delete family
		logger.Debug("Deleting family", "UID", UID)
		if err := db.Where("uid = ?", UID).Delete(&postgres.Families{}).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFamilies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all families
		logger.Debug("Listing all families")
		var families []postgres.Families
		if err := db.Find(&families).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(families); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		// decode family
		var family postgres.Families
		if err := json.NewDecoder(r.Body).Decode(&family); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check restricted fields
		if family.UID != "" {
			http.Error(w, "Restricted fields: UID", http.StatusBadRequest)
			return
		}
		if family.CreatedByUserID != "" {
			http.Error(w, "Restricted fields: CreatedByUserID", http.StatusBadRequest)
			return
		}

		// Check required fields
		if family.Name == "" {
			http.Error(w, "Missing required fields: Name", http.StatusBadRequest)
			return
		}

		// Generate a unique family UID
		familyUID_byte := make([]byte, FAMILYUIDLENGTH)
		for i := range familyUID_byte {
			familyUID_byte[i] = FAMILYUIDCHARSET[rand.Intn(len(FAMILYUIDCHARSET))]
		}
		family.UID = string(familyUID_byte)

		// Create new family
		logger.Debug("Creating new family")
		if err := db.Create(&family).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(family); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all users
		logger.Debug("Listing all users")
		var users []postgres.Users
		if err := db.Find(&users).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		// decode user
		logger.Debug("Decoding user")
		var user postgres.Users
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check required fields
		if user.Name == "" {
			http.Error(w, "Missing required fields: Name", http.StatusBadRequest)
			return
		}
		if user.FamilyUID == "" {
			http.Error(w, "Missing required fields: FamilyUID", http.StatusBadRequest)
			return
		}
		if user.UserID == "" {
			http.Error(w, "Missing required fields: UserID", http.StatusBadRequest)
			return
		}
		if user.Role == "" {
			http.Error(w, "Missing required fields: Role", http.StatusBadRequest)
			return
		}

		// Set default platform if not provided
		if user.Platform == "" {
			user.Platform = "telegram"
		}

		// check user role
		if user.Role != "parent" && user.Role != "child" {
			http.Error(w, "Invalid role: parent or child only", http.StatusBadRequest)
			return
		}

				// check if family exists
		logger.Debug("Checking if family exists", "family_uid", user.FamilyUID)
		var family postgres.Families
		if err := db.Where("uid = ?", user.FamilyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// Create new user
		logger.Debug("Creating new user")
		if err := db.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create tokens for child
		if user.Role == "child" {
			tokens := postgres.Tokens{
				UserID: user.UserID,
				Tokens: 0,
			}
			if err := db.Create(&tokens).Error; err != nil {
				logger.Error("Failed to create tokens for child", "error", err)
				// Don't fail the user creation, just log the error
			}
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/users/"):]

	switch r.Method {
	case http.MethodGet:
		// Get single user
		logger.Debug("Getting user", "UserID", userID)
		var user postgres.Users
		if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPut:
		// Decode user
		logger.Debug("Updating user", "UserID", userID)
		var user postgres.Users
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check user role
		if user.Role != "" && (user.Role != "parent" && user.Role != "child") {
			http.Error(w, "Invalid role: parent or child only", http.StatusBadRequest)
			return
		}

		// check if family exists
		if user.FamilyUID != "" {
			logger.Debug("Checking if family exists", "family_uid", user.FamilyUID)
			var family postgres.Families
			if err := db.Where("uid = ?", user.FamilyUID).First(&family).Error; err != nil {
				http.Error(w, "Family not found", http.StatusBadRequest)
				return
			}
		}

		// check if user exists
		var existingUser postgres.Users
		if err := db.Where("user_id = ?", userID).First(&existingUser).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Update user
		if err := db.Where("user_id = ?", userID).Updates(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		// Check if user exists
		var existingUser postgres.Users
		err := db.Where("user_id = ?", userID).First(&existingUser).Error
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Delete user
		logger.Debug("Deleting user", "UserID", userID)
		if err := db.Where("user_id = ?", userID).Delete(&postgres.Users{}).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all tokens
		logger.Debug("Listing all tokens")
		var tokens []postgres.Tokens
		if err := db.Find(&tokens).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(tokens); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUserTokens(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/tokens/"):]

	switch r.Method {
	case http.MethodGet:
		// Get user tokens
		logger.Debug("Getting user tokens", "UserID", userID)
		var tokens postgres.Tokens
		if err := db.Where("user_id = ?", userID).First(&tokens).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "User tokens not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if err := json.NewEncoder(w).Encode(tokens); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPut:
		// Update user tokens
		logger.Debug("Updating user tokens", "UserID", userID)
		var tokensUpdate postgres.Tokens
		if err := json.NewDecoder(r.Body).Decode(&tokensUpdate); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check if user tokens exist
		var existingTokens postgres.Tokens
		if err := db.Where("user_id = ?", userID).First(&existingTokens).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "User tokens not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Update tokens
		if err := db.Where("user_id = ?", userID).Updates(&tokensUpdate).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get updated tokens
		if err := db.Where("user_id = ?", userID).First(&existingTokens).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(existingTokens); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		// Add tokens to user (для начисления/списания токенов)
		logger.Debug("Adding tokens to user", "UserID", userID)

		var request struct {
			Amount      int    `json:"amount"`      // Может быть отрицательным для списания
			Type        string `json:"type"`        // task_completed, reward_claimed, manual_adjustment
			Description string `json:"description"` // Описание операции
			TaskID      *uint  `json:"task_id,omitempty"`
			RewardID    *uint  `json:"reward_id,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Валидация
		if request.Amount == 0 {
			http.Error(w, "Amount cannot be zero", http.StatusBadRequest)
			return
		}
		if request.Type == "" {
			http.Error(w, "Type is required", http.StatusBadRequest)
			return
		}

		// Получаем текущие токены
		var tokens postgres.Tokens
		if err := db.Where("user_id = ?", userID).First(&tokens).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Создаем новую запись
				tokens = postgres.Tokens{
					UserID: userID,
					Tokens: 0,
				}
				if err := db.Create(&tokens).Error; err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Проверяем, хватает ли токенов для списания
		if request.Amount < 0 && tokens.Tokens < -request.Amount {
			http.Error(w, "Insufficient tokens", http.StatusBadRequest)
			return
		}

		// Обновляем токены
		tokens.Tokens += request.Amount
		if err := db.Save(&tokens).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Записываем в историю
		history := postgres.TokenHistory{
			UserID:      userID,
			Amount:      request.Amount,
			Type:        request.Type,
			Description: request.Description,
			TaskID:      request.TaskID,
			RewardID:    request.RewardID,
		}
		if err := db.Create(&history).Error; err != nil {
			logger.Error("Failed to save token history", "error", err)
			// Не останавливаем операцию из-за ошибки в истории
		}

		if err := json.NewEncoder(w).Encode(tokens); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		// Delete user tokens - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTokenHistory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all token history
		logger.Debug("Listing all token history")
		var history []postgres.TokenHistory
		if err := db.Find(&history).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(history); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		// Create token history - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	case http.MethodPut:
		// Update all token history - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	case http.MethodDelete:
		// Delete all token history - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUserTokenHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/token-history/"):]

	switch r.Method {
	case http.MethodGet:
		// Get user token history
		logger.Debug("Getting user token history", "UserID", userID)

		// Получаем параметры пагинации
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit := 50 // по умолчанию
		offset := 0

		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		if offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		var history []postgres.TokenHistory
		if err := db.Where("user_id = ?", userID).
			Order("created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&history).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(history); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		// Create user token history - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	case http.MethodPut:
		// Update user token history - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	case http.MethodDelete:
		// Delete user token history - not implemented
		http.Error(w, "Not implemented", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling tasks")
	handleEntities(w, r, "tasks")
}

func handleTask(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling task")
	handleEntity(w, r)
}

func handleSingleTask(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling single task")
	handleSingleEntity(w, r, "tasks")
}

func handleRewards(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling rewards")
	handleEntities(w, r, "rewards")
}

func handleReward(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling reward")
	handleEntity(w, r)
}

func handleSingleReward(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling single reward")
	handleSingleEntity(w, r, "rewards")
}

func handleEntities(w http.ResponseWriter, r *http.Request, entityType string) {
	logger := logger.With("entityType", entityType)

	switch r.Method {
	case http.MethodGet:
		// List all entities
		logger.Debug("Listing all entities")
		switch entityType {
		case "tasks":
			var entities []postgres.Tasks
			if err := db.Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(entities); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "rewards":
			var entities []postgres.Rewards
			if err := db.Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(entities); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	case http.MethodPost:
		// Create new entity
		logger.Debug("Creating new entity")
		switch entityType {
		case "tasks":
			var entity postgres.Tasks
			if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Validate required fields
			if entity.FamilyUID == "" {
				http.Error(w, "Missing required fields: FamilyUID", http.StatusBadRequest)
				return
			}
			if entity.Name == "" {
				http.Error(w, "Missing required fields: Name", http.StatusBadRequest)
				return
			}
			if entity.Tokens <= 0 {
				http.Error(w, "Missing required fields: Tokens (must be > 0)", http.StatusBadRequest)
				return
			}

			// Check if family exists
			var family postgres.Families
			if err := db.Where("uid = ?", entity.FamilyUID).First(&family).Error; err != nil {
				http.Error(w, "Family not found", http.StatusBadRequest)
				return
			}

			if err := db.Create(&entity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.Debug("Created task", "task", entity)
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(entity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "rewards":
			var entity postgres.Rewards
			if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Validate required fields
			if entity.FamilyUID == "" {
				http.Error(w, "Missing required fields: FamilyUID", http.StatusBadRequest)
				return
			}
			if entity.Name == "" {
				http.Error(w, "Missing required fields: Name", http.StatusBadRequest)
				return
			}
			if entity.Tokens <= 0 {
				http.Error(w, "Missing required fields: Tokens (must be > 0)", http.StatusBadRequest)
				return
			}

			// Check if family exists
			var family postgres.Families
			if err := db.Where("uid = ?", entity.FamilyUID).First(&family).Error; err != nil {
				http.Error(w, "Family not found", http.StatusBadRequest)
				return
			}

			if err := db.Create(&entity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.Debug("Created reward", "reward", entity)
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(entity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	case http.MethodPut:
		// Decode entity
		var entity postgres.Entities
		if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check if family uid is provided
		if entity.FamilyUID == "" {
			http.Error(w, "Missing required fields: FamilyUID", http.StatusBadRequest)
			return
		}
		logger = logger.With("family_uid", entity.FamilyUID)

		// check if name is provided
		if entity.Name == "" {
			http.Error(w, "Missing required fields: Name", http.StatusBadRequest)
			return
		}
		logger = logger.With("name", entity.Name)

		// check if tokens is provided
		if entity.Tokens == 0 {
			http.Error(w, "Missing required fields: Tokens", http.StatusBadRequest)
			return
		}
		logger = logger.With("tokens", entity.Tokens)

		// check if family exists
		logger.Debug("Checking if family exists")
		var family postgres.Families
		if err := db.Where("uid = ?", entity.FamilyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// check if entity exists
		logger.Debug("Checking if entity exists")
		where := db.Where("family_uid = ?", entity.FamilyUID).Where("name = ?", entity.Name)
		switch entityType {
		case "tasks":
			var existingEntity postgres.Tasks
			if err := where.First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// Update entity
			logger.Debug("Updating entity")
			existingEntity.Tokens = entity.Tokens
			existingEntity.Description = entity.Description
			if err := db.Save(&existingEntity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(existingEntity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case "rewards":
			var existingEntity postgres.Rewards
			if err := where.First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// Update entity
			logger.Debug("Updating entity")
			existingEntity.Tokens = entity.Tokens
			existingEntity.Description = entity.Description
			if err := db.Save(&existingEntity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(existingEntity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleEntity(w http.ResponseWriter, r *http.Request) {
	familyUID := r.URL.Path[len("/"+r.URL.Path[1:strings.Index(r.URL.Path[1:], "/")+1]+"/"):]

	switch r.Method {
	case http.MethodGet:
		// check if family exists
		var family postgres.Families
		if err := db.Where("uid = ?", familyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// List all entities in family
		logger.Debug("Listing all entities", "entityType", r.URL.Path, "familyUID", family.UID)
		if strings.Contains(r.URL.Path, "tasks") {
			var entities []postgres.Tasks
			if err := db.Where("family_uid = ?", family.UID).Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(entities); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if strings.Contains(r.URL.Path, "rewards") {
			var entities []postgres.Rewards
			if err := db.Where("family_uid = ?", family.UID).Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(entities); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	case http.MethodDelete:
		// check if family exists
		var family postgres.Families
		if err := db.Where("uid = ?", familyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// check if entity exists
		logger.Debug("Checking if entity exists for deletion")
		where := db.Where("family_uid = ?", family.UID)
		if strings.Contains(r.URL.Path, "tasks") {
			var existingEntity postgres.Tasks
			if err := where.Where("name = ?", r.URL.Path[strings.Index(r.URL.Path, "/")+1:]).First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if err := db.Where("family_uid = ? AND name = ?", family.UID, r.URL.Path[strings.Index(r.URL.Path, "/")+1:]).Delete(&postgres.Tasks{}).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		} else if strings.Contains(r.URL.Path, "rewards") {
			var existingEntity postgres.Rewards
			if err := where.Where("name = ?", r.URL.Path[strings.Index(r.URL.Path, "/")+1:]).First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if err := db.Where("family_uid = ? AND name = ?", family.UID, r.URL.Path[strings.Index(r.URL.Path, "/")+1:]).Delete(&postgres.Rewards{}).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSingleEntity(w http.ResponseWriter, r *http.Request, entityType string) {
	// Parse URL path: /task/{family_uid}/{entity_name} or /reward/{family_uid}/{entity_name}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	familyUID := pathParts[1]
	entityNameEncoded := pathParts[2]

	// Decode URL-encoded entity name
	entityName, err := url.QueryUnescape(entityNameEncoded)
	if err != nil {
		http.Error(w, "Invalid entity name encoding", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// check if family exists
		var family postgres.Families
		if err := db.Where("uid = ?", familyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// Get single entity
		logger.Debug("Getting single entity", "entityType", entityType, "familyUID", family.UID, "name", entityName)
		where := db.Where("family_uid = ?", family.UID).Where("name = ?", entityName)
		if entityType == "tasks" {
			var singleEntity postgres.Tasks
			if err := where.First(&singleEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if err := json.NewEncoder(w).Encode(singleEntity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if entityType == "rewards" {
			var singleEntity postgres.Rewards
			if err := where.First(&singleEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if err := json.NewEncoder(w).Encode(singleEntity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

	case http.MethodPut:
		// Decode entity
		var entity postgres.Entities
		if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check if family uid is provided
		if entity.FamilyUID == "" {
			http.Error(w, "Missing required fields: FamilyUID", http.StatusBadRequest)
			return
		}
		logger = logger.With("family_uid", entity.FamilyUID)

		// check if name is provided
		if entity.Name == "" {
			http.Error(w, "Missing required fields: Name", http.StatusBadRequest)
			return
		}
		logger = logger.With("name", entity.Name)

		// check if tokens is provided
		if entity.Tokens == 0 {
			http.Error(w, "Missing required fields: Tokens", http.StatusBadRequest)
			return
		}
		logger = logger.With("tokens", entity.Tokens)

		// check if family exists
		logger.Debug("Checking if family exists")
		var family postgres.Families
		if err := db.Where("uid = ?", entity.FamilyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// check if entity exists
		logger.Debug("Checking if entity exists")
		where := db.Where("family_uid = ?", entity.FamilyUID).Where("name = ?", entity.Name)
		if entityType == "tasks" {
			var existingEntity postgres.Tasks
			if err := where.First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// Update entity
			logger.Debug("Updating entity")
			existingEntity.Tokens = entity.Tokens
			existingEntity.Description = entity.Description
			if err := db.Save(&existingEntity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(existingEntity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if entityType == "rewards" {
			var existingEntity postgres.Rewards
			if err := where.First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// Update entity
			logger.Debug("Updating entity")
			existingEntity.Tokens = entity.Tokens
			existingEntity.Description = entity.Description
			if err := db.Save(&existingEntity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(existingEntity); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	case http.MethodDelete:
		// check if family exists
		var family postgres.Families
		if err := db.Where("uid = ?", familyUID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// check if entity exists
		logger.Debug("Checking if entity exists for deletion")
		where := db.Where("family_uid = ?", family.UID)
		if entityType == "tasks" {
			var existingEntity postgres.Tasks
			if err := where.Where("name = ?", entityName).First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if err := db.Where("family_uid = ? AND name = ?", family.UID, entityName).Delete(&postgres.Tasks{}).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		} else if entityType == "rewards" {
			var existingEntity postgres.Rewards
			if err := where.Where("name = ?", entityName).First(&existingEntity).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Entity not found", http.StatusNotFound)
					return
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if err := db.Where("family_uid = ? AND name = ?", family.UID, entityName).Delete(&postgres.Rewards{}).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
