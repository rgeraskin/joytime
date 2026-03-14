package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func longString(n int) string {
	return string(make([]byte, n))
}

func TestValidateName(t *testing.T) {
	t.Run("required and empty", func(t *testing.T) {
		assert.ErrorIs(t, validateName("", true), ErrValidation)
	})
	t.Run("optional and empty", func(t *testing.T) {
		assert.NoError(t, validateName("", false))
	})
	t.Run("at max length", func(t *testing.T) {
		assert.NoError(t, validateName(longString(MaxNameLength), true))
	})
	t.Run("over max length", func(t *testing.T) {
		assert.ErrorIs(t, validateName(longString(MaxNameLength+1), false), ErrValidation)
	})
}

func TestValidateDescription(t *testing.T) {
	assert.NoError(t, validateDescription(longString(MaxDescriptionLength)))
	assert.ErrorIs(t, validateDescription(longString(MaxDescriptionLength+1)), ErrValidation)
}

func TestValidateTokensRequired(t *testing.T) {
	tests := []struct {
		name    string
		tokens  int
		wantErr bool
	}{
		{"zero", 0, true},
		{"negative", -1, true},
		{"valid", 10, false},
		{"at max", MaxTokens, false},
		{"over max", MaxTokens + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTokensRequired(tt.tokens)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrValidation)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTokensOptional(t *testing.T) {
	t.Run("nil is valid", func(t *testing.T) {
		assert.NoError(t, validateTokensOptional(nil))
	})
	t.Run("zero is valid", func(t *testing.T) {
		v := 0
		assert.NoError(t, validateTokensOptional(&v))
	})
	t.Run("negative is invalid", func(t *testing.T) {
		v := -1
		assert.ErrorIs(t, validateTokensOptional(&v), ErrValidation)
	})
	t.Run("over max is invalid", func(t *testing.T) {
		v := MaxTokens + 1
		assert.ErrorIs(t, validateTokensOptional(&v), ErrValidation)
	})
}

func TestValidateRole(t *testing.T) {
	assert.NoError(t, validateRole(""))
	assert.NoError(t, validateRole("parent"))
	assert.NoError(t, validateRole("child"))
	assert.ErrorIs(t, validateRole("admin"), ErrValidation)
}

func TestValidateStatus(t *testing.T) {
	assert.NoError(t, validateStatus(""))
	assert.NoError(t, validateStatus(TaskStatusNew))
	assert.NoError(t, validateStatus(TaskStatusCheck))
	assert.NoError(t, validateStatus(TaskStatusCompleted))
	assert.ErrorIs(t, validateStatus("invalid"), ErrValidation)
}

func TestValidateFamilyCreate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, ValidateFamilyCreate("My Family", "", ""))
	})
	t.Run("rejects uid", func(t *testing.T) {
		assert.ErrorIs(t, ValidateFamilyCreate("My Family", "uid123", ""), ErrValidation)
	})
	t.Run("rejects created_by", func(t *testing.T) {
		assert.ErrorIs(t, ValidateFamilyCreate("My Family", "", "user1"), ErrValidation)
	})
	t.Run("rejects empty name", func(t *testing.T) {
		assert.ErrorIs(t, ValidateFamilyCreate("", "", ""), ErrValidation)
	})
}

func TestValidateEntityCreate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, ValidateEntityCreate("Task", "Do stuff", 10))
	})
	t.Run("empty name", func(t *testing.T) {
		assert.ErrorIs(t, ValidateEntityCreate("", "desc", 10), ErrValidation)
	})
	t.Run("long description", func(t *testing.T) {
		assert.ErrorIs(t, ValidateEntityCreate("Task", longString(501), 10), ErrValidation)
	})
	t.Run("zero tokens", func(t *testing.T) {
		assert.ErrorIs(t, ValidateEntityCreate("Task", "", 0), ErrValidation)
	})
}

func TestValidateTokenTransaction(t *testing.T) {
	t.Run("valid add", func(t *testing.T) {
		assert.NoError(t, ValidateTokenTransaction(10, TokenTypeManualAdjustment, "bonus"))
	})
	t.Run("valid deduct", func(t *testing.T) {
		assert.NoError(t, ValidateTokenTransaction(-10, TokenTypeManualAdjustment, "penalty"))
	})
	t.Run("zero amount", func(t *testing.T) {
		assert.ErrorIs(t, ValidateTokenTransaction(0, TokenTypeManualAdjustment, ""), ErrValidation)
	})
	t.Run("over max", func(t *testing.T) {
		assert.ErrorIs(t, ValidateTokenTransaction(1001, TokenTypeManualAdjustment, ""), ErrValidation)
	})
	t.Run("under negative max", func(t *testing.T) {
		assert.ErrorIs(t, ValidateTokenTransaction(-1001, TokenTypeManualAdjustment, ""), ErrValidation)
	})
	t.Run("empty type", func(t *testing.T) {
		assert.ErrorIs(t, ValidateTokenTransaction(10, "", ""), ErrValidation)
	})
	t.Run("invalid type", func(t *testing.T) {
		assert.ErrorIs(t, ValidateTokenTransaction(10, "bogus", ""), ErrValidation)
	})
}

func TestUpdateRequestValidation(t *testing.T) {
	t.Run("family rejects empty name", func(t *testing.T) {
		r := &UpdateFamilyRequest{Name: ""}
		assert.ErrorIs(t, r.Validate(), ErrValidation)
	})
	t.Run("family accepts valid name", func(t *testing.T) {
		r := &UpdateFamilyRequest{Name: "New Name"}
		assert.NoError(t, r.Validate())
	})

	t.Run("user rejects bad role", func(t *testing.T) {
		r := &UpdateUserRequest{Role: "admin"}
		assert.ErrorIs(t, r.Validate(), ErrValidation)
	})
	t.Run("user accepts empty fields", func(t *testing.T) {
		r := &UpdateUserRequest{}
		assert.NoError(t, r.Validate())
	})

	t.Run("task rejects invalid status", func(t *testing.T) {
		r := &UpdateTaskRequest{Status: "invalid"}
		assert.ErrorIs(t, r.Validate(), ErrValidation)
	})
	t.Run("task rejects negative tokens", func(t *testing.T) {
		v := -1
		r := &UpdateTaskRequest{Tokens: &v}
		assert.ErrorIs(t, r.Validate(), ErrValidation)
	})
	t.Run("task accepts valid update", func(t *testing.T) {
		v := 50
		r := &UpdateTaskRequest{Name: "New", Tokens: &v, Status: TaskStatusCheck}
		assert.NoError(t, r.Validate())
	})

	t.Run("reward rejects tokens over max", func(t *testing.T) {
		v := 1001
		r := &UpdateRewardRequest{Tokens: &v}
		assert.ErrorIs(t, r.Validate(), ErrValidation)
	})
	t.Run("reward accepts valid update", func(t *testing.T) {
		r := &UpdateRewardRequest{Name: "Prize", Description: "Nice"}
		assert.NoError(t, r.Validate())
	})
}
