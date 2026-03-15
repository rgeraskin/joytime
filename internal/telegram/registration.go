package telegram

import (
	"errors"
	"fmt"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func (b *Bot) showWelcome(c tele.Context) error {
	name := extractName(c.Sender())
	kb := inlineKeyboard(
		btnRow(
			btn("🏠 Создать семью", "family_create"),
			btn("🔑 Ввести код", "invite_join"),
		),
	)
	return c.Send(
		fmt.Sprintf("👋 Привет, %s! Создай семью или введи код приглашения.", name),
		kb,
	)
}

func (b *Bot) onFamilyCreate(c tele.Context) error {
	// Create parent user if not exists
	userID := makeUserID(c.Sender().ID)
	user, _ := b.findUser(c.Sender().ID)
	if user == nil {
		newUser := &models.Users{
			UserID:   userID,
			Name:     extractName(c.Sender()),
			Role:     string(domain.RoleParent),
			Platform: "telegram",
		}
		if err := b.services.UserService.CreateUser(bgCtx(), newUser); err != nil {
			return b.internalError(c, "Error creating user", err)
		}
	}

	family := &models.Families{
		Name:            fmt.Sprintf("Семья %s", extractName(c.Sender())),
		CreatedByUserID: userID,
	}
	if err := b.services.FamilyService.CreateFamily(bgCtx(), family); err != nil {
		return b.internalError(c, "Error creating family", err)
	}

	if err := b.services.UserService.UpdateFamilyUID(bgCtx(), userID, family.UID); err != nil {
		return b.internalError(c, "Error updating user family", err)
	}

	if err := c.Send("🎉 Семья создана! Теперь пригласи участников через меню «Семья»."); err != nil {
		return err
	}
	return b.showParentMenu(c)
}

func (b *Bot) onInviteJoinPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateJoinFamily, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи код приглашения")
}

func (b *Bot) onInviteJoinText(c tele.Context, code string) error {
	invite, err := b.services.InviteService.UseInvite(bgCtx(), code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := b.setState(c.Sender().ID, stateJoinFamily, ""); err != nil {
				return b.internalError(c, "Error setting state", err)
			}
			return c.Send("❌ Код приглашения не найден или уже использован. Попробуй ещё раз")
		}
		return b.internalError(c, "Error using invite", err)
	}

	// Create user with role from invite
	userID := makeUserID(c.Sender().ID)
	user, _ := b.findUser(c.Sender().ID)
	if user == nil {
		newUser := &models.Users{
			UserID:    userID,
			Name:      extractName(c.Sender()),
			Role:      invite.Role,
			FamilyUID: invite.FamilyUID,
			Platform:  "telegram",
		}
		if err := b.services.UserService.CreateUser(bgCtx(), newUser); err != nil {
			return b.internalError(c, "Error creating user", err)
		}
	} else {
		// Update existing user
		if err := b.services.UserService.UpdateFamilyUID(bgCtx(), userID, invite.FamilyUID); err != nil {
			return b.internalError(c, "Error joining family", err)
		}
	}

	b.clearState(c.Sender().ID)

	if err := c.Send("🎉 Добро пожаловать в семью!"); err != nil {
		return err
	}

	// Notify parents about new member
	roleName := "родитель"
	if invite.Role == string(domain.RoleChild) {
		roleName = "ребёнок"
	}
	userName := extractName(c.Sender())
	if user != nil {
		userName = user.Name
	}
	b.notifyParents(invite.FamilyUID, c.Sender().ID,
		fmt.Sprintf("👋 %s присоединился к семье (%s)", userName, roleName))

	if invite.Role == string(domain.RoleChild) {
		return b.showChildMenu(c)
	}
	return b.showParentMenu(c)
}
