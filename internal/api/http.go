package api

import (
	"encoding/json"
	"math/rand"
	"net/http"

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
	mux.HandleFunc("/rewards", handleRewards)
	mux.HandleFunc("/rewards/", handleReward)
	mux.HandleFunc("/families", handleFamilies)
	mux.HandleFunc("/families/", handleFamily)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/", handleUser)

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
		json.NewEncoder(w).Encode(family)

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
		json.NewEncoder(w).Encode(family)

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
		json.NewEncoder(w).Encode(families)

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
		if family.CreatedByUserID != 0 {
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
		json.NewEncoder(w).Encode(family)

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
		json.NewEncoder(w).Encode(users)

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
		if user.FamilyID == 0 {
			http.Error(w, "Missing required fields: FamilyID", http.StatusBadRequest)
			return
		}
		if user.UID == "" {
			http.Error(w, "Missing required fields: UID", http.StatusBadRequest)
			return
		}
		if user.Role == "" {
			http.Error(w, "Missing required fields: Role", http.StatusBadRequest)
			return
		}

		// check user role
		if user.Role != "parent" && user.Role != "child" {
			http.Error(w, "Invalid role: parent or child only", http.StatusBadRequest)
			return
		}

		// check if family exists
		logger.Debug("Checking if family exists", "family_id", user.FamilyID)
		var family postgres.Families
		if err := db.Where("id = ?", user.FamilyID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// Create new user
		logger.Debug("Creating new user")
		if err := db.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	UID := r.URL.Path[len("/users/"):]

	switch r.Method {
	case http.MethodGet:
		// Get single user
		logger.Debug("Getting user", "UID", UID)
		var user postgres.Users
		if err := db.Where("uid = ?", UID).First(&user).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)

	case http.MethodPut:
		// Decode user
		logger.Debug("Updating user", "UID", UID)
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
		var family postgres.Families
		if user.FamilyID != 0 {
			logger.Debug("Checking if family exists", "family_id", user.FamilyID)
			if err := db.Where("id = ?", user.FamilyID).First(&family).Error; err != nil {
				http.Error(w, "Family not found", http.StatusBadRequest)
				return
			}
		}

		// check if user exists
		var existingUser postgres.Users
		if err := db.Where("uid = ?", UID).First(&existingUser).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Update user
		if err := db.Where("uid = ?", UID).Updates(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(user)

	case http.MethodDelete:
		// Check if user exists
		var existingUser postgres.Users
		err := db.Where("uid = ?", UID).First(&existingUser).Error
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Delete user
		logger.Debug("Deleting user", "UID", UID)
		if err := db.Where("uid = ?", UID).Delete(&postgres.Users{}).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

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
	handleEntity(w, r, "tasks")
}

func handleRewards(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling rewards")
	handleEntities(w, r, "rewards")
}

func handleReward(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling reward")
	handleEntity(w, r, "rewards")
}

func handleEntities(w http.ResponseWriter, r *http.Request, entityType string) {
	logger := logger.With("entityType", entityType)

	switch r.Method {
	case http.MethodGet:
		// List all entities
		logger.Debug("Listing all entities")
		if entityType == "tasks" {
			var entities []postgres.Tasks
			if err := db.Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(entities)
		} else if entityType == "rewards" {
			var entities []postgres.Rewards
			if err := db.Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(entities)
		}

	case http.MethodPost:
		// Create new entity
		logger.Debug("Creating new entity")
		if entityType == "tasks" {
			var entity postgres.Tasks
			if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := db.Create(&entity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.Debug("Created task", "task", entity)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(entity)
		} else if entityType == "rewards" {
			var entity postgres.Rewards
			if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := db.Create(&entity).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.Debug("Created reward", "reward", entity)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(entity)
		}

	case http.MethodPut:
		// Decode entity
		var entity postgres.Entities
		if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check if family id is provided
		if entity.FamilyID == 0 {
			http.Error(w, "Missing required fields: FamilyID", http.StatusBadRequest)
			return
		}
		logger = logger.With("family_id", entity.FamilyID)

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
		if err := db.Where("id = ?", entity.FamilyID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// check if entity exists
		logger.Debug("Checking if entity exists")
		where := db.Where("family_id = ?", entity.FamilyID).Where("name = ?", entity.Name)
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
			json.NewEncoder(w).Encode(existingEntity)

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
			json.NewEncoder(w).Encode(existingEntity)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleEntity(w http.ResponseWriter, r *http.Request, entityType string) {
	familyID := r.URL.Path[len("/"+entityType+"/"):]

	switch r.Method {
	case http.MethodGet:
		// check if family exists
		var family postgres.Families
		if err := db.Where("id = ?", familyID).First(&family).Error; err != nil {
			http.Error(w, "Family not found", http.StatusBadRequest)
			return
		}

		// List all entities in family
		logger.Debug("Listing all entities", "entityType", entityType, "familyID", family.ID)
		if entityType == "tasks" {
			var entities []postgres.Tasks
			if err := db.Where("family_id = ?", family.ID).Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(entities)
		} else if entityType == "rewards" {
			var entities []postgres.Rewards
			if err := db.Where("family_id = ?", family.ID).Find(&entities).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(entities)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
