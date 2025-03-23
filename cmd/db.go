package main

import (
	"fmt"

	"golang.org/x/exp/rand"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	dbFamilyUIDCharset = "abcdefghjkmnpqrstuvwxyz23456789"
	dbFamilyUIDLength  = 6
)

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

type DB struct {
	DB     *gorm.DB
	Config *DBConfig
}

func (db *DB) Open() error {
	var err error
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		db.Config.Host,
		db.Config.User,
		db.Config.Password,
		db.Config.Database,
		db.Config.Port,
	)

	db.DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.New(
			logger,
			gormlogger.Config{
				// SlowThreshold:             time.Second,       // Slow SQL threshold
				// LogLevel:                  gormlogger.Silent, // Log level
				IgnoreRecordNotFoundError: true, // Ignore ErrRecordNotFound error for logger
				// ParameterizedQueries:      true,              // Don't include params in the SQL log
				// Colorful:                  false,             // Disable color
			},
		),
	})
	if err != nil {
		return err
	}

	db.DB.AutoMigrate(&Users{})
	db.DB.AutoMigrate(&Families{})
	db.DB.AutoMigrate(&Tokens{})
	db.DB.AutoMigrate(&Tasks{})
	db.DB.AutoMigrate(&Rewards{})
	return nil
}

func (db *DB) UserFind(user *Users) error {
	logger := logger.With("tg_id", user.TgID)

	logger.Info("Looking for user...")
	err := db.DB.Where("tg_id = ?", user.TgID).First(&user).Error
	return err
}

func (db *DB) FamilyJoin(user *Users) error {
	logger := logger.With("tg_id", user.TgID)

	logger.Info("Checking a family to join exists...")
	var family Families
	err := db.DB.Where("uid = ?", user.FamilyUID).First(&family).Error
	if err != nil {
		return err
	}

	user.FamilyUID = family.UID
	logger.Info(fmt.Sprintf(
		"Updating user with family_uid %s...", user.FamilyUID,
	))
	return db.DB.Save(&user).Error
}

func (db *DB) FamilyCreate(user *Users) error {
	logger := logger.With("user_id", user.ID)

	// Generate a unique family UID
	familyUID_byte := make([]byte, dbFamilyUIDLength)
	for i := range familyUID_byte {
		familyUID_byte[i] = dbFamilyUIDCharset[rand.Intn(len(dbFamilyUIDCharset))]
	}
	familyUID := string(familyUID_byte)

	logger.Info(fmt.Sprintf("Creating a new family with uid %s...", familyUID))
	family := Families{UID: familyUID, CreatedBy: int(user.ID)}
	err := db.DB.Create(&family).Error
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Updating user %d with family_uid %s...", user.ID, familyUID))
	user.FamilyUID = familyUID
	return db.DB.Save(&user).Error
}

func (db *DB) UserRegister(user *Users) error {
	logger := logger.With("role", user.Role)

	logger.Info("Registering user...")
	err := db.DB.Create(&user).Error
	if err != nil {
		return err
	}
	logger = logger.With("user_id", user.ID)
	logger.Info("User registered successfully")

	if user.Role == "child" {
		logger.Info("It's child, creating tokens...")
		token := Tokens{UserID: int(user.ID), Tokens: 0}
		return db.DB.Create(&token).Error
	}
	return nil
}

func (db *DB) TokensGet(user *Users) (int, error) {
	logger := logger.With("user_id", user.ID)

	logger.Info("Getting tokens...")
	var tokens Tokens
	err := db.DB.Where("user_id = ?", user.ID).First(&tokens).Error
	return tokens.Tokens, err
}

func (db *DB) TokensAdd(user *Users, tokens int) error {
	logger := logger.
		With("user_id", user.ID).
		With("tokens", tokens)

	logger.Info("Adding tokens...")
	var token Tokens
	err := db.DB.Where("user_id = ?", user.ID).First(&token).Error
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get tokens for user %d: %v", user.ID, err))
		return err
	}
	token.Tokens += tokens
	return db.DB.Save(&token).Error
}

func (db *DB) TasksStatusChange(user *Users, task *Tasks, status string) error {
	logger := logger.
		With("user_id", user.ID).
		With("task_name", task.Name).
		With("family_uid", user.FamilyUID).
		With("status", status)

	logger.Info("Changing status of task...")
	return db.DB.Where("user_id = ?", user.ID).
		Where("name = ?", task.Name).
		Update("status", status).
		Error
}

func (db *DB) RewardsClaim(user *Users, reward *Rewards) (int, error) {
	logger := logger.
		With("user_id", user.ID).
		With("reward_name", reward.Name).
		With("family_uid", user.FamilyUID)

	logger.Info("Claiming reward...")

	// get price of the reward
	var price int
	err := db.DB.Where("family_uid = ?", user.FamilyUID).
		Where("name = ?", reward.Name).
		First(&price).
		Error
	if err != nil {
		logger.Error("Failed to get price of reward")
		return 0, err
	}
	logger.Info(fmt.Sprintf("Price: %d", price))

	// get user tokens
	var userTokens Tokens
	err = db.DB.Where("user_id = ?", user.ID).First(&userTokens).Error
	if err != nil {
		logger.Error("Failed to get user tokens")
		return price, err
	}
	logger.Info(fmt.Sprintf("User tokens: %d", userTokens.Tokens))

	// check if user has enough tokens
	if userTokens.Tokens < price {
		logger.Info("Not enough tokens")
		return price, fmt.Errorf("not enough tokens")
	}

	// update user tokens
	userTokens.Tokens -= price
	return price, db.DB.Save(&userTokens).Error
}

func (db *DB) UsersGet(familyUID string, role string) ([]Users, error) {
	logger := logger.With("family_uid", familyUID, "role", role)

	logger.Info("Getting users...")
	var users []Users

	err := db.DB.Where("family_uid = ?", familyUID).Where("role = ?", role).Find(&users).Error
	return users, err
}

func (db *DB) TasksGet(user *Users, tasks *[]Tasks) error {
	return db.DB.Where("family_uid = ?", user.FamilyUID).Find(&tasks).Error
}

func (db *DB) TasksAdd(task *Tasks) error {
	return db.DB.Create(&task).Error
}

func (db *DB) TasksEdit(task *Tasks) error {
	logger := logger.With("task_id", task.ID)

	logger.Info("Editing task...")
	return db.DB.Where("name = ?", task.Name).Save(&task).Error
}

func (db *DB) TasksDelete(task *Tasks) error {
	return db.DB.Delete(&task).Error
}

func (db *DB) RewardsGet(user *Users, rewards *[]Rewards) error {
	return db.DB.Where("family_uid = ?", user.FamilyUID).Find(&rewards).Error
}

func (db *DB) RewardsAdd(reward *Rewards) error {
	return db.DB.Create(&reward).Error
}

func (db *DB) RewardsEdit(reward *Rewards) error {
	return db.DB.Save(&reward).Error
}

func (db *DB) RewardsDelete(reward *Rewards) error {
	return db.DB.Delete(&reward).Error
}

// func dbCommonsAdd(
// 	common string,
// 	db *sql.DB,
// 	familyUID sql.NullString,
// 	name string,
// 	tokens int,
// ) error {
// 	logger.Info(fmt.Sprintf("Adding %s '%s' for family %s...", common, name, familyUID))
// 	var id int
// 	query := fmt.Sprintf(
// 		"INSERT INTO %s(family_uid, name, tokens) VALUES($1, $2, $3) RETURNING id",
// 		common,
// 	)
// 	return db.QueryRow(
// 		query,
// 		familyUID,
// 		name,
// 		tokens,
// 	).Scan(&id)
// }

// func dbCommonsDelete(common string, db *sql.DB, user DBRecordUser, name string) (int, error) {
// 	logger.Info(
// 		fmt.Sprintf("Removing %s '%s' for family %s...", common, name, user.Family_UID.String),
// 	)
// 	var id int
// 	query := fmt.Sprintf(
// 		"DELETE FROM %s WHERE family_uid = $1 AND name = $2 RETURNING id",
// 		common,
// 	)
// 	return 0, db.QueryRow(
// 		query,
// 		user.Family_UID,
// 		name,
// 	).Scan(&id)
// }

// func dbCommonsEdit(common string, db *sql.DB, familyUID string, name string, tokens int) error {
// 	logger.Info(fmt.Sprintf("Editing %s '%s' for family %s...", common, name, familyUID))
// 	var id int
// 	query := fmt.Sprintf(
// 		"UPDATE %s SET tokens = $1 WHERE family_uid = $2 AND name = $3 RETURNING id",
// 		common,
// 	)
// 	return db.QueryRow(
// 		query,
// 		tokens,
// 		familyUID,
// 		name,
// 	).Scan(&id)
// }

// func dbCommonsList(common string, db *sql.DB, familyUID sql.NullString) ([]DBRecordCommon, error) {
// 	logger.Info(fmt.Sprintf("Getting %s list for family %s...", common, familyUID))
// 	var records []DBRecordCommon
// 	query := fmt.Sprintf(
// 		"SELECT name, tokens FROM %s WHERE family_uid = $1",
// 		common,
// 	)
// 	rows, err := db.Query(query, familyUID)
// 	if err != nil {
// 		return records, err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var (
// 			name   string
// 			tokens int
// 		)

// 		if err := rows.Scan(&name, &tokens); err != nil {
// 			return records, err
// 		}
// 		records = append(records, DBRecordCommon{name, tokens})
// 	}

// 	return records, nil
// }

// func dbCommonsGet(
// 	common string,
// 	db *sql.DB,
// 	familyUID string,
// 	name string,
// ) (DBRecordCommon, error) {
// 	logger.Info(fmt.Sprintf("Getting %s '%s' for family %s...", common, name, familyUID))
// 	var record DBRecordCommon
// 	query := fmt.Sprintf(
// 		"SELECT name, tokens FROM %s WHERE family_uid = $1 AND name = $2",
// 		common,
// 	)
// 	err := db.QueryRow(
// 		query,
// 		familyUID,
// 		name,
// 	).Scan(&record.Name, &record.Tokens)
// 	return record, err
// }

// // func dbTasksAdd(db *sql.DB, familyUID string, name string, tokens int) error {
// // 	return dbCommonsAdd("tasks", db, familyUID, name, tokens)
// // }
// // func dbTasksDelete(db *sql.DB, familyUID string, name string) error {
// // 	return dbCommonsDelete("tasks", db, familyUID, name)
// // }
// // func dbTasksEdit(db *sql.DB, familyUID string, name string, tokens int) error {
// // 	return dbCommonsEdit("tasks", db, familyUID, name, tokens)
// // }
// // func dbTasksList(db *sql.DB, familyUID string) ([]DBRecordCommon, error) {
// // 	return dbCommonsList("tasks", db, familyUID)
// // }
// // func dbTasksGet(db *sql.DB, familyUID string, name string) (DBRecordCommon, error) {
// // 	return dbCommonsGet("tasks", db, familyUID, name)
// // }

// // func dbRewardsAdd(db *sql.DB, familyUID string, name string, tokens int) error {
// // 	return dbCommonsAdd("rewards", db, familyUID, name, tokens)
// // }
// // func dbRewardsDelete(db *sql.DB, familyUID string, name string) error {
// // 	return dbCommonsDelete("rewards", db, familyUID, name)
// // }
// // func dbRewardsEdit(db *sql.DB, familyUID string, name string, tokens int) error {
// // 	return dbCommonsEdit("rewards", db, familyUID, name, tokens)
// // }
// // func dbRewardsList(db *sql.DB, familyUID string) ([]DBRecordCommon, error) {
// // 	return dbCommonsList("rewards", db, familyUID)
// // }
// // func dbRewardsGet(db *sql.DB, familyUID string, name string) (DBRecordCommon, error) {
// // 	return dbCommonsGet("rewards", db, familyUID, name)
// // }

// func dbUsersSetTextInput(db *sql.DB, tgID int64, textInput DBTextInput) error {
// 	var id int

// 	logger.Info(fmt.Sprintf(
// 		"Updating user %d with text_input for %s with arg '%s'...",
// 		tgID,
// 		textInput.For.String,
// 		textInput.Arg.String,
// 	))
// 	return db.QueryRow(
// 		"UPDATE users SET text_input_for = $1, text_input_arg = $2 WHERE tg_id = $3 RETURNING id",
// 		textInput.For,
// 		textInput.Arg,
// 		tgID,
// 	).Scan(&id)
// }

// func dbUsersGetTextInput(db *sql.DB, tgID int64) (DBTextInput, error) {
// 	var textInput DBTextInput

// 	logger.Info(fmt.Sprintf("Getting text_input for user %d...", tgID))
// 	err := db.QueryRow(
// 		"SELECT text_input_for, text_input_arg FROM users WHERE tg_id = $1",
// 		tgID,
// 	).Scan(&textInput.For, &textInput.Arg)

// 	return textInput, err
// }
