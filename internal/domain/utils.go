package domain

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"

	"gorm.io/gorm"
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

// Keys returns the field names that were actually set, sorted for deterministic ordering.
func (uf UpdateFields) Keys() []string {
	keys := make([]string, 0, len(uf))
	for k := range uf {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ToMap returns the underlying map
func (uf UpdateFields) ToMap() map[string]any {
	return map[string]any(uf)
}

// updateAndReload applies selective updates to an entity and re-reads it in one transaction.
// The entity must have an ID field (via gorm.Model embedding) for the reload.
func updateAndReload[T any](db *gorm.DB, ctx context.Context, entity *T, id uint, fields UpdateFields) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(entity).
			Select(fields.Keys()).
			Updates(fields.ToMap()).
			Error; err != nil {
			return err
		}
		return tx.First(entity, id).Error
	})
}

// findByFamilyAndName looks up an entity scoped to a family by its unique name.
func findByFamilyAndName[T any](db *gorm.DB, ctx context.Context, familyUID, name string) (*T, error) {
	var entity T
	err := db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, name).
		First(&entity).
		Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// deleteByFamilyAndName deletes an entity scoped to a family by name.
// Returns gorm.ErrRecordNotFound if no rows were affected.
func deleteByFamilyAndName[T any](db *gorm.DB, ctx context.Context, familyUID, name string) error {
	result := db.WithContext(ctx).
		Where("family_uid = ? AND name = ?", familyUID, name).
		Delete(new(T))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
