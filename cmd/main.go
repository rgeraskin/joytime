package main

import (
	"log/slog"
	"os"

	"github.com/NicoNex/echotron/v3"

	"github.com/lmittmann/tint"

	_ "github.com/lib/pq"
)

func main() {

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelDebug,
		}),
	))

	db, err := dbOpen()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	dsp := echotron.NewDispatcher(os.Getenv("TOKEN"), func(chatID int64) echotron.Bot {
		return newBot(chatID, db) // Pass the database instance
	})
	slog.Info(dsp.Poll().Error())
}
