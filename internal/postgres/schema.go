package postgres

import (
	"gorm.io/gorm"
)

type Families struct {
	gorm.Model
	Name            string         `json:"name"`
	UID             string         `json:"uid"                gorm:"unique"`
	CreatedByUserID int            `json:"created_by_user_id"`
	DeletedAt       gorm.DeletedAt `json:"-"`
}

type Users struct {
	gorm.Model
	UID       string         `json:"uid"       gorm:"unique"`
	Name      string         `json:"name"`
	Role      string         `json:"role"`
	FamilyID  int            `json:"family_id"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

type Entities struct {
	// entities are tasks or rewards
	gorm.Model
	FamilyID    int            `json:"family_id"   gorm:"uniqueIndex:idx_name"`
	Name        string         `json:"name"        gorm:"uniqueIndex:idx_name"`
	Description string         `json:"description"`
	Tokens      int            `json:"tokens"`
	DeletedAt   gorm.DeletedAt `json:"-"`
	// OneOff      bool           `json:"one_off"`
}

type Tasks struct {
	Entities
}

type Rewards struct {
	Entities
}

type Tokens struct {
	gorm.Model
	UserID    int            `json:"user_id" gorm:"unique"`
	Tokens    int            `json:"tokens"`
	DeletedAt gorm.DeletedAt `json:"-"`
}
