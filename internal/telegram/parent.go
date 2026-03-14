package telegram

import (
	"errors"
	"fmt"
	"strconv"
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
		btnRow(btn("Добавить", "task_add"), btn("Удалить", "task_delete")),
		btnRow(btn("Изменить цену", "task_edit"), btn("Назад", "back_parent")),
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
		return b.internalError(c, "Error creating task", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("Задание добавлено!"); err != nil {
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
		btnRow(btn("Добавить", "reward_add"), btn("Удалить", "reward_delete")),
		btnRow(btn("Изменить цену", "reward_edit"), btn("Назад", "back_parent")),
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
		return b.internalError(c, "Error creating reward", err)
	}

	b.clearState(c.Sender().ID)
	if err := c.Send("Награда добавлена!"); err != nil {
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

// --- Task Review (number grid → immediate action) ---

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
	completedTask, err := b.services.TaskService.CompleteTask(bgCtx(), auth, auth.FamilyUID, task.Name)
	if err != nil {
		return b.internalError(c, "Error completing task", err)
	}

	if err := c.Send(fmt.Sprintf("Задание \"%s\" подтверждено! %d 💎 начислено", task.Name, task.Tokens)); err != nil {
		return err
	}

	// Notify the child who completed the task
	if completedTask.AssignedToUserID != "" {
		child, _ := b.services.UserService.FindUser(bgCtx(), completedTask.AssignedToUserID)
		if child != nil {
			tgIDStr := strings.TrimPrefix(child.UserID, "user_")
			if tgID, err := strconv.ParseInt(tgIDStr, 10, 64); err == nil {
				_, _ = b.bot.Send(&tele.User{ID: tgID},
					fmt.Sprintf("Задание \"%s\" подтверждено! Ты получил %d 💎", task.Name, task.Tokens))
			}
		}
	}

	return b.showParentMenu(c)
}
