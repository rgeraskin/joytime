package handlers

const (
	// User roles
	RoleParent = "parent"
	RoleChild  = "child"

	// Platforms
	PlatformTelegram = "telegram"
	PlatformWeb      = "web"
	PlatformMobile   = "mobile"

	// Token operation types
	TokenTypeTaskCompleted    = "task_completed"
	TokenTypeRewardClaimed    = "reward_claimed"
	TokenTypeManualAdjustment = "manual_adjustment"

	// Task statuses
	TaskStatusNew       = "new"
	TaskStatusCheck     = "check"
	TaskStatusCompleted = "completed"

	// HTTP Content-Type
	ContentTypeJSON = "application/json"

	// Error messages
	ErrFamilyNotFound         = "family not found"
	ErrUserNotFound           = "user not found"
	ErrEntityNotFound         = "entity not found"
	ErrUserTokensNotFound     = "user tokens not found"
	ErrInvalidRole            = "invalid role: parent or child only"
	ErrInsufficientTokens     = "insufficient tokens"
	ErrMethodNotAllowed       = "method not allowed"
	ErrNotImplemented         = "not implemented"
	ErrMissingRequiredFields  = "missing required fields"
	ErrRestrictedFields       = "restricted fields"
	ErrInvalidURLPath         = "invalid URL path"
	ErrInvalidEntityEncoding  = "invalid entity name encoding"
	ErrInvalidJSONFormat      = "Invalid JSON format"
	ErrFamilyUIDRequired      = "Family UID is required"
	ErrUserIDRequired         = "User ID is required"
	ErrInvalidPlatform        = "Invalid platform"
	ErrInvalidTokenType       = "Invalid token type"
	ErrNameRequired           = "Name is required"
	ErrRoleRequired           = "Role is required"
	ErrUserIDRequiredField    = "UserID is required"
	ErrFamilyUIDRequiredField = "FamilyUID is required"
	ErrNameOrUIDRequired      = "Name or UID is required"
)