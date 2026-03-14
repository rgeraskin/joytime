package domain

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

// ToMap returns the underlying map
func (uf UpdateFields) ToMap() map[string]any {
	return map[string]any(uf)
}