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

	children, err := b.services.UserService.FindFamilyUsersByRole(bgCtx(), auth.FamilyUID, string(domain.RoleChild))
	if err != nil {
		return b.internalError(c, "Error getting children", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Код семьи: `%s`\n\n", auth.FamilyUID))

	if len(children) == 0 {
		sb.WriteString("Дети еще не добавлены\\.\\.\\.\n")
	} else {
		for _, child := range children {
			tokens, err := b.services.TokenService.GetUserTokens(bgCtx(), auth, child.UserID)
			if err != nil {
				return b.internalError(c, "Error getting tokens", err)
			}
			sb.WriteString(fmt.Sprintf("%s: %d 💎\n", escapeMarkdownV2(child.Name), tokens.Tokens))
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
		btnRow(btn("Задания", "parent_tasks"), btn("Награды", "parent_rewards")),
	}
	if pendingCount > 0 {
		rows = append(
			[][]tele.InlineButton{
				btnRow(btn(fmt.Sprintf("Проверить (%d)", pendingCount), "parent_review")),
			},
			rows...,
		)
	}

	return c.Send(sb.String(), tele.ModeMarkdownV2, inlineKeyboard(rows...))
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

	msg := formatList("Задания", items)
	kb := inlineKeyboard(
		btnRow(btn("Добавить", "task_add"), btn("Добавить списком", "task_add_bulk")),
		btnRow(btn("Удалить", "task_delete"), btn("Изменить цену", "task_edit")),
		btnRow(btn("Назад", "back_parent")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onAddTaskPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddTaskName, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("Введи название задания")
}

func (b *Bot) onAddTaskName(c tele.Context, name string) error {
	if err := b.setState(c.Sender().ID, stateAddTaskTokens, name); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("Сколько токенов за это задание?")
}

func (b *Bot) onAddTaskTokens(c tele.Context, text, taskName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("Количество токенов должно быть числом")
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
			return c.Send("Задание с таким именем уже существует")
		}
		return b.internalError(c, "Error creating task", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("Задание добавлено!"); err != nil {
		return err
	}
	return b.showTasks(c)
}

// --- Task bulk add ---

func (b *Bot) onAddTaskBulkPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddTaskBulk, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("Введи задания списком, каждое на новой строке.\nПоследнее слово — количество токенов.\n\nПример:\nЗагрузить посудомойку 2\nВынести мусор 5\nЧитать час 12")
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
		return c.Send("Нет заданий", inlineKeyboard(btnRow(btn("Назад", "parent_tasks"))))
	}

	items := make([]string, len(tasks))
	for i, t := range tasks {
		items[i] = fmt.Sprintf("%s: %d 💎", t.Name, t.Tokens)
	}

	msg := formatList("Выбери задание для изменения", items)
	grid := numberGrid(len(tasks), "pick_edit_task")
	grid = append(grid, btnRow(btn("Назад", "parent_tasks")))
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
	return c.Send(fmt.Sprintf("Задание: %s\nСколько токенов за это задание?", taskName))
}

func (b *Bot) onEditTaskTokens(c tele.Context, text, taskName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("Количество токенов должно быть числом")
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
	if err := c.Send("Задание изменено!"); err != nil {
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
		return c.Send("Нет заданий", inlineKeyboard(btnRow(btn("Назад", "parent_tasks"))))
	}

	items := make([]string, len(tasks))
	for i, t := range tasks {
		items[i] = fmt.Sprintf("%s: %d 💎", t.Name, t.Tokens)
	}

	msg := formatList("Выбери задание для удаления", items)
	grid := numberGrid(len(tasks), "pick_del_task")
	grid = append(grid, btnRow(btn("Назад", "parent_tasks")))
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

	if err := c.Send("Задание удалено!"); err != nil {
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

	msg := formatList("Награды", items)
	kb := inlineKeyboard(
		btnRow(btn("Добавить", "reward_add"), btn("Добавить списком", "reward_add_bulk")),
		btnRow(btn("Удалить", "reward_delete"), btn("Изменить цену", "reward_edit")),
		btnRow(btn("Назад", "back_parent")),
	)
	return c.Send(msg, kb)
}

func (b *Bot) onAddRewardPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddRewardName, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("Введи название награды")
}

func (b *Bot) onAddRewardName(c tele.Context, name string) error {
	if err := b.setState(c.Sender().ID, stateAddRewardTokens, name); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("Сколько токенов стоит награда?")
}

func (b *Bot) onAddRewardTokens(c tele.Context, text, rewardName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("Количество токенов должно быть числом")
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
			return c.Send("Награда с таким именем уже существует")
		}
		return b.internalError(c, "Error creating reward", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("Награда добавлена!"); err != nil {
		return err
	}
	return b.showRewards(c)
}

// --- Reward bulk add ---

func (b *Bot) onAddRewardBulkPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateAddRewardBulk, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("Введи награды списком, каждая на новой строке.\nПоследнее слово — количество токенов.\n\nПример:\nСмотреть YouTube 15м 5\nИграть в Роблокс 60м 16")
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
		return c.Send("Нет наград", inlineKeyboard(btnRow(btn("Назад", "parent_rewards"))))
	}

	items := make([]string, len(rewards))
	for i, r := range rewards {
		items[i] = fmt.Sprintf("%s: %d 💎", r.Name, r.Tokens)
	}

	msg := formatList("Выбери награду для изменения", items)
	grid := numberGrid(len(rewards), "pick_edit_reward")
	grid = append(grid, btnRow(btn("Назад", "parent_rewards")))
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
	return c.Send(fmt.Sprintf("Награда: %s\nСколько токенов стоит награда?", rewardName))
}

func (b *Bot) onEditRewardTokens(c tele.Context, text, rewardName string) error {
	tokens, err := parseNumber(text)
	if err != nil {
		return c.Send("Количество токенов должно быть числом")
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
	if err := c.Send("Награда изменена!"); err != nil {
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
		return c.Send("Нет наград", inlineKeyboard(btnRow(btn("Назад", "parent_rewards"))))
	}

	items := make([]string, len(rewards))
	for i, r := range rewards {
		items[i] = fmt.Sprintf("%s: %d 💎", r.Name, r.Tokens)
	}

	msg := formatList("Выбери награду для удаления", items)
	grid := numberGrid(len(rewards), "pick_del_reward")
	grid = append(grid, btnRow(btn("Назад", "parent_rewards")))
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

	if err := c.Send("Награда удалена!"); err != nil {
		return err
	}
	return b.showRewards(c)
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
		return c.Send("Нет заданий для проверки", inlineKeyboard(btnRow(btn("Назад", "back_parent"))))
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
	grid = append(grid, btnRow(btn("Назад", "back_parent")))
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
			btn("Подтвердить", "review_approve"),
			btn("Отклонить", "review_reject"),
		),
		btnRow(btn("Назад", "parent_review")),
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

	if err := c.Send(fmt.Sprintf("Задание \"%s\" подтверждено! %d 💎 начислено", task.Name, task.Tokens)); err != nil {
		return err
	}

	// Notify the child
	if assignedChildID != "" {
		b.notifyChild(assignedChildID,
			fmt.Sprintf("Задание \"%s\" подтверждено! Ты получил %d 💎", task.Name, task.Tokens))
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

	if err := c.Send(fmt.Sprintf("Задание \"%s\" отклонено", task.Name)); err != nil {
		return err
	}

	// Notify the child
	if assignedChildID != "" {
		b.notifyChild(assignedChildID,
			fmt.Sprintf("Задание \"%s\" отклонено. Попробуй еще раз!", task.Name))
	}

	return b.showParentMenu(c)
}
