package main

import (
	"database/sql"
	"fmt"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// Спрашиваем у ребенка, какое задание отметить выполненным
func tgHandleChildTasksDone(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("child_tasks_done", "Какое задание отметить выполненным? Введи номер", c, db)
}

func tgHandleTextChildTasksDone(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	return nil
	// task_n, err := strconv.Atoi(c.Text())
	// if err != nil {
	// 	return c.Send("Номер задания должен быть числом")
	// }

	// user, err := dbUserFind(db, c.Sender().ID)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	return c.Send("Internal error")
	// }

	// tasks, err := dbCommonsList("tasks", db, user.Family_UID.String)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	return c.Send("Internal error")
	// }

	// if task_n < 1 || task_n > len(tasks) {
	// 	return c.Send("Нет задания с таким номером")
	// }

	// task_name := tasks[task_n-1].Name
	// err = dbCommonsDelete("tasks", db, user.Family_UID.String, task_name)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	return err
	// }

	// err = dbTokenAdd(db, user.Family_UID.String, tasks[task_n-1].Tokens)
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	return err
	// }

	// return c.Send("Задание выполнено!", selector)
}

func tgHandleChildRewardsClaim(c tele.Context, db *sql.DB) error {
	return tgHandleParentCommonsAction("child_rewards_claim", "Какую награду получить? Введи номер", c, db)
}

func tgHandleTextChildRewardsClaim(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	return nil
}

// Главное меню для ребенка
func tgHandleChild(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	tokens, err := dbTokenGet(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}
	return c.Send(
		fmt.Sprintf("Твой баланс: %d 💎", tokens),
		selector,
	)
}
