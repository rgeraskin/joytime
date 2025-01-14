package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"
)

const (
	dbFamilyUIDCharset = "abcdefghjkmnpqrstuvwxyz23456789"
	dbFamilyUIDLength  = 6
)

type DBTextInput struct {
	For sql.NullString
	Arg sql.NullString
}

type DBRecordUser struct {
	Tg_ID      int64
	Role       string
	Family_UID sql.NullString
	Created_at time.Time
}

type DBRecordCommon struct {
	Name   string
	Tokens int
}

func dbOpen() (*sql.DB, error) {
	slog.Info("Connecting to database...")
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		os.Getenv("PGUSER"),
		os.Getenv("PGPASSWORD"),
		os.Getenv("PGHOST"),
		os.Getenv("PGDATABASE"),
	)

	slog.Info(fmt.Sprintf("Connection string: %s", connStr))
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func dbUsersSetTextInput(db *sql.DB, tgID int64, textInput DBTextInput) error {
	var id int

	slog.Info(fmt.Sprintf(
		"Updating user %d with text_input for %s with arg '%s'...",
		tgID,
		textInput.For.String,
		textInput.Arg.String,
	))
	return db.QueryRow(
		"UPDATE users SET text_input_for = $1, text_input_arg = $2 WHERE tg_id = $3 RETURNING id",
		textInput.For,
		textInput.Arg,
		tgID,
	).Scan(&id)
}

func dbUsersGetTextInput(db *sql.DB, tgID int64) (DBTextInput, error) {
	var textInput DBTextInput

	slog.Info(fmt.Sprintf("Getting text_input for user %d...", tgID))
	err := db.QueryRow(
		"SELECT text_input_for, text_input_arg FROM users WHERE tg_id = $1",
		tgID,
	).Scan(&textInput.For, &textInput.Arg)

	return textInput, err
}

func dbFamilyJoin(db *sql.DB, user DBRecordUser) error {
	var id int

	slog.Info("Checking a family to join exists...")
	err := db.QueryRow(
		"SELECT id FROM families WHERE uid = $1",
		user.Family_UID,
	).Scan(&id)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf(
		"Updating user %d with family_uid %s...", user.Tg_ID, user.Family_UID.String,
	))
	return db.QueryRow(
		"UPDATE users SET family_uid = $1 WHERE tg_id = $2 RETURNING id",
		user.Family_UID,
		user.Tg_ID,
	).Scan(&id)
}

func dbFamilyCreate(db *sql.DB, userID int64) error {
	var id int

	// Generate a unique family UID
	familyUID := make([]byte, dbFamilyUIDLength)
	for i := range familyUID {
		familyUID[i] = dbFamilyUIDCharset[rand.Intn(len(dbFamilyUIDCharset))]
	}

	slog.Info(fmt.Sprintf("Creating a new family with uid %s...", familyUID))
	err := db.QueryRow(
		"INSERT INTO families(uid, created_by) VALUES($1, $2) RETURNING id",
		familyUID,
		userID,
	).Scan(&id)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Updating user %d with family_uid %s...", userID, familyUID))
	return db.QueryRow(
		"UPDATE users SET family_uid = $1 WHERE tg_id = $2 RETURNING id",
		familyUID,
		userID,
	).Scan(&id)
}

func dbUserRegister(db *sql.DB, user DBRecordUser) error {
	slog.Info(fmt.Sprintf("Registering user %d in the database...", user.Tg_ID))
	var id int
	err := db.QueryRow(
		"INSERT INTO users(tg_id, role) VALUES($1, $2) RETURNING id",
		user.Tg_ID,
		user.Role,
	).Scan(&id)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	if user.Role == "child" {
		return db.QueryRow(
			"INSERT INTO tokens(user_id) VALUES($1) RETURNING id",
			user.Tg_ID,
		).Scan(&id)
	}
	return nil
}

func dbTokenGet(db *sql.DB, tgID int64) (int, error) {
	slog.Info(fmt.Sprintf("Getting tokens for user %d...", tgID))
	var tokens int
	err := db.QueryRow("SELECT balance FROM tokens WHERE user_id = $1", tgID).Scan(&tokens)
	return tokens, err
}

func dbTokenAdd(db *sql.DB, tgID int64, tokens int) error {
	slog.Info(fmt.Sprintf("Adding tokens for user %d...", tgID))
	var id int
	return db.QueryRow(
		"UPDATE tokens SET balance = balance + $1 WHERE user_id = $2 RETURNING id",
		tokens,
		tgID,
	).Scan(&id)
}

func dbUserFind(db *sql.DB, tgID int64) (DBRecordUser, error) {
	slog.Info(fmt.Sprintf("Looking for user tg_id %d in the database...", tgID))
	var record DBRecordUser
	err := db.QueryRow("SELECT tg_id, role, family_uid FROM users WHERE tg_id = $1", tgID).Scan(
		&record.Tg_ID,
		&record.Role,
		&record.Family_UID,
	)
	if err != nil && err != sql.ErrNoRows {
		slog.Error(err.Error())
		return record, err
	}
	return record, nil
}

func dbCommonsAdd(common string, db *sql.DB, familyID string, name string, tokens int) error {
	slog.Info(fmt.Sprintf("Adding %s '%s' for family %s...", common, name, familyID))
	var id int
	query := fmt.Sprintf(
		"INSERT INTO %s(family_uid, name, tokens) VALUES($1, $2, $3) RETURNING id",
		common,
	)
	return db.QueryRow(
		query,
		familyID,
		name,
		tokens,
	).Scan(&id)
}

func dbCommonsDelete(common string, db *sql.DB, familyID string, name string) error {
	slog.Info(fmt.Sprintf("Removing %s '%s' for family %s...", common, name, familyID))
	var id int
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE family_uid = $1 AND name = $2 RETURNING id",
		common,
	)
	return db.QueryRow(
		query,
		familyID,
		name,
	).Scan(&id)
}

func dbCommonsEdit(common string, db *sql.DB, familyID string, name string, tokens int) error {
	slog.Info(fmt.Sprintf("Editing %s '%s' for family %s...", common, name, familyID))
	var id int
	query := fmt.Sprintf(
		"UPDATE %s SET tokens = $1 WHERE family_uid = $2 AND name = $3 RETURNING id",
		common,
	)
	return db.QueryRow(
		query,
		tokens,
		familyID,
		name,
	).Scan(&id)
}

func dbCommonsList(common string, db *sql.DB, familyID string) ([]DBRecordCommon, error) {
	slog.Info(fmt.Sprintf("Getting %s list for family %s...", common, familyID))
	var records []DBRecordCommon
	query := fmt.Sprintf(
		"SELECT name, tokens FROM %s WHERE family_uid = $1",
		common,
	)
	rows, err := db.Query(query, familyID)
	if err != nil {
		return records, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			name   string
			tokens int
		)

		if err := rows.Scan(&name, &tokens); err != nil {
			return records, err
		}
		records = append(records, DBRecordCommon{name, tokens})
	}

	return records, nil
}

func dbCommonsGet(common string, db *sql.DB, familyID string, name string) (DBRecordCommon, error) {
	slog.Info(fmt.Sprintf("Getting %s '%s' for family %s...", common, name, familyID))
	var record DBRecordCommon
	query := fmt.Sprintf(
		"SELECT name, tokens FROM %s WHERE family_uid = $1 AND name = $2",
		common,
	)
	err := db.QueryRow(
		query,
		familyID,
		name,
	).Scan(&record.Name, &record.Tokens)
	return record, err
}

// func dbTasksAdd(db *sql.DB, familyID string, name string, tokens int) error {
// 	return dbCommonsAdd("tasks", db, familyID, name, tokens)
// }
// func dbTasksDelete(db *sql.DB, familyID string, name string) error {
// 	return dbCommonsDelete("tasks", db, familyID, name)
// }
// func dbTasksEdit(db *sql.DB, familyID string, name string, tokens int) error {
// 	return dbCommonsEdit("tasks", db, familyID, name, tokens)
// }
// func dbTasksList(db *sql.DB, familyID string) ([]DBRecordCommon, error) {
// 	return dbCommonsList("tasks", db, familyID)
// }
// func dbTasksGet(db *sql.DB, familyID string, name string) (DBRecordCommon, error) {
// 	return dbCommonsGet("tasks", db, familyID, name)
// }

// func dbRewardsAdd(db *sql.DB, familyID string, name string, tokens int) error {
// 	return dbCommonsAdd("rewards", db, familyID, name, tokens)
// }
// func dbRewardsDelete(db *sql.DB, familyID string, name string) error {
// 	return dbCommonsDelete("rewards", db, familyID, name)
// }
// func dbRewardsEdit(db *sql.DB, familyID string, name string, tokens int) error {
// 	return dbCommonsEdit("rewards", db, familyID, name, tokens)
// }
// func dbRewardsList(db *sql.DB, familyID string) ([]DBRecordCommon, error) {
// 	return dbCommonsList("rewards", db, familyID)
// }
// func dbRewardsGet(db *sql.DB, familyID string, name string) (DBRecordCommon, error) {
// 	return dbCommonsGet("rewards", db, familyID, name)
// }
