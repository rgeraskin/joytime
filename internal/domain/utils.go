package domain

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// codeCharset excludes visually confusing characters: 0/O and 1/I
const codeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generateRandomCode generates a random alphanumeric code of the given length.
func generateRandomCode(length int) (string, error) {
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeCharset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random code: %w", err)
		}
		code[i] = codeCharset[n.Int64()]
	}
	return string(code), nil
}

// UpdateFields helps build selective update maps for GORM
type UpdateFields map[string]any

// AddStringIfNotEmpty adds a string field if non-empty
func (uf UpdateFields) AddStringIfNotEmpty(field, value string) UpdateFields {
	if value != "" {
		uf[field] = value
	}
	return uf
}

// AddIntIfSet adds an int field if the pointer is non-nil (allows setting to 0)
func (uf UpdateFields) AddIntIfSet(field string, value *int) UpdateFields {
	if value != nil {
		uf[field] = *value
	}
	return uf
}

// Keys returns the field names that were actually set
func (uf UpdateFields) Keys() []string {
	keys := make([]string, 0, len(uf))
	for k := range uf {
		keys = append(keys, k)
	}
	return keys
}

// ToMap returns the underlying map
func (uf UpdateFields) ToMap() map[string]any {
	return map[string]any(uf)
}
