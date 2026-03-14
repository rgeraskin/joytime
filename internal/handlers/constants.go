package handlers

const (
	// HTTP Content-Type
	ContentTypeJSON = "application/json"

	// Error messages
	ErrEntityNotFound         = "entity not found"
	ErrInvalidRole            = "invalid role: parent or child only"
	ErrInsufficientTokens     = "insufficient tokens"
	ErrMethodNotAllowed       = "method not allowed"
	ErrInvalidJSONFormat      = "invalid JSON format"
	ErrFamilyUIDRequired      = "family UID is required"
	ErrUserIDRequired         = "user ID is required"
	ErrInvalidPlatform        = "invalid platform"
	ErrInvalidTokenType       = "invalid token type"
	ErrInvalidEntityEncoding  = "invalid entity name encoding"
	ErrNameRequired           = "name is required"
	ErrUserIDRequiredField    = "user_id is required"
	ErrFamilyUIDRequiredField = "family_uid is required"
)
