# 🎯 JoyTime

Универсальное приложение для учета задач и наград детей в семье с поддержкой различных интерфейсов.

## 📋 Описание

JoyTime - это система управления семейными задачами и наградами, которая помогает родителям мотивировать детей выполнять домашние дела. Система разработана для поддержки **любых интерфейсов** - Telegram, веб, мобильные приложения.

### Основные компоненты:

- **REST API** - серверная часть для управления данными (✅ готов)
- **Telegram Bot** - интерфейс через Telegram (🚧 в разработке)
- **Web UI** - веб-интерфейс (📋 запланирован)
- **Mobile App** - мобильное приложение (📋 запланирован)
- **PostgreSQL** - база данных

### 🎯 Основные функции:

- 👨‍👩‍👧‍👦 Управление семьями и пользователями (родители/дети)
- 📝 Создание и управление заданиями
- 🎁 Создание и управление наградами
- 💎 Система токенов (валюта для обмена заданий на награды)
- 📊 API для интеграции с любыми UI
- 🌐 **Мультиплатформенность** - поддержка telegram, web, mobile
- 📜 **История операций** - полный аудит всех действий с токенами
- 🔄 **Гибкие операции с токенами** - начисление, списание, ручная корректировка

## 🚀 Быстрый старт

### Предварительные требования

- [mise](https://mise.jdx.dev/) - универсальный инструмент разработки
- Docker (для PostgreSQL)

> mise автоматически установит Go 1.23+

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd joytime
```

### 2. Настройка окружения

```bash
# Создать файл .env с настройками
mise run setup-env

# Проверить, что все инструменты установлены
mise install
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
mise run db.up

# Проверить статус
docker ps

# Альтернативно через docker-compose
mise run docker.up
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
mise run dev-deps

# Собрать приложение
mise run build

# Заполнить БД тестовыми данными
mise run fill

# Запустить сервер
mise run run

# Или запустить всю среду разработки одной командой
mise run dev
```

Сервер будет доступен по адресу: http://localhost:8080

## 🧪 Тестирование

### Автоматические тесты

```bash
# Запустить все тесты
mise run test

# Только unit тесты
mise run test-unit

# Тесты с покрытием кода
mise run test-coverage
```

### Ручное тестирование API

```bash
# Интеграционные тесты API
mise run test-integration

# Тесты операций с токенами
mise run test-tokens
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
    "user_id": "user_ivan_123",
    "name": "Иван Иванов",
    "role": "parent",
    "family_uid": "abc123",
    "platform": "telegram"
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

#### Начисление токенов за выполнение задания

```bash
curl -X POST http://localhost:8080/tokens/user_child_456 \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 10,
    "type": "task_completed",
    "description": "Выполнил задание: Убраться в комнате"
  }'
```

#### Получение истории токенов

```bash
curl -X GET "http://localhost:8080/token-history/user_child_456?limit=10&offset=0"
```

## 📚 API Документация

### Основные эндпоинты

#### Семьи
- `GET /families` - Список всех семей
- `POST /families` - Создать семью
- `GET /families/{uid}` - Получить семью
- `PUT /families/{uid}` - Обновить семью
- `DELETE /families/{uid}` - Удалить семью

#### Пользователи
- `GET /users` - Список всех пользователей
- `POST /users` - Создать пользователя
- `GET /users/{user_id}` - Получить пользователя
- `PUT /users/{user_id}` - Обновить пользователя
- `DELETE /users/{user_id}` - Удалить пользователя

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
- `GET /tokens/{user_id}` - Получить токены пользователя
- `PUT /tokens/{user_id}` - Установить токены пользователя
- `POST /tokens/{user_id}` - **Начислить/списать токены** (новый!)

#### История токенов (новое!)
- `GET /token-history` - Вся история операций
- `GET /token-history/{user_id}` - История пользователя (с пагинацией)

### Структуры данных для POST /tokens/{user_id}

```json
{
  "amount": 10,           // Положительное - начисление, отрицательное - списание
  "type": "task_completed", // task_completed, reward_claimed, manual_adjustment
  "description": "Выполнил задание: Убраться в комнате",
  "task_id": 123,         // Опционально - ID связанного задания
  "reward_id": 456        // Опционально - ID связанной награды
}
```

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
- `created_by_user_id` - UserID создателя

#### Users (Пользователи)
- `id` - ID пользователя
- `user_id` - **Универсальный ID пользователя** (строка)
- `name` - Имя пользователя
- `role` - Роль (parent/child)
- `family_uid` - Код семьи
- `platform` - **Платформа** (telegram, web, mobile)
- `input_state` - Состояние ввода (для UI)
- `input_context` - Контекст ввода

#### Tasks (Задания)
- `id` - ID задания
- `family_uid` - Код семьи
- `name` - Название задания
- `tokens` - Количество токенов за выполнение
- `description` - Описание
- `status` - Статус (new/check/completed)
- `one_off` - Одноразовое задание

#### Rewards (Награды)
- `id` - ID награды
- `family_uid` - Код семьи
- `name` - Название награды
- `tokens` - Стоимость в токенах
- `description` - Описание

#### Tokens (Токены)
- `id` - ID записи
- `user_id` - Универсальный ID пользователя
- `tokens` - Текущее количество токенов

#### **TokenHistory (История токенов) - НОВОЕ!**
- `id` - ID операции
- `user_id` - ID пользователя
- `amount` - Изменение токенов (+/-)
- `type` - Тип операции (task_completed, reward_claimed, manual_adjustment)
- `description` - Описание операции
- `task_id` - Связанное задание (опционально)
- `reward_id` - Связанная награда (опционально)
- `created_at` - Время операции

### ✨ Новые возможности

#### 🌐 Мультиплатформенность
- Поле `platform` в Users позволяет отслеживать, через какой интерфейс работает пользователь
- Поддержка: `telegram`, `web`, `mobile`, и других
- Универсальные `user_id` (строки) вместо привязки к Telegram ID

#### 📜 Полная история операций
- Каждая операция с токенами записывается в `TokenHistory`
- Возможность отследить: кто, когда, за что получил/потратил токены
- Пагинация для больших объемов данных

#### 🔄 Гибкие операции с токенами
- `POST /tokens/{user_id}` для начисления/списания
- Автоматическая проверка достаточности токенов
- Связывание операций с заданиями/наградами

### Полезные команды

```bash
# Показать все доступные команды
mise tasks

# Очистить артефакты сборки
mise run clean

# Посмотреть логи PostgreSQL
mise run db.logs

# Остановить PostgreSQL
mise run db.down

# Подключиться к БД
mise run db.shell

# Полный сброс проекта
mise run full-reset

# Форматирование кода
mise run fmt

# Проверка линтером
mise run lint

# CI проверки
mise run ci
```

## 🐛 Отладка

### Логи приложения

Приложение выводит подробные логи в консоль. Debug режим уже настроен в mise:

```bash
# Запуск с логами
mise run run

# Посмотреть логи PostgreSQL
mise run db.logs
```

### Подключение к базе данных

```bash
# Быстрое подключение через mise
mise run db.shell

# Напрямую через Docker
docker exec -it joytime-postgres psql -U joytime -d joytime

# Локально (если PostgreSQL установлен)
psql -h localhost -U joytime -d joytime
```

### Полезные SQL запросы

```sql
-- Посмотреть всех пользователей
SELECT user_id, name, role, platform, family_uid FROM users;

-- Посмотреть баланс токенов
SELECT u.name, t.tokens FROM users u JOIN tokens t ON u.user_id = t.user_id;

-- Посмотреть историю операций
SELECT th.*, u.name
FROM token_history th
JOIN users u ON th.user_id = u.user_id
ORDER BY th.created_at DESC;
```

## 🚀 Следующие шаги

### Готово к реализации:
1. **Telegram Bot integration** - подключить бота к новому API
2. **Web UI** - создать веб-интерфейс
3. **Mobile App** - разработать мобильное приложение

### API готов для:
- ✅ Управления пользователями любых платформ
- ✅ Операций с заданиями и наградами
- ✅ Гибкой системы токенов с историей
- ✅ Полного аудита всех действий
- ✅ Мультиплатформенной разработки

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