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
    \"tg_id\": 123456789,
    \"name\": \"Test Parent\",
    \"role\": \"parent\",
    \"family_uid\": \"$FAMILY_UID\"
}")
echo "$PARENT_RESPONSE"

echo ""
echo "3. 👶 Создание ребенка..."
CHILD_RESPONSE=$(send_request POST "/users" "{
    \"tg_id\": 987654321,
    \"name\": \"Test Child\",
    \"role\": \"child\",
    \"family_uid\": \"$FAMILY_UID\"
}")
echo "$CHILD_RESPONSE"

echo ""
echo "4. ⚡ Проверка токенов ребенка..."
TOKENS_RESPONSE=$(send_request GET "/tokens/987654321")
echo "$TOKENS_RESPONSE"

echo ""
echo "5. 📝 Создание задания..."
TASK_RESPONSE=$(send_request POST "/tasks" "{
    \"family_uid\": \"$FAMILY_UID\",
    \"name\": \"Убраться в комнате\",
    \"tokens\": 10,
    \"description\": \"Навести порядок\"
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
echo "9. 💰 Обновление токенов ребенка (+10)..."
UPDATE_TOKENS=$(send_request PUT "/tokens/987654321" '{"tokens": 10}')
echo "$UPDATE_TOKENS"

echo ""
echo "✅ Тесты завершены успешно!"