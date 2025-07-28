package postgres

import "gorm.io/gorm"

func fill(db *gorm.DB) error {
	// Create family first
	family := Families{
		Name:            "Family 1",
		UID:             "sa726q",
		CreatedByUserID: 220328701,
	}
	db.Create(&family)

	// Create user
	user := Users{
		TgID:      220328701,
		Name:      "Test Parent",
		Role:      "parent",
		FamilyUID: "sa726q",
	}
	db.Create(&user)

	// Create child user
	child := Users{
		TgID:      123456789,
		Name:      "Test Child",
		Role:      "child",
		FamilyUID: "sa726q",
	}
	db.Create(&child)

	// Create tokens for child
	tokens := Tokens{
		TgID:   123456789,
		Tokens: 10,
	}
	db.Create(&tokens)

	// Create tasks
	db.Create(&[]Tasks{
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Загрузить посудомойку",
				Tokens:    2,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Достать и расставить посуду из посудомойки",
				Tokens:    2,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Помыть посуду у папы",
				Tokens:    5,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Постирать",
				Tokens:    2,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Вынести мусор",
				Tokens:    5,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Покормить кошку",
				Tokens:    2,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Почистить туалет кошки",
				Tokens:    2,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Занятие по шахматам",
				Tokens:    10,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Занятие по шахматам онлайн",
				Tokens:    5,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Занятие по футболу",
				Tokens:    10,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Турнир по шахматам",
				Tokens:    30,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Читать час",
				Tokens:    12,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Побрызгаться дезодорантом",
				Tokens:    1,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Поставить будильник и самому по нему проснуться",
				Tokens:    1,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Приготовить себе еду",
				Tokens:    6,
			},
		},
	})

	// Create rewards
	db.Create(&[]Rewards{
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Смотреть YouTube/VK 15м",
				Tokens:    5,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Смотреть YouTube/VK 60м",
				Tokens:    20,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Роблокс/Melon 15м",
				Tokens:    4,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Роблокс/Melon 60м",
				Tokens:    16,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м",
				Tokens:    2,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 60м",
				Tokens:    8,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в остальные игры 15м",
				Tokens:    3,
			},
		},
		{
			Entities: Entities{
				FamilyUID: "sa726q",
				Name:      "Играть в остальные игры 60м",
				Tokens:    12,
			},
		},
	})

	return nil
}
