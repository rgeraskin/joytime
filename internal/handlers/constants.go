package handlers

const (
	// HTTP Content-Type
	ContentTypeJSON = "application/json"

	// Error messages
	ErrEntityNotFound        = "entity not found"
	ErrInsufficientTokens    = "insufficient tokens"
	ErrMethodNotAllowed      = "method not allowed"
	ErrInvalidJSONFormat     = "invalid JSON format"
	ErrFamilyUIDRequired     = "family_uid is required"
	ErrUserIDRequired        = "user_id is required"
	ErrInvalidTokenType      = "invalid token type"
	ErrInvalidEntityEncoding = "invalid entity name encoding"
	ErrNameRequired          = "name is required"
)
