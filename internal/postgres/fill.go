package postgres

import "gorm.io/gorm"

func fill(db *gorm.DB) error {
	db.Create(&Users{
		UID:      "tg220328701",
		Role:     "parent",
		FamilyID: 1,
	})
	db.Create(&Families{
		Name:            "Family 1",
		UID:             "sa726q",
		CreatedByUserID: 1,
	})

	db.Create(&[]Tasks{
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Загрузить посудомойку",
				Tokens:   2,
			},
		},

		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Достать и расставить посуду из посудомойки",
				Tokens:   2,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Помыть посуду у папы",
				Tokens:   5,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Постирать",
				Tokens:   2,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Вынести мусор",
				Tokens:   5,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Покормить кошку",
				Tokens:   2,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Почистить туалет кошки",
				Tokens:   2,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Занятие по шахматам",
				Tokens:   10,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Занятие по шахматам онлайн",
				Tokens:   5,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Занятие по футболу",
				Tokens:   10,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Турнир по шахматам",
				Tokens:   30,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Читать час",
				Tokens:   12,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Побрызгаться дезодорантом",
				Tokens:   1,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Поставить будильник и самому по нему проснуться",
				Tokens:   1,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Приготовить себе еду",
				Tokens:   6,
			},
		},
	})

	db.Create(&[]Rewards{
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Смотреть YouTube/VK 15м",
				Tokens:   5,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Смотреть YouTube/VK 60м",
				Tokens:   20,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Смотреть YouTube/VK 60м",
				Tokens:   20,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в Роблокс/Melon 15м",
				Tokens:   4,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в Роблокс/Melon 60м",
				Tokens:   16,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в Роблокс/Melon 60м",
				Tokens:   16,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м",
				Tokens:   2,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м",
				Tokens:   2,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 60м",
				Tokens:   8,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в остальные игры 15м",
				Tokens:   3,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в остальные игры 60м",
				Tokens:   12,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в остальные игры 15м",
				Tokens:   3,
			},
		},
		{
			Entities: Entities{
				FamilyID: 1,
				Name:     "Играть в остальные игры 60м",
				Tokens:   12,
			},
		},
	})

	return nil
}
