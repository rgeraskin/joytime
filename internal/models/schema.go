package models

import (
	"gorm.io/gorm"
)

type Families struct {
	gorm.Model
	Name            string         `json:"name"`
	UID             string         `json:"uid"                gorm:"uniqueIndex"`
	CreatedByUserID string         `json:"created_by_user_id" gorm:"index"`
	DeletedAt       gorm.DeletedAt `json:"-"` // override gorm.Model to hide from JSON
}

type Users struct {
	gorm.Model
	UserID    string         `json:"user_id"     gorm:"uniqueIndex"`
	Name      string         `json:"name"`
	Role      string         `json:"role"`
	FamilyUID string         `json:"family_uid"  gorm:"index"`
	Platform  string         `json:"platform"    gorm:"default:telegram"` // telegram, web, mobile, etc.
	DeletedAt gorm.DeletedAt `json:"-"`                                   // override gorm.Model to hide from JSON
	// Поля для состояния ввода текста (универсальные)
	InputState   string `json:"input_state"`   // Что пользователь сейчас вводит
	InputContext string `json:"input_context"` // Контекст ввода
}

type Entities struct {
	// Entities is the base struct for tasks, rewards, and penalties
	gorm.Model
	FamilyUID   string         `json:"family_uid"  gorm:"index"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Tokens      int            `json:"tokens"`
	DeletedAt   gorm.DeletedAt `json:"-"` // override gorm.Model to hide from JSON
}

type Tasks struct {
	Entities
	AssignedToUserID string `json:"assigned_to_user_id" gorm:"index"`
	Status           string `json:"status" gorm:"default:new"` // new, check, completed
}

type Rewards struct {
	Entities
}

type Penalties struct {
	Entities
}

type Tokens struct {
	gorm.Model
	UserID    string         `json:"user_id" gorm:"uniqueIndex"`
	Tokens    int            `json:"tokens" gorm:"default:0"`
	DeletedAt gorm.DeletedAt `json:"-"` // override gorm.Model to hide from JSON
}

// Invites stores one-time invite codes for joining families
type Invites struct {
	gorm.Model
	Code            string `json:"code" gorm:"uniqueIndex"`
	FamilyUID       string `json:"family_uid" gorm:"index"`
	Role            string `json:"role"`
	Used            bool   `json:"used" gorm:"default:false"`
	CreatedByUserID string `json:"created_by_user_id"`
}

// TokenHistory tracks all token operations
type TokenHistory struct {
	gorm.Model
	UserID      string `json:"user_id" gorm:"index"`
	Amount      int    `json:"amount"`      // Положительное - заработал, отрицательное - потратил
	Type        string `json:"type" gorm:"index"` // task_completed, reward_claimed, penalty, manual_adjustment
	Description string `json:"description"` // Описание операции
	TaskID      *uint  `json:"task_id,omitempty"`
	RewardID    *uint  `json:"reward_id,omitempty"`
	PenaltyID   *uint  `json:"penalty_id,omitempty"`
}
