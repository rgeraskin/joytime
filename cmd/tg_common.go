package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/NicoNex/echotron/v3"
)

// Recursive type definition of the bot state function.
type stateFn func(*echotron.Update) stateFn
type bot struct {
	chatID int64
	state  stateFn
	name   string
	echotron.API
	db   *sql.DB
	user DBRecordUser
}

func newBot(chatID int64, db *sql.DB) echotron.Bot {
	bot := &bot{
		chatID: chatID,
		API:    echotron.NewAPI(os.Getenv("TOKEN")),
	}
	// We set the default state to the bot.handleMessage method.
	bot.state = bot.handleStart
	bot.db = db
	return bot
}

func (b *bot) Update(update *echotron.Update) {
	// Here we execute the current state and set the next one.
	b.state = b.state(update)
}

func (b *bot) handleStart(update *echotron.Update) stateFn {
	slog.Debug("handleStart")
	if strings.HasPrefix(update.Message.Text, "/start") {
		var user = update.Message.From

		// find user by ID in the database
		if b.user.Role == "" && !b.user.Family_UID.Valid {
			slog.Info("User role is empty, family ID is empty. Fetching user record...")
			record, err := dbUserFind(b.db, user.ID)
			if err != nil {
				slog.Error(err.Error())
				b.SendMessage("Internal error", b.chatID, nil)
			}
			b.user = record
		}

		username := tgGetUsername(user)
		// check that user record is empty
		// it means that user is not registered yet
		if b.user.Role == "" {
			slog.Info("User not found. Select role to create...")
			opts := echotron.MessageOptions{
				ReplyMarkup: &echotron.InlineKeyboardMarkup{
					InlineKeyboard: [][]echotron.InlineKeyboardButton{
						{
							{
								Text:         "Я родитель",
								CallbackData: "register_parent",
							},
							{
								Text:         "Я ребенок",
								CallbackData: "register_child",
							},
						},
					},
				},
			}
			b.SendMessage(
				fmt.Sprintf("Вижу, мы еще не знакомы. Кто ты, %s? 😊", username),
				b.chatID, &opts,
			)
			return b.handleRegisterUser(update)
		}

		slog.Info("HEEEERE")
		// check that family ID is empty
		// it means that user is not in the family yet
		if !b.user.Family_UID.Valid {
			slog.Info("here5")
			slog.Info("User family not found. Managing family...")
			opts := echotron.MessageOptions{
				ReplyMarkup: &echotron.InlineKeyboardMarkup{
					InlineKeyboard: [][]echotron.InlineKeyboardButton{
						{
							{
								Text:         "Создать новую",
								CallbackData: "register_family_create",
							},
							{
								Text:         "Присоединиться",
								CallbackData: "register_family_join",
							},
						},
					},
				},
			}
			b.SendMessage(
				"Ты еще не добавлен в семью. Создать новую или присоединиться к существующей?",
				b.chatID, &opts,
			)
			return b.handleRegisterFamily
		}
		slog.Info("here6")

		switch b.user.Role {
		case "parent":
			{
				slog.Info("here7")
				return b.handleParent(update)
			}
		case "child":
			{
				slog.Info("here8")
				// return b.handleChild
				return b.handleStart
			}
		}
	}
	slog.Info("here9")
	return b.handleStart
}

func (b *bot) handleRegisterUser(update *echotron.Update) stateFn {
	// this magic is only to make func calls look similar
	return func(update *echotron.Update) stateFn {
		ret := func() stateFn {
			username := tgGetUsername(update.CallbackQuery.From)
			b.SendMessage(fmt.Sprintf("Привет, %s! 😊", username), b.chatID, nil)
			return b.handleStart
		}

		switch update.CallbackQuery.Data {
		case "register_parent":
			{
				err := dbUserRegister(b.db, DBRecordUser{
					Tg_ID: update.CallbackQuery.From.ID,
					Role:  "parent",
				})
				if err != nil {
					slog.Error(err.Error())
					b.SendMessage("Internal error", b.chatID, nil)
				}
				return ret()
			}
		case "register_child":
			{
				err := dbUserRegister(b.db, DBRecordUser{
					Tg_ID: update.CallbackQuery.From.ID,
					Role:  "child",
				})
				if err != nil {
					slog.Error(err.Error())
					b.SendMessage("Internal error", b.chatID, nil)
				}
				return ret()
			}
		}
		return b.handleRegisterUser
	}
}

func (b *bot) handleRegisterFamily(update *echotron.Update) stateFn {
	switch update.CallbackQuery.Data {
	case "register_family_create":
		{
			familyID, err := dbFamilyCreate(b.db, update.CallbackQuery.From.ID)
			b.user.Family_UID = familyID
			if err != nil {
				slog.Error(err.Error())
				b.SendMessage("Internal error", b.chatID, nil)
			}

			slog.Info("Sending greeting message...")
			opts := echotron.MessageOptions{
				ParseMode: echotron.MarkdownV2,
			}
			b.SendMessage(`ID твоей семьи *%s*

			Его нужно будет ввести остальным членам твоей семьи,
			когда они начнут пользоваться ботом`,
				b.chatID, &opts,
			)
			return b.handleStart
		}
	case "register_family_join":
		{
			b.SendMessage("Введи ID семьи, к которой хочешь присоединиться", b.chatID, nil)
			return b.handleRegisterFamilyJoin
		}
	}
	return b.handleRegisterFamily
}

func (b *bot) handleRegisterFamilyJoin(update *echotron.Update) stateFn {
	FamilyID := update.Message.Text
	tgID := update.Message.From.ID

	err := dbFamilyJoin(b.db, DBRecordUser{
		Tg_ID:      tgID,
		Family_UID: sql.NullString{String: FamilyID, Valid: true},
	})
	if err != nil {
		slog.Error(err.Error())
		b.SendMessage(
			"Семьи с таким ID не найдено. Убедись, что ввел правильный ID.",
			b.chatID,
			nil,
		)
	}

	// find user by ID in the database
	record, err := dbUserFind(b.db, tgID)
	if err != nil {
		slog.Error(err.Error())
		b.SendMessage("Internal error", b.chatID, nil)
	}

	b.SendMessage("Добро пожаловать в семью! Начнем?", b.chatID, nil)
	switch record.Role {
	case "parent":
		{
			return b.handleParent
		}
	case "child":
		{
			// return b.handleChild
			return b.handleStart
		}
	}
	return b.handleRegisterFamilyJoin
}

func (b *bot) handleParent(update *echotron.Update) stateFn {
	slog.Debug("handleParent")
	children, err := dbUsersGet(b.db, b.user.Family_UID, "child")
	if err != nil {
		slog.Error(err.Error())
		b.SendMessage("Internal error", b.chatID, nil)
	}

	var messageStrings []string
	var opts echotron.MessageOptions
	for _, tgID := range children {
		tokens, err := dbTokensGet(b.db, tgID)
		if err != nil {
			slog.Error(err.Error())
			b.SendMessage("Internal error", b.chatID, nil)
		}

		opts = echotron.MessageOptions{
			ReplyMarkup: &echotron.InlineKeyboardMarkup{
				InlineKeyboard: [][]echotron.InlineKeyboardButton{
					// {
					// 	{
					// 		Text:         "Проверить сделанное",
					// 		CallbackData: "parent_tasks_check",
					// 	},
					// },
					{
						{
							Text:         "Задания",
							CallbackData: "parent_tasks",
						},
						{
							Text:         "Награды",
							CallbackData: "parent_rewards",
						},
					},
				},
			},
		}
		messageStrings = append(messageStrings, fmt.Sprintf("Баланс ребенка: %d 💎", tokens))
	}
	message := strings.Join(messageStrings, "\n")
	b.SendMessage(message, b.chatID, &opts)

	return func(update *echotron.Update) stateFn {
		switch update.CallbackQuery.Data {
		case "parent_tasks":
			{
				return b.handleTasks(update)
			}
		case "parent_rewards":
			{
				// return b.handleRewards
				return b.handleStart
			}
		}
		return b.handleParent
	}
}

func (b *bot) handleTasks(update *echotron.Update) stateFn {
	s := TGShowCommonsListStrings{
		Common:   "tasks",
		Empty:    "Список заданий пуст",
		Many:     "Список заданий",
		Question: "Что делаем с заданиями?",
	}

	message, err := tgShowCommonsList(b.db, s, b.user.Family_UID)
	if err != nil {
		slog.Error(err.Error())
		b.SendMessage("Internal error", b.chatID, nil)
	}
	opts := echotron.MessageOptions{
		ReplyMarkup: &echotron.InlineKeyboardMarkup{
			InlineKeyboard: [][]echotron.InlineKeyboardButton{
				{
					{
						Text:         "Добавить",
						CallbackData: "parent_tasks_add",
					},
					{
						Text:         "Удалить",
						CallbackData: "parent_tasks_delete",
					},
				},
				{
					{
						Text:         "Изменить награду",
						CallbackData: "parent_tasks_edit",
					},
					{
						Text:         "Назад",
						CallbackData: "parent_tasks_back",
					},
				},
			},
		},
	}
	b.SendMessage(message, b.chatID, &opts)

	return func(update *echotron.Update) stateFn {
		switch update.CallbackQuery.Data {
		case "parent_tasks_add":
			{
				return b.handleParentTasksAdd
			}
		case "parent_tasks_edit":
			{
				// return b.handleParentTasksEdit
				return b.handleStart
			}
		case "parent_tasks_delete":
			{
				// return b.handleParentTasksDelete
				return b.handleStart
			}
		case "parent_tasks_back":
			{
				return b.handleParent
			}
		}
		return b.handleTasks
	}
}

func (b *bot) handleParentTasksAdd(update *echotron.Update) stateFn {
	b.SendMessage("Введи название задания", b.chatID, nil)
	task_name := update.Message.Text

	b.SendMessage("Сколько токенов за это задание?", b.chatID, nil)
	tokens, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		slog.Error(err.Error())
		b.SendMessage("Количество токенов должно быть числом", b.chatID, nil)
	}

	err = dbCommonsAdd("tasks", b.db, b.user.Family_UID, task_name, tokens)
	if err != nil {
		slog.Error(err.Error())
		b.SendMessage("Internal error", b.chatID, nil)
	}
	return b.handleTasks
}

// func tgBot(db *sql.DB) (*tele.Bot, error) {
// 	var (
// 		selectorRole  = &tele.ReplyMarkup{}
// 		btnRoleParent = selectorRole.Data("Я родитель", "register_parent")
// 		btnRoleChild  = selectorRole.Data("Я ребенок", "register_child")

// 		selectorFamily = &tele.ReplyMarkup{}
// 		btnFamilyNew   = selectorFamily.Data("Создать", "family_new")
// 		btnFamilyJoin  = selectorFamily.Data("Присоединиться", "family_join")

// 		selectorParent   = &tele.ReplyMarkup{}
// 		btnParentChecks  = selectorParent.Data("Проверить сделанное", "parent_tasks_check")
// 		btnParentTasks   = selectorParent.Data("Задания", "parent_tasks")
// 		btnParentRewards = selectorParent.Data("Награды", "parent_rewards")
// 		btnParentHistory = selectorParent.Data("История", "parent_history")

// 		selectorParentTasks  = &tele.ReplyMarkup{}
// 		btnParentTasksAdd    = selectorParentTasks.Data("Добавить", "parent_tasks_add")
// 		btnParentTasksDelete = selectorParentTasks.Data("Удалить", "parent_tasks_delete")
// 		btnParentTasksEdit   = selectorParentTasks.Data("Изменить награду", "parent_tasks_edit")
// 		btnParentTasksBack   = selectorParentTasks.Data("Назад", "parent_tasks_back")

// 		selectorParentRewards  = &tele.ReplyMarkup{}
// 		btnParentRewardsAdd    = selectorParentRewards.Data("Добавить", "parent_rewards_add")
// 		btnParentRewardsEdit   = selectorParentRewards.Data("Изменить цену", "parent_rewards_edit")
// 		btnParentRewardsDelete = selectorParentRewards.Data("Удалить", "parent_rewards_delete")
// 		btnParentRewardsBack   = selectorParentRewards.Data("Назад", "parent_rewards_back")

// 		selectorParentChecks   = &tele.ReplyMarkup{}
// 		btnParentChecksApprove = selectorParentChecks.Data("Подтвердить ✅", "parent_check_approve")
// 		btnParentChecksReject  = selectorParentChecks.Data("Отклонить ❌", "parent_check_reject")
// 		btnParentChecksBack    = selectorParentChecks.Data("Назад ⬅️", "parent_check_back")

// 		selectorParentHistory = &tele.ReplyMarkup{}
// 		btnParentHistoryBack  = selectorParentHistory.Data("Назад", "parent_history_back")

// 		selectorChild   = &tele.ReplyMarkup{}
// 		btnChildTasks   = selectorChild.Data("Задания", "child_tasks")
// 		btnChildRewards = selectorChild.Data("Награды", "child_rewards")

// 		selectorChildTasks = &tele.ReplyMarkup{}
// 		btnChildTasksDone  = selectorChildTasks.Data("Выполнить", "child_tasks_done")
// 		btnChildTasksBack  = selectorChildTasks.Data("Назад", "child_tasks_back")

// 		selectorChildRewards = &tele.ReplyMarkup{}
// 		btnChildRewardsClaim = selectorChildRewards.Data("Получить", "child_rewards_claim")
// 		btnChildRewardsBack  = selectorChildRewards.Data("Назад", "child_rewards_back")
// 	)
// 	selectorRole.Inline(
// 		selectorRole.Row(btnRoleParent, btnRoleChild),
// 	)
// 	selectorFamily.Inline(
// 		selectorFamily.Row(btnFamilyNew, btnFamilyJoin),
// 	)
// 	selectorParent.Inline(
// 		selectorParent.Row(btnParentChecks),
// 		selectorParent.Row(
// 			// btnParentHistory,
// 			btnParentTasks,
// 			btnParentRewards,
// 		),
// 	)
// 	selectorParentTasks.Inline(
// 		selectorParentTasks.Row(btnParentTasksAdd, btnParentTasksDelete),
// 		selectorParentTasks.Row(btnParentTasksEdit, btnParentTasksBack),
// 	)
// 	selectorParentRewards.Inline(
// 		selectorParentRewards.Row(
// 			btnParentRewardsAdd,
// 			btnParentRewardsEdit,
// 			btnParentRewardsDelete,
// 		),
// 		selectorParentRewards.Row(btnParentRewardsBack),
// 	)
// 	selectorParentChecks.Inline(
// 		selectorParentChecks.Row(btnParentChecksApprove, btnParentChecksReject),
// 		selectorParentChecks.Row(btnParentChecksBack),
// 	)
// 	selectorParentHistory.Inline(
// 		selectorParentHistory.Row(btnParentHistoryBack),
// 	)

// 	selectorChild.Inline(
// 		selectorChild.Row(btnChildTasks, btnChildRewards),
// 	)
// 	selectorChildTasks.Inline(
// 		selectorChildTasks.Row(btnChildTasksDone, btnChildTasksBack),
// 	)
// 	selectorChildRewards.Inline(
// 		selectorChildRewards.Row(btnChildRewardsClaim, btnChildRewardsBack),
// 	)

// 	slog.Info("Configuring bot...")
// 	pref := tele.Settings{
// 		Token:  os.Getenv("TOKEN"),
// 		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
// 	}

// 	b, err := tele.NewBot(pref)
// 	if err != nil {
// 		return nil, err
// 	}

// 	slog.Info("Configuring handlers...")

// 	// start command
// 	b.Handle("/start", func(c tele.Context) error {
// 		slog.Info("Received /start command")
// 		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
// 	})

// 	// new parent
// 	b.Handle(&btnRoleParent, func(c tele.Context) error {
// 		slog.Info("Received register parent command")
// 		err = dbUserRegister(db, DBRecordUser{
// 			Tg_ID: c.Sender().ID,
// 			Role:  "parent",
// 		})
// 		if err != nil {
// 			slog.Error(err.Error())
// 			return c.Send("Internal error")
// 		}
// 		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
// 	})

// 	// new child
// 	b.Handle(&btnRoleChild, func(c tele.Context) error {
// 		slog.Info("Received register child command")
// 		err = dbUserRegister(db, DBRecordUser{
// 			Tg_ID: c.Sender().ID,
// 			Role:  "child",
// 		})
// 		if err != nil {
// 			slog.Error(err.Error())
// 			return c.Send("Internal error")
// 		}
// 		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
// 	})

// 	// new family
// 	b.Handle(&btnFamilyNew, func(c tele.Context) error {
// 		slog.Info("Received create family command")
// 		err = dbFamilyCreate(db, c.Sender().ID)
// 		if err != nil {
// 			slog.Error(err.Error())
// 			return c.Send("Internal error")
// 		}
// 		return tgHandleStart(c, db, selectorRole, selectorFamily, selectorParent, selectorChild)
// 	})

// 	// join family
// 	b.Handle(&btnFamilyJoin, func(c tele.Context) error {
// 		slog.Info("Received join family command")

// 		err = dbUsersSetTextInput(
// 			db,
// 			c.Sender().ID,
// 			DBTextInput{
// 				For: sql.NullString{String: "family_id", Valid: true},
// 			},
// 		)
// 		if err != nil {
// 			slog.Error(err.Error())
// 			return c.Send("Internal error")
// 		}
// 		return c.Send("Введите ID семьи, к которой хотите присоединиться")
// 	})

// 	// text message
// 	b.Handle(tele.OnText, func(c tele.Context) error {
// 		slog.Info("Received text message")
// 		return tgHandleText(
// 			c, db, b,
// 			selectorParentTasks,
// 			selectorParentRewards,
// 			selectorParentChecks,
// 			selectorParent,
// 			selectorChild,
// 			selectorChildTasks,
// 			selectorChildRewards,
// 		)
// 	})

// 	// parent tasks
// 	b.Handle(&btnParentTasks, func(c tele.Context) error {
// 		slog.Info("Received parent tasks command")
// 		return tgHandleTasks(c, db, selectorParentTasks)
// 	})
// 	// parent rewards
// 	b.Handle(&btnParentRewards, func(c tele.Context) error {
// 		slog.Info("Received parent rewards command")
// 		return tgHandleRewards(c, db, selectorParentRewards)
// 	})
// 	// parent history
// 	b.Handle(&btnParentHistory, func(c tele.Context) error {
// 		slog.Info("Received parent history command")
// 		return c.Send("История тут когда-нибудь будет", selectorParentHistory)
// 	})
// 	// parent checks
// 	b.Handle(&btnParentChecks, func(c tele.Context) error {
// 		slog.Info("Received parent check command")
// 		return c.Send("Проверить сделанное")
// 	})
// 	// parent tasks add
// 	b.Handle(&btnParentTasksAdd, func(c tele.Context) error {
// 		slog.Info("Received parent tasks add command")
// 		return tgHandleParentTasksAdd(c, db)
// 	})
// 	// parent tasks edit
// 	b.Handle(&btnParentTasksEdit, func(c tele.Context) error {
// 		slog.Info("Received parent tasks edit command")
// 		return tgHandleParentTasksEdit(c, db)
// 	})
// 	// parent tasks delete
// 	b.Handle(&btnParentTasksDelete, func(c tele.Context) error {
// 		slog.Info("Received parent tasks delete command")
// 		return tgHandleParentTasksDelete(c, db)
// 	})
// 	// parent tasks back
// 	b.Handle(&btnParentTasksBack, func(c tele.Context) error {
// 		slog.Info("Received parent tasks back command")
// 		return c.Send("Главное меню", selectorParent)
// 	})
// 	// parent rewards add
// 	b.Handle(&btnParentRewardsAdd, func(c tele.Context) error {
// 		slog.Info("Received parent rewards add command")
// 		return tgHandleParentRewardsAdd(c, db)
// 	})
// 	// parent rewards edit
// 	b.Handle(&btnParentRewardsEdit, func(c tele.Context) error {
// 		slog.Info("Received parent rewards edit command")
// 		return tgHandleParentRewardsEdit(c, db)
// 	})
// 	// parent rewards delete
// 	b.Handle(&btnParentRewardsDelete, func(c tele.Context) error {
// 		slog.Info("Received parent rewards delete command")
// 		return tgHandleParentRewardsDelete(c, db)
// 	})
// 	// parent rewards back
// 	b.Handle(&btnParentRewardsBack, func(c tele.Context) error {
// 		slog.Info("Received parent rewards back command")
// 		return c.Send("Главное меню", selectorParent)
// 	})
// 	// parent checks approve
// 	b.Handle(&btnParentChecksApprove, func(c tele.Context) error {
// 		slog.Info("Received parent checks approve command")
// 		return c.Send("Подтвердить")
// 	})
// 	// parent checks reject
// 	b.Handle(&btnParentChecksReject, func(c tele.Context) error {
// 		slog.Info("Received parent checks reject command")
// 		return c.Send("Отклонить")
// 	})
// 	// parent checks back
// 	b.Handle(&btnParentChecksBack, func(c tele.Context) error {
// 		slog.Info("Received parent checks back command")
// 		return c.Send(selectorParent)
// 	})
// 	// parent history back
// 	b.Handle(&btnParentHistoryBack, func(c tele.Context) error {
// 		slog.Info("Received parent history back command")
// 		return c.Send("Главное меню", selectorParent)
// 	})

// 	// child tasks
// 	b.Handle(&btnChildTasks, func(c tele.Context) error {
// 		slog.Info("Received child tasks command")
// 		return tgHandleTasks(c, db, selectorChildTasks)
// 	})
// 	// child rewards
// 	b.Handle(&btnChildRewards, func(c tele.Context) error {
// 		slog.Info("Received child rewards command")
// 		return tgHandleRewards(c, db, selectorChildRewards)
// 	})
// 	// child tasks done
// 	b.Handle(&btnChildTasksDone, func(c tele.Context) error {
// 		slog.Info("Received child tasks done command")
// 		return tgHandleChildTasksDone(c, db)
// 	})
// 	// child tasks back
// 	b.Handle(&btnChildTasksBack, func(c tele.Context) error {
// 		slog.Info("Received child tasks back command")
// 		return tgHandleChild(c, db, selectorChild)
// 	})
// 	// child rewards claim
// 	b.Handle(&btnChildRewardsClaim, func(c tele.Context) error {
// 		slog.Info("Received child rewards claim command")
// 		return tgHandleChildRewardsClaim(c, db)
// 	})
// 	// child rewards back
// 	b.Handle(&btnChildRewardsBack, func(c tele.Context) error {
// 		slog.Info("Received child rewards back command")
// 		return tgHandleChild(c, db, selectorChild)
// 	})

// 	return b, err
// }

// // rewards
// func tgHandleRewards(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
// 	s := TGShowCommonsListStrings{
// 		Common:   "rewards",
// 		Empty:    "Список наград пуст",
// 		Many:     "Список наград",
// 		Question: "Что делаем с наградами?",
// 	}
// 	return tgHandleCommmons(c, db, selector, s)
// }

// func tgHandleCommmons(
// 	c tele.Context,
// 	db *sql.DB,
// 	selector *tele.ReplyMarkup,
// 	s TGShowCommonsListStrings,
// ) error {
// 	user, err := dbUserFind(db, c.Sender().ID)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Internal error")
// 	}
// 	return tgShowCommonsList(c, db, selector, s, user)
// }

// func tgHandleTextFamilyID(
// 	c tele.Context,
// 	db *sql.DB,
// 	selectorParent *tele.ReplyMarkup,
// 	selectorChild *tele.ReplyMarkup,
// ) error {
// 	tgID := c.Sender().ID

// 	err := dbFamilyJoin(db, DBRecordUser{
// 		Tg_ID:      tgID,
// 		Family_UID: sql.NullString{String: c.Text(), Valid: true},
// 	})
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Семьи с таким ID не найдено. Убедись, что ввел правильный ID.")
// 	}

// 	err = dbUsersSetTextInput(db, tgID, DBTextInput{
// 		sql.NullString{Valid: false},
// 		sql.NullString{Valid: false},
// 	})
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return err
// 	}

// 	// find user by ID in the database
// 	record, err := dbUserFind(db, tgID)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Internal error")
// 	}

// 	switch record.Role {
// 	case "parent":
// 		return c.Send("Добро пожаловать в семью! Начнем?", selectorParent)
// 	}
// 	// case "child":
// 	return c.Send("Добро пожаловать в семью! Начнем?", selectorChild)
// }

// // handle text message input
// func tgHandleText(
// 	c tele.Context,
// 	db *sql.DB,
// 	b *tele.Bot,
// 	selectorParentTasks *tele.ReplyMarkup,
// 	selectorParentRewards *tele.ReplyMarkup,
// 	selectorParentChecks *tele.ReplyMarkup,
// 	selectorParent *tele.ReplyMarkup,
// 	selectorChild *tele.ReplyMarkup,
// 	selectorChildTasks *tele.ReplyMarkup,
// 	selectorChildRewards *tele.ReplyMarkup,
// ) error {
// 	// to distinguish what is the type of input we've got
// 	// we need to get explanation from the database
// 	textInput, err := dbUsersGetTextInput(db, c.Sender().ID)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Internal error")
// 	}

// 	// based on the explanation we can handle the input
// 	switch textInput.For.String {
// 	// the input is family ID number. We need to join the family
// 	case "family_id":
// 		return tgHandleTextFamilyID(c, db, selectorParent, selectorChild)
// 		// the input is task name. We need it to add new task
// 	case "parent_tasks_add_name":
// 		{
// 			return tgHandleTextParentTasksAddName(c, db, selectorParentTasks)
// 		}
// 		// the input is reward name. We need it to add new reward
// 	case "parent_rewards_add_name":
// 		{
// 			return tgHandleTextParentRewardsAddName(c, db, selectorParentRewards)
// 		}
// 		// the input is token number for task. We need it to add new task
// 	case "parent_tasks_add_tokens":
// 		{
// 			return tgHandleTextParentTasksAddTokens(c, db, selectorParentTasks, textInput)
// 		}
// 		// the input is token number for reward. We need it to add new reward
// 	case "parent_rewards_add_tokens":
// 		{
// 			return tgHandleTextParentRewardsAddTokens(c, db, selectorParentRewards, textInput)
// 		}
// 		// the input is task number. We need it to edit task
// 	case "parent_tasks_edit_name":
// 		{
// 			return tgHandleTextParentTasksEditName(c, db)
// 		}
// 		// the input is reward number. We need it to edit reward
// 	case "parent_rewards_edit_name":
// 		{
// 			return tgHandleTextParentRewardsEditName(c, db)
// 		}
// 		// the input is token number for task. We need it to edit task
// 	case "parent_tasks_edit_tokens":
// 		{
// 			return tgHandleTextParentTasksEditTokens(c, db, selectorParentTasks, textInput)
// 		}
// 		// the input is token number for reward. We need it to edit reward
// 	case "parent_rewards_edit_tokens":
// 		{
// 			return tgHandleTextParentRewardsEditTokens(c, db, selectorParentRewards, textInput)
// 		}
// 		// the input is task number. We need it to delete task
// 	case "parent_tasks_delete_name":
// 		{
// 			return tgHandleParentTasksDeleteName(c, db, selectorParentTasks)
// 		}
// 		// the input is reward number. We need it to delete reward
// 	case "parent_rewards_delete_name":
// 		{
// 			return tgHandleParentRewardsDeleteName(c, db, selectorParentRewards)
// 		}
// 		// the input is task number. We need it to mark task as done
// 	case "child_tasks_done":
// 		{
// 			return tgHandleTextChildTasksDone(c, db, selectorChildTasks, selectorParentChecks, b)
// 		}
// 		// the input is reward number. We need it to claim reward
// 	case "child_rewards_claim":
// 		{
// 			return tgHandleTextChildRewardsClaim(c, db, selectorChildRewards, b)
// 		}
// 	}
// 	return c.Send("Не понимаю, что ты хочешь от меня 😕")
// }

// // register new user, family, show main menu
// func tgHandleStart(
// 	c tele.Context,
// 	db *sql.DB,
// 	selectorRole *tele.ReplyMarkup,
// 	selectorFamily *tele.ReplyMarkup,
// 	selectorParent *tele.ReplyMarkup,
// 	selectorChild *tele.ReplyMarkup,
// ) error {
// 	var user = c.Sender()
// 	username := tgGetUsername(user)

// 	// find user by ID in the database
// 	record, err := dbUserFind(db, user.ID)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Internal error")
// 	}

// 	// check that user record is empty
// 	// it means that user is not registered yet
// 	if record.Role == "" {
// 		slog.Info("User not found. Select role to create...")
// 		return c.Send(
// 			fmt.Sprintf("Вижу, мы еще не знакомы. Кто ты, %s? 😊", username),
// 			selectorRole,
// 		)
// 	}

// 	// check that family ID is empty
// 	// it means that user is not in the family yet
// 	if !record.Family_UID.Valid {
// 		slog.Info("User family not found. Managing family...")
// 		return c.Send(
// 			"Ты еще не добавлен в семью. Создать новую или присоединиться к существующей?",
// 			selectorFamily,
// 		)
// 	}

// 	slog.Info("Sending greeting message...")
// 	err = c.Send(fmt.Sprintf(
// 		`Привет, %s\! 😊 ID твоей семьи *%s*

// Его нужно будет ввести остальным членам твоей семьи,
// когда они начнут пользоваться ботом`,
// 		username,
// 		record.Family_UID.String),
// 		&tele.SendOptions{ParseMode: tele.ModeMarkdownV2},
// 	)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return err
// 	}

// 	switch record.Role {
// 	case "parent":
// 		return c.Send("Начнем?", selectorParent)
// 	}
// 	// case "child":
// 	return c.Send("Начнем?", selectorChild)
// }

// // Action on task or reward without text input
// // input strings
// type TGHandleCommonsActionStrings struct {
// 	NextHandler string // "parent_tasks_add_name"
// 	Question    string // "Как назвать новое задание?"
// }

// // prepare bot to handle next next input from this user
// func tgHandleCommonsAction(
// 	c tele.Context,
// 	db *sql.DB,
// 	s TGHandleCommonsActionStrings,
// ) error {
// 	err := dbUsersSetTextInput(
// 		db,
// 		c.Sender().ID,
// 		DBTextInput{
// 			For: sql.NullString{String: s.NextHandler, Valid: true},
// 		},
// 	)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Internal error")
// 	}

// 	return c.Send(s.Question)
// }

// //

// // Action on task or reward by its number
// type TGHandleTextCommonsActionOnNumberFunc func(
// 	arg string, db *sql.DB, user DBRecordUser, name string,
// ) (int, error)

// // input strings
// type TGHandleTextCommonsActionOnNumberStrings struct {
// 	Common           string // 'task' or 'reward'
// 	ShouldBeNumber   string // "Номер таски/награды должен быть числом"
// 	NoWithThisNumber string // "Нет таски/награды с таким номером"
// 	ActionDone       string // "Действие выполнено!"
// 	Arg              string // "check"
// }

// // do some things with task or reward by its number provided by user as input
// func tgHandleTextCommonsActionOnNumber(
// 	c tele.Context,
// 	db *sql.DB,
// 	selector *tele.ReplyMarkup,
// 	s TGHandleTextCommonsActionOnNumberStrings,
// 	f TGHandleTextCommonsActionOnNumberFunc,
// ) (DBRecordUser, string, int, error) {
// 	// convert text to number
// 	record_n, err := strconv.Atoi(c.Text())
// 	if err != nil {
// 		return DBRecordUser{}, "", 0, c.Send(s.ShouldBeNumber)
// 	}

// 	// get user by ID
// 	user, err := dbUserFind(db, c.Sender().ID)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return user, "", 0, c.Send("Internal error")
// 	}

// 	// get list of tasks or rewards
// 	records, err := dbCommonsList(s.Common, db, user.Family_UID.String)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return user, "", 0, c.Send("Internal error")
// 	}

// 	// check if task or reward number is in valid range
// 	if record_n < 1 || record_n > len(records) {
// 		return user, "", 0, c.Send(s.NoWithThisNumber)
// 	}

// 	// do some action with task or reward (delete)
// 	record_name := records[record_n-1].Name
// 	arg := s.Arg
// 	if arg == "" {
// 		arg = s.Common
// 	}
// 	res, err := f(
// 		arg, db, user, record_name,
// 	)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return user, record_name, res, err
// 	}

// 	// send message about successful action
// 	err = c.Send(s.ActionDone)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return user, record_name, res, err
// 	}

// 	// show list of tasks or rewards to do other actions
// 	if s.Common == "tasks" {
// 		return user, record_name, res, tgHandleTasks(c, db, selector)
// 	}
// 	// case "rewards":
// 	return user, record_name, res, tgHandleRewards(c, db, selector)
// }

// //

// func tgNotifyParents(
// 	c tele.Context,
// 	db *sql.DB,
// 	b *tele.Bot,
// 	user DBRecordUser,
// 	message string,
// 	selector *tele.ReplyMarkup,
// ) error {
// 	return tgNotifyUsers(c, db, b, user, "parent", message, selector)
// }

// func tgNotifyChilds(
// 	c tele.Context,
// 	db *sql.DB,
// 	b *tele.Bot,
// 	user DBRecordUser,
// 	message string,
// 	selector *tele.ReplyMarkup,
// ) error {
// 	return tgNotifyUsers(c, db, b, user, "child", message, selector)
// }

// func tgNotifyUsers(
// 	c tele.Context,
// 	db *sql.DB,
// 	b *tele.Bot,
// 	user DBRecordUser,
// 	notifyRole string,
// 	message string,
// 	selector *tele.ReplyMarkup,
// ) error {
// 	// get users with roles for user's family
// 	users, err := dbUsersGet(db, user.Family_UID, notifyRole)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return c.Send("Internal error")
// 	}
// 	// send notification to users
// 	for _, user_ := range users {
// 		if user_ == user.Tg_ID {
// 			continue
// 		}
// 		slog.Info(fmt.Sprintf("Sending notification to %s: %d", notifyRole, user_))
// 		recipient := &tele.User{ID: user_}

// 		_, err = b.Send(recipient, message, selector)
// 		if err != nil {
// 			slog.Error(err.Error())
// 			return c.Send("Internal error")
// 		}
// 	}
// 	return nil
// }
