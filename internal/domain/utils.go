package domain

// UpdateFields helps build selective update maps for GORM
type UpdateFields map[string]any

// AddFieldIfNotEmpty adds a field to the update map if it's not empty/zero value
func (uf UpdateFields) AddFieldIfNotEmpty(field string, value any) UpdateFields {
	switch v := value.(type) {
	case string:
		if v != "" {
			uf[field] = v
		}
	case int:
		if v != 0 {
			uf[field] = v
		}
	case bool:
		uf[field] = v // Always include booleans
	}
	return uf
}

// ToMap returns the underlying map
func (uf UpdateFields) ToMap() map[string]any {
	return map[string]any(uf)
}