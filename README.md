# JoyTime

Семейное приложение для управления задачами и наградами с токен-экономикой. Родители назначают задания, дети выполняют их за токены, токены обмениваются на награды. Основной интерфейс — Telegram-бот, также доступен REST API.

## Основные функции

- Управление семьями и пользователями (родители/дети)
- Одноразовые коды приглашения с deep link (`t.me/bot?start=CODE`)
- Создание и управление заданиями, наградами и штрафами
- Система токенов с полной историей операций
- Задания повторяемые — сбрасываются после одобрения родителем
- Двухэтапная проверка заданий: ребёнок отмечает → родитель подтверждает/отклоняет
- Массовый импорт заданий/наград/штрафов списком
- Ручная коррекция токенов с указанием причины
- Уведомления между родителями и детьми
- Управление семьёй: переименование и удаление участников
- История токенов для родителя и ребёнка
- RBAC+ABAC авторизация через Casbin (роли + изоляция семей)
- SQLite по умолчанию, PostgreSQL опционально
- REST API (`/api/v1/`) для интеграции с web/mobile
- Telegram-бот с inline-клавиатурами и эмодзи

## Быстрый старт

### Требования

- [mise](https://mise.jdx.dev/) — автоматически установит Go 1.23+
- Docker (только для PostgreSQL, необязателен для SQLite)

### Установка и запуск

```bash
git clone <repository-url>
cd joytime

mise install              # Установить зависимости
mise run setup-env        # Создать .env файл (указать TOKEN бота)
mise run run              # Собрать и запустить (SQLite + Telegram бот + HTTP на :8080)
```

Для PostgreSQL:
```bash
mise run db:up            # Запустить PostgreSQL
# В .env: DB_TYPE=postgres + PG* переменные
mise run run
```

### Тестирование

```bash
mise run test             # Все тесты (автоматически поднимает PostgreSQL)
mise run test-coverage    # Отчет покрытия
mise run ci               # Формат + линт + тесты
```

Отдельный тест: `go test ./internal/handlers/ -run TestName -v` (БД должна быть запущена).

## Telegram-бот

### Регистрация

- `/start` → «Создать семью» или «Ввести код приглашения»
- Deep link: `t.me/botname?start=CODE` — автоматический вход по ссылке
- Код приглашения одноразовый, содержит роль (родитель/ребёнок) и семью

### Родитель

- Главное меню: баланс детей, кнопка проверки заданий
- **Задания**: список (по убыванию токенов), добавить (по одному или списком), изменить цену, удалить
- **Награды**: список, добавить (по одному или списком), изменить цену, удалить
- **Штрафы**: список, добавить (по одному или списком), изменить цену, удалить, применить к ребёнку
- **Коррекция**: ручное начисление/списание токенов с указанием причины
- **Семья**: пригласить (родитель/ребёнок), переименовать, удалить участника
- **История**: просмотр истории токенов по ребёнку (последние 20 записей)
- **Проверка**: одобрить или отклонить выполненные ребёнком задания

### Ребёнок

- Главное меню: баланс токенов
- **Выполнить задание**: выбрать задание → отправить на проверку родителю
- **Получить награду**: выбрать награду → списать токены
- **Штрафы**: просмотр списка (только чтение)
- **История**: просмотр своей истории токенов

### Массовый импорт

Задания, награды и штрафы можно добавлять списком — каждый на новой строке, последнее слово — количество токенов:

```
Загрузить посудомойку 2
Вынести мусор 5
Читать час 12
```

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

## Архитектура

Трёхуровневая архитектура в `internal/`:

```
cmd/
├── main.go                    # Точка входа (HTTP + Telegram)
└── config.go                  # Конфигурация (SQLite/PostgreSQL)

internal/
├── handlers/                  # HTTP-слой
│   ├── http.go                # Маршруты и сервер
│   ├── middleware.go           # Auth middleware (X-User-ID → AuthContext)
│   ├── handler_families.go    # Эндпоинты семей
│   ├── handler_users.go       # Эндпоинты пользователей
│   ├── handler_tasks.go       # Эндпоинты заданий
│   ├── handler_rewards.go     # Эндпоинты наград
│   ├── handler_tokens.go      # Эндпоинты токенов
│   ├── types.go               # Типы, хелперы ответов
│   └── validation.go          # Валидация
│
├── telegram/                  # Telegram-бот
│   ├── bot.go                 # Bot struct, маршрутизация, хелперы
│   ├── registration.go        # Регистрация, deep link, приглашения
│   ├── parent.go              # Меню родителя, CRUD задания/награды/штрафы
│   └── child.go               # Меню ребёнка, выполнение, получение наград
│
├── domain/                    # Бизнес-логика
│   ├── services.go            # Агрегатор всех сервисов
│   ├── service_task.go        # TaskService (CRUD + complete/reject)
│   ├── service_reward.go      # RewardService (CRUD)
│   ├── service_penalty.go     # PenaltyService (CRUD + apply)
│   ├── service_token.go       # TokenService (баланс, история, claim)
│   ├── service_user.go        # UserService (профили, AuthContext, состояние)
│   ├── service_family.go      # FamilyService (семьи, UID генерация)
│   ├── service_invite.go      # InviteService (одноразовые коды приглашения)
│   ├── casbin_auth.go         # Casbin RBAC+ABAC (программная модель)
│   ├── types.go               # AuthContext, DTO, ошибки, валидация
│   └── utils.go               # UpdateFields хелпер
│
├── database/                  # Подключение к БД
│   ├── client.go              # GORM setup (SQLite/PostgreSQL), миграция
│   └── fill.go                # Тестовые данные
│
└── models/                    # GORM модели
    └── schema.go              # Families, Users, Tasks, Rewards, Penalties, Invites, Tokens, TokenHistory
```

## Окружение

Настраивается через `.env` (см. `mise run setup-env`):

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `TOKEN` | (обязательно) | Токен Telegram-бота |
| `DB_TYPE` | `sqlite` | Тип БД: `sqlite` или `postgres` |
| `DB_PATH` | `joytime.db` | Путь к файлу SQLite |
| `PGHOST` | — | Хост PostgreSQL (обязательно при postgres) |
| `PGPORT` | — | Порт PostgreSQL |
| `PGUSER` | — | Пользователь PostgreSQL |
| `PGPASSWORD` | — | Пароль PostgreSQL |
| `PGDATABASE` | — | Имя базы PostgreSQL |
| `DEBUG` | — | Включает отладочное логирование |

## Полезные команды

```bash
mise run build              # Сборка
mise run run                # Сборка и запуск
mise run test               # Тесты
mise run fmt                # Форматирование
mise run lint               # Линтер
mise run db:up              # Запустить PostgreSQL
mise run db:down            # Остановить PostgreSQL
mise run db:reset           # Полный сброс с тестовыми данными
mise run db:dump [file]     # Дамп базы (по умолчанию dump.sql)
mise run db:restore [file]  # Восстановление базы
mise run db:shell           # psql в базу данных
```
