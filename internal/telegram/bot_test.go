package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/rgeraskin/joytime/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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

func TestFormatHistory(t *testing.T) {
	now := time.Date(2026, 3, 17, 14, 30, 0, 0, time.UTC)

	t.Run("empty history", func(t *testing.T) {
		result := formatHistory("", nil, 20)
		assert.Contains(t, result, "Пока пусто")
	})

	t.Run("with prefix", func(t *testing.T) {
		result := formatHistory("Child Name", nil, 20)
		assert.Contains(t, result, "Child Name")
	})

	t.Run("date and reversed order", func(t *testing.T) {
		history := []models.TokenHistory{
			{Amount: 10, Description: "Second", Model: gorm.Model{CreatedAt: now}},
			{Amount: -5, Description: "First", Model: gorm.Model{CreatedAt: now.Add(-time.Hour)}},
		}
		result := formatHistory("", history, 20)
		// Recent at bottom: "First" line should come before "Second" line
		firstIdx := strings.Index(result, "First")
		secondIdx := strings.Index(result, "Second")
		assert.Greater(t, secondIdx, firstIdx, "recent entries should be at the bottom")
		// Date format present
		assert.Contains(t, result, "17.03 14:30")
	})

	t.Run("strips description prefixes", func(t *testing.T) {
		history := []models.TokenHistory{
			{Amount: 10, Description: "Задание: Homework", Model: gorm.Model{CreatedAt: now}},
			{Amount: -5, Description: "Награда: Ice Cream", Model: gorm.Model{CreatedAt: now}},
			{Amount: -3, Description: "Штраф: Bad Behavior", Model: gorm.Model{CreatedAt: now}},
		}
		result := formatHistory("", history, 20)
		assert.NotContains(t, result, "Задание:")
		assert.NotContains(t, result, "Награда:")
		assert.NotContains(t, result, "Штраф:")
		assert.Contains(t, result, "Homework")
		assert.Contains(t, result, "Ice Cream")
		assert.Contains(t, result, "Bad Behavior")
	})

	t.Run("manual adjustment description preserved", func(t *testing.T) {
		history := []models.TokenHistory{
			{Amount: 5, Description: "Bonus for helping", Model: gorm.Model{CreatedAt: now}},
		}
		result := formatHistory("", history, 20)
		assert.Contains(t, result, "Bonus for helping")
	})

	t.Run("sign formatting", func(t *testing.T) {
		history := []models.TokenHistory{
			{Amount: 10, Description: "gain", Model: gorm.Model{CreatedAt: now}},
			{Amount: -5, Description: "loss", Model: gorm.Model{CreatedAt: now}},
		}
		result := formatHistory("", history, 20)
		assert.Contains(t, result, "+10")
		assert.Contains(t, result, "-5")
	})

	t.Run("limit applied", func(t *testing.T) {
		history := []models.TokenHistory{
			{Amount: 1, Description: "a", Model: gorm.Model{CreatedAt: now}},
			{Amount: 2, Description: "b", Model: gorm.Model{CreatedAt: now}},
			{Amount: 3, Description: "c", Model: gorm.Model{CreatedAt: now}},
		}
		result := formatHistory("", history, 2)
		// Only first 2 entries (reversed), "c" should not appear
		assert.NotContains(t, result, " c\n")
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
