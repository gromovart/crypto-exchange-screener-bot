.PHONY: help debug debug-enhanced debug-diagnostic analyzer-test debug-super-sensitive debug-all \
        build release run run-prod setup install test clean lint \
        debug-counter test-counter test-counter-quick counter-test-all

# ============================================
# ОТЛАДКА И ТЕСТИРОВАНИЕ
# ============================================

debug:
	@echo "🐛 Базовая отладка..."
	go run ./cmd/debug/basic/main.go

debug-enhanced:
	@echo "🔬 Расширенная отладка..."
	go run ./cmd/debug/enhanced/main.go

debug-diagnostic:
	@echo "🏥 Глубокая диагностика системы..."
	@echo ""
	@echo "Эта команда проверит:"
	@echo "  1. Конфигурацию"
	@echo "  2. Данные в хранилище"
	@echo "  3. Работу анализаторов вручную"
	@echo "  4. Полную систему"
	@echo ""
	@echo "Пороги: 0.001% (одна тысячная процента!)"
	@echo ""
	go run ./cmd/debug/diagnostic/main.go

analyzer-test:
	@echo "🧪 Тестирование анализаторов..."
	@echo ""
	@echo "Проверяем работу каждого анализатора отдельно"
	@echo "С тестовыми данными (рост 1%, падение 0.5%)"
	@echo ""
	go run ./cmd/debug/analyzer/main.go

debug-super-sensitive:
	@echo "🚀 Супер-чувствительная отладка..."
	go run ./cmd/debug/supersensitive/main.go

# ============================================
# COUNTER ANALYZER ТЕСТЫ
# ============================================

## debug-counter: Тестирование CounterAnalyzer (базовый тест)
debug-counter:
	@echo "🔢 Тестирование CounterAnalyzer..."
	@echo ""
	@echo "📊 Проверяем:"
	@echo "  • Базовый подсчет сигналов"
	@echo "  • Уведомления"
	@echo "  • Периоды анализа"
	@echo "  • Статистику"
	@echo ""
	go run ./cmd/debug/counter_test/main.go

## test-counter: Полный тест CounterAnalyzer
test-counter:
	@echo "🧪 ПОЛНЫЙ ТЕСТ COUNTER ANALYZER"
	@echo "================================"
	@echo ""
	@echo "1. Базовый функционал..."
	go run ./cmd/debug/analyzer/main.go 2>&1 | grep -A 30 "ТЕСТ COUNTER ANALYZER"
	@echo ""
	@echo "2. Детальный тест..."
	go run ./cmd/debug/counter_test/main.go
	@echo ""
	@echo "3. Интеграция с системой..."
	go run ./cmd/debug/enhanced/main.go 2>&1 | grep -A 40 "ТЕСТ 3: COUNTER ANALYZER"

## test-counter-quick: Быстрый тест CounterAnalyzer
test-counter-quick:
	@echo "⚡ Быстрый тест CounterAnalyzer..."
	go run ./cmd/debug/counter_test/main.go 2>&1 | grep -B5 -A20 "БАЗОВЫЙ ТЕСТ" || true

## counter-test-all: Все тесты CounterAnalyzer
counter-test-all:
	@echo "🚀 ЗАПУСК ВСЕХ ТЕСТОВ COUNTER ANALYZER"
	@echo "======================================"
	@echo ""
	@echo "Этап 1/4: Базовый тест"
	@echo "----------------------"
	go run ./cmd/debug/analyzer/main.go 2>&1 | grep -A 35 "ТЕСТ COUNTER ANALYZER" || true
	@echo ""

	@echo "Этап 2/4: Полный тест"
	@echo "---------------------"
	go run ./cmd/debug/counter_test/main.go 2>&1 | tail -50 || true
	@echo ""

	@echo "Этап 3/4: Интеграционный тест"
	@echo "------------------------------"
	go run ./cmd/debug/enhanced/main.go 2>&1 | grep -B5 -A40 "COUNTER ANALYZER" || true
	@echo ""

	@echo "Этап 4/4: Диагностический тест"
	@echo "-------------------------------"
	go run ./cmd/debug/diagnostic/main.go 2>&1 | grep -B5 -A20 "ТЕСТ COUNTER ANALYZER" || true
	@echo ""
	@echo "✅ Все тесты CounterAnalyzer завершены"

# ============================================
# ВСЕ ТЕСТЫ
# ============================================

debug-all:
	@echo "🚀 Полный набор тестов..."
	@echo ""
	@echo "1. Тест анализаторов..."
	@$(MAKE) analyzer-test
	@echo ""
	@echo "2. Тест CounterAnalyzer..."
	@$(MAKE) test-counter-quick
	@echo ""
	@echo "3. Диагностика системы..."
	@$(MAKE) debug-diagnostic
	@echo ""
	@echo "4. Расширенная отладка..."
	@$(MAKE) debug-enhanced
	@echo ""
	@echo "5. Супер-чувствительный тест..."
	@$(MAKE) debug-super-sensitive

# ============================================
# ПРОДАКШЕН КОМАНДЫ
# ============================================

## build: Сборка продакшен версии
build:
	@echo "🔨 Building Crypto Growth Monitor..."
	@mkdir -p bin
	CGO_ENABLED=0 go build \
		-ldflags="-s -w -X main.version=1.0.0 -X 'main.buildTime=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")'" \
		-o bin/growth-monitor ./cmd/bot
	@echo "✅ Built: bin/growth-monitor"

## release: Сборка релизных версий для всех платформ
release:
	@echo "🚀 Building release versions..."
	@mkdir -p releases

	# Linux
	@echo "📦 Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=1.0.0" \
		-o releases/growth-monitor-linux ./cmd/bot

	# macOS
	@echo "🍏 Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=1.0.0" \
		-o releases/growth-monitor-macos ./cmd/bot

	# Windows
	@echo "🪟 Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=1.0.0" \
		-o releases/growth-monitor-windows.exe ./cmd/bot

	@echo "✅ Release builds created in releases/"

## run: Запуск в режиме разработки (из исходников)
run:
	@echo "🚀 Запуск основного бота (разработка)..."
	go run ./cmd/bot/main.go

## run-prod: Запуск собранной версии
run-prod: build
	@echo "🚀 Запуск в продакшен режиме..."
	@if [ ! -f ".env" ]; then \
		echo "⚠️  Warning: .env file not found, using .env.example"; \
		cp .env.example .env 2>/dev/null || true; \
	fi
	./bin/growth-monitor --config=.env --log-level=info

## setup: Настройка окружения для продакшена
setup:
	@echo "📦 Setting up production environment..."
	@mkdir -p logs bin
	@if [ ! -f ".env" ]; then \
		cp .env.example .env 2>/dev/null || true; \
		echo "✅ Created .env from .env.example"; \
		echo "📝 Please edit .env file with your API keys"; \
	else \
		echo "✅ .env file already exists"; \
	fi
	@echo "🔧 Environment ready!"
	@echo "👉 Run 'make build' to build the binary"
	@echo "👉 Run 'make run-prod' to start the monitor"

## install: Установка в систему
install: build
	@echo "📦 Installing to system..."
	@if [ -d "$(GOPATH)/bin" ]; then \
		cp bin/growth-monitor $(GOPATH)/bin/; \
		echo "✅ Installed to $(GOPATH)/bin/growth-monitor"; \
		echo "👉 Run: growth-monitor --help"; \
	else \
		echo "⚠️  GOPATH/bin not found, copying to /usr/local/bin"; \
		sudo cp bin/growth-monitor /usr/local/bin/ 2>/dev/null || \
		cp bin/growth-monitor ~/.local/bin/ 2>/dev/null || \
		echo "❌ Could not install, try manually: cp bin/growth-monitor /usr/local/bin/"; \
	fi

# ============================================
# ВСПОМОГАТЕЛЬНЫЕ КОМАНДЫ
# ============================================

## test: Запуск тестов
test:
	@echo "🧪 Running tests..."
	go test ./... -v

## clean: Очистка проекта
clean:
	@echo "🧹 Cleaning project..."
	rm -rf bin/ releases/ logs/*.log
	go clean
	@echo "✅ Cleaned"

## lint: Проверка кода
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
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
	@if [ ! -f ".env" ]; then \
		echo "⚠️  Warning: .env file not found"; \
		echo "👉 Create .env file first: cp .env.example .env"; \
		exit 1; \
	fi
	docker run --env-file .env crypto-growth-monitor:latest

# ============================================
# ДОПОЛНИТЕЛЬНЫЕ КОМАНДЫ ДЛЯ УДОБСТВА
# ============================================

debug-analyzer:
	@echo "🧪 Отладка анализаторов..."
	go run ./cmd/debug/analyzer/main.go

debug-basic:
	@echo "🐛 Базовая отладка..."
	go run ./cmd/debug/basic/main.go

debug-enhanced-full:
	@echo "🔬 Полная расширенная отладка..."
	go run ./cmd/debug/enhanced/main.go

debug-super:
	@echo "🚀 Супер-чувствительный тест..."
	go run ./cmd/debug/supersensitive/main.go

debug-counter-quick:
	@echo "🔢 Быстрый тест CounterAnalyzer..."
	go run ./cmd/debug/counter_test/main.go 2>&1 | grep -B2 -A15 "БАЗОВЫЙ ТЕСТ\|ПОЛНЫЙ ТЕСТ\|СТАТИСТИКИ" || true

list-debug:
	@echo "📁 Доступные отладочные программы:"
	@echo "  make analyzer-test       - Тест анализаторов"
	@echo "  make debug               - Базовая отладка"
	@echo "  make debug-diagnostic    - Глубокая диагностика"
	@echo "  make debug-enhanced      - Расширенная отладка"
	@echo "  make debug-super-sensitive - Супер-чувствительный"
	@echo "  make debug-all           - Все тесты сразу"
	@echo ""
	@echo "🧮 COUNTER ANALYZER ТЕСТЫ:"
	@echo "  make debug-counter       - Базовый тест CounterAnalyzer"
	@echo "  make test-counter        - Полный тест CounterAnalyzer"
	@echo "  make test-counter-quick  - Быстрый тест CounterAnalyzer"
	@echo "  make counter-test-all    - Все тесты CounterAnalyzer"

## help: Показать помощь
help:
	@echo "📈 Crypto Growth Monitor - Makefile Help"
	@echo ""
	@echo "🚀 Основные команды:"
	@echo "  make build       - Сборка продакшен версии"
	@echo "  make run-prod    - Запуск собранной версии"
	@echo "  make setup       - Настройка окружения"
	@echo "  make install     - Установка в систему"
	@echo "  make release     - Сборка для всех платформ"
	@echo ""
	@echo "🐛 Отладка и тестирование:"
	@echo "  make debug       - Базовая отладка"
	@echo "  make debug-all   - Все тесты сразу"
	@echo "  make analyzer-test - Тест анализаторов"
	@echo "  make test        - Запуск unit тестов"
	@echo ""
	@echo "🧮 COUNTER ANALYZER:"
	@echo "  make debug-counter      - Тест CounterAnalyzer"
	@echo "  make test-counter       - Полный тест CounterAnalyzer"
	@echo "  make counter-test-all   - Все тесты CounterAnalyzer"
	@echo ""
	@echo "🔧 Утилиты:"
	@echo "  make clean       - Очистка проекта"
	@echo "  make lint        - Проверка кода"
	@echo "  make deps        - Обновление зависимостей"
	@echo "  make docker-build - Сборка Docker образа"
	@echo ""
	@echo "📖 Подробнее:"
	@echo "  make help        - Показать это сообщение"
	@echo "  make list-debug  - Список отладочных программ"

# ============================================
# СКРИПТЫ ДЛЯ ТЕСТИРОВАНИЯ
# ============================================

## run-counter-tests: Запуск всех тестов CounterAnalyzer через скрипт
run-counter-tests:
	@echo "🧪 Запуск тестов CounterAnalyzer..."
	@chmod +x ./scripts/test_counter.sh
	@./scripts/test_counter.sh

## create-counter-test-dir: Создание структуры для тестов CounterAnalyzer
create-counter-test-dir:
	@echo "📁 Создание структуры директорий для тестов..."
	@mkdir -p ./cmd/debug/counter_test
	@echo "✅ Создана директория: ./cmd/debug/counter_test"
	@echo "👉 Добавьте файл main.go для тестов CounterAnalyzer"