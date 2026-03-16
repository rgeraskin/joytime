package telegram

import (
	"errors"
	"fmt"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
)

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
		items[i] = formatEntityItem(r.Name, r.Tokens)
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

	items, errs := parseBulkInput(text)
	var added []string
	for _, item := range items {
		reward := &models.Rewards{
			Entities: models.Entities{
				FamilyUID: auth.FamilyUID,
				Name:      item.Name,
				Tokens:    item.Tokens,
			},
		}
		if err := b.services.RewardService.CreateReward(bgCtx(), auth, reward); err != nil {
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

	if err := c.Send(formatBulkResult(added, errs, "Не найдено наград для добавления")); err != nil {
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
		items[i] = formatEntityItem(r.Name, r.Tokens)
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
		items[i] = formatEntityItem(r.Name, r.Tokens)
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
