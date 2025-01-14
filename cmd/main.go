package main

// 2DO
// 1. ERR pq: duplicate key value violates unique constraint "unique_tg_id"
// 1. ERR pq: duplicate key value violates unique constraint "unique_family_id"

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"

	_ "github.com/lib/pq"
)

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, nil)))

	db, err := dbOpen()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	b, err := tgBot(db)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	b.Start()
}
