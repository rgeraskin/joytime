package main

import (
	"time"

	"gorm.io/gorm"
)

type Families struct {
	gorm.Model
	Name      string
	UID       string // unique
	CreatedBy int
	CreatedAt time.Time
}

type Users struct {
	gorm.Model
	TgID      int64 // unique
	Name      string
	Role      string
	FamilyUID string
	CreatedAt time.Time
}

type Tasks struct {
	gorm.Model
	FamilyUID   string
	Name        string
	Description string
	Tokens      int
	OneOff      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Rewards struct {
	gorm.Model
	FamilyUID   string
	Name        string
	Description string
	Tokens      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Tokens struct {
	gorm.Model
	UserID    int
	Tokens    int
	CreatedAt time.Time
	UpdatedAt time.Time
}
