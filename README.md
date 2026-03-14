# JoyTime

Семейное приложение для управления задачами и наградами с токен-экономикой. Родители назначают задания, дети выполняют их за токены, токены обмениваются на награды. Мультиплатформенное (Telegram, web, mobile).

## Основные функции

- Управление семьями и пользователями (родители/дети)
- Создание и управление заданиями и наградами
- Система токенов с полной историей операций
- RBAC+ABAC авторизация через Casbin (роли + изоляция семей)
- REST API (`/api/v1/`) для интеграции с любыми UI
- Мультиплатформенность (telegram, web, mobile)

## Быстрый старт

### Требования

- [mise](https://mise.jdx.dev/) — автоматически установит Go 1.23+
- Docker (для PostgreSQL)

### Установка и запуск

```bash
git clone <repository-url>
cd joytime

mise install              # Установить зависимости
mise run setup-env        # Создать .env файл
mise run db:up            # Запустить PostgreSQL
mise run db:fill          # Заполнить тестовыми данными
mise run run              # Собрать и запустить (порт 8080)
```

### Тестирование

```bash
mise run test             # Все тесты (автоматически поднимает БД)
mise run test-coverage    # Отчет покрытия
mise run ci               # Формат + линт + тесты
```

Отдельный тест: `go test ./internal/handlers/ -run TestName -v` (БД должна быть запущена).

## API

Все эндпоинты требуют заголовок `X-User-ID` для аутентификации (кроме `/api/v1/health`).

### Семьи

```
GET    /api/v1/families          # Список семей
POST   /api/v1/families          # Создать семью (UID генерируется автоматически)
GET    /api/v1/families/{uid}    # Получить семью
PUT    /api/v1/families/{uid}    # Обновить семью
```

### Пользователи

```
GET    /api/v1/users             # Список пользователей семьи
GET    /api/v1/users/{userID}    # Получить пользователя
PUT    /api/v1/users/{userID}    # Обновить пользователя
DELETE /api/v1/users/{userID}    # Удалить пользователя
```

### Задания

```
POST   /api/v1/tasks                          # Создать задание
GET    /api/v1/tasks/{familyUID}              # Задания семьи
GET    /api/v1/tasks/{familyUID}/{taskName}   # Конкретное задание
PUT    /api/v1/tasks/{familyUID}/{taskName}   # Обновить задание
DELETE /api/v1/tasks/{familyUID}/{taskName}   # Удалить задание
POST   /api/v1/tasks/{familyUID}/{taskName}   # Завершить задание
```

### Награды

```
POST   /api/v1/rewards                            # Создать награду
GET    /api/v1/rewards/{familyUID}                # Награды семьи
GET    /api/v1/rewards/{familyUID}/{rewardName}   # Конкретная награда
PUT    /api/v1/rewards/{familyUID}/{rewardName}   # Обновить награду
DELETE /api/v1/rewards/{familyUID}/{rewardName}   # Удалить награду
```

### Токены

```
POST   /api/v1/tokens                  # Начислить/списать токены
GET    /api/v1/tokens/users/{userID}   # Баланс пользователя
POST   /api/v1/tokens/users/{userID}   # Обновить баланс
GET    /api/v1/token-history           # Вся история
GET    /api/v1/token-history/{userID}  # История пользователя
```

### Примеры запросов

```bash
# Создать семью
curl -X POST http://localhost:8080/api/v1/families \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user_parent_123" \
  -d '{"name": "Семья Ивановых"}'

# Создать задание (только родитель)
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user_parent_123" \
  -d '{
    "family_uid": "abc123",
    "name": "Убраться в комнате",
    "tokens": 10,
    "description": "Навести порядок и пропылесосить"
  }'

# Создать награду (только родитель)
curl -X POST http://localhost:8080/api/v1/rewards \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user_parent_123" \
  -d '{
    "family_uid": "abc123",
    "name": "Мороженое",
    "tokens": 15,
    "description": "Поход в кафе за мороженым"
  }'

# Начислить токены
curl -X POST http://localhost:8080/api/v1/tokens \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user_parent_123" \
  -d '{
    "amount": 10,
    "type": "task_completed",
    "description": "Выполнил задание: Убраться в комнате"
  }'
```

### Формат ответов

Успех:
```json
{
  "data": { ... },
  "message": "optional"
}
```

Ошибка:
```json
{
  "error": "описание ошибки"
}
```

## Архитектура

Трёхуровневая архитектура в `internal/`:

```
cmd/
├── main.go                    # Точка входа
└── config.go                  # Конфигурация

internal/
├── handlers/                  # HTTP-слой
│   ├── http.go                # Маршруты и сервер
│   ├── middleware.go           # Auth middleware (X-User-ID → AuthContext)
│   ├── handler_families.go    # Эндпоинты семей
│   ├── handler_users.go       # Эндпоинты пользователей
│   ├── handler_tasks.go       # Эндпоинты заданий
│   ├── tasks_business.go      # Бизнес-логика заданий в handlers
│   ├── handler_rewards.go     # Эндпоинты наград
│   ├── handler_tokens.go      # Эндпоинты токенов
│   ├── types.go               # Типы, хелперы ответов
│   ├── constants.go           # Константы ошибок и ролей
│   └── validation.go          # Валидация и санитизация
│
├── domain/                    # Бизнес-логика
│   ├── services.go            # Агрегатор всех сервисов
│   ├── service_task.go        # TaskService (CRUD + RBAC)
│   ├── service_reward.go      # RewardService (CRUD + RBAC)
│   ├── service_token.go       # TokenService (баланс, история, claim)
│   ├── service_user.go        # UserService (профили, AuthContext)
│   ├── service_family.go      # FamilyService (семьи, UID генерация)
│   ├── casbin_auth.go         # Casbin RBAC+ABAC (программная модель)
│   ├── types.go               # AuthContext, DTO обновлений, ошибки
│   └── utils.go               # UpdateFields хелпер
│
├── database/                  # Подключение к БД
│   ├── client.go              # GORM setup, миграция
│   └── fill.go                # Тестовые данные
│
└── models/                    # GORM модели
    └── schema.go              # Families, Users, Tasks, Rewards, Tokens, TokenHistory
```

**Поток запроса**: HTTP → валидация → auth middleware (строит AuthContext) → handler → domain service (Casbin проверка + бизнес-правила) → GORM → PostgreSQL.

## Авторизация (Casbin RBAC+ABAC)

Модель определена программно в `domain/casbin_auth.go` (in-memory, без файлов политик).

| Ресурс   | Действие      | Родитель | Ребёнок | Примечание                                |
|----------|---------------|----------|---------|-------------------------------------------|
| tasks    | create        | +        | -       | Только родители создают задания           |
| tasks    | read          | +        | +       | Все видят задания семьи                   |
| tasks    | update        | +        | -       | Только родители редактируют               |
| tasks    | delete        | +        | -       | Только родители удаляют                   |
| tasks    | complete      | +        | +       | Ребёнок → "check", родитель → "completed" |
| rewards  | create        | +        | -       | Только родители создают награды           |
| rewards  | read          | +        | +       | Все видят награды                         |
| rewards  | update        | +        | -       | Только родители редактируют               |
| rewards  | delete        | +        | -       | Только родители удаляют                   |
| rewards  | claim         | -        | +       | Только дети обменивают токены на награды   |
| tokens   | read          | +        | +       | Свои токены                               |
| tokens   | read_others   | +        | -       | Родители видят токены всей семьи          |
| tokens   | add           | +        | -       | Только родители начисляют                 |
| users    | read          | +        | +       | Свой профиль                              |
| users    | update        | +        | +       | Свой профиль                              |
| users    | update_others | +        | -       | Родители редактируют детей                |
| users    | delete        | +        | -       | Родители удаляют пользователей            |
| family   | read          | +        | +       | Все видят семью                           |
| family   | update        | +        | -       | Только родители редактируют               |
| family   | create        | +        | -       | Создание семьи при регистрации            |

Семьи полностью изолированы — пользователи не могут получить доступ к данным чужих семей.

## Структуры данных

| Модель       | Ключевые поля                                                        |
|--------------|----------------------------------------------------------------------|
| Families     | name, uid (auto), created_by_user_id                                 |
| Users        | user_id, name, role (parent/child), family_uid, platform             |
| Tasks        | family_uid, name, description, tokens, status (new/check/completed)  |
| Rewards      | family_uid, name, description, tokens                                |
| Tokens       | user_id, tokens (баланс)                                            |
| TokenHistory | user_id, amount (+/-), type, description, task_id, reward_id         |

## Окружение

Настраивается через `.env` (см. `mise run setup-env`):

```env
PGHOST=localhost
PGPORT=5432
PGUSER=joytime
PGPASSWORD=password
PGDATABASE=joytime
DEBUG=1
```

## Полезные команды

```bash
mise run build          # Сборка
mise run run            # Сборка и запуск
mise run test           # Тесты
mise run fmt            # Форматирование
mise run lint           # Линтер
mise run db:up          # Запустить PostgreSQL
mise run db:down        # Остановить PostgreSQL
mise run db:reset       # Полный сброс с тестовыми данными
mise run db:shell       # psql в базу данных
```
