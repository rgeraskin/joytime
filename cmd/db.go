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

func dbFamilyCreate(db *sql.DB, userID int64) (sql.NullString, error) {
	var id int

	// Generate a unique family UID
	familyUID_byte := make([]byte, dbFamilyUIDLength)
	for i := range familyUID_byte {
		familyUID_byte[i] = dbFamilyUIDCharset[rand.Intn(len(dbFamilyUIDCharset))]
	}
	familyUID := sql.NullString{String: string(familyUID_byte), Valid: true}

	slog.Info(fmt.Sprintf("Creating a new family with uid %s...", familyUID.String))
	err := db.QueryRow(
		"INSERT INTO families(uid, created_by) VALUES($1, $2) RETURNING id",
		familyUID,
		userID,
	).Scan(&id)
	if err != nil {
		return familyUID, err
	}

	slog.Info(fmt.Sprintf("Updating user %d with family_uid %s...", userID, familyUID.String))

	return familyUID, db.QueryRow(
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
		slog.Info(fmt.Sprintf("It's child, creating tokens for user %d...", user.Tg_ID))
		return db.QueryRow(
			"INSERT INTO tokens(tg_id) VALUES($1) RETURNING id",
			user.Tg_ID,
		).Scan(&id)
	}
	return nil
}

func dbTokensGet(db *sql.DB, tgID int64) (int, error) {
	slog.Info(fmt.Sprintf("Getting tokens for user %d...", tgID))
	var tokens int
	err := db.QueryRow("SELECT tokens FROM tokens WHERE tg_id = $1", tgID).Scan(&tokens)
	return tokens, err
}

func dbTokensAdd(db *sql.DB, tgID int64, tokens int) error {
	slog.Info(fmt.Sprintf("Adding tokens for user %d...", tgID))
	var id int
	return db.QueryRow(
		"UPDATE tokens SET tokens = tokens + $1 WHERE tg_id = $2 RETURNING id",
		tokens,
		tgID,
	).Scan(&id)
}

func dbTasksStatusChange(status string, db *sql.DB, user DBRecordUser, name string) (int, error) {
	slog.Info(fmt.Sprintf(
		"Changing status of task '%s' for family %s to %s...", name, user.Family_UID.String, status,
	))
	var id int
	return 0, db.QueryRow(
		"UPDATE tasks SET status = $1 WHERE family_uid = $2 AND name = $3 RETURNING id",
		status,
		user.Family_UID,
		name,
	).Scan(&id)
}

func dbRewardsClaim(_ string, db *sql.DB, user DBRecordUser, name string) (int, error) {
	slog.Info(fmt.Sprintf(
		"Claiming reward '%s' for family %s to %d...", name, user.Family_UID.String, user.Tg_ID,
	))

	// get price of the reward
	var price int
	err := db.QueryRow(
		"SELECT tokens FROM rewards WHERE family_uid = $1 AND name = $2",
		user.Family_UID,
		name,
	).Scan(&price)
	if err != nil {
		slog.Error(err.Error())
		return 0, err
	}
	slog.Info(fmt.Sprintf("Price: %d", price))

	// get user tokens
	var tokens int
	err = db.QueryRow(
		"SELECT tokens FROM tokens WHERE tg_id = $1",
		user.Tg_ID,
	).Scan(&tokens)
	if err != nil {
		slog.Error(err.Error())
		return price, err
	}
	slog.Info(fmt.Sprintf("User tokens: %d", tokens))

	// check if user has enough tokens
	if tokens < price {
		slog.Info(fmt.Sprintf("Not enough tokens for user %d", user.Tg_ID))
		return price, fmt.Errorf("not enough tokens")
	}

	// update user tokens
	var id int
	return price, db.QueryRow(
		"UPDATE tokens SET tokens = $1 WHERE tg_id = $2 RETURNING id",
		tokens-price,
		user.Tg_ID,
	).Scan(&id)
}

func dbUsersGet(db *sql.DB, familyUID sql.NullString, role string) ([]int64, error) {
	slog.Info(fmt.Sprintf("Getting parents for family %s...", familyUID.String))
	var users []int64
	query := "SELECT tg_id FROM users WHERE family_uid = $1 AND role = $2"
	rows, err := db.Query(query, familyUID, role)
	if err != nil {
		return users, err
	}
	defer rows.Close()

	for rows.Next() {
		var tgID int64
		if err := rows.Scan(&tgID); err != nil {
			return users, err
		}
		users = append(users, tgID)
	}

	return users, nil
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

func dbCommonsAdd(
	common string,
	db *sql.DB,
	familyUID sql.NullString,
	name string,
	tokens int,
) error {
	slog.Info(fmt.Sprintf("Adding %s '%s' for family %s...", common, name, familyUID))
	var id int
	query := fmt.Sprintf(
		"INSERT INTO %s(family_uid, name, tokens) VALUES($1, $2, $3) RETURNING id",
		common,
	)
	return db.QueryRow(
		query,
		familyUID,
		name,
		tokens,
	).Scan(&id)
}

func dbCommonsDelete(common string, db *sql.DB, user DBRecordUser, name string) (int, error) {
	slog.Info(
		fmt.Sprintf("Removing %s '%s' for family %s...", common, name, user.Family_UID.String),
	)
	var id int
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE family_uid = $1 AND name = $2 RETURNING id",
		common,
	)
	return 0, db.QueryRow(
		query,
		user.Family_UID,
		name,
	).Scan(&id)
}

func dbCommonsEdit(common string, db *sql.DB, familyUID string, name string, tokens int) error {
	slog.Info(fmt.Sprintf("Editing %s '%s' for family %s...", common, name, familyUID))
	var id int
	query := fmt.Sprintf(
		"UPDATE %s SET tokens = $1 WHERE family_uid = $2 AND name = $3 RETURNING id",
		common,
	)
	return db.QueryRow(
		query,
		tokens,
		familyUID,
		name,
	).Scan(&id)
}

func dbCommonsList(common string, db *sql.DB, familyUID sql.NullString) ([]DBRecordCommon, error) {
	slog.Info(fmt.Sprintf("Getting %s list for family %s...", common, familyUID))
	var records []DBRecordCommon
	query := fmt.Sprintf(
		"SELECT name, tokens FROM %s WHERE family_uid = $1",
		common,
	)
	rows, err := db.Query(query, familyUID)
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

func dbCommonsGet(
	common string,
	db *sql.DB,
	familyUID string,
	name string,
) (DBRecordCommon, error) {
	slog.Info(fmt.Sprintf("Getting %s '%s' for family %s...", common, name, familyUID))
	var record DBRecordCommon
	query := fmt.Sprintf(
		"SELECT name, tokens FROM %s WHERE family_uid = $1 AND name = $2",
		common,
	)
	err := db.QueryRow(
		query,
		familyUID,
		name,
	).Scan(&record.Name, &record.Tokens)
	return record, err
}

// func dbTasksAdd(db *sql.DB, familyUID string, name string, tokens int) error {
// 	return dbCommonsAdd("tasks", db, familyUID, name, tokens)
// }
// func dbTasksDelete(db *sql.DB, familyUID string, name string) error {
// 	return dbCommonsDelete("tasks", db, familyUID, name)
// }
// func dbTasksEdit(db *sql.DB, familyUID string, name string, tokens int) error {
// 	return dbCommonsEdit("tasks", db, familyUID, name, tokens)
// }
// func dbTasksList(db *sql.DB, familyUID string) ([]DBRecordCommon, error) {
// 	return dbCommonsList("tasks", db, familyUID)
// }
// func dbTasksGet(db *sql.DB, familyUID string, name string) (DBRecordCommon, error) {
// 	return dbCommonsGet("tasks", db, familyUID, name)
// }

// func dbRewardsAdd(db *sql.DB, familyUID string, name string, tokens int) error {
// 	return dbCommonsAdd("rewards", db, familyUID, name, tokens)
// }
// func dbRewardsDelete(db *sql.DB, familyUID string, name string) error {
// 	return dbCommonsDelete("rewards", db, familyUID, name)
// }
// func dbRewardsEdit(db *sql.DB, familyUID string, name string, tokens int) error {
// 	return dbCommonsEdit("rewards", db, familyUID, name, tokens)
// }
// func dbRewardsList(db *sql.DB, familyUID string) ([]DBRecordCommon, error) {
// 	return dbCommonsList("rewards", db, familyUID)
// }
// func dbRewardsGet(db *sql.DB, familyUID string, name string) (DBRecordCommon, error) {
// 	return dbCommonsGet("rewards", db, familyUID, name)
// }
