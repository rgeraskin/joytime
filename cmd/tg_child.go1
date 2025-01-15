package main

import (
	"database/sql"
	"fmt"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// Главное меню для ребенка
func tgHandleChild(c tele.Context, db *sql.DB, selector *tele.ReplyMarkup) error {
	tokens, err := dbTokensGet(db, c.Sender().ID)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}
	return c.Send(
		fmt.Sprintf("Твой баланс: %d 💎", tokens),
		selector,
	)
}

// Спрашиваем у ребенка, какое задание отметить выполненным
func tgHandleChildTasksDone(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "child_tasks_done",
		Question:    "Какое задание отметить выполненным? Введи номер",
	}
	return tgHandleCommonsAction(c, db, s)
}

// Меняем статус задания new => check
func tgHandleTextChildTasksDone(
	c tele.Context,
	db *sql.DB,
	selectorChildTasks *tele.ReplyMarkup,
	selectorParentChecks *tele.ReplyMarkup,
	b *tele.Bot,
) error {
	s := TGHandleTextCommonsActionOnNumberStrings{
		Common:           "tasks",
		ShouldBeNumber:   "Номер задания должен быть числом",
		NoWithThisNumber: "Нет задания с таким номером",
		ActionDone:       "Задание отмечено выполненным ✅\nПосле проверки родителем ты получишь 💎",
		Arg:              "check",
	}
	user, task_name, task_price, err := tgHandleTextCommonsActionOnNumber(
		c,
		db,
		selectorChildTasks,
		s,
		dbTasksStatusChange,
	)
	if err != nil {
		slog.Error(err.Error())
		return c.Send("Internal error")
	}

	message := fmt.Sprintf("Ребенок выполнил задание: %s за %d 💎", task_name, task_price)
	return tgNotifyParents(c, db, b, user, message, selectorParentChecks)
}

func tgHandleChildRewardsClaim(c tele.Context, db *sql.DB) error {
	s := TGHandleCommonsActionStrings{
		NextHandler: "child_rewards_claim",
		Question:    "Какую награду ты хочешь оплатить? Введи номер",
	}
	return tgHandleCommonsAction(c, db, s)
}

func tgHandleTextChildRewardsClaim(
	c tele.Context,
	db *sql.DB,
	selector *tele.ReplyMarkup,
	b *tele.Bot,
) error {
	s := TGHandleTextCommonsActionOnNumberStrings{
		Common:           "rewards",
		ShouldBeNumber:   "Номер награды должен быть числом",
		NoWithThisNumber: "Нет награды с таким номером 🙁",
		ActionDone:       "Награда куплена 🎉\nРодителю отправлено уведомление",
	}
	user, reward_name, reward_price, err := tgHandleTextCommonsActionOnNumber(
		c,
		db,
		selector,
		s,
		dbRewardsClaim,
	)
	if err != nil {
		if err.Error() == "not enough tokens" {
			// 2DO fix red error message
			return c.Send("У тебя недостаточно 💎")
		}
		slog.Error(err.Error())
		return c.Send("Internal error")
	}
	// send notification to parents
	message := fmt.Sprintf("Ребенок купил награду: %s за %d 💎", reward_name, reward_price)
	return tgNotifyParents(c, db, b, user, message, nil)
}
