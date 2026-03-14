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
	)
	return c.Send(fmt.Sprintf("Твой баланс: %d 💎", tokens.Tokens), kb)
}

// --- Task completion ---

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
	if err := b.setState(c.Sender().ID, stateTaskDone, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(msg+"\nВведи номер задания",
		inlineKeyboard(btnRow(btn("Назад", "back_child"))))
}

func (b *Bot) onTaskDoneText(c tele.Context, text string) error {
	num, err := parseNumber(text)
	if err != nil {
		return c.Send("Номер задания должен быть числом")
	}

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

	b.clearState(c.Sender().ID)
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

// --- Reward claiming ---

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
	if err := b.setState(c.Sender().ID, stateRewardClaim, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send(msg+"\nВведи номер награды",
		inlineKeyboard(btnRow(btn("Назад", "back_child"))))
}

func (b *Bot) onRewardClaimText(c tele.Context, text string) error {
	num, err := parseNumber(text)
	if err != nil {
		return c.Send("Номер награды должен быть числом")
	}

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

	b.clearState(c.Sender().ID)
	if err := c.Send(fmt.Sprintf("Награда \"%s\" куплена!", reward.Name)); err != nil {
		return err
	}

	// Notify parents
	user, _ := b.findUser(c.Sender().ID)
	childName := extractName(c.Sender())
	if user != nil {
		childName = user.Name
	}
	b.notifyParents(auth.FamilyUID, c.Sender().ID,
		fmt.Sprintf("%s купил награду: %s (%d 💎)", childName, reward.Name, reward.Tokens))

	return b.showChildMenu(c)
}
