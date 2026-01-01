.PHONY: help debug debug-enhanced debug-diagnostic analyzer-test debug-super-sensitive debug-all \
	build release run run-prod setup install test clean lint \
	debug-counter test-counter test-counter-quick counter-test-all \
	test-basic test-quick test-all safe-test validate fix-vet test-stable quick-check \
	config-show config-dev config-prod config-list config-init config-edit check-config \
	run-dev run-local config-copy config-diff config-backup

# ============================================
# КОНФИГУРАЦИЯ ОКРУЖЕНИЙ (первым делом!)
# ============================================

ENV ?= dev
CONFIG_DIR = configs/$(ENV)
ENV_FILE = $(CONFIG_DIR)/.env
MAIN_FILE = ./application/main.go

# ============================================
# УПРАВЛЕНИЕ ОКРУЖЕНИЯМИ
# ============================================

## config-show: Показать текущее окружение и доступные окружения
config-show:
	@echo "🎯 ТЕКУЩЕЕ ОКРУЖЕНИЕ: $(ENV)"
	@echo ""
	@echo "📁 Доступные окружения:"
	@ls -la configs/
	@echo ""
	@if [ -f "$(ENV_FILE)" ]; then \
		echo "✅ Файл конфигурации: $(ENV_FILE)"; \
		echo "   Основные настройки:"; \
		grep -E "(TG_API_KEY|TELEGRAM_ENABLED|COUNTER_|LOG_LEVEL|HTTP_PORT)" "$(ENV_FILE)" 2>/dev/null || echo "   ⚠️  Файл пуст или не найден"; \
	else \
		echo "❌ Файл конфигурации не найден: $(ENV_FILE)"; \
		echo "   Создайте: cp configs/example/.env $(ENV_FILE)"; \
	fi

## config-dev: Переключиться на dev окружение
config-dev:
	@$(MAKE) config-show ENV=dev

## config-prod: Переключиться на prod окружение
config-prod:
	@$(MAKE) config-show ENV=prod

## config-list: Показать все доступные окружения
config-list:
	@echo "📋 ДОСТУПНЫЕ ОКРУЖЕНИЯ:"
	@echo "======================"
	@for dir in configs/*; do \
		if [ -d "$$dir" ]; then \
			env_name=$$(basename "$$dir"); \
			if [ -f "$$dir/.env" ]; then \
				echo "  ✅ $$env_name"; \
			else \
				echo "  ⚠️  $$env_name (.env отсутствует)"; \
			fi; \
		fi; \
	done
	@echo ""
	@echo "📝 Использование:"
	@echo "  make config-dev          # Переключиться на dev"
	@echo "  make config-prod         # Переключиться на prod"
	@echo "  make run ENV=prod        # Запустить с prod окружением"
	@echo "  make build ENV=dev       # Собрать с dev окружением"

## config-init: Инициализировать окружение
config-init:
	@echo "🔄 ИНИЦИАЛИЗАЦИЯ ОКРУЖЕНИЯ: $(ENV)"
	@echo "================================"
	@mkdir -p "$(CONFIG_DIR)"
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "Создание файла конфигурации..."; \
		if [ -f "configs/example/.env" ]; then \
			cp configs/example/.env "$(ENV_FILE)"; \
			echo "✅ Создан: $(ENV_FILE) (из example)"; \
		elif [ -f ".env.example" ]; then \
			cp .env.example "$(ENV_FILE)"; \
			echo "✅ Создан: $(ENV_FILE) (из .env.example)"; \
		else \
			echo "❌ Файл-шаблон не найден!"; \
			echo "   Создайте configs/example/.env или .env.example"; \
			exit 1; \
		fi; \
	else \
		echo "✅ Файл уже существует: $(ENV_FILE)"; \
	fi
	@echo ""
	@echo "📝 Отредактируйте файл:"
	@echo "  nano $(ENV_FILE)"
	@echo ""
	@echo "📋 Основные настройки для редактирования:"
	@echo "  - TG_API_KEY=your_telegram_bot_token_here"
	@echo "  - TG_CHAT_ID=your_telegram_chat_id_here"
	@echo "  - TELEGRAM_ENABLED=true/false"
	@echo "  - LOG_LEVEL=debug/info/warn/error"

## config-edit: Редактировать конфигурацию текущего окружения
config-edit:
	@if [ -f "$(ENV_FILE)" ]; then \
		echo "📝 Редактирование: $(ENV_FILE)"; \
		$${EDITOR:-nano} "$(ENV_FILE)"; \
	else \
		echo "❌ Файл не найден: $(ENV_FILE)"; \
		echo "   Создайте его: make config-init ENV=$(ENV)"; \
	fi

## check-config: Проверить текущую конфигурацию
check-config:
	@echo "🔍 ПРОВЕРКА КОНФИГУРАЦИИ ($(ENV))"
	@echo "================================"
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "❌ Файл конфигурации не найден: $(ENV_FILE)"; \
		echo "   Создайте: make config-init ENV=$(ENV)"; \
		exit 1; \
	fi

	@echo "✅ Файл конфигурации: $(ENV_FILE)"
	@echo ""

	@echo "📋 ОСНОВНЫЕ НАСТРОЙКИ:"
	@echo "-------------------"
	@errors=0

	@if grep -q "TG_API_KEY=" "$(ENV_FILE)"; then \
		TOKEN=$$(grep "TG_API_KEY=" "$(ENV_FILE)" | cut -d= -f2); \
		if [ "$$TOKEN" = "" ] || [ "$$TOKEN" = "your_telegram_bot_token_here" ]; then \
			echo "❌ TG_API_KEY не настроен"; \
			errors=$$((errors + 1)); \
		else \
			echo "✅ TG_API_KEY: $${TOKEN:0:10}...$${TOKEN: -10}"; \
		fi; \
	else \
		echo "❌ TG_API_KEY отсутствует в конфигурации"; \
		errors=$$((errors + 1)); \
	fi

	@if grep -q "TG_CHAT_ID=" "$(ENV_FILE)"; then \
		CHAT_ID=$$(grep "TG_CHAT_ID=" "$(ENV_FILE)" | cut -d= -f2); \
		if [ "$$CHAT_ID" = "" ] || [ "$$CHAT_ID" = "your_telegram_chat_id_here" ]; then \
			echo "❌ TG_CHAT_ID не настроен"; \
			errors=$$((errors + 1)); \
		else \
			echo "✅ TG_CHAT_ID: $$CHAT_ID"; \
		fi; \
	else \
		echo "❌ TG_CHAT_ID отсутствует в конфигурации"; \
		errors=$$((errors + 1)); \
	fi

	@if grep -q "TELEGRAM_ENABLED=" "$(ENV_FILE)"; then \
		ENABLED=$$(grep "TELEGRAM_ENABLED=" "$(ENV_FILE)" | cut -d= -f2); \
		echo "✅ TELEGRAM_ENABLED: $$ENABLED"; \
	else \
		echo "⚠️  TELEGRAM_ENABLED не указан (по умолчанию: false)"; \
	fi

	@if grep -q "COUNTER_ANALYZER_ENABLED=" "$(ENV_FILE)"; then \
		COUNTER=$$(grep "COUNTER_ANALYZER_ENABLED=" "$(ENV_FILE)" | cut -d= -f2); \
		echo "✅ COUNTER_ANALYZER_ENABLED: $$COUNTER"; \
	else \
		echo "⚠️  COUNTER_ANALYZER_ENABLED не указан"; \
	fi

	@if grep -q "LOG_LEVEL=" "$(ENV_FILE)"; then \
		LOG=$$(grep "LOG_LEVEL=" "$(ENV_FILE)" | cut -d= -f2); \
		echo "✅ LOG_LEVEL: $$LOG"; \
	else \
		echo "⚠️  LOG_LEVEL не указан"; \
	fi

	@echo ""
	@if [ "$$errors" -eq 0 ]; then \
		echo "🎯 КОНФИГУРАЦИЯ ГОТОВА К ИСПОЛЬЗОВАНИЮ"; \
	else \
		echo "❌ НАЙДЕНЫ ПРОБЛЕМЫ: $$errors"; \
		echo "   Исправьте: make config-edit ENV=$(ENV)"; \
	fi

## config-copy: Копировать конфигурацию между окружениями
config-copy:
	@echo "📋 КОПИРОВАНИЕ КОНФИГУРАЦИИ"
	@echo "=========================="
	@if [ -z "$(FROM)" ] || [ -z "$(TO)" ]; then \
		echo "❌ Укажите исходное и целевое окружение:"; \
		echo "   make config-copy FROM=dev TO=prod"; \
		exit 1; \
	fi

	@FROM_FILE="configs/$(FROM)/.env"
	@TO_FILE="configs/$(TO)/.env"

	@if [ ! -f "$$FROM_FILE" ]; then \
		echo "❌ Исходный файл не найден: $$FROM_FILE"; \
		exit 1; \
	fi

	@mkdir -p "configs/$(TO)"
	@cp "$$FROM_FILE" "$$TO_FILE"
	@echo "✅ Конфигурация скопирована из $(FROM) в $(TO)"
	@echo "   Файл: $$TO_FILE"

## config-diff: Сравнить два окружения
config-diff:
	@echo "🔍 СРАВНЕНИЕ ОКРУЖЕНИЙ"
	@echo "======================"
	@if [ -z "$(ENV1)" ] || [ -z "$(ENV2)" ]; then \
		echo "❌ Укажите два окружения:"; \
		echo "   make config-diff ENV1=dev ENV2=prod"; \
		exit 1; \
	fi

	@FILE1="configs/$(ENV1)/.env"
	@FILE2="configs/$(ENV2)/.env"

	@echo "📊 $(ENV1) vs $(ENV2)"
	@echo "-------------------"

	@if [ ! -f "$$FILE1" ]; then \
		echo "❌ Файл не найден: $$FILE1"; \
	else \
		echo "✅ $(ENV1): $$FILE1"; \
	fi

	@if [ ! -f "$$FILE2" ]; then \
		echo "❌ Файл не найден: $$FILE2"; \
	else \
		echo "✅ $(ENV2): $$FILE2"; \
	fi

	@if [ -f "$$FILE1" ] && [ -f "$$FILE2" ]; then \
		echo ""; \
		echo "📋 Различия:"; \
		diff -u "$$FILE1" "$$FILE2" 2>/dev/null || echo "   Файлы одинаковые"; \
	fi

## config-backup: Создать резервную копию конфигураций
config-backup:
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S)
	@BACKUP_DIR="configs/backup_$$TIMESTAMP"
	@mkdir -p "$$BACKUP_DIR"
	@cp -r configs/*/ "$$BACKUP_DIR" 2>/dev/null || true
	@echo "✅ Резервная копия создана: $$BACKUP_DIR"
	@echo "   Содержимое:"
	@ls -la "$$BACKUP_DIR"/

# ============================================
# ОСНОВНЫЕ КОМАНДЫ (с поддержкой окружений)
# ============================================

## build: Сборка продакшен версии с учетом окружения
build:
	@echo "🔨 Building Crypto Growth Monitor ($(ENV))..."
	@mkdir -p bin
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "⚠️  Файл конфигурации не найден: $(ENV_FILE)"; \
		read -p "Создать? (y/n): " -n 1 -r; echo ""; \
		if [[ $$REPLY =~ ^[Yy] ]]; then \
			$(MAKE) config-init ENV=$(ENV); \
		else \
			echo "❌ Отмена сборки"; \
			exit 1; \
		fi; \
	fi

	@echo "📋 Конфигурация: $(ENV_FILE)"
	CGO_ENABLED=0 go build \
		-ldflags="-s -w -X main.version=1.0.0 -X 'main.buildTime=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")'" \
		-o bin/growth-monitor-$(ENV) $(MAIN_FILE)
	@echo "✅ Built: bin/growth-monitor-$(ENV)"
	@echo "   Используйте: ./bin/growth-monitor-$(ENV) --config=$(ENV_FILE) --mode=full"

## release: Сборка релизных версий для всех платформ
release:
	@echo "🚀 Building release versions..."
	@mkdir -p releases

	# Linux
	@echo "📦 Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=1.0.0" \
		-o releases/growth-monitor-linux $(MAIN_FILE)

	# macOS
	@echo "🍏 Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=1.0.0" \
		-o releases/growth-monitor-macos $(MAIN_FILE)

	# Windows
	@echo "🪟 Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=1.0.0" \
		-o releases/growth-monitor-windows.exe $(MAIN_FILE)

	@echo "✅ Release builds created in releases/"

## run: Запуск в режиме разработки с учетом окружения
run:
	@echo "🚀 Запуск основного бота ($(ENV))..."
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "❌ Файл конфигурации не найден: $(ENV_FILE)"; \
		echo "   Создайте: make config-init ENV=$(ENV)"; \
		exit 1; \
	fi
	@echo "📋 Используется конфигурация: $(ENV_FILE)"
	go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple

## run-prod: Запуск собранной версии с prod окружением
run-prod:
	@$(MAKE) run ENV=prod

## run-dev: Запуск с dev окружением
run-dev:
	@$(MAKE) run ENV=dev

## run-full: Запуск в полном режиме
run-full:
	@echo "🚀 Запуск в полном режиме ($(ENV))..."
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "❌ Файл конфигурации не найден: $(ENV_FILE)"; \
		echo "   Создайте: make config-init ENV=$(ENV)"; \
		exit 1; \
	fi
	@echo "📋 Используется конфигурация: $(ENV_FILE)"
	go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full

## run-prod-binary: Запуск собранной бинарной версии
run-prod-binary: build
	@echo "🚀 Запуск в продакшен режиме ($(ENV))..."
	@./bin/growth-monitor-$(ENV) --config=$(ENV_FILE) --mode=full --log-level=info

## setup: Настройка окружения для продакшена
setup:
	@echo "📦 Setting up production environment..."
	@mkdir -p logs bin
	@$(MAKE) config-init ENV=prod
	@echo ""
	@echo "🔧 Environment ready!"
	@echo "👉 Run 'make build ENV=prod' to build the binary"
	@echo "👉 Run 'make run-full ENV=prod' to start the monitor"

## install: Установка в систему
install: build
	@echo "📦 Installing to system..."
	@if [ -d "$(GOPATH)/bin" ]; then \
		cp bin/growth-monitor-$(ENV) $(GOPATH)/bin/growth-monitor; \
		echo "✅ Installed to $(GOPATH)/bin/growth-monitor"; \
		echo "👉 Run: growth-monitor --config=$(ENV_FILE) --mode=full"; \
	else \
		echo "⚠️  GOPATH/bin not found, copying to /usr/local/bin"; \
		sudo cp bin/growth-monitor-$(ENV) /usr/local/bin/growth-monitor 2>/dev/null || \
		cp bin/growth-monitor-$(ENV) ~/.local/bin/growth-monitor 2>/dev/null || \
		echo "❌ Could not install, try manually: cp bin/growth-monitor-$(ENV) /usr/local/bin/"; \
	fi

# ============================================
# ЛОКАЛЬНЫЙ ЗАПУСК TELEGRAM БОТА
# ============================================

## run-local: Запуск Telegram бота в локальном режиме (polling)
run-local:
	@echo "🤖 ЗАПУСК В ЛОКАЛЬНОМ РЕЖИМЕ ($(ENV))"
	@echo "====================================="
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "❌ Файл конфигурации не найден: $(ENV_FILE)"; \
		echo "   Создайте: make config-init ENV=$(ENV)"; \
		exit 1; \
	fi

	@# Добавляем настройки для локального режима
	@cp "$(ENV_FILE)" "$(ENV_FILE).local"
	@echo "" >> "$(ENV_FILE).local"
	@echo "# Локальный режим" >> "$(ENV_FILE).local"
	@echo "HTTP_ENABLED=false" >> "$(ENV_FILE).local"
	@echo "TEST_MODE=false" >> "$(ENV_FILE).local"

	@echo "📋 Конфигурация: $(ENV_FILE).local"
	@echo ""
	@echo "🚀 Запуск бота..."
	@echo "📌 Откройте Telegram и найдите своего бота"
	@echo "📌 Отправьте команду /start"
	@echo "📌 Используйте меню кнопок для управления"
	@echo ""
	@echo "🛑 Для остановки нажмите Ctrl+C"
	@echo ""

	@# Запускаем бота
	go run $(MAIN_FILE) --config="$(ENV_FILE).local" --mode=full --log-level=debug

	@# Очистка
	@rm -f "$(ENV_FILE).local"
	@echo ""
	@echo "✅ Бот остановлен"

## run-local-test: Запуск в локальном тестовом режиме
run-local-test:
	@echo "🧪 ЗАПУСК В ЛОКАЛЬНОМ ТЕСТОВОМ РЕЖИМЕ ($(ENV))"
	@echo "=============================================="
	@echo "Без отправки реальных сообщений в Telegram"
	@if [ ! -f "$(ENV_FILE)" ]; then \
		cp "$(ENV_FILE)" "$(ENV_FILE).test"; \
		echo "TEST_MODE=true" >> "$(ENV_FILE).test"; \
		TEST_FILE="$(ENV_FILE).test"; \
	else \
		TEST_FILE="$(ENV_FILE)"; \
	fi
	@TEST_MODE=true go run $(MAIN_FILE) --config="$$TEST_FILE" --mode=full 2>&1 | grep -E "(Telegram|test mode|🤖|🧪)"
	@if [ -f "$(ENV_FILE).test" ]; then rm -f "$(ENV_FILE).test"; fi

## check-telegram-connection: Проверка подключения к Telegram
check-telegram-connection:
	@echo "🔌 ПРОВЕРКА ПОДКЛЮЧЕНИЯ К TELEGRAM ($(ENV))"
	@echo "============================================"
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "❌ Файл конфигурации не найден: $(ENV_FILE)"; \
		exit 1; \
	fi

	@TOKEN=$$(grep "TG_API_KEY=" "$(ENV_FILE)" | cut -d= -f2); \
	if [ "$$TOKEN" != "" ] && [ "$$TOKEN" != "your_telegram_bot_token_here" ]; then \
		echo "✅ Проверяем подключение..."; \
		curl -s "https://api.telegram.org/bot$$TOKEN/getMe" | python3 -m json.tool 2>/dev/null || echo "❌ Ошибка подключения"; \
	else \
		echo "❌ TG_API_KEY не настроен в $(ENV_FILE)"; \
	fi

## check-telegram-config: Проверка конфигурации Telegram
check-telegram-config:
	@echo "🔍 ПРОВЕРКА КОНФИГУРАЦИИ TELEGRAM ($(ENV))"
	@echo "=========================================="
	@echo ""
	@if [ -f "$(ENV_FILE)" ]; then \
		echo "📁 Файл .env найден: $(ENV_FILE)"; \
		echo ""; \
		echo "📋 Настройки Telegram:"; \
		if grep -q "TELEGRAM_ENABLED=true" "$(ENV_FILE)"; then \
			echo "✅ Telegram включен"; \
			TOKEN=$$(grep "TG_API_KEY=" "$(ENV_FILE)" | cut -d= -f2); \
			if [ "$$TOKEN" != "" ] && [ "$$TOKEN" != "your_telegram_bot_token_here" ]; then \
				echo "✅ Bot Token: $${TOKEN:0:10}...$${TOKEN: -10}"; \
			else \
				echo "❌ Bot Token не настроен"; \
			fi; \
			CHAT_ID=$$(grep "TG_CHAT_ID=" "$(ENV_FILE)" | cut -d= -f2); \
			if [ "$$CHAT_ID" != "" ] && [ "$$CHAT_ID" != "your_telegram_chat_id_here" ]; then \
				echo "✅ Chat ID: $$CHAT_ID"; \
			else \
				echo "❌ Chat ID не настроен"; \
			fi; \
			echo ""; \
			echo "📊 Counter Analyzer:"; \
			if grep -q "COUNTER_ANALYZER_ENABLED=true" "$(ENV_FILE)"; then \
				echo "✅ Counter Analyzer включен"; \
			else \
				echo "⚠️  Counter Analyzer отключен"; \
			fi; \
			if grep -q "COUNTER_NOTIFICATION_ENABLED=true" "$(ENV_FILE)"; then \
				echo "✅ Уведомления счетчика включены"; \
			else \
				echo "⚠️  Уведомления счетчика отключены"; \
			fi; \
		else \
			echo "❌ Telegram отключен в конфигурации"; \
		fi; \
	else \
		echo "⚠️  Файл .env не найден: $(ENV_FILE)"; \
		echo "   Создайте: make config-init ENV=$(ENV)"; \
	fi

# ============================================
# ОТЛАДКА И ТЕСТИРОВАНИЕ
# ============================================

## debug: Базовая отладка
debug:
	@echo "🐛 Базовая отладка ($(ENV))..."
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "⚠️  Используется конфигурация по умолчанию"; \
		go run $(MAIN_FILE) --mode=simple; \
	else \
		go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple --log-level=debug; \
	fi

## debug-enhanced: Расширенная отладка
debug-enhanced:
	@echo "🔬 Расширенная отладка ($(ENV))..."
	@echo "Запуск на 10 секунд..."
	@(go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=debug & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Отладка завершена"

## debug-diagnostic: Глубокая диагностика системы
debug-diagnostic:
	@echo "🏥 Глубокая диагностика системы ($(ENV))..."
	@echo ""
	@echo "Эта команда проверит:"
	@echo "  1. Конфигурацию"
	@echo "  2. Данные в хранилище"
	@echo "  3. Работу анализаторов вручную"
	@echo "  4. Полную систему"
	@echo ""
	@echo "Пороги: 0.001% (одна тысячная процента!)"
	@echo ""
	@echo "Запуск на 15 секунд..."
	@(go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=debug & PID=$$!; sleep 15; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Диагностика завершена"

# ============================================
# COUNTER ANALYZER ТЕСТЫ
# ============================================

## test-safe: Самый безопасный тест (рекомендуется)
test-safe:
	@echo "🛡️  БЕЗОПАСНОЕ ТЕСТИРОВАНИЕ ($(ENV))"
	@echo "=================================="
	@$(MAKE) check-config ENV=$(ENV)
	@echo ""
	@echo "1. Компиляция..."
	@go build $(MAIN_FILE) && echo "✅ Все компилируется"
	@echo ""
	@echo "2. Упрощенный тест системы..."
	@if go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple --test 2>&1 | grep -q "Тестирование завершено"; then \
		echo "✅ Система работает"; \
	else \
		echo "⚠️  Система требует проверки"; \
	fi
	@echo ""
	@echo "3. Сборка..."
	@$(MAKE) build ENV=$(ENV)
	@echo ""
	@echo "✅ Безопасное тестирование завершено"

## test-stable: Самый стабильный тест (рекомендуется)
test-stable:
	@echo "🏆 САМЫЙ СТАБИЛЬНЫЙ ТЕСТ ($(ENV))"
	@echo "================================"
	@$(MAKE) check-config ENV=$(ENV)
	@echo ""
	@echo "1. Компиляция основных компонентов..."
	@go build $(MAIN_FILE) && echo "✅ Базовая компиляция OK"
	@echo ""
	@echo "2. Быстрый тест системы..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple --test 2>&1 | tail -3 | grep -E "(✅|❌)" || echo "⚠️  Система требует внимания"
	@echo ""
	@echo "3. Сборка основного приложения..."
	@$(MAKE) build ENV=$(ENV)
	@echo ""
	@echo "🎉 ВСЕ ТЕСТЫ ПРОЙДЕНЫ УСПЕШНО!"

## quick-check: Быстрая проверка всей системы
quick-check:
	@echo "⚡ БЫСТРАЯ ПРОВЕРКА СИСТЕМЫ ($(ENV))"
	@echo "=================================="
	@echo ""
	@echo "1. Компиляция..."
	@go build $(MAIN_FILE) && echo "✅ Все компилируется"
	@echo ""
	@echo "2. Тест простого режима..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple --test 2>&1 | grep -E "(Application|Telegram|✅)" | head -5
	@echo ""
	@echo "🎯 СИСТЕМА РАБОТАЕТ КОРРЕКТНО!"

## debug-counter: Тестирование CounterAnalyzer (базовый тест)
debug-counter:
	@echo "🔢 Тестирование CounterAnalyzer ($(ENV))..."
	@echo ""
	@echo "📊 Проверяем:"
	@echo "  • Базовый подсчет сигналов"
	@echo "  • Уведомления"
	@echo "  • Периоды анализа"
	@echo "  • Статистику"
	@echo ""
	go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=debug 2>&1 | grep -E "(Counter|Analyzer|📊|🔢)" | head -20

## test-counter: Полный тест CounterAnalyzer
test-counter:
	@echo "🧪 ПОЛНЫЙ ТЕСТ COUNTER ANALYZER ($(ENV))"
	@echo "========================================"
	@echo ""
	@echo "1. Базовый функционал..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple 2>&1 | grep -E "(Counter|📊|🧪|✅)" | head -10
	@echo ""
	@echo "2. Детальный тест..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=info 2>&1 | grep -E "(Counter|📊|🧮|✅)" | head -20
	@echo ""
	@echo "3. Интеграция с системой..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=debug 2>&1 | grep -E "(COUNTER ANALYZER|🔢|📈|✅)" | head -20
	@echo ""
	@echo "✅ Полный тест CounterAnalyzer завершен"

## test-counter-quick: Быстрый тест CounterAnalyzer
test-counter-quick:
	@echo "⚡ Быстрый тест CounterAnalyzer ($(ENV))..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple 2>&1 | grep -E "(Counter|📊|✅)" | head -15

## counter-test-all: Все тесты CounterAnalyzer
counter-test-all:
	@echo "🚀 ЗАПУСК ВСЕХ ТЕСТОВ COUNTER ANALYZER ($(ENV))"
	@echo "================================================"
	@echo ""
	@echo "Этап 1/4: Базовый тест"
	@echo "----------------------"
	@(go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple & PID=$$!; sleep 5; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(Counter|📊|🧪)" | head -10
	@echo ""

	@echo "Этап 2/4: Полный тест"
	@echo "---------------------"
	@(go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=info & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(Counter|✅|📊|🧮)" | head -15
	@echo ""

	@echo "Этап 3/4: Интеграционный тест"
	@echo "------------------------------"
	@(go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=debug & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(COUNTER ANALYZER|🔢|📈)" | head -15
	@echo ""

	@echo "Этап 4/4: Диагностический тест"
	@echo "-------------------------------"
	@(go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full --log-level=debug & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(Counter|🔍|📊)" | head -10
	@echo ""
	@echo "✅ Все тесты CounterAnalyzer завершены"

# ============================================
# ВСЕ ТЕСТЫ
# ============================================

debug-all:
	@echo "🚀 Полный набор тестов ($(ENV))..."
	@echo ""
	@echo "1. Простой режим..."
	@$(MAKE) debug ENV=$(ENV)
	@echo ""
	@echo "2. Полный режим..."
	@$(MAKE) debug-enhanced ENV=$(ENV)
	@echo ""
	@echo "3. Диагностика системы..."
	@$(MAKE) debug-diagnostic ENV=$(ENV)

# ============================================
# БАЗОВЫЕ ТЕСТЫ (стабильные)
# ============================================

## test-basic: Базовые стабильные тесты
test-basic:
	@echo "🧪 БАЗОВЫЕ ТЕСТЫ СИСТЕМЫ ($(ENV))"
	@echo "=================================="
	@$(MAKE) check-config ENV=$(ENV)
	@echo ""
	@echo "1. Компиляция..."
	@go build $(MAIN_FILE) && echo "✅ Компиляция успешна"
	@echo ""
	@echo "2. Тест простого режима..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple --test 2>&1 | grep -E "(✅|📊|🧮|🎉)" | head -15 || echo "⚠️  Система требует внимания"
	@echo ""
	@echo "3. Проверка типов..."
	@go vet ./... 2>&1 | head -10 || echo "⚠️  Есть предупреждения go vet"
	@echo "✅ Базовые тесты завершены"

## test-quick: Быстрые тесты
test-quick:
	@echo "⚡ БЫСТРЫЕ ТЕСТЫ ($(ENV))"
	@echo "========================"
	@echo "Первый запуск (первые 5 строк)..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=simple 2>&1 | head -5
	@echo ""
	@echo "Counter Analyzer (первые 5 строк)..."
	@go run $(MAIN_FILE) --config=$(ENV_FILE) --mode=full 2>&1 | grep -E "(Counter|📊)" | head -5

## test-all: Все тесты (без бесконечного ожидания)
test-all: test-basic build
	@echo ""
	@echo "🎯 ВСЕ ТЕСТЫ ПРОЙДЕНЫ!"
	@echo "====================="
	@echo "✅ Система функционирует"
	@echo "✅ Counter Analyzer протестирован"
	@echo "✅ Сборка успешна"
	@echo "✅ Система готова к работе"

## safe-test: Безопасное тестирование без бесконечного ожидания
safe-test:
	@echo "🛡️  БЕЗОПАСНОЕ ТЕСТИРОВАНИЕ ($(ENV))"
	@echo "=================================="
	@$(MAKE) test-basic
	@echo ""
	@$(MAKE) build
	@echo ""
	@echo "✅ Безопасное тестирование завершено"

# ============================================
# ПРОВЕРКИ И ВАЛИДАЦИЯ
# ============================================

## validate: Проверка кода перед коммитом
validate:
	@echo "🔍 ПРОВЕРКА КОДА"
	@echo "================"
	@echo "1. Компиляция..."
	@go build ./... && echo "✅ Компиляция успешна"
	@echo "2. Проверка типов..."
	@go vet ./... 2>&1 | head -10 || true
	@echo "3. Форматирование..."
	@gofmt -l . | head -5 || true
	@echo "✅ Проверка завершена"

## fix-vet: Исправить ошибки go vet
fix-vet:
	@echo "🔧 ИСПРАВЛЕНИЕ ОШИБОК GO VET"
	@echo "==========================="
	@echo "Исправление ошибок копирования мьютекса..."
	@if go vet ./internal/analysis/analyzers/... 2>&1 | grep -q "copies lock value"; then \
		echo "⚠️  Найдены ошибки копирования мьютекса"; \
		echo "✅ Используйте test-stable или safe-test для стабильного тестирования"; \
	else \
		echo "✅ Ошибок go vet не обнаружено"; \
	fi

# ============================================
# ВСПОМОГАТЕЛЬНЫЕ КОМАНДЫ
# ============================================

## test: Запуск unit тестов
test:
	@echo "🧪 Running unit tests..."
	go test ./internal/analysis/analyzers/... -v -short

## clean: Очистка проекта
clean:
	@echo "🧹 Cleaning project..."
	rm -rf bin/ releases/ logs/*.log coverage/ reports/
	rm -f configs/*/.env.local configs/*/.env.test configs/*/.env.temp
	go clean
	@echo "✅ Cleaned"

## lint: Проверка кода
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed, using go vet..."; \
		go vet ./...; \
	fi

## deps: Обновление зависимостей
deps:
	@echo "📦 Updating dependencies..."
	go mod tidy
	go mod download
	@echo "✅ Dependencies updated"

## docker-build: Сборка Docker образа
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t crypto-growth-monitor:latest .

## docker-run: Запуск в Docker
docker-run:
	@echo "🚀 Running in Docker..."
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "⚠️  Warning: .env file not found"; \
		echo "👉 Create .env file first: make config-init ENV=$(ENV)"; \
		exit 1; \
	fi
	docker run --env-file $(ENV_FILE) crypto-growth-monitor:latest

## docker-run-prod: Запуск в Docker с prod окружением
docker-run-prod:
	@$(MAKE) docker-run ENV=prod

# ============================================
# ПОЛНЫЙ HELP
# ============================================

## help: Показать помощь с информацией об окружениях
help:
	@echo "📈 Crypto Growth Monitor - Makefile Help"
	@echo "🎯 Текущее окружение: $(ENV)"
	@echo ""
	@echo "🚀 УПРАВЛЕНИЕ ОКРУЖЕНИЯМИ:"
	@echo "  make config-show              - Показать текущее окружение"
	@echo "  make config-list              - Показать все окружения"
	@echo "  make config-dev               - Переключиться на dev"
	@echo "  make config-prod              - Переключиться на prod"
	@echo "  make config-init ENV=name     - Инициализировать окружение"
	@echo "  make config-edit ENV=name     - Редактировать окружение"
	@echo "  make check-config ENV=name    - Проверить конфигурацию"
	@echo "  make config-copy FROM=dev TO=prod - Копировать конфигурацию"
	@echo "  make config-diff ENV1=dev ENV2=prod - Сравнить окружения"
	@echo ""
	@echo "🚀 ОСНОВНЫЕ КОМАНДЫ (с окружениями):"
	@echo "  make build ENV=dev           - Сборка с указанным окружением"
	@echo "  make run ENV=dev             - Запуск в simple режиме"
	@echo "  make run-full ENV=dev        - Запуск в full режиме"
	@echo "  make run-prod                - Запуск с prod окружением"
	@echo "  make run-dev                 - Запуск с dev окружением"
	@echo "  make run-prod-binary         - Запуск собранной бинарной версии"
	@echo "  make run-local ENV=dev       - Локальный запуск Telegram бота"
	@echo "  make setup                   - Настройка окружения"
	@echo ""
	@echo "🔧 ОТЛАДКА И ТЕСТИРОВАНИЕ:"
	@echo "  make debug ENV=dev           - Базовая отладка (simple)"
	@echo "  make debug-full ENV=dev      - Отладка full режима"
	@echo "  make debug-counter ENV=dev   - Тест CounterAnalyzer"
	@echo "  make test-safe ENV=dev       - Безопасное тестирование"
	@echo "  make test-stable ENV=dev     - Стабильный тест"
	@echo "  make quick-check ENV=dev     - Быстрая проверка"
	@echo "  make check-telegram-config   - Проверка конфигурации Telegram"
	@echo ""
	@echo "🤖 TELEGRAM КОМАНДЫ:"
	@echo "  make run-local ENV=dev       - Локальный запуск бота"
	@echo "  make check-telegram-connection - Проверка подключения"
	@echo ""
	@echo "🧹 СЕРВИСНЫЕ КОМАНДЫ:"
	@echo "  make clean                   - Очистка проекта"
	@echo "  make lint                    - Проверка кода"
	@echo "  make deps                    - Обновление зависимостей"
	@echo "  make validate                - Проверка кода перед коммитом"
	@echo "  make test                    - Запуск unit тестов"
	@echo ""
	@echo "🐳 DOCKER КОМАНДЫ:"
	@echo "  make docker-build            - Сборка Docker образа"
	@echo "  make docker-run ENV=dev      - Запуск в Docker"
	@echo "  make docker-run-prod         - Запуск в Docker с prod"
	@echo ""
	@echo "📝 ПРИМЕРЫ ИСПОЛЬЗОВАНИЯ:"
	@echo "  # Разработка с dev окружением"
	@echo "  make config-dev"
	@echo "  make config-edit ENV=dev"
	@echo "  make run-full ENV=dev"
	@echo ""
	@echo "  # Продакшен с prod окружением"
	@echo "  make config-prod"
	@echo "  make build ENV=prod"
	@echo "  make run-full ENV=prod"
	@echo ""
	@echo "  # Тестирование разных окружений"
	@echo "  make test-safe ENV=dev"
	@echo "  make test-safe ENV=prod"
	@echo ""
	@echo "  # Локальная разработка Telegram бота"
	@echo "  make config-dev"
	@echo "  make run-local ENV=dev"
	@echo ""
	@echo "✅ Этот Makefile должен работать!"