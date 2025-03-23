package main

func (db *DB) Fill() error {
	db.DB.Create(&Users{
		TgID:      220328701,
		Role:      "parent",
		FamilyUID: "sa726q",
	})
	db.DB.Create(&Families{
		Name:      "Family 1",
		UID:       "sa726q",
		CreatedBy: 220328701,
	})

	db.DB.Create(&[]Tasks{
		{
			FamilyUID: "sa726q",
			Name:      "Загрузить посудомойку",
			Tokens:    2,
		},

		{
			FamilyUID: "sa726q",
			Name:      "Достать и расставить посуду из посудомойки",
			Tokens:    2,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Помыть посуду у папы",
			Tokens:    5,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Постирать",
			Tokens:    2,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Вынести мусор",
			Tokens:    5,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Покормить кошку",
			Tokens:    2,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Почистить туалет кошки",
			Tokens:    2,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Занятие по шахматам",
			Tokens:    10,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Занятие по шахматам онлайн",
			Tokens:    5,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Занятие по футболу",
			Tokens:    10,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Турнир по шахматам",
			Tokens:    30,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Читать час",
			Tokens:    12,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Побрызгаться дезодорантом",
			Tokens:    1,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Поставить будильник и самому по нему проснуться",
			Tokens:    1,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Приготовить себе еду",
			Tokens:    6,
		},
	})

	db.DB.Create(&[]Rewards{
		{
			FamilyUID: "sa726q",
			Name:      "Смотреть YouTube/VK 15м",
			Tokens:    5,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Смотреть YouTube/VK 60м",
			Tokens:    20,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Смотреть YouTube/VK 60м",
			Tokens:    20,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в Роблокс/Melon 15м",
			Tokens:    4,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в Роблокс/Melon 60м",
			Tokens:    16,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в Роблокс/Melon 60м",
			Tokens:    16,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м",
			Tokens:    2,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м",
			Tokens:    2,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 60м",
			Tokens:    8,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в остальные игры 15м",
			Tokens:    3,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в остальные игры 60м",
			Tokens:    12,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в остальные игры 15м",
			Tokens:    3,
		},
		{
			FamilyUID: "sa726q",
			Name:      "Играть в остальные игры 60м",
			Tokens:    12,
		},
	})

	return nil
}
