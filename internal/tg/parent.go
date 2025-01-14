package main

import (
	"database/sql"
	"log/slog"
	"strconv"

	tele "gopkg.in/telebot.v4"
)

func tgHandleParentTasksAdd(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("parent_tasks_add_name", tgGetCommonsDiffs("tasks").questionAdd, c, db)
}
func tgHandleParentTasksDelete(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("parent_tasks_delete_name", tgGetCommonsDiffs("tasks").questionDelete, c, db)

}
func tgHandleParentTasksEdit(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("parent_tasks_edit_name", tgGetCommonsDiffs("tasks").questionEdit, c, db)
}

func tgHandleParentRewardsAdd(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("parent_rewards_add_name", tgGetCommonsDiffs("rewards").questionAdd, c, db)
}
func tgHandleParentRewardsDelete(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("parent_rewards_delete_name", tgGetCommonsDiffs("rewards").questionDelete, c, db)
}
func tgHandleParentRewardsEdit(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("parent_rewards_edit_name", tgGetCommonsDiffs("rewards").questionEdit, c, db)
}

func tgHandleTextParentCommonsAddName(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	common string,
	sAlreadyExists string,
	sNextQuestion string,
	nextHandler string,
) error {
	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	record, err := dbCommonsGet(common, db, user.Family_UID.String, c.Text())
	if err != nil && err != sql.ErrNoRows {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}
	if record.Name != "" {
		return c.Send(sAlreadyExists, selector)
	}

	err = dbUsersSetTextInput(db, c.Sender().ID, DBTextInput{
		sql.NullString{String: nextHandler, Valid: true},
		sql.NullString{String: c.Text(), Valid: true},
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return c.Send(sNextQuestion)
}

func tgHandleTextParentCommonsAddReward(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	textInput DBTextInput,
	common string,
	sShouldBeNumber string,
	sAdded string,
) error {
	reward, err := strconv.Atoi(c.Text())
	if err != nil {
		return c.Send(sShouldBeNumber)
	}

	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	err = dbCommonsAdd(common, db, user.Family_UID.String, textInput.Arg.String, reward)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	err = c.Send(sAdded)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return tgShowCommonsList(tgGetCommonsDiffs(common), user, selector, c, db)
}

func tgHandleTextParentCommonsEditName(
	c tele.Context,
	db *sql.DB,
	common string,
	sShouldBeNumber string,
	sNoWithThisNumber string,
	sWhatPrice string,
	nextHandler string,
) error {
	record_n, err := strconv.Atoi(c.Text())
	if err != nil {
		return c.Send(sShouldBeNumber)
	}

	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	records, err := dbCommonsList(common, db, user.Family_UID.String)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	if record_n < 1 || record_n > len(records) {
		return c.Send(sNoWithThisNumber)
	}

	task_name := records[record_n-1].Name
	slog.Info(task_name)
	err = dbUsersSetTextInput(db, c.Sender().ID, DBTextInput{
		For: sql.NullString{String: nextHandler, Valid: true},
		Arg: sql.NullString{String: task_name, Valid: true},
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return c.Send(sWhatPrice)
}

func tgHandleTextParentCommonsEditTokens(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	textInput DBTextInput,
	common string,
	sShouldBeNumber string,
	sChanged string,
) error {
	reward, err := strconv.Atoi(c.Text())
	if err != nil {
		return c.Send(sShouldBeNumber)
	}

	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	err = dbCommonsEdit(common, db, user.Family_UID.String, textInput.Arg.String, reward)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	err = c.Send(sChanged)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return tgShowCommonsList(tgGetCommonsDiffs(common), user, selector, c, db)
}

func tgHandleParentCommonsDeleteName(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	common string,
	sShouldBeNumber string,
	sNoWithThisNumber string,
	sDeleted string,
) error {
	user, err := dbUserFind(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	record_n, err := strconv.Atoi(c.Text())
	if err != nil {
		return c.Send(sShouldBeNumber)
	}

	records, err := dbCommonsList(common, db, user.Family_UID.String)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	if record_n < 1 || record_n > len(records) {
		return c.Send(sNoWithThisNumber)
	}

	record_name := records[record_n-1].Name
	err = dbCommonsDelete(common, db, user.Family_UID.String, record_name)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	err = c.Send(sDeleted)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return tgShowCommonsList(tgGetCommonsDiffs(common), user, selector, c, db)

}
