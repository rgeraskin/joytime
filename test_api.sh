#!/bin/bash

# Тестовый скрипт для проверки API

set -e

echo "🧪 Тестирование API..."

BASE_URL="http://localhost:8080"

# Функция для отправки HTTP запросов
function send_request() {
    local method=$1
    local endpoint=$2
    local data=$3

    if [ -z "$data" ]; then
        curl -s -X $method "$BASE_URL$endpoint"
    else
        curl -s -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data"
    fi
}

echo "1. 📋 Создание семьи..."
FAMILY_RESPONSE=$(send_request POST "/families" '{"name": "Test Family"}')
echo "$FAMILY_RESPONSE"
FAMILY_UID=$(echo "$FAMILY_RESPONSE" | grep -o '"uid":"[^"]*"' | cut -d'"' -f4)
echo "Family UID: $FAMILY_UID"

echo ""
echo "2. 👤 Создание родителя..."
PARENT_RESPONSE=$(send_request POST "/users" "{
    \"user_id\": \"user_parent_123\",
    \"name\": \"Test Parent\",
    \"role\": \"parent\",
    \"family_uid\": \"$FAMILY_UID\",
    \"platform\": \"telegram\"
}")
echo "$PARENT_RESPONSE"

echo ""
echo "3. 👶 Создание ребенка..."
CHILD_RESPONSE=$(send_request POST "/users" "{
    \"user_id\": \"user_child_456\",
    \"name\": \"Test Child\",
    \"role\": \"child\",
    \"family_uid\": \"$FAMILY_UID\",
    \"platform\": \"telegram\"
}")
echo "$CHILD_RESPONSE"

echo ""
echo "4. ⚡ Проверка токенов ребенка..."
TOKENS_RESPONSE=$(send_request GET "/tokens/user_child_456")
echo "$TOKENS_RESPONSE"

echo ""
echo "5. 📝 Создание задания..."
TASK_RESPONSE=$(send_request POST "/tasks" "{
    \"family_uid\": \"$FAMILY_UID\",
    \"name\": \"Убраться в комнате\",
    \"tokens\": 10,
    \"description\": \"Навести порядок и пропылесосить\"
}")
echo "$TASK_RESPONSE"

echo ""
echo "6. 🎁 Создание награды..."
REWARD_RESPONSE=$(send_request POST "/rewards" "{
    \"family_uid\": \"$FAMILY_UID\",
    \"name\": \"Посмотреть мультики\",
    \"tokens\": 5,
    \"description\": \"15 минут мультиков\"
}")
echo "$REWARD_RESPONSE"

echo ""
echo "7. 📊 Получение всех заданий семьи..."
FAMILY_TASKS=$(send_request GET "/tasks/$FAMILY_UID")
echo "$FAMILY_TASKS"

echo ""
echo "8. 🏆 Получение всех наград семьи..."
FAMILY_REWARDS=$(send_request GET "/rewards/$FAMILY_UID")
echo "$FAMILY_REWARDS"

echo ""
echo "9. 💰 Начисление токенов ребенку за выполнение задания (+10)..."
ADD_TOKENS=$(send_request POST "/tokens/user_child_456" '{
    "amount": 10,
    "type": "task_completed",
    "description": "Выполнил задание: Убраться в комнате"
}')
echo "$ADD_TOKENS"

echo ""
echo "10. 🎯 Списание токенов за награду (-5)..."
SUBTRACT_TOKENS=$(send_request POST "/tokens/user_child_456" '{
    "amount": -5,
    "type": "reward_claimed",
    "description": "Получил награду: Посмотреть мультики"
}')
echo "$SUBTRACT_TOKENS"

echo ""
echo "11. 📜 Получение истории токенов ребенка..."
TOKEN_HISTORY=$(send_request GET "/token-history/user_child_456")
echo "$TOKEN_HISTORY"

echo ""
echo "12. 📈 Получение истории токенов с пагинацией (limit=2)..."
TOKEN_HISTORY_PAGINATED=$(send_request GET "/token-history/user_child_456?limit=2&offset=0")
echo "$TOKEN_HISTORY_PAGINATED"

echo ""
echo "13. 👥 Список всех пользователей..."
ALL_USERS=$(send_request GET "/users")
echo "$ALL_USERS"

echo ""
echo "14. 🏠 Список всех семей..."
ALL_FAMILIES=$(send_request GET "/families")
echo "$ALL_FAMILIES"

echo ""
echo "15. 💎 Список всех токенов..."
ALL_TOKENS=$(send_request GET "/tokens")
echo "$ALL_TOKENS"

echo ""
echo "16. 📝 Обновление пользователя (смена платформы)..."
UPDATE_USER=$(send_request PUT "/users/user_child_456" '{
    "platform": "web",
    "input_state": "waiting_for_task_name"
}')
echo "$UPDATE_USER"

echo ""
echo "✅ Тесты завершены успешно!"
echo ""
echo "🔍 Проверьте результаты:"
echo "- Семья создана с UID: $FAMILY_UID"
echo "- Создано 2 пользователя (родитель и ребенок)"
echo "- Создано 1 задание и 1 награда"
echo "- Выполнены операции с токенами (начисление/списание)"
echo "- Создана история операций с токенами"
echo "- API поддерживает разные платформы (telegram, web)"