package database

import (
	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	"gorm.io/gorm"
)

func Fill(db *gorm.DB) error {
	return db.Transaction(func(db *gorm.DB) error {
		return fillData(db)
	})
}

func fillData(db *gorm.DB) error {
	// Create family first
	family := models.Families{
		Name:            "Family 1",
		UID:             "sa726q",
		CreatedByUserID: "user_220328701",
	}
	if err := db.Create(&family).Error; err != nil {
		return err
	}

	// Create parent user
	parent := models.Users{
		UserID:    "user_220328701",
		Name:      "Test Parent",
		Role:      "parent",
		FamilyUID: "sa726q",
		Platform:  "telegram",
	}
	if err := db.Create(&parent).Error; err != nil {
		return err
	}

	// Create child user
	child := models.Users{
		UserID:    "user_123456789",
		Name:      "Test Child",
		Role:      "child",
		FamilyUID: "sa726q",
		Platform:  "telegram",
	}
	if err := db.Create(&child).Error; err != nil {
		return err
	}

	// Create tokens for child
	tokens := models.Tokens{
		UserID: "user_123456789",
		Tokens: 10,
	}
	if err := db.Create(&tokens).Error; err != nil {
		return err
	}

	// Create initial token history
	history := models.TokenHistory{
		UserID:      "user_123456789",
		Amount:      10,
		Type:        domain.TokenTypeManualAdjustment,
		Description: "Начальный баланс",
	}
	if err := db.Create(&history).Error; err != nil {
		return err
	}

	// Create tasks
	if err := db.Create(&[]models.Tasks{
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Загрузить посудомойку",
				Tokens:    2,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Достать и расставить посуду из посудомойки",
				Tokens:    2,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Помыть посуду у папы",
				Tokens:    5,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Постирать",
				Tokens:    2,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Вынести мусор",
				Tokens:    5,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Покормить кошку",
				Tokens:    2,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Почистить туалет кошки",
				Tokens:    2,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Занятие по шахматам",
				Tokens:    10,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Занятие по шахматам онлайн",
				Tokens:    5,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Занятие по футболу",
				Tokens:    10,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Турнир по шахматам",
				Tokens:    30,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Читать час",
				Tokens:    12,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Побрызгаться дезодорантом",
				Tokens:    1,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Поставить будильник и самому по нему проснуться",
				Tokens:    1,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Приготовить себе еду",
				Tokens:    6,
			},
		},
	}).Error; err != nil {
		return err
	}

	// Create rewards
	if err := db.Create(&[]models.Rewards{
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Смотреть YouTube/VK 15м",
				Tokens:    5,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Смотреть YouTube/VK 60м",
				Tokens:    20,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Роблокс/Melon 15м",
				Tokens:    4,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Роблокс/Melon 60м",
				Tokens:    16,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м",
				Tokens:    2,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 60м",
				Tokens:    8,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в остальные игры 15м",
				Tokens:    3,
			},
		},
		{
			Entities: models.Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в остальные игры 60м",
				Tokens:    12,
			},
		},
	}).Error; err != nil {
		return err
	}

	return nil
}
