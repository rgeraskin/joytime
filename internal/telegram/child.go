package telegram

import (
	"errors"
	"fmt"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) showChildMenu(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tokens, err := b.services.TokenService.GetUserTokens(bgCtx(), auth, auth.UserID)
	if err != nil {
		return b.internalError(c, "Error getting tokens", err)
	}

	kb := inlineKeyboard(
		btnRow(
			btn("Выполнить задание", "child_task_done"),
			btn("Получить награду", "child_reward_claim"),
		),
		btnRow(btn("Штрафы", "child_penalties")),
	)
	return c.Send(fmt.Sprintf("Твой баланс: %d 💎", tokens.Tokens), kb)
}

// --- Penalties (read-only) ---

func (b *Bot) showChildPenalties(c tele.Context) error {
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

	msg := formatList("Штрафы", items)
	return c.Send(msg, inlineKeyboard(btnRow(btn("Назад", "back_child"))))
}

// --- Task completion (number grid → immediate action) ---

func (b *Bot) onTaskDonePrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	// Show only available tasks (status: new)
	var available []models.Tasks
	for _, t := range tasks {
		if t.Status == domain.TaskStatusNew {
			available = append(available, t)
		}
	}

	if len(available) == 0 {
		return c.Send("Нет доступных заданий", inlineKeyboard(btnRow(btn("Назад", "back_child"))))
	}

	items := make([]string, len(available))
	for i, t := range available {
		items[i] = fmt.Sprintf("%s: %d 💎", t.Name, t.Tokens)
	}

	msg := formatList("Доступные задания", items)
	grid := numberGrid(len(available), "pick_task_done")
	grid = append(grid, btnRow(btn("Назад", "back_child")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onTaskDonePick(c tele.Context, num int) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	tasks, err := b.services.TaskService.GetTasksForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting tasks", err)
	}

	var available []models.Tasks
	for _, t := range tasks {
		if t.Status == domain.TaskStatusNew {
			available = append(available, t)
		}
	}

	if num < 1 || num > len(available) {
		return c.Send("Нет задания с таким номером")
	}

	task := available[num-1]
	_, err = b.services.TaskService.CompleteTask(bgCtx(), auth, auth.FamilyUID, task.Name)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotAssignedToUser) {
			return c.Send("Это задание назначено другому ребенку")
		}
		return b.internalError(c, "Error completing task", err)
	}

	if err := c.Send(fmt.Sprintf(
		"Задание \"%s\" отмечено выполненным!\nПосле проверки родителем ты получишь %d 💎",
		task.Name, task.Tokens,
	)); err != nil {
		return err
	}

	// Notify parents
	user, _ := b.findUser(c.Sender().ID)
	childName := extractName(c.Sender())
	if user != nil {
		childName = user.Name
	}
	b.notifyParents(auth.FamilyUID, c.Sender().ID,
		fmt.Sprintf("%s выполнил задание: %s (%d 💎)", childName, task.Name, task.Tokens))

	return b.showChildMenu(c)
}

// --- Reward claiming (number grid → immediate action) ---

func (b *Bot) onRewardClaimPrompt(c tele.Context) error {
	auth, err := b.authCtx(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error creating auth context", err)
	}

	rewards, err := b.services.RewardService.GetRewardsForFamily(bgCtx(), auth, auth.FamilyUID)
	if err != nil {
		return b.internalError(c, "Error getting rewards", err)
	}

	if len(rewards) == 0 {
		return c.Send("Нет доступных наград", inlineKeyboard(btnRow(btn("Назад", "back_child"))))
	}

	items := make([]string, len(rewards))
	for i, r := range rewards {
		items[i] = fmt.Sprintf("%s: %d 💎", r.Name, r.Tokens)
	}

	msg := formatList("Награды", items)
	grid := numberGrid(len(rewards), "pick_reward_claim")
	grid = append(grid, btnRow(btn("Назад", "back_child")))
	return c.Send(msg, inlineKeyboard(grid...))
}

func (b *Bot) onRewardClaimPick(c tele.Context, num int) error {
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

	reward := rewards[num-1]
	err = b.services.TokenService.ClaimReward(bgCtx(), auth, auth.FamilyUID, reward.Name)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientTokens) {
			return c.Send("У тебя недостаточно 💎")
		}
		return b.internalError(c, "Error claiming reward", err)
	}

	if err := c.Send(fmt.Sprintf("Награда \"%s\" получена!", reward.Name)); err != nil {
		return err
	}

	// Notify parents
	user, _ := b.findUser(c.Sender().ID)
	childName := extractName(c.Sender())
	if user != nil {
		childName = user.Name
	}
	b.notifyParents(auth.FamilyUID, c.Sender().ID,
		fmt.Sprintf("%s получил награду: %s (%d 💎)", childName, reward.Name, reward.Tokens))

	return b.showChildMenu(c)
}
