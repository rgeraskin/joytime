package postgres

import (
	"gorm.io/gorm"
)

type Families struct {
	gorm.Model
	Name            string         `json:"name"`
	UID             string         `json:"uid"                gorm:"unique"`
	CreatedByUserID string         `json:"created_by_user_id"`
	DeletedAt       gorm.DeletedAt `json:"-"`
}

type Users struct {
	gorm.Model
	UserID    string         `json:"user_id"     gorm:"unique"`
	Name      string         `json:"name"`
	Role      string         `json:"role"`
	FamilyUID string         `json:"family_uid"`
	Platform  string         `json:"platform"    gorm:"default:telegram"` // telegram, web, mobile, etc.
	DeletedAt gorm.DeletedAt `json:"-"`
	// Поля для состояния ввода текста (универсальные)
	InputState   string `json:"input_state"`   // Что пользователь сейчас вводит
	InputContext string `json:"input_context"` // Контекст ввода
}

type Entities struct {
	// entities are tasks or rewards
	gorm.Model
	FamilyUID   string         `json:"family_uid"  gorm:"uniqueIndex:idx_name"`
	Name        string         `json:"name"        gorm:"uniqueIndex:idx_name"`
	Description string         `json:"description"`
	Tokens      int            `json:"tokens"`
	DeletedAt   gorm.DeletedAt `json:"-"`
}

type Tasks struct {
	Entities
	Status string `json:"status" gorm:"default:new"` // new, check, completed
	OneOff bool   `json:"one_off" gorm:"default:false"`
}

type Rewards struct {
	Entities
}

type Tokens struct {
	gorm.Model
	UserID    string         `json:"user_id" gorm:"unique"`
	Tokens    int            `json:"tokens" gorm:"default:0"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

// История операций с токенами
type TokenHistory struct {
	gorm.Model
	UserID      string `json:"user_id"`
	Amount      int    `json:"amount"`      // Положительное - заработал, отрицательное - потратил
	Type        string `json:"type"`        // task_completed, reward_claimed, manual_adjustment
	Description string `json:"description"` // Описание операции
	TaskID      *uint  `json:"task_id,omitempty"`
	RewardID    *uint  `json:"reward_id,omitempty"`
}
