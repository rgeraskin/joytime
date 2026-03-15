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
		items[i] = fmt.Sprintf("%s: %d 💎", t.Name, t.Tokens)
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

	lines := strings.Split(text, "\n")
	var added []string
	var errs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lastSpace := strings.LastIndex(line, " ")
		if lastSpace < 0 {
			errs = append(errs, fmt.Sprintf("'%s' — нет токенов", line))
			continue
		}

		name := strings.TrimSpace(line[:lastSpace])
		tokensStr := strings.TrimSpace(line[lastSpace+1:])
		tokens, err := parseNumber(tokensStr)
		if err != nil || name == "" {
			errs = append(errs, fmt.Sprintf("'%s' — неверный формат", line))
			continue
		}

		task := &models.Tasks{
			Entities: models.Entities{
				FamilyUID: auth.FamilyUID,
				Name:      name,
				Tokens:    tokens,
			},
		}
		if err := b.services.TaskService.CreateTask(bgCtx(), auth, task); err != nil {
			if isDuplicateKey(err) {
				errs = append(errs, fmt.Sprintf("'%s' — уже существует", name))
			} else {
				errs = append(errs, fmt.Sprintf("'%s' — ошибка", name))
			}
			continue
		}
		added = append(added, fmt.Sprintf("%s: %d 💎", name, tokens))
	}

	b.clearState(c.Sender().ID)

	var sb strings.Builder
	if len(added) > 0 {
		sb.WriteString(fmt.Sprintf("Добавлено %d:\n", len(added)))
		for _, a := range added {
			sb.WriteString("  + " + a + "\n")
		}
	}
	if len(errs) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Ошибки:\n")
		for _, e := range errs {
			sb.WriteString("  - " + e + "\n")
		}
	}
	if len(added) == 0 && len(errs) == 0 {
		sb.WriteString("Не найдено заданий для добавления")
	}

	if err := c.Send(sb.String()); err != nil {
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
		items[i] = fmt.Sprintf("%s: %d 💎", t.Name, t.Tokens)
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
		items[i] = fmt.Sprintf("%s: %d 💎", t.Name, t.Tokens)
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

// --- Rewards ---

func (b *Bot) showRewards(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	rewards, err := b.services.RewardService.GetRewardsForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting rewards", err)
	}

	items := make([]string, len(rewards))
	for i, r := range rewards {
		items[i] = fmt.Sprintf("%s: %d 💎", r.Name, r.Tokens)
	}

	msg := formatList("🎁 Награды", items)
	kb := inlineKeyboard(
		btnRow(btn("➕ Добавить", "reward_add"), btn("📝 Списком", "reward_add_bulk")),
		btnRow(btn("🗑 Удалить", "reward_delete"), btn("✏️ Изменить", "reward_edit")),
		btnRow(btn("⬅️ Назад", "back_parent")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onAddRewardPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddRewardName, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи название награды", backKeyboard("parent_rewards"))
}

func (b *Bot) onAddRewardName(c tele.Context, name string) error {
	if err := b.setState(c.Sender().ID, stateAddRewardTokens, name); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("💰 Сколько токенов стоит награда?", backKeyboard("parent_rewards"))
}

func (b *Bot) onAddRewardTokens(c tele.Context, text, rewardName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	reward := &models.Rewards{
		Entities: models.Entities{
			FamilyUID: auth.FamilyUID,
			Name:      rewardName,
			Tokens:    tokens,
		},
	}
	if err := b.services.RewardService.CreateReward(bgCtx(), auth, reward); err != nil {
		if errors.Is(err, domain.ErrValidation) {
			return c.Send(err.Error())
		}
		if isDuplicateKey(err) {
			return c.Send("❌ Награда с таким именем уже существует")
		}
		return b.internalError(c, "Error creating reward", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Награда добавлена!"); err != nil {
		return err
	}

	b.notifyChildren(auth.FamilyUID, fmt.Sprintf("🎁 Новая награда: %s (%d 💎)", rewardName, tokens))

	return b.showRewards(c)
}

// --- Reward bulk add ---

func (b *Bot) onAddRewardBulkPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddRewardBulk, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		"Введи награды списком, каждая на новой строке.\nПоследнее слово — количество токенов.\n\n<b>Пример:</b>\nСмотреть YouTube 15м 5\nИграть в Роблокс 60м 16",
		tele.ModeHTML,
		backKeyboard("parent_rewards"),
	)
}

func (b *Bot) onAddRewardBulk(c tele.Context, text string) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	lines := strings.Split(text, "\n")
	var added []string
	var errs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lastSpace := strings.LastIndex(line, " ")
		if lastSpace < 0 {
			errs = append(errs, fmt.Sprintf("'%s' — нет токенов", line))
			continue
		}

		name := strings.TrimSpace(line[:lastSpace])
		tokensStr := strings.TrimSpace(line[lastSpace+1:])
		tokens, err := parseNumber(tokensStr)
		if err != nil || name == "" {
			errs = append(errs, fmt.Sprintf("'%s' — неверный формат", line))
			continue
		}

		reward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID: auth.FamilyUID,
				Name:      name,
				Tokens:    tokens,
			},
		}
		if err := b.services.RewardService.CreateReward(bgCtx(), auth, reward); err != nil {
			if isDuplicateKey(err) {
				errs = append(errs, fmt.Sprintf("'%s' — уже существует", name))
			} else {
				errs = append(errs, fmt.Sprintf("'%s' — ошибка", name))
			}
			continue
		}
		added = append(added, fmt.Sprintf("%s: %d 💎", name, tokens))
	}

	b.clearState(c.Sender().ID)

	var sb strings.Builder
	if len(added) > 0 {
		sb.WriteString(fmt.Sprintf("Добавлено %d:\n", len(added)))
		for _, a := range added {
			sb.WriteString("  + " + a + "\n")
		}
	}
	if len(errs) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Ошибки:\n")
		for _, e := range errs {
			sb.WriteString("  - " + e + "\n")
		}
	}
	if len(added) == 0 && len(errs) == 0 {
		sb.WriteString("Не найдено наград для добавления")
	}

	if err := c.Send(sb.String()); err != nil {
		return err
	}

	if len(added) > 0 {
		b.notifyChildren(auth.FamilyUID, fmt.Sprintf("🎁 Добавлено %d новых наград", len(added)))
	}

	return b.showRewards(c)
}

// --- Reward edit (number grid → text input for tokens) ---

func (b *Bot) onEditRewardPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	rewards, err := b.services.RewardService.GetRewardsForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting rewards", err)
	}

	if len(rewards) == 0 {
		return c.Send("Нет наград", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_rewards"))))
	}

	items := make([]string, len(rewards))
	for i, r := range rewards {
		items[i] = fmt.Sprintf("%s: %d 💎", r.Name, r.Tokens)
	}

	msg := formatList("Выбери награду для изменения", items)
	grid := numberGrid(len(rewards), "pick_edit_reward")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_rewards")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onEditRewardPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	rewards, err := b.services.RewardService.GetRewardsForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting rewards", err)
	}

	if num < 1 || num > len(rewards) {
		return c.Send("Нет награды с таким номером")
	}

	rewardName := rewards[num-1].Name
	if err := b.setState(c.Sender().ID, stateEditRewardTokens, rewardName); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(
		fmt.Sprintf("🎁 Награда: %s\n\n💰 Сколько токенов стоит награда?", rewardName),
		backKeyboard("parent_rewards"),
	)
}

func (b *Bot) onEditRewardTokens(c tele.Context, text, rewardName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("❌ Количество токенов должно быть числом")
	}

	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	updates := &domain.UpdateRewardRequest{Tokens: &tokens}
	if _, err := b.services.RewardService.UpdateReward(bgCtx(), auth, auth.FamilyUID, rewardName, updates); err != nil {
		return b.internalError(c, "Error updating reward", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Награда изменена!"); err != nil {
		return err
	}
	return b.showRewards(c)
}

// --- Reward delete (number grid → immediate action) ---

func (b *Bot) onDeleteRewardPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	rewards, err := b.services.RewardService.GetRewardsForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting rewards", err)
	}

	if len(rewards) == 0 {
		return c.Send("Нет наград", inlineKeyboard(btnRow(btn("⬅️ Назад", "parent_rewards"))))
	}

	items := make([]string, len(rewards))
	for i, r := range rewards {
		items[i] = fmt.Sprintf("%s: %d 💎", r.Name, r.Tokens)
	}

	msg := formatList("Выбери награду для удаления", items)
	grid := numberGrid(len(rewards), "pick_del_reward")
	grid = append(grid, btnRow(btn("⬅️ Назад", "parent_rewards")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onDeleteRewardPick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	rewards, err := b.services.RewardService.GetRewardsForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting rewards", err)
	}

	if num < 1 || num > len(rewards) {
		return c.Send("Нет награды с таким номером")
	}

	rewardName := rewards[num-1].Name
	if err := b.services.RewardService.DeleteReward(bgCtx(), auth, auth.FamilyUID, rewardName); err != nil {
		return b.internalError(c, "Error deleting reward", err)
	}

	if err := c.Send("✅ Награда удалена!"); err != nil {
		return err
	}
	return b.showRewards(c)
}

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
		items[i] = fmt.Sprintf("%s: %d 💎", p.Name, p.Tokens)
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

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Штраф добавлен!"); err != nil {
		return err
	}

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

	lines := strings.Split(text, "\n")
	var added []string
	var errs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lastSpace := strings.LastIndex(line, " ")
		if lastSpace < 0 {
			errs = append(errs, fmt.Sprintf("'%s' — нет токенов", line))
			continue
		}

		name := strings.TrimSpace(line[:lastSpace])
		tokensStr := strings.TrimSpace(line[lastSpace+1:])
		tokens, err := parseNumber(tokensStr)
		if err != nil || name == "" {
			errs = append(errs, fmt.Sprintf("'%s' — неверный формат", line))
			continue
		}

		penalty := &models.Penalties{
			Entities: models.Entities{
				FamilyUID: auth.FamilyUID,
				Name:      name,
				Tokens:    tokens,
			},
		}
		if err := b.services.PenaltyService.CreatePenalty(bgCtx(), auth, penalty); err != nil {
			if isDuplicateKey(err) {
				errs = append(errs, fmt.Sprintf("'%s' — уже существует", name))
			} else {
				errs = append(errs, fmt.Sprintf("'%s' — ошибка", name))
			}
			continue
		}
		added = append(added, fmt.Sprintf("%s: %d 💎", name, tokens))
	}

	b.clearState(c.Sender().ID)

	var sb strings.Builder
	if len(added) > 0 {
		sb.WriteString(fmt.Sprintf("Добавлено %d:\n", len(added)))
		for _, a := range added {
			sb.WriteString("  + " + a + "\n")
		}
	}
	if len(errs) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Ошибки:\n")
		for _, e := range errs {
			sb.WriteString("  - " + e + "\n")
		}
	}
	if len(added) == 0 && len(errs) == 0 {
		sb.WriteString("Не найдено штрафов для добавления")
	}

	if err := c.Send(sb.String()); err != nil {
		return err
	}

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
		items[i] = fmt.Sprintf("%s: %d 💎", p.Name, p.Tokens)
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

	updates := &domain.UpdateRewardRequest{Tokens: &tokens}
	if _, err := b.services.PenaltyService.UpdatePenalty(bgCtx(), auth, auth.FamilyUID, penaltyName, updates); err != nil {
		return b.internalError(c, "Error updating penalty", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Штраф изменен!"); err != nil {
		return err
	}
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
		items[i] = fmt.Sprintf("%s: %d 💎", p.Name, p.Tokens)
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
		items[i] = fmt.Sprintf("%s: %d 💎", p.Name, p.Tokens)
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

	b.clearState(c.Sender().ID)
	if err := c.Send("✅ Имя изменено!"); err != nil {
		return err
	}
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

	err = b.services.TokenService.AddTokensToUser(
		bgCtx(), auth, childUserID, tokens,
		domain.TokenTypeManualAdjustment, "Коррекция: "+reason, nil, nil,
	)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientTokens) {
			return c.Send("Недостаточно токенов для снятия")
		}
		return b.internalError(c, "Error adjusting tokens", err)
	}

	b.clearState(c.Sender().ID)

	child, _ := b.services.UserService.FindUser(bgCtx(), childUserID)
	childName := childUserID
	if child != nil {
		childName = child.Name
	}

	sign := "+"
	if tokens < 0 {
		sign = ""
	}
	if err := c.Send(fmt.Sprintf("✅ Коррекция: %s%d 💎 для %s\n\nПричина: %s", sign, tokens, childName, reason)); err != nil {
		return err
	}

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
