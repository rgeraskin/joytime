package domain

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

// UserService handles user-related business logic
type UserService struct {
	db     *gorm.DB
	logger *log.Logger
	auth   *CasbinAuthService
}

// NewUserService creates a new user service
func NewUserService(db *gorm.DB, logger *log.Logger, auth *CasbinAuthService) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
		auth:   auth,
	}
}

// GetUser retrieves a user with business rule enforcement
func (s *UserService) GetUser(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) (*models.Users, error) {
	// Get the target user first to determine family context for authorization
	var user models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}

	// Determine the correct action based on self vs other access
	action := "read"
	if authCtx.UserID != userID {
		action = "read_others"
	}

	// Use ABAC for authorization - handles role permissions and family access control
	if err := s.auth.RequirePermission(authCtx, "users", action, user.FamilyUID); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetFamilyUsers retrieves all users in a family
func (s *UserService) GetFamilyUsers(
	ctx context.Context,
	authCtx *AuthContext,
	familyUID string,
) ([]models.Users, error) {
	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "users", "read_others", familyUID); err != nil {
		return nil, err
	}

	var users []models.Users
	err := s.db.WithContext(ctx).Where("family_uid = ?", familyUID).Find(&users).Error
	return users, err
}

// UpdateUser updates user information with business rule enforcement
func (s *UserService) UpdateUser(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
	updates *UpdateUserRequest,
) (*models.Users, error) {
	// Get existing user first to check family and authorization
	var existingUser models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&existingUser).Error
	if err != nil {
		return nil, err
	}

	// ABAC: Check appropriate permission based on self vs other user update
	if authCtx.UserID == userID {
		// User updating themselves
		if err := s.auth.RequirePermission(authCtx, "users", "update", existingUser.FamilyUID); err != nil {
			return nil, err
		}
	} else {
		// User updating someone else - requires special permission (only parents have this)
		if err := s.auth.RequirePermission(authCtx, "users", "update_others", existingUser.FamilyUID); err != nil {
			return nil, err
		}
	}

	if err := updates.Validate(); err != nil {
		return nil, err
	}

	// Build selective update fields
	updateFields := make(UpdateFields)

	// Children can only update their name, parents can update name and role of others
	if authCtx.UserRole == RoleChild && authCtx.UserID == userID {
		// Children can only update their own name
		updateFields.AddStringIfNotEmpty("name", updates.Name)
	} else if authCtx.UserRole == RoleParent {
		updateFields.AddStringIfNotEmpty("name", updates.Name)
		if authCtx.UserID != userID {
			// Parents can change role of OTHER users (not themselves)
			updateFields.AddStringIfNotEmpty("role", updates.Role)
		}
	}

	// Apply updates only if there are fields to update
	if len(updateFields) > 0 {
		err = s.db.WithContext(ctx).
			Model(&existingUser).
			Select(updateFields.Keys()).
			Updates(updateFields.ToMap()).
			Error
		if err != nil {
			return nil, err
		}

		// Re-read to return current state
		if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&existingUser).Error; err != nil {
			return nil, err
		}
	}

	return &existingUser, nil
}

// DeleteUser deletes a user with business rule enforcement
func (s *UserService) DeleteUser(
	ctx context.Context,
	authCtx *AuthContext,
	userID string,
) error {
	// Get user to verify family and check business rules
	var user models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		return err
	}

	// Check permission and family access using Casbin
	if err := s.auth.RequirePermission(authCtx, "users", "delete", user.FamilyUID); err != nil {
		return err
	}

	// Business Rule: Parents cannot delete themselves
	if authCtx.UserID == userID {
		return ErrCannotDeleteSelf
	}

	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.Users{}).Error
}

// FindUser retrieves a user by ID without authorization checks.
// Used internally by the Telegram bot for user lookup during startup.
func (s *UserService) FindUser(ctx context.Context, userID string) (*models.Users, error) {
	var user models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	return &user, err
}

// CreateUser creates a new user without authorization checks.
// Used during Telegram registration when no auth context exists yet.
func (s *UserService) CreateUser(ctx context.Context, user *models.Users) error {
	return s.db.WithContext(ctx).Create(user).Error
}

// SetInputState updates the user's conversation input state.
// Used by the Telegram bot for multi-step conversation flows.
func (s *UserService) SetInputState(ctx context.Context, userID, state, inputContext string) error {
	return s.db.WithContext(ctx).
		Model(&models.Users{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"input_state":   state,
			"input_context": inputContext,
		}).Error
}

// UpdateFamilyUID updates a user's family UID.
// Used during Telegram registration when a user joins a family.
func (s *UserService) UpdateFamilyUID(ctx context.Context, userID, familyUID string) error {
	return s.db.WithContext(ctx).
		Model(&models.Users{}).
		Where("user_id = ?", userID).
		Update("family_uid", familyUID).Error
}

// FindFamilyUsersByRole retrieves users in a family filtered by role.
// Used internally for notifications (e.g., notifying parents when a child completes a task).
func (s *UserService) FindFamilyUsersByRole(ctx context.Context, familyUID, role string) ([]models.Users, error) {
	var users []models.Users
	err := s.db.WithContext(ctx).
		Where("family_uid = ? AND role = ?", familyUID, role).
		Find(&users).Error
	return users, err
}

// CreateAuthContext creates an authentication context for a user
func (s *UserService) CreateAuthContext(
	ctx context.Context,
	userID string,
) (*AuthContext, error) {
	var user models.Users
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &AuthContext{
		UserID:    user.UserID,
		UserRole:  UserRole(user.Role),
		FamilyUID: user.FamilyUID,
	}, nil
}
