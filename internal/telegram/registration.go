package telegram

import (
	"errors"
	"fmt"

	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

func (b *Bot) showRoleSelection(c tele.Context) error {
	name := extractName(c.Sender())
	kb := inlineKeyboard(
		btnRow(
			btn("Я родитель", "role_parent"),
			btn("Я ребенок", "role_child"),
		),
	)
	return c.Send(
		fmt.Sprintf("👋 Привет, %s! Кто ты?", name),
		kb,
	)
}

func (b *Bot) showFamilySetup(c tele.Context, role string) error {
	if role == string(domain.RoleParent) {
		kb := inlineKeyboard(
			btnRow(
				btn("Создать новую", "family_create"),
				btn("Присоединиться", "family_join"),
			),
		)
		return c.Send("Ты еще не в семье. Создать новую или присоединиться?", kb)
	}

	// Child
	if err := b.setState(c.Sender().ID, stateJoinFamily, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Попроси родителей дать тебе код семьи и введи его здесь")
}

func (b *Bot) onSelectRole(c tele.Context, role string) error {
	user := &models.Users{
		UserID:   makeUserID(c.Sender().ID),
		Name:     extractName(c.Sender()),
		Role:     role,
		Platform: "telegram",
	}
	if err := b.services.UserService.CreateUser(bgCtx(), user); err != nil {
		return b.internalError(c, "Error creating user", err)
	}

	return b.showFamilySetup(c, role)
}

func (b *Bot) onFamilyCreate(c tele.Context) error {
	family := &models.Families{
		Name:            fmt.Sprintf("Семья %s", extractName(c.Sender())),
		CreatedByUserID: makeUserID(c.Sender().ID),
	}
	if err := b.services.FamilyService.CreateFamily(bgCtx(), family); err != nil {
		return b.internalError(c, "Error creating family", err)
	}

	if err := b.services.UserService.UpdateFamilyUID(bgCtx(), makeUserID(c.Sender().ID), family.UID); err != nil {
		return b.internalError(c, "Error updating user family", err)
	}

	if err := c.Send(fmt.Sprintf(
		"🎉 Семья создана\\! Код: `%s`\n\nЕго нужно будет ввести остальным членам семьи",
		family.UID,
	), tele.ModeMarkdownV2); err != nil {
		return err
	}
	return b.showParentMenu(c)
}

func (b *Bot) onFamilyJoinPrompt(c tele.Context) error {
	if err := b.setState(c.Sender().ID, stateJoinFamily, ""); err != nil {
		return b.internalError(c, "Error setting state", err)
	}
	return c.Send("✏️ Введи код семьи")
}

func (b *Bot) onFamilyJoinText(c tele.Context, familyUID string) error {
	// Verify family exists
	_, err := b.services.FamilyService.FindFamily(bgCtx(), familyUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := b.setState(c.Sender().ID, stateJoinFamily, ""); err != nil {
				return b.internalError(c, "Error setting state", err)
			}
			return c.Send("❌ Семья с таким кодом не найдена. Проверь код и попробуй еще раз")
		}
		return b.internalError(c, "Error finding family", err)
	}

	if err := b.services.UserService.UpdateFamilyUID(bgCtx(), makeUserID(c.Sender().ID), familyUID); err != nil {
		return b.internalError(c, "Error joining family", err)
	}

	user, _ := b.findUser(c.Sender().ID)

	if err := c.Send("🎉 Добро пожаловать в семью!"); err != nil {
		return err
	}

	// Notify parents about new member
	if user != nil {
		roleName := "родитель"
		if user.Role == string(domain.RoleChild) {
			roleName = "ребёнок"
		}
		b.notifyParents(familyUID, c.Sender().ID,
			fmt.Sprintf("👋 %s присоединился к семье (%s)", user.Name, roleName))
	}

	if user != nil && user.Role == string(domain.RoleChild) {
		return b.showChildMenu(c)
	}
	return b.showParentMenu(c)
}
