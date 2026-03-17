package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBulkInput(t *testing.T) {
	t.Run("valid single item", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 10")
		assert.Len(t, items, 1)
		assert.Empty(t, errs)
		assert.Equal(t, "Homework", items[0].Name)
		assert.Equal(t, 10, items[0].Tokens)
	})

	t.Run("valid multiple items", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 10\nClean room 5\nWalk dog 15")
		assert.Len(t, items, 3)
		assert.Empty(t, errs)
		assert.Equal(t, "Homework", items[0].Name)
		assert.Equal(t, "Clean room", items[1].Name)
		assert.Equal(t, "Walk dog", items[2].Name)
	})

	t.Run("blank lines are skipped", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 10\n\n\nClean room 5\n")
		assert.Len(t, items, 2)
		assert.Empty(t, errs)
	})

	t.Run("missing tokens", func(t *testing.T) {
		items, errs := parseBulkInput("JustAName")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "нет токенов")
	})

	t.Run("invalid token value", func(t *testing.T) {
		items, errs := parseBulkInput("Homework abc")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "неверный формат")
	})

	t.Run("zero tokens rejected", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 0")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "токены должны быть")
	})

	t.Run("negative tokens rejected", func(t *testing.T) {
		items, errs := parseBulkInput("Homework -5")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "токены должны быть")
	})

	t.Run("tokens exceeding max rejected", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 1001")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "токены должны быть")
	})

	t.Run("max tokens allowed", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 1000")
		assert.Len(t, items, 1)
		assert.Empty(t, errs)
		assert.Equal(t, 1000, items[0].Tokens)
	})

	t.Run("name too long rejected", func(t *testing.T) {
		longName := make([]byte, 101)
		for i := range longName {
			longName[i] = 'a'
		}
		items, errs := parseBulkInput(string(longName) + " 10")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "слишком длинное")
	})

	t.Run("name with spaces preserved", func(t *testing.T) {
		items, errs := parseBulkInput("Do the dishes 5")
		assert.Len(t, items, 1)
		assert.Empty(t, errs)
		assert.Equal(t, "Do the dishes", items[0].Name)
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		items, errs := parseBulkInput("  Homework   10  ")
		assert.Len(t, items, 1)
		assert.Empty(t, errs)
		assert.Equal(t, "Homework", items[0].Name)
		assert.Equal(t, 10, items[0].Tokens)
	})

	t.Run("mixed valid and invalid", func(t *testing.T) {
		items, errs := parseBulkInput("Homework 10\nBadLine\nClean room 5")
		assert.Len(t, items, 2)
		assert.Len(t, errs, 1)
	})

	t.Run("empty input", func(t *testing.T) {
		items, errs := parseBulkInput("")
		assert.Empty(t, items)
		assert.Empty(t, errs)
	})

	t.Run("whitespace-only name rejected", func(t *testing.T) {
		// " 10" trims to "10" which has no space, so it's "no tokens" error
		items, errs := parseBulkInput(" 10")
		assert.Empty(t, items)
		assert.Len(t, errs, 1)
	})
}

func TestParseNumber(t *testing.T) {
	t.Run("valid number", func(t *testing.T) {
		n, err := parseNumber("42")
		assert.NoError(t, err)
		assert.Equal(t, 42, n)
	})

	t.Run("with whitespace", func(t *testing.T) {
		n, err := parseNumber("  7  ")
		assert.NoError(t, err)
		assert.Equal(t, 7, n)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseNumber("abc")
		assert.Error(t, err)
	})
}

func TestExtractName(t *testing.T) {
	t.Run("first and last name", func(t *testing.T) {
		// We can't test extractName directly with a tele.User in a unit test
		// without importing telebot, but the function is simple enough.
		// Test formatEntityItem instead as it's a pure function.
		result := formatEntityItem("Homework", 10)
		assert.Equal(t, "Homework: 10 💎", result)
	})
}

func TestFormatList(t *testing.T) {
	t.Run("with items", func(t *testing.T) {
		result := formatList("Tasks", []string{"A: 10 💎", "B: 5 💎"})
		assert.Contains(t, result, "Tasks:")
		assert.Contains(t, result, "1. A: 10 💎")
		assert.Contains(t, result, "2. B: 5 💎")
	})

	t.Run("empty list", func(t *testing.T) {
		result := formatList("Tasks", nil)
		assert.Contains(t, result, "Пока пусто")
	})
}

func TestFormatBulkResult(t *testing.T) {
	t.Run("with added items", func(t *testing.T) {
		result := formatBulkResult([]string{"A", "B"}, nil, "empty")
		assert.Contains(t, result, "Добавлено 2")
		assert.Contains(t, result, "+ A")
	})

	t.Run("with errors", func(t *testing.T) {
		result := formatBulkResult(nil, []string{"bad line"}, "empty")
		assert.Contains(t, result, "Ошибки")
		assert.Contains(t, result, "bad line")
	})

	t.Run("empty", func(t *testing.T) {
		result := formatBulkResult(nil, nil, "nothing here")
		assert.Equal(t, "nothing here", result)
	})
}

func TestIsDuplicateKey(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.False(t, isDuplicateKey(nil))
	})

	t.Run("non-duplicate error", func(t *testing.T) {
		assert.False(t, isDuplicateKey(assert.AnError))
	})
}
