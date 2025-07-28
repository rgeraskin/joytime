package main

// type TGShowCommonsListStrings struct {
// 	Common   string // 'tasks' or 'rewards'
// 	Empty    string // "Список пуст"
// 	Many     string // "Список заданий"
// 	Question string // "Что делаем?"
// }

// func tgShowCommonsList(
// 	db *sql.DB,
// 	s TGShowCommonsListStrings,
// 	familyID sql.NullString,
// ) (string, error) {
// 	commons, err := dbCommonsList(s.Common, db, familyID)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return "", err
// 	}

// 	if len(commons) == 0 {
// 		return s.Empty, nil
// 	}

// 	message := fmt.Sprintf("%s:\n", s.Many)
// 	for i, common := range commons {
// 		message += fmt.Sprintf("%d. %s - %d 💎\n", i+1, common.Name, common.Tokens)
// 	}
// 	message += fmt.Sprintf("\n%s\n", s.Question)
// 	return message, nil
// }
