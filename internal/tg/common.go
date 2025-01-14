package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
)

func tgGetUsername(user *tele.User) string {
	// How how bot call the user

	// join user's first and last names
	username := strings.Join([]string{user.FirstName, user.LastName}, " ")

	if username == "" {
		username = user.Username
	}
	if username == "" {
		username = "Незнакомец"
	}
	return username
}

func tgBot(db *sql.DB) (*tele.Bot, error) {
	var (
		selectorRole  = &tele.ReplyMarkup{}
		btnRoleParent = selectorRole.Data("Я родитель", "register_parent")
		btnRoleChild  = selectorRole.Data("Я ребенок", "register_child")

		selectorFamily = &tele.ReplyMarkup{}
		btnFamilyNew   = selectorFamily.Data("Создать", "family_new")
		btnFamilyJoin  = selectorFamily.Data("Присоединиться", "family_join")

		selectorParent   = &tele.ReplyMarkup{}
		btnParentChecks  = selectorParent.Data("Проверить сделанное", "parent_tasks_check")
		btnParentTasks   = selectorParent.Data("Задания", "parent_tasks")
		btnParentRewards = selectorParent.Data("Награды", "parent_rewards")
		btnParentHistory = selectorParent.Data("История", "parent_history")

		selectorParentTasks  = &tele.ReplyMarkup{}
		btnParentTasksAdd    = selectorParentTasks.Data("Добавить", "parent_tasks_add")
		btnParentTasksDelete = selectorParentTasks.Data("Удалить", "parent_tasks_delete")
		btnParentTasksEdit   = selectorParentTasks.Data("Изменить награду", "parent_tasks_edit")
		btnParentTasksBack   = selectorParentTasks.Data("Назад", "parent_tasks_back")

		selectorParentRewards  = &tele.ReplyMarkup{}
		btnParentRewardsAdd    = selectorParentRewards.Data("Добавить", "parent_rewards_add")
		btnParentRewardsEdit   = selectorParentRewards.Data("Изменить цену", "parent_rewards_edit")
		btnParentRewardsDelete = selectorParentRewards.Data("Удалить", "parent_rewards_delete")
		btnParentRewardsBack   = selectorParentRewards.Data("Назад", "parent_rewards_back")

		selectorParentChecks   = &tele.ReplyMarkup{}
		btnParentChecksApprove = selectorParentChecks.Data("Подтвердить", "parent_check_approve")
		btnParentChecksReject  = selectorParentChecks.Data("Отклонить", "parent_check_reject")
		btnParentChecksBack    = selectorParentChecks.Data("Назад", "parent_check_back")

		selectorParentHistory = &tele.ReplyMarkup{}
		btnParentHistoryBack  = selectorParentHistory.Data("Назад", "parent_history_back")

		selectorChild   = &tele.ReplyMarkup{}
		btnChildTasks   = selectorChild.Data("Задания", "child_tasks")
		btnChildRewards = selectorChild.Data("Награды", "child_rewards")

		selectorChildTasks = &tele.ReplyMarkup{}
		btnChildTasksDone  = selectorChildTasks.Data("Выполнить", "child_tasks_done")
		btnChildTasksBack  = selectorChildTasks.Data("Назад", "child_tasks_back")

		selectorChildRewards = &tele.ReplyMarkup{}
		btnChildRewardsClaim = selectorChildRewards.Data("Получить", "child_rewards_claim")
		btnChildRewardsBack  = selectorChildRewards.Data("Назад", "child_rewards_back")
	)
	selectorRole.Inline(
		selectorRole.Row(btnRoleParent, btnRoleChild),
	)
	selectorFamily.Inline(
		selectorFamily.Row(btnFamilyNew, btnFamilyJoin),
	)
	selectorParent.Inline(
		selectorParent.Row(btnParentChecks),
		selectorParent.Row(btnParentHistory, btnParentTasks, btnParentRewards),
	)
	selectorParentTasks.Inline(
		selectorParentTasks.Row(btnParentTasksAdd, btnParentTasksDelete),
		selectorParentTasks.Row(btnParentTasksEdit, btnParentTasksBack),
	)
	selectorParentRewards.Inline(
		selectorParentRewards.Row(btnParentRewardsAdd, btnParentRewardsEdit, btnParentRewardsDelete),
		selectorParentRewards.Row(btnParentRewardsBack),
	)
	selectorParentChecks.Inline(
		selectorParentChecks.Row(btnParentChecksApprove, btnParentChecksReject),
		selectorParentChecks.Row(btnParentChecksBack),
	)
	selectorParentHistory.Inline(
		selectorParentHistory.Row(btnParentHistoryBack),
	)

	selectorChild.Inline(
		selectorChild.Row(btnChildTasks, btnChildRewards),
	)
	selectorChildTasks.Inline(
		selectorChildTasks.Row(btnChildTasksDone, btnChildTasksBack),
	)
	selectorChildRewards.Inline(
		selectorChildRewards.Row(btnChildRewardsClaim, btnChildRewardsBack),
	)

	slog.Info("Configuring bot...")
	pref := tele.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	slog.Info("Configuring handlers...")

	// start command
	b.Handle("/start", func(c tele.Context) error {
		slog.Info("Received /start command")
		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
	})

	// new parent
	b.Handle(&btnRoleParent, func(c tele.Context) error {
		slog.Info("Received register parent command")
		err = dbUserRegister(db, DBRecordUser{
			Tg_ID: c.Sender().ID,
			Role:  "parent",
		})
		if err != nil {
			slog.Error(err.Error())
			return c.Send("Internal error")
		}
		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
	})

	// new child
	b.Handle(&btnRoleChild, func(c tele.Context) error {
		slog.Info("Received register child command")
		err = dbUserRegister(db, DBRecordUser{
			Tg_ID: c.Sender().ID,
			Role:  "child",
		})
		if err != nil {
			slog.Error(err.Error())
			return c.Send("Internal error")
		}
		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
	})

	// new family
	b.Handle(&btnFamilyNew, func(c tele.Context) error {
		slog.Info("Received create family command")
		err = dbFamilyCreate(db, c.Sender().ID)
		if err != nil {
			slog.Error(err.Error())
			return c.Send("Internal error")
		}
		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
	})

	// join family
	b.Handle(&btnFamilyJoin, func(c tele.Context) error {
		slog.Info("Received join family command")

		err = dbUsersSetTextInput(
			db,
			c.Sender().ID,
			DBTextInput{
				For: sql.NullString{String: "family_id", Valid: true},
			},
		)
		if err != nil {
			slog.Error(err.Error())
			return c.Send("Internal error")
		}
		return c.Send("Введите ID семьи, к которой хотите присоединиться")
	})

	// text message
	b.Handle(tele.OnText, func(c tele.Context) error {
		slog.Info("Received text message")
		return tgHandleText(
			c, db,
			selectorParentTasks,
			selectorParentRewards,
			selectorParent,
			selectorChild,
		)
	})

	// parent tasks
	b.Handle(&btnParentTasks, func(c tele.Context) error {
		slog.Info("Received parent tasks command")
		return tgHandleTasks(c, db, selectorParentTasks)
	})
	// parent rewards
	b.Handle(&btnParentRewards, func(c tele.Context) error {
		slog.Info("Received parent rewards command")
		return tgHandleRewards(c, db, selectorParentRewards)
	})
	// parent history
	b.Handle(&btnParentHistory, func(c tele.Context) error {
		slog.Info("Received parent history command")
		return c.Send("История тут когда-нибудь будет", selectorParentHistory)
	})
	// parent checks
	b.Handle(&btnParentChecks, func(c tele.Context) error {
		slog.Info("Received parent check command")
		return c.Send("Проверить сделанное")
	})
	// parent tasks add
	b.Handle(&btnParentTasksAdd, func(c tele.Context) error {
		slog.Info("Received parent tasks add command")
		return tgHandleParentTasksAdd(c, db)
	})
	// parent tasks edit
	b.Handle(&btnParentTasksEdit, func(c tele.Context) error {
		slog.Info("Received parent tasks edit command")
		return tgHandleParentTasksEdit(c, db)
	})
	// parent tasks delete
	b.Handle(&btnParentTasksDelete, func(c tele.Context) error {
		slog.Info("Received parent tasks delete command")
		return tgHandleParentTasksDelete(c, db)
	})
	// parent tasks back
	b.Handle(&btnParentTasksBack, func(c tele.Context) error {
		slog.Info("Received parent tasks back command")
		return c.Send("Главное меню", selectorParent)
	})
	// parent rewards add
	b.Handle(&btnParentRewardsAdd, func(c tele.Context) error {
		slog.Info("Received parent rewards add command")
		return tgHandleParentRewardsAdd(c, db)
	})
	// parent rewards edit
	b.Handle(&btnParentRewardsEdit, func(c tele.Context) error {
		slog.Info("Received parent rewards edit command")
		return tgHandleParentRewardsEdit(c, db)
	})
	// parent rewards delete
	b.Handle(&btnParentRewardsDelete, func(c tele.Context) error {
		slog.Info("Received parent rewards delete command")
		return tgHandleParentRewardsDelete(c, db)
	})
	// parent rewards back
	b.Handle(&btnParentRewardsBack, func(c tele.Context) error {
		slog.Info("Received parent rewards back command")
		return c.Send("Главное меню", selectorParent)
	})
	// parent checks approve
	b.Handle(&btnParentChecksApprove, func(c tele.Context) error {
		slog.Info("Received parent checks approve command")
		return c.Send("Подтвердить")
	})
	// parent checks reject
	b.Handle(&btnParentChecksReject, func(c tele.Context) error {
		slog.Info("Received parent checks reject command")
		return c.Send("Отклонить")
	})
	// parent checks back
	b.Handle(&btnParentChecksBack, func(c tele.Context) error {
		slog.Info("Received parent checks back command")
		return c.Send(selectorParent)
	})
	// parent history back
	b.Handle(&btnParentHistoryBack, func(c tele.Context) error {
		slog.Info("Received parent history back command")
		return c.Send("Главное меню", selectorParent)
	})

	// child tasks
	b.Handle(&btnChildTasks, func(c tele.Context) error {
		slog.Info("Received child tasks command")
		return tgHandleTasks(c, db, selectorChildTasks)
	})
	// child rewards
	b.Handle(&btnChildRewards, func(c tele.Context) error {
		slog.Info("Received child rewards command")
		return tgHandleRewards(c, db, selectorChildRewards)
	})
	// child tasks done
	b.Handle(&btnChildTasksDone, func(c tele.Context) error {
		slog.Info("Received child tasks done command")
		return tgHandleChildTasksDone(c, db)
	})
	// child tasks back
	b.Handle(&btnChildTasksBack, func(c tele.Context) error {
		slog.Info("Received child tasks back command")
		return tgHandleChild(c, db, selectorChild)
	})
	// child rewards claim
	b.Handle(&btnChildRewardsClaim, func(c tele.Context) error {
		slog.Info("Received child rewards claim command")
		return c.Send("Получить")
	})
	// child rewards back
	b.Handle(&btnChildRewardsBack, func(c tele.Context) error {
		slog.Info("Received child rewards back command")
		return tgHandleChild(c, db, selectorChild)
	})

	return b, err
}

// это пиздец какой-то
type TGHandleCommmonsDiffs struct {
	common         string
	no             string
	many           string
	questionWhat   string
	questionAdd    string
	questionEdit   string
	questionDelete string
}

func tgGetCommonsDiffs(common string) TGHandleCommmonsDiffs {
	switch common {
	case "tasks":
		return TGHandleCommmonsDiffs{
			common:         common,
			no:             "Заданий нет",
			many:           "Задания",
			questionWhat:   "Что сделать с заданиями?",
			questionAdd:    "Как назвать новое задание?",
			questionEdit:   "Для какого задания изменить награду? Введи его номер",
			questionDelete: "Какое задание удалить? Введи его номер",
		}
	case "rewards":
		return TGHandleCommmonsDiffs{
			common:         common,
			no:             "Наград нет",
			many:           "Награды",
			questionWhat:   "Что сделать с наградами?",
			questionAdd:    "Как назвать новую награду?",
			questionEdit:   "Для какой награды изменить цену? Введи ее номер",
			questionDelete: "Какую награду удалить? Введи ее номер",
		}
	}
	return TGHandleCommmonsDiffs{}
}

// tasks
func tgHandleTasks(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	return tgHandleCommmons(tgGetCommonsDiffs("tasks"), selector, c, db)
}

// rewards
func tgHandleRewards(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	return tgHandleCommmons(tgGetCommonsDiffs("rewards"), selector, c, db)
}

func tgHandleParentCommonsAction(handler string, question string, c tele.Context, db *sql.DB) error {
	err := dbUsersSetTextInput(
		db,
		c.Sender().ID,
		DBTextInput{
			For: sql.NullString{String: handler, Valid: true},
		},
	)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	return c.Send(question)
}

func tgShowCommonsList(diffs TGHandleCommmonsDiffs, user DBRecordUser, selector *tele.ReplyMarkup, c tele.Context, db *sql.DB) error {
	commons, err := dbCommonsList(diffs.common, db, user.Family_UID.String)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	if len(commons) == 0 {
		return c.Send(diffs.no, selector)
	}

	message := fmt.Sprintf("%s:\n", diffs.many)
	for i, common := range commons {
		message += fmt.Sprintf("%d. %s - %d 💎\n", i+1, common.Name, common.Tokens)
	}
	message += fmt.Sprintf("\n%s\n", diffs.questionWhat)

	return c.Send(message, selector)
}

func tgHandleCommmons(diffs TGHandleCommmonsDiffs, selector *tele.ReplyMarkup, c tele.Context, db *sql.DB) error {
	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}
	return tgShowCommonsList(diffs, user, selector, c, db)
}

func tgHandleTextFamilyID(
	c tele.Context,
	db *sql.DB,
	selectorParent *tele.ReplyMarkup,
	selectorChild *tele.ReplyMarkup,
) error {
	tgID := c.Sender().ID

	err := dbFamilyJoin(db, DBRecordUser{
		Tg_ID:      tgID,
		Family_UID: sql.NullString{String: c.Text(), Valid: true},
	})
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Семьи с таким ID не найдено. Убедись, что ввел правильный ID.")
	}

	err = dbUsersSetTextInput(db, tgID, DBTextInput{
		sql.NullString{Valid: false},
		sql.NullString{Valid: false},
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// find user by ID in the database
	record, err := dbUserFind(db, tgID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	switch record.Role {
	case "parent":
		return c.Send("Добро пожаловать в семью! Начнем?", selectorParent)
	}
	// case "child":
	return c.Send("Добро пожаловать в семью! Начнем?", selectorChild)
}

func tgHandleText(
	c tele.Context,
	db *sql.DB,
	selectorParentTasks *tele.ReplyMarkup,
	selectorParentRewards *tele.ReplyMarkup,
	selectorParent *tele.ReplyMarkup,
	selectorChild *tele.ReplyMarkup,
) error {
	textInput, err := dbUsersGetTextInput(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	switch textInput.For.String {
	case "family_id":
		return tgHandleTextFamilyID(c, db, selectorParent, selectorChild)
	case "parent_tasks_add_name":
		{
			return tgHandleTextParentCommonsAddName(
				c, db, selectorParentTasks,
				"tasks",
				"Задание с таким именем уже существует",
				"Во сколько жетонов оценить задание?",
				"parent_tasks_add_tokens",
			)
		}
	case "parent_rewards_add_name":
		{
			return tgHandleTextParentCommonsAddName(
				c, db, selectorParentRewards,
				"rewards",
				"Награда с таким именем уже существует",
				"Во сколько жетонов оценить стоимость награды?",
				"parent_rewards_add_tokens",
			)
		}
	case "parent_tasks_add_tokens":
		{
			return tgHandleTextParentCommonsAddReward(
				c, db, selectorParentTasks,
				textInput,
				"tasks",
				"Награда должна быть числом",
				"Задание добавлено!",
			)
		}
	case "parent_rewards_add_tokens":
		{
			return tgHandleTextParentCommonsAddReward(
				c, db, selectorParentRewards,
				textInput,
				"rewards",
				"Награда должна быть числом",
				"Награда добавлена!",
			)
		}
	case "parent_tasks_edit_name":
		{
			return tgHandleTextParentCommonsEditName(
				c, db,
				"tasks",
				"Номер задания должен быть числом",
				"Нет задания с таким номером",
				"Во сколько жетонов оценить задание?",
				"parent_tasks_edit_tokens",
			)
		}
	case "parent_rewards_edit_name":
		{
			return tgHandleTextParentCommonsEditName(
				c, db,
				"rewards",
				"Номер награды должен быть числом",
				"Нет награды с таким номером",
				"Во сколько жетонов оценить награду?",
				"parent_rewards_edit_tokens",
			)
		}
	case "parent_tasks_edit_tokens":
		{
			return tgHandleTextParentCommonsEditTokens(
				c, db, selectorParentTasks, textInput,
				"tasks",
				"Награда за задание должна быть числом",
				"Задание изменено!",
			)
		}
	case "parent_rewards_edit_tokens":
		{
			return tgHandleTextParentCommonsEditTokens(
				c, db, selectorParentRewards, textInput,
				"rewards",
				"Цена награды должна быть числом",
				"Награда изменена!",
			)
		}
	case "parent_tasks_delete_name":
		{
			return tgHandleParentCommonsDeleteName(
				c, db, selectorParentTasks,
				"tasks",
				"Номер задания должен быть числом",
				"Нет задания с таким номером",
				"Задание удалено!",
			)
		}
	case "parent_rewards_delete_name":
		{
			return tgHandleParentCommonsDeleteName(
				c, db, selectorParentRewards,
				"rewards",
				"Номер награды должен быть числом",
				"Нет награды с таким номером",
				"Награда удалена!",
			)
		}
	case "child_tasks_done":
		{
			return tgHandleTextChildTasksDone(c, db, selectorChildTasks)
		}
	case "child_rewards_claim":
		{
			return tgHandleTextChildRewardsClaim(c, db, selectorChildRewards)
		}
	}
	return c.Send("Не понимаю, что ты хочешь от меня 😕")
}

func tgHandleStart(
	c tele.Context,
	db *sql.DB,
	selectorRole *tele.ReplyMarkup,
	selectorFamily *tele.ReplyMarkup,
	selectorParent *tele.ReplyMarkup,
	selectorChild *tele.ReplyMarkup,
) error {
	var user = c.Sender()
	username := tgGetUsername(user)

	// find user by ID in the database
	record, err := dbUserFind(db, user.ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	// check that user record is empty
	// it means that user is not registered yet
	if record.Role == "" {
		slog.Info("User not found. Select role to create...")
		return c.Send(
			fmt.Sprintf("Вижу, мы еще не знакомы. Кто ты, %s? 😊", username),
			selectorRole,
		)
	}

	// check that family ID is empty
	// it means that user is not in the family yet
	if !record.Family_UID.Valid {
		slog.Info("User family not found. Managing family...")
		return c.Send(
			"Ты еще не добавлен в семью. Создать новую или присоединиться к существующей?",
			selectorFamily,
		)
	}

	slog.Info("Sending greeting message...")
	err = c.Send(fmt.Sprintf(
		`Привет, %s\! 😊 ID твоей семьи *%s*

Его нужно будет ввести остальным членам твоей семьи,
когда они начнут пользоваться ботом`,
		username,
		record.Family_UID.String),
		&tele.SendOptions{ParseMode: tele.ModeMarkdownV2},
	)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	switch record.Role {
	case "parent":
		return c.Send("Начнем?", selectorParent)
	}
	// case "child":
	return c.Send("Начнем?", selectorChild)
}
