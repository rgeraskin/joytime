package telegram

import (
	"errors"
	"fmt"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
)

// --- Penalties ---

func (b *Bot) showPenalties(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	items := make([]string, len(penalties))
	for i, p := range penalties {
		items[i] = formatEntityItem(p.Name, p.Tokens)
	}

	msg := formatList("⚠️ Штрафы", items)
	kb := inlineKeyboard(
		btnRow(btn("➕ Добавить", "penalty_add"), btn("📝 Списком", "penalty_add_bulk")),
		btnRow(btn("🗑 Удалить", "penalty_delete"), btn("✏️ Изменить", "penalty_edit")),
		btnRow(btn("⚡ Применить", "penalty_apply"), btn("⬅️ Назад", "back_parent")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onAddPenaltyPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddPenaltyName, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи название штрафа", backKeyboard("parent_penalties"))
}

func (b *Bot) onAddPenaltyName(c tele.Context, name string) error {
	if len(name) > maxEntityNameLength {
		return c.Send(fmt.Sprintf("❌ Название слишком длинное (максимум %d символов)", maxEntityNameLength))
	}
	if err := b.setState(c.Sender().ID, stateAddPenaltyTokens, name); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("💰 Сколько токенов снимать за этот штраф?", backKeyboard("parent_penalties"))
}

func (b *Bot) onAddPenaltyTokens(c tele.Context, text, penaltyName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalty := &models.Penalties{
		Entities: models.Entities{
			FamilyUID: auth.FamilyUID,
			Name:      penaltyName,
			Tokens:    tokens,
		},
	}
	if err := b.services.PenaltyService.CreatePenalty(bgCtx(), auth, penalty); err != nil {
		if isDuplicateKey(err) {
			return c.Send("❌ Штраф с таким именем уже существует")
		}
		return b.internalError(c, "Error creating penalty", err)
	}

	if err := c.Send("✅ Штраф добавлен!"); err != nil {
		return err
	}
	b.clearState(c.Sender().ID)

	b.notifyChildren(auth.FamilyUID, fmt.Sprintf("⚠️ Новый штраф: %s (%d 💎)", penaltyName, tokens))

	return b.showPenalties(c)
}

// --- Penalty bulk add ---

func (b *Bot) onAddPenaltyBulkPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddPenaltyBulk, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		"Введи штрафы списком, каждый на новой строке.\nПоследнее слово — количество токенов.\n\n<b>Пример:</b>\nНе убрал комнату 5\nГрубость 10",
		tele.ModeHTML,
		backKeyboard("parent_penalties"),
	)
}

func (b *Bot) onAddPenaltyBulk(c tele.Context, text string) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	items, errs := parseBulkInput(text)
	var added []string
	for _, item := range items {
		penalty := &models.Penalties{
			Entities: models.Entities{
				FamilyUID: auth.FamilyUID,
				Name:      item.Name,
				Tokens:    item.Tokens,
			},
		}
		if err := b.services.PenaltyService.CreatePenalty(bgCtx(), auth, penalty); err != nil {
			if isDuplicateKey(err) {
				errs = append(errs, fmt.Sprintf("'%s' — уже существует", item.Name))
			} else {
				errs = append(errs, fmt.Sprintf("'%s' — ошибка", item.Name))
			}
			continue
		}
		added = append(added, formatEntityItem(item.Name, item.Tokens))
	}

	if err := c.Send(formatBulkResult(added, errs, "Не найдено штрафов для добавления")); err != nil {
		return err
	}
	b.clearState(c.Sender().ID)

	if len(added) > 0 {
		b.notifyChildren(auth.FamilyUID, fmt.Sprintf("⚠️ Добавлено %d новых штрафов", len(added)))
	}

	return b.showPenalties(c)
}

// --- Penalty edit ---

func (b *Bot) onEditPenaltyPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	if len(penalties) == 0 {
		return c.Send("Нет штрафов", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_penalties"))))
	}

	items := make([]string, len(penalties))
	for i, p := range penalties {
		items[i] = formatEntityItem(p.Name, p.Tokens)
	}

	msg := formatList("Выбери штраф для изменения", items)
	grid := numberGrid(len(penalties), "pick_edit_penalty")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_penalties")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onEditPenaltyPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	if num < 1 || num > len(penalties) {
		return c.Send("Нет штрафа с таким номером")
	}

	penaltyName := penalties[num-1].Name
	if err := b.setState(c.Sender().ID, stateEditPenaltyTokens, penaltyName); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		fmt.Sprintf("⚠️ Штраф: %s\n\n💰 Сколько токенов снимать?", penaltyName),
		backKeyboard("parent_penalties"),
	)
}

func (b *Bot) onEditPenaltyTokens(c tele.Context, text, penaltyName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	updates := &domain.UpdatePenaltyRequest{Tokens: &tokens}
	if _, err := b.services.PenaltyService.UpdatePenalty(bgCtx(), auth, auth.FamilyUID, penaltyName, updates); err != nil {
		return b.internalError(c, "Error updating penalty", err)
	}

	if err := c.Send("✅ Штраф изменен!"); err != nil {
		return err
	}
	b.clearState(c.Sender().ID)
	return b.showPenalties(c)
}

// --- Penalty delete ---

func (b *Bot) onDeletePenaltyPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	if len(penalties) == 0 {
		return c.Send("Нет штрафов", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_penalties"))))
	}

	items := make([]string, len(penalties))
	for i, p := range penalties {
		items[i] = formatEntityItem(p.Name, p.Tokens)
	}

	msg := formatList("Выбери штраф для удаления", items)
	grid := numberGrid(len(penalties), "pick_del_penalty")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_penalties")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onDeletePenaltyPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	if num < 1 || num > len(penalties) {
		return c.Send("Нет штрафа с таким номером")
	}

	penaltyName := penalties[num-1].Name
	if err := b.services.PenaltyService.DeletePenalty(bgCtx(), auth, auth.FamilyUID, penaltyName); err != nil {
		return b.internalError(c, "Error deleting penalty", err)
	}

	if err := c.Send("✅ Штраф удален!"); err != nil {
		return err
	}
	return b.showPenalties(c)
}

// --- Penalty apply (pick penalty → pick child → apply) ---

func (b *Bot) onApplyPenaltyPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	if len(penalties) == 0 {
		return c.Send("Нет штрафов", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_penalties"))))
	}

	items := make([]string, len(penalties))
	for i, p := range penalties {
		items[i] = formatEntityItem(p.Name, p.Tokens)
	}

	msg := formatList("Выбери штраф для применения", items)
	grid := numberGrid(len(penalties), "pick_apply_penalty")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_penalties")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onApplyPenaltyPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	penalties, err := b.services.PenaltyService.GetPenaltiesForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting penalties", err)
	}

	if num < 1 || num > len(penalties) {
		return c.Send("Нет штрафа с таким номером")
	}

	penaltyName := penalties[num-1].Name

	// Get children to select who to penalize
	children, err := b.services.UserService.FindFamilyUsersByRole(
		bgCtx(),
		auth.FamilyUID,
		string(domain.RoleChild),
	)
	if err != nil {
		return b.internalError(c, "Error getting children", err)
	}

	if len(children) == 0 {
		return c.Send(
			"Нет детей в семье",
			inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_penalties"))),
		)
	}

	// If only one child, apply directly
	if len(children) == 1 {
		return b.applyPenalty(c, auth, penaltyName, children[0])
	}

	// Multiple children — store penalty name and show child picker
	if err := b.setState(c.Sender().ID, stateApplyPenaltyChild, penaltyName); err != nil {
		return b.internalError(c, "Error setting state", err)
	}

	items := make([]string, len(children))
	for i, ch := range children {
		items[i] = ch.Name
	}

	msg := formatList("Кому применить штраф", items)
	grid := numberGrid(len(children), "pick_penalty_child")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_penalties")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onApplyPenaltyChildPick(c tele.Context, num int) error {
	// Read penalty name from state (before clearState)
	user, err := b.findUser(c.Sender().ID)
	if err != nil || user == nil {
		return b.internalError(c, "Error finding user", err)
	}
	penaltyName := user.InputContext
	b.clearState(c.Sender().ID)

	if penaltyName == "" {
		return c.Send("Штраф не выбран. Попробуй /start")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	children, err := b.services.UserService.FindFamilyUsersByRole(
		bgCtx(),
		auth.FamilyUID,
		string(domain.RoleChild),
	)
	if err != nil {
		return b.internalError(c, "Error getting children", err)
	}

	if num < 1 || num > len(children) {
		return c.Send("Неверный номер")
	}

	return b.applyPenalty(c, auth, penaltyName, children[num-1])
}

func (b *Bot) applyPenalty(
	c tele.Context,
	auth *domain.AuthContext,
	penaltyName string,
	child models.Users,
) error {
	penalty, err := b.services.PenaltyService.ApplyPenalty(
		bgCtx(),
		auth,
		auth.FamilyUID,
		penaltyName,
		child.UserID,
	)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientTokens) {
			return c.Send(fmt.Sprintf("У %s недостаточно 💎 для штрафа", child.Name))
		}
		return b.internalError(c, "Error applying penalty", err)
	}

	if err := c.Send(fmt.Sprintf("✅ Штраф \"%s\" (%d 💎) применен к %s", penalty.Name, penalty.Tokens, child.Name)); err != nil {
		return err
	}

	// Notify child
	b.notifyChild(child.UserID,
		fmt.Sprintf("⚠️ Штраф: -%d 💎\n\nПричина: %s", penalty.Tokens, penalty.Name))

	return b.showParentMenu(c)
}
