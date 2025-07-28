.PHONY: build run test test-api fill clean help

# Переменные
APP_NAME=joytime
BUILD_DIR=./build
DOCKER_COMPOSE=docker-compose

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Показать справку
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

build: ## Собрать приложение
	@echo "$(GREEN)Сборка приложения...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd
	@echo "$(GREEN)✅ Приложение собрано: $(BUILD_DIR)/$(APP_NAME)$(NC)"

run: build ## Запустить приложение
	@echo "$(GREEN)Запуск приложения...$(NC)"
	@$(BUILD_DIR)/$(APP_NAME)

run-with-fill: build ## Запустить приложение с заполнением БД
	@echo "$(GREEN)Запуск приложения с заполнением БД...$(NC)"
	@$(BUILD_DIR)/$(APP_NAME) -fill

test: ## Запустить тесты
	@echo "$(GREEN)Запуск тестов...$(NC)"
	@go test -v ./...

test-api: ## Запустить тесты API через curl
	@echo "$(GREEN)Тестирование API...$(NC)"
	@chmod +x test_api.sh
	@./test_api.sh

fill: build ## Заполнить БД тестовыми данными
	@echo "$(GREEN)Заполнение БД тестовыми данными...$(NC)"
	@$(BUILD_DIR)/$(APP_NAME) -fill

dev-deps: ## Установить зависимости для разработки
	@echo "$(GREEN)Установка зависимостей...$(NC)"
	@go mod download
	@go mod tidy

clean: ## Очистить артефакты сборки
	@echo "$(GREEN)Очистка...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(APP_NAME)

# Docker команды
docker-up: ## Запустить PostgreSQL в Docker
	@echo "$(GREEN)Запуск PostgreSQL...$(NC)"
	@$(DOCKER_COMPOSE) up -d postgres

docker-down: ## Остановить Docker контейнеры
	@echo "$(GREEN)Остановка Docker контейнеров...$(NC)"
	@$(DOCKER_COMPOSE) down

docker-logs: ## Показать логи Docker контейнеров
	@$(DOCKER_COMPOSE) logs -f

# Настройка окружения
setup-env: ## Создать файл .env с примером настроек
	@echo "$(GREEN)Создание файла .env...$(NC)"
	@echo "# Telegram Bot Token" > .env
	@echo "TOKEN=your_telegram_bot_token_here" >> .env
	@echo "" >> .env
	@echo "# PostgreSQL настройки" >> .env
	@echo "PGHOST=localhost" >> .env
	@echo "PGPORT=5432" >> .env
	@echo "PGUSER=joytime" >> .env
	@echo "PGPASSWORD=password" >> .env
	@echo "PGDATABASE=joytime" >> .env
	@echo "" >> .env
	@echo "# Debug режим" >> .env
	@echo "DEBUG=1" >> .env
	@echo "$(YELLOW)⚠️  Не забудьте обновить настройки в .env файле!$(NC)"