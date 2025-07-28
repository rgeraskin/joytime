package main

import (
	"database/sql"
	"log/slog"
	"strconv"

	tele "gopkg.in/telebot.v4"
)

// Задания
// Добавляем задания
func tgHandleParentTasksAdd(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "parent_tasks_add_name",
		Question:    "Какое задание добавить? Введи название",
	}
	return tgHandleCommonsAction(c, db, s)
}
func tgHandleTextParentTasksAddName(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	s := TGHandleTextParentCommonsAddNameStrings{
		Common:        "tasks",
		AlreadyExists: "Задание с таким именем уже существует",
		NextQuestion:  "Во сколько жетонов оценить задание?",
		NextHandler:   "parent_tasks_add_tokens",
	}
	return tgHandleTextParentCommonsAddName(c, db, selector, s)
}
func tgHandleTextParentTasksAddTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	textInput DBTextInput,
) error {
	s := TGHandleTextParentCommonsChangeTokensStrings{
		Common:         "tasks",
		ShouldBeNumber: "Количество жетонов должно быть числом",
		Changed:        "Задание добавлено!",
	}
	return tgHandleTextParentCommonsAddTokens(c, db, selector, s, textInput)
}

// Редактируем задания
func tgHandleParentTasksEdit(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "parent_tasks_edit_name",
		Question:    "Какое задание редактировать? Введи номер",
	}
	return tgHandleCommonsAction(c, db, s)
}
func tgHandleTextParentTasksEditName(c tele.Context, db *sql.DB) error {
	s := TGHandleTextParentCommonsEditNameStrings{
		Common:           "tasks",
		ShouldBeNumber:   "Номер задания должен быть числом",
		NoWithThisNumber: "Нет задания с таким номером",
		WhatPrice:        "Сколько будет стоить задание?",
		NextHandler:      "parent_tasks_edit_tokens",
	}
	return tgHandleTextParentCommonsEditName(c, db, s)
}
func tgHandleTextParentTasksEditTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	textInput DBTextInput,
) error {
	s := TGHandleTextParentCommonsChangeTokensStrings{
		Common:         "tasks",
		ShouldBeNumber: "Количество жетонов должно быть числом",
		Changed:        "Задание изменено!",
	}
	return tgHandleTextParentCommonsEditTokens(c, db, selector, s, textInput)
}

// Удаляем задания
// попросим ввести номер задания для удаления
func tgHandleParentTasksDelete(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "parent_tasks_delete_name",
		Question:    "Какое задание удалить? Введи номер",
	}
	return tgHandleCommonsAction(c, db, s)
}

// удалим задание введенному по номеру
func tgHandleParentTasksDeleteName(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
) error {
	s := TGHandleTextCommonsActionOnNumberStrings{
		Common:           "tasks",
		ShouldBeNumber:   "Номер задания должен быть числом",
		NoWithThisNumber: "Нет задания с таким номером",
		ActionDone:       "Задание удалено!",
	}
	_, _, _, err := tgHandleTextCommonsActionOnNumber(c, db, selector, s, dbCommonsDelete)
	return err
}

//

// Награды
// Добавляем награды
func tgHandleParentRewardsAdd(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "parent_rewards_add_name",
		Question:    "Какую награду добавить? Введи название",
	}
	return tgHandleCommonsAction(c, db, s)
}

func tgHandleTextParentRewardsAddName(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
) error {
	s := TGHandleTextParentCommonsAddNameStrings{
		Common:        "rewards",
		AlreadyExists: "Награда с таким именем уже существует",
		NextQuestion:  "Сколько жетонов стоит награда?",
		NextHandler:   "parent_rewards_add_tokens",
	}
	return tgHandleTextParentCommonsAddName(c, db, selector, s)
}
func tgHandleTextParentRewardsAddTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	textInput DBTextInput,
) error {
	s := TGHandleTextParentCommonsChangeTokensStrings{
		Common:         "rewards",
		ShouldBeNumber: "Количество жетонов должно быть числом",
		Changed:        "Награда добавлена!",
	}
	return tgHandleTextParentCommonsAddTokens(c, db, selector, s, textInput)
}

// Редактируем награды
func tgHandleParentRewardsEdit(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "parent_rewards_edit_name",
		Question:    "Какую награду редактировать? Введи номер",
	}
	return tgHandleCommonsAction(c, db, s)
}
func tgHandleTextParentRewardsEditName(c tele.Context, db *sql.DB) error {
	s := TGHandleTextParentCommonsEditNameStrings{
		Common:           "rewards",
		ShouldBeNumber:   "Номер награды должен быть числом",
		NoWithThisNumber: "Нет награды с таким номером",
		WhatPrice:        "Сколько будет стоить награда?",
		NextHandler:      "parent_rewards_edit_tokens",
	}
	return tgHandleTextParentCommonsEditName(c, db, s)
}
func tgHandleTextParentRewardsEditTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	textInput DBTextInput,
) error {
	s := TGHandleTextParentCommonsChangeTokensStrings{
		Common:         "rewards",
		ShouldBeNumber: "Количество жетонов должно быть числом",
		Changed:        "Награда изменена!",
	}
	return tgHandleTextParentCommonsEditTokens(c, db, selector, s, textInput)
}

// Удаляем награды
// попросим ввести номер награды для удаления
func tgHandleParentRewardsDelete(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "parent_rewards_delete_name",
		Question:    "Какую награду удалить? Введи номер",
	}
	return tgHandleCommonsAction(c, db, s)
}

// удалим награду введенному по номеру
func tgHandleParentRewardsDeleteName(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
) error {
	s := TGHandleTextCommonsActionOnNumberStrings{
		Common:           "rewards",
		ShouldBeNumber:   "Нет награды с таким номером",
		NoWithThisNumber: "Номер награды должен быть числом",
		ActionDone:       "Награда удалена!",
	}
	_, _, _, err := tgHandleTextCommonsActionOnNumber(c, db, selector, s, dbCommonsDelete)
	return err
}

//

// Задания или награды
// Add new task or reward
// input strings
type TGHandleTextParentCommonsAddNameStrings struct {
	Common        string // 'task' or 'reward'
	AlreadyExists string // "Такое задание/награда уже существует"
	NextQuestion  string // "Сколько будет стоить задание/награда?"
	NextHandler   string // "parent_tasks_add_tokens"
}

// check that task or reward with this name is not already exists
// if not exists, prepare bot to handle next input from this user
// storing task or reward name in the database
func tgHandleTextParentCommonsAddName(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	s TGHandleTextParentCommonsAddNameStrings,
) error {
	// get user by ID
	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	// get task or reward by name
	// Should get sql.ErrNoRows for new task or reward
	record, err := dbCommonsGet(s.Common, db, user.Family_UID.String, c.Text())
	if err != nil && err != sql.ErrNoRows {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}
	// if task or reward with this name already exists
	if record.Name != "" {
		return c.Send(s.AlreadyExists, selector)
	}

	// prepare bot to handle next input from this user
	err = dbUsersSetTextInput(db, c.Sender().ID, DBTextInput{
		sql.NullString{String: s.NextHandler, Valid: true},
		sql.NullString{String: c.Text(), Valid: true},
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// ask user to provide next text input
	return c.Send(s.NextQuestion)
}

func tgHandleTextParentCommonsAddTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	s TGHandleTextParentCommonsChangeTokensStrings,
	textInput DBTextInput,
) error {
	return tgHandleTextParentCommonsChangeTokens(
		c, db, selector, s, textInput, dbCommonsAdd,
	)
}

// Edit existing task or reward
// input strings
type TGHandleTextParentCommonsEditNameStrings struct {
	Common           string // 'task' or 'reward'
	ShouldBeNumber   string // "Номер задания должен быть числом"
	NoWithThisNumber string // "Нет задания с таким номером"
	WhatPrice        string // "Сколько будет стоить задание/награда?"
	NextHandler      string // "parent_tasks_edit_tokens"
}

// check that task or reward with this number exists
// if exists, prepare bot to handle next input from this user
// storing task or reward name in the database
func tgHandleTextParentCommonsEditName(
	c tele.Context,
	db *sql.DB,
	s TGHandleTextParentCommonsEditNameStrings,
) error {
	// convert text to number
	record_n, err := strconv.Atoi(c.Text())
	if err != nil {
		return c.Send(s.ShouldBeNumber)
	}

	// get user by ID
	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	// get list of tasks or rewards
	records, err := dbCommonsList(s.Common, db, user.Family_UID.String)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	// check that task or reward is in valid range
	if record_n < 1 || record_n > len(records) {
		return c.Send(s.NoWithThisNumber)
	}

	// prepare bot to handle next input from this user
	task_name := records[record_n-1].Name
	slog.Info(task_name)
	err = dbUsersSetTextInput(db, c.Sender().ID, DBTextInput{
		For: sql.NullString{String: s.NextHandler, Valid: true},
		Arg: sql.NullString{String: task_name, Valid: true},
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// ask user to provide next text input
	return c.Send(s.WhatPrice)
}

func tgHandleTextParentCommonsEditTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	s TGHandleTextParentCommonsChangeTokensStrings,
	textInput DBTextInput,
) error {
	return tgHandleTextParentCommonsChangeTokens(
		c, db, selector, s, textInput, dbCommonsEdit,
	)
}

type TGHandleTextParentCommonsChangeTokensFunc func(
	common string, db *sql.DB, familyID string, name string, tokens int,
) error
type TGHandleTextParentCommonsChangeTokensStrings struct {
	Common         string // 'task' or 'reward'
	ShouldBeNumber string // "Награда должна быть числом"
	Changed        string // "Задание изменено!"
}

// almost the same as tgHandleTextCommonsActionOnNumber
func tgHandleTextParentCommonsChangeTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	s TGHandleTextParentCommonsChangeTokensStrings,
	textInput DBTextInput,
	f TGHandleTextParentCommonsChangeTokensFunc,
) error {
	// convert text to number
	record, err := strconv.Atoi(c.Text())
	if err != nil {
		return c.Send(s.ShouldBeNumber)
	}

	// get user by ID
	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	// add/edit/etc... task or reward in the database
	err = f(
		s.Common, db, user.Family_UID.String, textInput.Arg.String, record,
	)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// send message about successful action
	err = c.Send(s.Changed)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// show list of tasks or rewards to do other actions
	if s.Common == "tasks" {
		return tgHandleTasks(c, db, selector)
	}
	// case "rewards":
	return tgHandleRewards(c, db, selector)
}
