# 🔄 Миграция с Makefile на mise

Проект переведен с Makefile на [mise](https://mise.jdx.dev/) для улучшения управления инструментами и задачами разработки.

## 📋 Таблица соответствия команд

| Makefile             | mise                 | Описание                   |
|----------------------|----------------------|----------------------------|
| `make help`          | `mise tasks`         | Показать доступные команды |
| `make build`         | `mise run build`     | Собрать приложение         |
| `make run`           | `mise run run`       | Запустить приложение       |
| `make run-with-fill` | `mise run run-fill`  | Запустить с заполнением БД |
| `make test`          | `mise run test`      | Запустить все тесты        |
| `make test-api`      | `mise run test-api`  | Тестировать API            |
| `make fill`          | `mise run fill`      | Заполнить БД данными       |
| `make dev-deps`      | `mise run dev-deps`  | Установить зависимости     |
| `make clean`         | `mise run clean`     | Очистить артефакты         |
| `make docker-up`     | `mise run db.up`     | Запустить PostgreSQL       |
| `make docker-down`   | `mise run db.down`   | Остановить PostgreSQL      |
| `make docker-logs`   | `mise run db.logs`   | Логи PostgreSQL            |
| `make setup-env`     | `mise run setup-env` | Создать .env файл          |

## ✨ Новые возможности mise

### 🔧 Автоматическое управление инструментами

```bash
# mise автоматически установит Go 1.23+
mise install

# Проверить установленные инструменты
mise list
```

### 🚀 Дополнительные команды

```bash
# Запустить полную среду разработки
mise run dev

# CI проверки (форматирование, линтинг, тесты)
mise run ci

# Полный сброс проекта
mise run full-reset

# Только unit тесты
mise run test-unit

# Тесты с покрытием
mise run test-coverage

# Форматирование кода
mise run fmt

# Линтинг
mise run lint

# Подключиться к PostgreSQL
mise run db.shell

# Сбросить БД и заполнить данными
mise run db.reset
```

### 🎯 Зависимости задач

mise автоматически выполняет зависимые задачи:

```bash
# Автоматически выполнит build перед запуском
mise run run

# Автоматически запустит dev-deps и db.up
mise run dev
```

### ⚙️ Переменные окружения

mise управляет переменными окружения из `.mise.toml`:

```toml
[env]
APP_NAME = "joytime"
BUILD_DIR = "./build"
DEBUG = "1"
```

## 🚀 Быстрый старт с mise

```bash
# 1. Установить mise (если еще не установлен)
curl https://mise.jdx.dev/install.sh | sh

# 2. Установить инструменты проекта
mise install

# 3. Запустить среду разработки
mise run dev
```

## 📚 Дополнительные ресурсы

- [Документация mise](https://mise.jdx.dev/)
- [Конфигурация задач](https://mise.jdx.dev/tasks/)
- [Управление инструментами](https://mise.jdx.dev/dev-tools/)

## 🗑️ Удаление Makefile

Makefile больше не нужен и может быть удален:

```bash
rm Makefile
```

> **Примечание**: Все команды из Makefile переведены в mise и работают аналогично или лучше.