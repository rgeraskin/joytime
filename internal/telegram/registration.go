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
		fmt.Sprintf(
			"👋 Привет, %s!\n\nJoyTime — семейный бот с заданиями и наградами. Дети выполняют задания, получают токены 💎 и обменивают их на награды.\n\nСоздай семью или введи код приглашения.",
			name,
		),
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

	if err := c.Send("🎉 Семья создана!\n\nТеперь пригласи участников через меню «Семья»."); err != nil {
		return err
	}
	return b.showParentMenu(c)
}

func (b *Bot) onInviteJoinPrompt(c tele.Context) error {
	// Ensure user record exists so setState and handleText work
	user, _ := b.findUser(c.Sender().ID)
	if user == nil {
		newUser := &models.Users{
			UserID:   makeUserID(c.Sender().ID),
			Name:     extractName(c.Sender()),
			Platform: "telegram",
		}
		if err := b.services.UserService.CreateUser(bgCtx(), newUser); err != nil {
			return b.internalError(c, "Error creating user", err)
		}
	}

	if err := b.setState(c.Sender().ID, stateJoinFamily, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи код приглашения", backKeyboard("back_welcome"))
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

	// Update existing user with role and familyUID from invite
	userID := makeUserID(c.Sender().ID)
	if err := b.services.UserService.SetRoleAndFamily(bgCtx(), userID, invite.Role, invite.FamilyUID); err != nil {
		return b.internalError(c, "Error joining family", err)
	}

	b.clearState(c.Sender().ID)

	welcomeMsg := "🎉 Добро пожаловать в семью!"
	if invite.Role == string(domain.RoleChild) {
		welcomeMsg += "\n\nВыполняй задания, получай токены 💎 и обменивай их на награды. Выполненные задания проверяет родитель."
	} else {
		welcomeMsg += "\n\nДобавляй задания и награды, проверяй выполнение и управляй семьей через меню."
	}
	if err := c.Send(welcomeMsg); err != nil {
		return err
	}

	// Notify parents about new member
	roleName := "родитель"
	if invite.Role == string(domain.RoleChild) {
		roleName = "ребёнок"
	}
	user, _ := b.findUser(c.Sender().ID)
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
