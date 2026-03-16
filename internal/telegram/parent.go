package telegram

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) showParentMenu(c tele.Context) error {
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

	var sb strings.Builder
	if len(children) == 0 {
		sb.WriteString("Дети еще не добавлены...\n")
	} else {
		for _, child := range children {
			tokens, err := b.services.TokenService.GetUserTokens(bgCtx(), auth, child.UserID)
			if err != nil {
				return b.internalError(c, "Error getting tokens", err)
			}
			sb.WriteString(fmt.Sprintf("%s: %d 💎\n", child.Name, tokens.Tokens))
		}
	}

	// Check for pending review tasks
	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}
	pendingCount := 0
	for _, t := range tasks {
		if t.Status == domain.TaskStatusCheck {
			pendingCount++
		}
	}

	rows := [][]tele.InlineButton{
		btnRow(btn("📋 Задания", "parent_tasks"), btn("🎁 Награды", "parent_rewards")),
		btnRow(btn("⚠️ Штрафы", "parent_penalties"), btn("🔧 Коррекция", "manual_adjust")),
		btnRow(btn("👨‍👩‍👧‍👦 Семья", "parent_family"), btn("📜 История", "parent_history")),
	}
	if pendingCount > 0 {
		rows = append(
			[][]tele.InlineButton{
				btnRow(btn(fmt.Sprintf("🔍 Проверить (%d)", pendingCount), "parent_review")),
			},
			rows...,
		)
	}

	return c.Send(sb.String(), inlineKeyboard(rows...))
}

// --- Tasks ---

func (b *Bot) showTasks(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	items := make([]string, len(tasks))
	for i, t := range tasks {
		items[i] = formatEntityItem(t.Name, t.Tokens)
	}

	msg := formatList("📋 Задания", items)
	kb := inlineKeyboard(
		btnRow(btn("➕ Добавить", "task_add"), btn("📝 Списком", "task_add_bulk")),
		btnRow(btn("🗑 Удалить", "task_delete"), btn("✏️ Изменить", "task_edit")),
		btnRow(btn("⬅️ Назад", "back_parent")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onAddTaskPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddTaskName, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи название задания", backKeyboard("parent_tasks"))
}

func (b *Bot) onAddTaskName(c tele.Context, name string) error {
	if len(name) > maxEntityNameLength {
		return c.Send(fmt.Sprintf("❌ Название слишком длинное (максимум %d символов)", maxEntityNameLength))
	}
	if err := b.setState(c.Sender().ID, stateAddTaskTokens, name); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("💰 Сколько токенов за это задание?", backKeyboard("parent_tasks"))
}

func (b *Bot) onAddTaskTokens(c tele.Context, text, taskName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	task := &models.Tasks{
		Entities: models.Entities{
			FamilyUID: auth.FamilyUID,
			Name:      taskName,
			Tokens:    tokens,
		},
	}
	if err := b.services.TaskService.CreateTask(bgCtx(), auth, task); err != nil {
		if errors.Is(err, domain.ErrValidation) {
			return c.Send(err.Error())
		}
		if isDuplicateKey(err) {
			return c.Send("❌ Задание с таким именем уже существует")
		}
		return b.internalError(c, "Error creating task", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Задание добавлено!"); err != nil {
		return err
	}

	b.notifyChildren(auth.FamilyUID, fmt.Sprintf("📋 Новое задание: %s (%d 💎)", taskName, tokens))

	return b.showTasks(c)
}

// --- Task bulk add ---

func (b *Bot) onAddTaskBulkPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddTaskBulk, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		"Введи задания списком, каждое на новой строке.\nПоследнее слово — количество токенов.\n\n<b>Пример:</b>\nЗагрузить посудомойку 2\nВынести мусор 5\nЧитать час 12",
		tele.ModeHTML,
		backKeyboard("parent_tasks"),
	)
}

func (b *Bot) onAddTaskBulk(c tele.Context, text string) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	items, errs := parseBulkInput(text)
	var added []string
	for _, item := range items {
		task := &models.Tasks{
			Entities: models.Entities{
				FamilyUID: auth.FamilyUID,
				Name:      item.Name,
				Tokens:    item.Tokens,
			},
		}
		if err := b.services.TaskService.CreateTask(bgCtx(), auth, task); err != nil {
			if isDuplicateKey(err) {
				errs = append(errs, fmt.Sprintf("'%s' — уже существует", item.Name))
			} else {
				errs = append(errs, fmt.Sprintf("'%s' — ошибка", item.Name))
			}
			continue
		}
		added = append(added, formatEntityItem(item.Name, item.Tokens))
	}

	b.clearState(c.Sender().ID)

	if err := c.Send(formatBulkResult(added, errs, "Не найдено заданий для добавления")); err != nil {
		return err
	}

	if len(added) > 0 {
		b.notifyChildren(auth.FamilyUID, fmt.Sprintf("📋 Добавлено %d новых заданий", len(added)))
	}

	return b.showTasks(c)
}

// --- Task edit (number grid → text input for tokens) ---

func (b *Bot) onEditTaskPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	if len(tasks) == 0 {
		return c.Send("Нет заданий", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_tasks"))))
	}

	items := make([]string, len(tasks))
	for i, t := range tasks {
		items[i] = formatEntityItem(t.Name, t.Tokens)
	}

	msg := formatList("Выбери задание для изменения", items)
	grid := numberGrid(len(tasks), "pick_edit_task")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_tasks")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onEditTaskPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	if num < 1 || num > len(tasks) {
		return c.Send("Нет задания с таким номером")
	}

	taskName := tasks[num-1].Name
	if err := b.setState(c.Sender().ID, stateEditTaskTokens, taskName); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		fmt.Sprintf("📋 Задание: %s\n\n💰 Сколько токенов за это задание?", taskName),
		backKeyboard("parent_tasks"),
	)
}

func (b *Bot) onEditTaskTokens(c tele.Context, text, taskName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	updates := &domain.UpdateTaskRequest{Tokens: &tokens}
	if _, err := b.services.TaskService.UpdateTask(bgCtx(), auth, auth.FamilyUID, taskName, updates); err != nil {
		return b.internalError(c, "Error updating task", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Задание изменено!"); err != nil {
		return err
	}
	return b.showTasks(c)
}

// --- Task delete (number grid → immediate action) ---

func (b *Bot) onDeleteTaskPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	if len(tasks) == 0 {
		return c.Send("Нет заданий", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_tasks"))))
	}

	items := make([]string, len(tasks))
	for i, t := range tasks {
		items[i] = formatEntityItem(t.Name, t.Tokens)
	}

	msg := formatList("Выбери задание для удаления", items)
	grid := numberGrid(len(tasks), "pick_del_task")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_tasks")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onDeleteTaskPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	if num < 1 || num > len(tasks) {
		return c.Send("Нет задания с таким номером")
	}

	taskName := tasks[num-1].Name
	if err := b.services.TaskService.DeleteTask(bgCtx(), auth, auth.FamilyUID, taskName); err != nil {
		return b.internalError(c, "Error deleting task", err)
	}

	if err := c.Send("✅ Задание удалено!"); err != nil {
		return err
	}
	return b.showTasks(c)
}
