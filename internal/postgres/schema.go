package postgres

import (
	"gorm.io/gorm"
)

type Families struct {
	gorm.Model
	Name            string         `json:"name"`
	UID             string         `json:"uid"                gorm:"unique"`
	CreatedByUserID int64          `json:"created_by_user_id"`
	DeletedAt       gorm.DeletedAt `json:"-"`
}

type Users struct {
	gorm.Model
	TgID      int64          `json:"tg_id"     gorm:"unique"`
	Name      string         `json:"name"`
	Role      string         `json:"role"`
	FamilyUID string         `json:"family_uid"`
	DeletedAt gorm.DeletedAt `json:"-"`
	// Поля для состояния ввода текста в телеграм боте
	TextInputFor string `json:"text_input_for"`
	TextInputArg string `json:"text_input_arg"`
}

type Entities struct {
	// entities are tasks or rewards
	gorm.Model
	FamilyUID   string         `json:"family_uid"  gorm:"uniqueIndex:idx_name"`
	Name        string         `json:"name"        gorm:"uniqueIndex:idx_name"`
	Description string         `json:"description"`
	Tokens      int            `json:"tokens"`
	DeletedAt   gorm.DeletedAt `json:"-"`
	// OneOff      bool           `json:"one_off"`
}

type Tasks struct {
	Entities
	Status string `json:"status" gorm:"default:new"` // new, check
	OneOff bool   `json:"one_off" gorm:"default:false"`
}

type Rewards struct {
	Entities
}

type Tokens struct {
	gorm.Model
	TgID      int64          `json:"tg_id" gorm:"unique"`
	Tokens    int            `json:"tokens" gorm:"default:0"`
	DeletedAt gorm.DeletedAt `json:"-"`
}
