# 🎯 JoyTime

Приложение для учета задач и наград детей в семье через Telegram бота.

## 📋 Описание

JoyTime - это система управления семейными задачами и наградами, которая помогает родителям мотивировать детей выполнять домашние дела. Система состоит из:

- **REST API** - серверная часть для управления данными
- **Telegram Bot** - пользовательский интерфейс (в разработке)
- **PostgreSQL** - база данных

### Основные функции:

- 👨‍👩‍👧‍👦 Управление семьями и пользователями (родители/дети)
- 📝 Создание и управление заданиями
- 🎁 Создание и управление наградами
- 💎 Система токенов (валюта для обмена заданий на награды)
- 📊 API для интеграции с внешними системами

## 🚀 Быстрый старт

### Предварительные требования

- Go 1.23+
- PostgreSQL (или Docker)
- Make (опционально)

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd joytime
```

### 2. Настройка окружения

```bash
# Создать файл .env с настройками
make setup-env

# Или создать вручную
cp .env.example .env
```

Обновите настройки в `.env`:

```env
# Telegram Bot Token (получить у @BotFather)
TOKEN=your_telegram_bot_token_here

# PostgreSQL настройки
PGHOST=localhost
PGPORT=5432
PGUSER=joytime
PGPASSWORD=password
PGDATABASE=joytime

# Debug режим
DEBUG=1
```

### 3. Запуск базы данных

#### Вариант A: Docker (рекомендуется)

```bash
# Запустить PostgreSQL в Docker
make docker-up

# Проверить статус
docker ps
```

#### Вариант B: Локальная установка PostgreSQL

Установите PostgreSQL и создайте базу данных:

```sql
CREATE DATABASE joytime;
CREATE USER joytime WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE joytime TO joytime;
```

### 4. Сборка и запуск

```bash
# Установить зависимости
make dev-deps

# Собрать приложение
make build

# Заполнить БД тестовыми данными
make fill

# Запустить сервер
make run
```

Сервер будет доступен по адресу: http://localhost:8080

## 🧪 Тестирование

### Автоматические тесты

```bash
# Запустить все тесты
make test

# Только тесты API
go test ./internal/api -v
```

### Ручное тестирование API

```bash
# Тестирование через curl
make test-api
```

### Примеры API запросов

#### Создание семьи

```bash
curl -X POST http://localhost:8080/families \
  -H "Content-Type: application/json" \
  -d '{"name": "Семья Ивановых"}'
```

#### Создание пользователя

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "tg_id": 123456789,
    "name": "Иван Иванов",
    "role": "parent",
    "family_uid": "abc123"
  }'
```

#### Создание задания

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "family_uid": "abc123",
    "name": "Убраться в комнате",
    "tokens": 10,
    "description": "Навести порядок и пропылесосить"
  }'
```

## 📚 API Документация

### Эндпоинты

#### Семьи
- `GET /families` - Список всех семей
- `POST /families` - Создать семью
- `GET /families/{uid}` - Получить семью
- `PUT /families/{uid}` - Обновить семью
- `DELETE /families/{uid}` - Удалить семью

#### Пользователи
- `GET /users` - Список всех пользователей
- `POST /users` - Создать пользователя
- `GET /users/{tg_id}` - Получить пользователя
- `PUT /users/{tg_id}` - Обновить пользователя
- `DELETE /users/{tg_id}` - Удалить пользователя

#### Задания
- `GET /tasks` - Список всех заданий
- `POST /tasks` - Создать задание
- `GET /tasks/{family_uid}` - Получить задания семьи
- `PUT /tasks` - Обновить задание

#### Награды
- `GET /rewards` - Список всех наград
- `POST /rewards` - Создать награду
- `GET /rewards/{family_uid}` - Получить награды семьи
- `PUT /rewards` - Обновить награду

#### Токены
- `GET /tokens` - Список всех токенов
- `GET /tokens/{tg_id}` - Получить токены пользователя
- `PUT /tokens/{tg_id}` - Обновить токены пользователя

## 🏗️ Архитектура

```
cmd/                 # Точка входа приложения
├── main.go         # Основная логика запуска
└── config.go       # Конфигурация

internal/           # Внутренние пакеты
├── api/           # HTTP API
│   ├── http.go    # Handlers и маршруты
│   └── http_test.go # Тесты API
├── postgres/      # Работа с БД
│   ├── client.go  # Подключение к БД
│   ├── schema.go  # Модели данных
│   └── fill.go    # Тестовые данные
└── telegram/      # Telegram Bot (в разработке)

db/                # Миграции БД
├── migrations/
└── init/

test_api.sh        # Скрипт тестирования API
Makefile           # Команды автоматизации
docker-compose.yml # Docker окружение
```

## 🛠️ Разработка

### Структура данных

#### Families (Семьи)
- `id` - ID семьи
- `name` - Название семьи
- `uid` - Уникальный код семьи
- `created_by_user_id` - ID создателя

#### Users (Пользователи)
- `id` - ID пользователя
- `tg_id` - Telegram ID
- `name` - Имя пользователя
- `role` - Роль (parent/child)
- `family_uid` - Код семьи

#### Tasks (Задания)
- `id` - ID задания
- `family_uid` - Код семьи
- `name` - Название задания
- `tokens` - Количество токенов за выполнение
- `description` - Описание
- `status` - Статус (new/check)

#### Rewards (Награды)
- `id` - ID награды
- `family_uid` - Код семьи
- `name` - Название награды
- `tokens` - Стоимость в токенах
- `description` - Описание

#### Tokens (Токены)
- `id` - ID записи
- `tg_id` - Telegram ID пользователя
- `tokens` - Количество токенов

### Полезные команды

```bash
# Показать все доступные команды
make help

# Очистить артефакты сборки
make clean

# Посмотреть логи Docker
make docker-logs

# Остановить Docker
make docker-down
```

## 🐛 Отладка

### Логи приложения

Приложение выводит подробные логи в консоль. Для включения debug режима:

```bash
export DEBUG=1
make run
```

### Подключение к базе данных

```bash
# Через Docker
docker exec -it joytime-postgres psql -U joytime -d joytime

# Локально
psql -h localhost -U joytime -d joytime
```

## 📝 TODO

Смотрите файл `_todo.md` для списка запланированных функций.

## 🤝 Вклад в проект

1. Создайте форк репозитория
2. Создайте ветку для функции (`git checkout -b feature/amazing-feature`)
3. Закоммитьте изменения (`git commit -m 'feat: add amazing feature'`)
4. Запушьте ветку (`git push origin feature/amazing-feature`)
5. Создайте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT.