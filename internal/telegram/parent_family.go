package telegram

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
)

// --- Token History (parent) ---

func (b *Bot) showParentHistoryPrompt(c tele.Context) error {
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

	if len(children) == 0 {
		return c.Send("Нет детей в семье", inlineKeyboard(btnRow(btn("⬅️ Назад", "back_parent"))))
	}

	// Single child — go directly
	if len(children) == 1 {
		return b.showHistoryForChild(c, auth, children[0])
	}

	// Multiple children — show picker
	items := make([]string, len(children))
	for i, ch := range children {
		items[i] = ch.Name
	}

	msg := formatList("Чью историю показать", items)
	grid := numberGrid(len(children), "pick_history_child")
	grid = append(grid, btnRow(btn("⬅️ Назад", "back_parent")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onHistoryChildPick(c tele.Context, num int) error {
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
		return c.Send("❌ Неверный номер")
	}

	return b.showHistoryForChild(c, auth, children[num-1])
}

func (b *Bot) showHistoryForChild(
	c tele.Context,
	auth *domain.AuthContext,
	child models.Users,
) error {
	history, err := b.services.TokenService.GetTokenHistory(bgCtx(), auth, child.UserID)
	if err != nil {
		return b.internalError(c, "Error getting history", err)
	}

	msg := formatHistory(child.Name, history, 20)
	return c.Send(msg, inlineKeyboard(btnRow(btn("⬅️ Назад", "back_parent"))))
}

// --- Family management ---

func (b *Bot) showFamilyMembers(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	users, err := b.services.UserService.GetFamilyUsers(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting family users", err)
	}

	items := make([]string, len(users))
	for i, u := range users {
		roleName := "родитель"
		if u.Role == string(domain.RoleChild) {
			roleName = "ребёнок"
		}
		items[i] = fmt.Sprintf("%s (%s)", u.Name, roleName)
	}

	msg := formatList("👨‍👩‍👧‍👦 Семья", items)
	kb := inlineKeyboard(
		btnRow(btn("➕ Пригласить", "family_invite")),
		btnRow(btn("✏️ Переименовать", "family_rename"), btn("🗑 Удалить", "family_delete")),
		btnRow(btn("⬅️ Назад", "back_parent")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onFamilyInviteRolePrompt(c tele.Context) error {
	kb := inlineKeyboard(
		btnRow(
			btn("👨‍👩‍👧 Родитель", "invite_role_parent"),
			btn("👶 Ребёнок", "invite_role_child"),
		),
		btnRow(btn("⬅️ Назад", "parent_family")),
	)
	return c.Send("Кого приглашаем?", kb)
}

func (b *Bot) onFamilyInviteCreate(c tele.Context, role string) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	invite, err := b.services.InviteService.CreateInvite(bgCtx(), auth, auth.FamilyUID, role)
	if err != nil {
		return b.internalError(c, "Error creating invite", err)
	}

	roleName := "родителя"
	if role == string(domain.RoleChild) {
		roleName = "ребёнка"
	}

	link := b.inviteLink(invite.Code)
	msg := fmt.Sprintf(
		"🔑 Приглашение для %s:\n\n%s\n\nСсылка одноразовая",
		roleName,
		link,
	)
	if err := c.Send(msg); err != nil {
		return err
	}
	return b.showFamilyMembers(c)
}

func (b *Bot) onRenameMemberPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	users, err := b.services.UserService.GetFamilyUsers(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting family users", err)
	}

	if len(users) == 0 {
		return c.Send("Нет участников", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_family"))))
	}

	items := make([]string, len(users))
	for i, u := range users {
		items[i] = u.Name
	}

	msg := formatList("Кого переименовать", items)
	grid := numberGrid(len(users), "pick_rename_member")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_family")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onRenameMemberPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	users, err := b.services.UserService.GetFamilyUsers(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting family users", err)
	}

	if num < 1 || num > len(users) {
		return c.Send("❌ Неверный номер")
	}

	targetUserID := users[num-1].UserID
	if err := b.setState(c.Sender().ID, stateRenameMemberName, targetUserID); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		fmt.Sprintf("✏️ Текущее имя: %s\nВведи новое имя:", users[num-1].Name),
		backKeyboard("parent_family"),
	)
}

func (b *Bot) onRenameMemberName(c tele.Context, newName, targetUserID string) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	updates := &domain.UpdateUserRequest{Name: newName}
	if _, err := b.services.UserService.UpdateUser(bgCtx(), auth, targetUserID, updates); err != nil {
		if errors.Is(err, domain.ErrValidation) {
			return c.Send("❌ " + err.Error())
		}
		return b.internalError(c, "Error updating user", err)
	}

	if err := c.Send("✅ Имя изменено!"); err != nil {
		return err
	}
	b.clearState(c.Sender().ID)
	return b.showFamilyMembers(c)
}

func (b *Bot) onDeleteMemberPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	users, err := b.services.UserService.GetFamilyUsers(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting family users", err)
	}

	// Exclude self from deletion list
	var others []models.Users
	for _, u := range users {
		if u.UserID != auth.UserID {
			others = append(others, u)
		}
	}

	if len(others) == 0 {
		return c.Send(
			"Нет участников для удаления",
			inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_family"))),
		)
	}

	items := make([]string, len(others))
	for i, u := range others {
		items[i] = u.Name
	}

	msg := formatList("Кого удалить", items)
	grid := numberGrid(len(others), "pick_delete_member")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_family")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onDeleteMemberPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	users, err := b.services.UserService.GetFamilyUsers(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting family users", err)
	}

	var others []models.Users
	for _, u := range users {
		if u.UserID != auth.UserID {
			others = append(others, u)
		}
	}

	if num < 1 || num > len(others) {
		return c.Send("❌ Неверный номер")
	}

	targetUser := others[num-1]
	if err := b.services.UserService.DeleteUser(bgCtx(), auth, targetUser.UserID); err != nil {
		if errors.Is(err, domain.ErrCannotDeleteSelf) {
			return c.Send("❌ Нельзя удалить себя")
		}
		return b.internalError(c, "Error deleting user", err)
	}

	if err := c.Send(fmt.Sprintf("✅ %s удалён", targetUser.Name)); err != nil {
		return err
	}
	return b.showFamilyMembers(c)
}

// --- Manual token adjustment ---

func (b *Bot) onManualAdjustPrompt(c tele.Context) error {
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

	if len(children) == 0 {
		return c.Send("Нет детей в семье", inlineKeyboard(btnRow(btn("⬅️ Назад", "back_parent"))))
	}

	// Single child — go straight to reason prompt
	if len(children) == 1 {
		return b.startManualAdjust(c, children[0].UserID)
	}

	// Multiple children — show picker
	items := make([]string, len(children))
	for i, ch := range children {
		items[i] = ch.Name
	}

	msg := formatList("Кому скорректировать токены", items)
	grid := numberGrid(len(children), "pick_manual_child")
	grid = append(grid, btnRow(btn("⬅️ Назад", "back_parent")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onManualAdjustChildPick(c tele.Context, num int) error {
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

	return b.startManualAdjust(c, children[num-1].UserID)
}

func (b *Bot) startManualAdjust(c tele.Context, childUserID string) error {
	if err := b.setState(c.Sender().ID, stateManualAdjustReason, childUserID); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи причину коррекции", backKeyboard("back_parent"))
}

func (b *Bot) onManualAdjustReason(c tele.Context, reason, childUserID string) error {
	// Store childUserID|reason for the next step
	ctx := childUserID + "|" + reason
	if err := b.setState(c.Sender().ID, stateManualAdjustTokens, ctx); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		"💰 Сколько токенов? (положительное — добавить, отрицательное — снять)",
		backKeyboard("back_parent"),
	)
}

func (b *Bot) onManualAdjustTokens(c tele.Context, text, inputCtx string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}
	if tokens == 0 {
		return c.Send("Количество токенов не может быть 0")
	}

	// Parse childUserID|reason from context
	parts := strings.SplitN(inputCtx, "|", 2)
	if len(parts) != 2 {
		return c.Send("Ошибка. Попробуй /start")
	}
	childUserID := parts[0]
	reason := parts[1]

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	_, err = b.services.TokenService.AddTokensToUser(
		bgCtx(), auth, childUserID, tokens,
		domain.TokenTypeManualAdjustment, "Коррекция: "+reason, nil, nil, nil,
	)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientTokens) {
			return c.Send("Недостаточно токенов для снятия")
		}
		return b.internalError(c, "Error adjusting tokens", err)
	}

	childName := childUserID
	if child, _ := b.services.UserService.FindUser(bgCtx(), childUserID); child != nil {
		childName = child.Name
	}

	sign := "+"
	if tokens < 0 {
		sign = ""
	}
	if err := c.Send(fmt.Sprintf("✅ Коррекция: %s%d 💎 для %s\n\nПричина: %s", sign, tokens, childName, reason)); err != nil {
		return err
	}
	b.clearState(c.Sender().ID)

	// Notify child
	b.notifyChild(childUserID,
		fmt.Sprintf("🔧 Коррекция: %s%d 💎\n\nПричина: %s", sign, tokens, reason))

	return b.showParentMenu(c)
}

// --- Task Review (number grid → approve/reject buttons) ---

func (b *Bot) showPendingReview(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	var pending []models.Tasks
	for _, t := range tasks {
		if t.Status == domain.TaskStatusCheck {
			pending = append(pending, t)
		}
	}

	if len(pending) == 0 {
		return c.Send(
			"Нет заданий для проверки",
			inlineKeyboard(btnRow(btn("⬅️ Назад", "back_parent"))),
		)
	}

	items := make([]string, len(pending))
	for i, t := range pending {
		child, _ := b.services.UserService.FindUser(bgCtx(), t.AssignedToUserID)
		childName := "?"
		if child != nil {
			childName = child.Name
		}
		items[i] = fmt.Sprintf("%s: %d 💎 (%s)", t.Name, t.Tokens, childName)
	}

	msg := formatList("Задания на проверку", items)
	grid := numberGrid(len(pending), "pick_review")
	grid = append(grid, btnRow(btn("⬅️ Назад", "back_parent")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onReviewTaskPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	var pending []models.Tasks
	for _, t := range tasks {
		if t.Status == domain.TaskStatusCheck {
			pending = append(pending, t)
		}
	}

	if num < 1 || num > len(pending) {
		return c.Send("Нет задания с таким номером")
	}

	task := pending[num-1]
	child, _ := b.services.UserService.FindUser(bgCtx(), task.AssignedToUserID)
	childName := "?"
	if child != nil {
		childName = child.Name
	}

	// Store task name in state for the approve/reject callback
	if err := b.setState(c.Sender().ID, stateReviewConfirm, task.Name); err != nil {
		return b.internalError(c, "Error setting state", err)
	}

	msg := fmt.Sprintf("Задание: %s\nТокены: %d 💎\nВыполнил: %s", task.Name, task.Tokens, childName)
	kb := inlineKeyboard(
		btnRow(
			btn("✅ Подтвердить", "review_approve"),
			btn("❌ Отклонить", "review_reject"),
		),
		btnRow(btn("⬅️ Назад", "parent_review")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onReviewApprove(c tele.Context) error {
	user, err := b.findUser(c.Sender().ID)
	if err != nil || user == nil {
		return b.internalError(c, "Error finding user", err)
	}
	taskName := user.InputContext
	b.clearState(c.Sender().ID)

	if taskName == "" {
		return c.Send("Задание не выбрано. Попробуй /start")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	task, err := b.services.TaskService.GetTask(bgCtx(), auth, auth.FamilyUID, taskName)
	if err != nil {
		return b.internalError(c, "Error getting task", err)
	}

	// Remember assigned child before CompleteTask resets it
	assignedChildID := task.AssignedToUserID

	_, err = b.services.TaskService.CompleteTask(bgCtx(), auth, auth.FamilyUID, taskName)
	if err != nil {
		return b.internalError(c, "Error completing task", err)
	}

	if err := c.Send(fmt.Sprintf("✅ Задание \"%s\" подтверждено! %d 💎 начислено", task.Name, task.Tokens)); err != nil {
		return err
	}

	// Notify the child
	if assignedChildID != "" {
		b.notifyChild(assignedChildID,
			fmt.Sprintf("✅ Задание \"%s\" подтверждено!\nТы получил %d 💎", task.Name, task.Tokens))
	}

	return b.showParentMenu(c)
}

func (b *Bot) onReviewReject(c tele.Context) error {
	user, err := b.findUser(c.Sender().ID)
	if err != nil || user == nil {
		return b.internalError(c, "Error finding user", err)
	}
	taskName := user.InputContext
	b.clearState(c.Sender().ID)

	if taskName == "" {
		return c.Send("Задание не выбрано. Попробуй /start")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	task, err := b.services.TaskService.GetTask(bgCtx(), auth, auth.FamilyUID, taskName)
	if err != nil {
		return b.internalError(c, "Error getting task", err)
	}

	assignedChildID := task.AssignedToUserID

	_, err = b.services.TaskService.RejectTask(bgCtx(), auth, auth.FamilyUID, taskName)
	if err != nil {
		return b.internalError(c, "Error rejecting task", err)
	}

	if err := c.Send(fmt.Sprintf("❌ Задание \"%s\" отклонено", task.Name)); err != nil {
		return err
	}

	// Notify the child
	if assignedChildID != "" {
		b.notifyChild(assignedChildID,
			fmt.Sprintf("❌ Задание \"%s\" отклонено.\n\nПопробуй еще раз!", task.Name))
	}

	return b.showParentMenu(c)
}
