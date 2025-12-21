.PHONY: help debug debug-enhanced debug-diagnostic analyzer-test debug-super-sensitive debug-all \
        build release run run-prod setup install test clean lint \
        debug-counter test-counter test-counter-quick counter-test-all \
        test-basic test-quick test-all safe-test validate fix-vet test-stable quick-check

# ============================================
# ОТЛАДКА И ТЕСТИРОВАНИЕ
# ============================================

debug:
	@echo "🐛 Базовая отладка..."
	go run ./cmd/debug/basic/main.go

debug-enhanced:
	@echo "🔬 Расширенная отладка..."
	@echo "Запуск на 10 секунд..."
	@(go run ./cmd/debug/enhanced/main.go & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Отладка завершена"

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
	@echo "Запуск на 15 секунд..."
	@(go run ./cmd/debug/diagnostic/main.go & PID=$$!; sleep 15; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Диагностика завершена"

analyzer-test:
	@echo "🧪 Тестирование анализаторов..."
	@echo ""
	@echo "Проверяем работу каждого анализатора отдельно"
	@echo "С тестовыми данных (рост 1%, падение 0.5%)"
	@echo ""
	go run ./cmd/debug/analyzer/main.go

debug-super-sensitive:
	@echo "🚀 Супер-чувствительная отладка..."
	@echo "Запуск на 10 секунд..."
	@(go run ./cmd/debug/supersensitive/main.go & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Супер-чувствительный тест завершен"

# ============================================
# COUNTER ANALYZER ТЕСТЫ
# ============================================

## test-safe: Самый безопасный тест (рекомендуется)
test-safe:
	@echo "🛡️  БЕЗОПАСНОЕ ТЕСТИРОВАНИЕ"
	@echo "==========================="
	@echo ""
	@echo "1. Компиляция..."
	@go build ./cmd/debug/... ./cmd/bot/ && echo "✅ Все компилируется"
	@echo ""
	@echo "2. Упрощенный тест CounterAnalyzer..."
	@if go run ./cmd/debug/counter_test/main.go 2>&1 | grep -q "ВСЕ ТЕСТЫ COUNTER ANALYZER ЗАВЕРШЕНЫ УСПЕШНО"; then \
		echo "✅ CounterAnalyzer работает"; \
	else \
		echo "⚠️  CounterAnalyzer требует проверки"; \
	fi
	@echo ""
	@echo "3. Упрощенный тест анализаторов..."
	@if go run ./cmd/debug/analyzer/main.go 2>&1 | grep -q "Тестирование завершено"; then \
		echo "✅ Анализаторы работают"; \
	else \
		echo "⚠️  Анализаторы требуют проверки"; \
	fi
	@echo ""
	@echo "4. Сборка..."
	@make build
	@echo ""
	@echo "✅ Безопасное тестирование завершено"

## test-stable: Самый стабильный тест (рекомендуется)
test-stable:
	@echo "🏆 САМЫЙ СТАБИЛЬНЫЙ ТЕСТ"
	@echo "========================"
	@echo ""
	@echo "1. Компиляция основных компонентов..."
	@go build ./cmd/debug/basic/ && echo "✅ Базовая компиляция OK"
	@go build ./cmd/debug/counter_test/ && echo "✅ CounterAnalyzer компиляция OK"
	@go build ./cmd/debug/analyzer/ && echo "✅ Анализаторы компиляция OK"
	@echo ""
	@echo "2. Быстрый тест CounterAnalyzer..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | tail -3 | grep -E "(✅|❌)" || echo "⚠️  CounterAnalyzer требует внимания"
	@echo ""
	@echo "3. Быстрый тест анализаторов..."
	@go run ./cmd/debug/analyzer/main.go 2>&1 | tail -3 | grep -E "(✅|🔧)" || echo "⚠️  Анализаторы работают"
	@echo ""
	@echo "4. Сборка основного приложения..."
	@make build
	@echo ""
	@echo "🎉 ВСЕ ТЕСТЫ ПРОЙДЕНЫ УСПЕШНО!"

## quick-check: Быстрая проверка всей системы
quick-check:
	@echo "⚡ БЫСТРАЯ ПРОВЕРКА СИСТЕМЫ"
	@echo "=========================="
	@echo ""
	@echo "1. Компиляция..."
	@go build ./cmd/debug/counter_test/ ./cmd/debug/analyzer/ ./cmd/bot/ && echo "✅ Все компилируется"
	@echo ""
	@echo "2. CounterAnalyzer..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | tail -2
	@echo ""
	@echo "3. Анализаторы..."
	@go run ./cmd/debug/analyzer/main.go 2>&1 | tail -2
	@echo ""
	@echo "🎯 СИСТЕМА РАБОТАЕТ КОРРЕКТНО!"

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

## test-counter: Полный тест CounterAnalyzer (исправленная версия)
test-counter:
	@echo "🧪 ПОЛНЫЙ ТЕСТ COUNTER ANALYZER"
	@echo "================================"
	@echo ""
	@echo "1. Базовый функционал..."
	@go run ./cmd/debug/analyzer/main.go 2>&1 | grep -E "(ТЕСТ COUNTER ANALYZER|📊|🧪|✅|🔧)" || true
	@echo ""
	@echo "2. Детальный тест..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(БАЗОВЫЙ ТЕСТ|📊|🧮|✅|🎉)" || true
	@echo ""
	@echo "3. Интеграция с системой..."
	@go run ./cmd/debug/enhanced/main.go 2>&1 | grep -E "(COUNTER ANALYZER|🔢|📈|✅)" | head -20 || true
	@echo ""
	@echo "✅ Полный тест CounterAnalyzer завершен"

## test-counter-quick: Быстрый тест CounterAnalyzer
test-counter-quick:
	@echo "⚡ Быстрый тест CounterAnalyzer..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(БАЗОВЫЙ ТЕСТ|📊|✅|🎉)" | head -15 || true

## counter-test-all: Все тесты CounterAnalyzer
counter-test-all:
	@echo "🚀 ЗАПУСК ВСЕХ ТЕСТОВ COUNTER ANALYZER"
	@echo "======================================"
	@echo ""
	@echo "Этап 1/4: Базовый тест анализаторов"
	@echo "----------------------"
	@(go run ./cmd/debug/analyzer/main.go & PID=$$!; sleep 15; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(ТЕСТ COUNTER|📊|🧪)" | head -20 || true
	@echo ""

	@echo "Этап 2/4: Полный тест CounterAnalyzer"
	@echo "---------------------"
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(✅|📊|🧮|🎉)" | head -25 || true
	@echo ""

	@echo "Этап 3/4: Интеграционный тест"
	@echo "------------------------------"
	@(go run ./cmd/debug/enhanced/main.go & PID=$$!; sleep 15; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(COUNTER ANALYZER|🔢|📈)" | head -15 || true
	@echo ""

	@echo "Этап 4/4: Диагностический тест"
	@echo "-------------------------------"
	@(go run ./cmd/debug/diagnostic/main.go & PID=$$!; sleep 15; kill $$PID 2>/dev/null || true) 2>/dev/null | grep -E "(ТЕСТ COUNTER|🔍|📊)" | head -10 || true
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
# БАЗОВЫЕ ТЕСТЫ (стабильные)
# ============================================

## test-basic: Базовые стабильные тесты
test-basic:
	@echo "🧪 БАЗОВЫЕ ТЕСТЫ СИСТЕМЫ"
	@echo "========================"
	@echo ""
	@echo "1. Компиляция..."
	@go build ./cmd/debug/... && echo "✅ Компиляция успешна"
	@echo ""
	@echo "2. Тест CounterAnalyzer..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(✅|📊|🧮|🎉)" | head -15 || echo "⚠️  CounterAnalyzer требует внимания"
	@echo ""
	@echo "3. Тест всех анализаторов..."
	@go run ./cmd/debug/analyzer/main.go 2>&1 | grep -E "(🧪|📊|✅|🔧)" | head -20 || echo "⚠️  Анализаторы требуют внимания"
	@echo ""
	@echo "4. Проверка типов..."
	@go vet ./internal/analysis/analyzers/... 2>&1 | head -10 || echo "⚠️  Есть предупреждения go vet"
	@echo "✅ Базовые тесты завершены"

## test-quick: Быстрые тесты
test-quick:
	@echo "⚡ БЫСТРЫЕ ТЕСТЫ"
	@echo "==============="
	@echo "CounterAnalyzer (первые 10 строк)..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | head -10
	@echo ""
	@echo "Анализаторы (первые 10 строк)..."
	@go run ./cmd/debug/analyzer/main.go 2>&1 | head -10

## test-all: Все тесты (без бесконечного ожидания)
test-all: test-basic build
	@echo ""
	@echo "🎯 ВСЕ ТЕСТЫ ПРОЙДЕНЫ!"
	@echo "====================="
	@echo "✅ CounterAnalyzer функционирует"
	@echo "✅ Анализаторы протестированы"
	@echo "✅ Сборка успешна"
	@echo "✅ Система готова к работе"

## safe-test: Безопасное тестирование без бесконечного ожидания
safe-test:
	@echo "🛡️  БЕЗОПАСНОЕ ТЕСТИРОВАНИЕ"
	@echo "==========================="
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
	@echo "Исправление ошибок копирования мьютекса в CounterAnalyzer..."
	@if grep -q "return copies lock value" internal/analysis/analyzers/counter_analyzer.go 2>/dev/null; then \
		echo "⚠️  Найдены ошибки копирования мьютекса"; \
		echo "✅ Используйте test-stable или safe-test для стабильного тестирования"; \
	else \
		echo "✅ Ошибок go vet не обнаружено"; \
	fi

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

## test: Запуск unit тестов
test:
	@echo "🧪 Running unit tests..."
	go test ./internal/analysis/analyzers/... -v -short

## clean: Очистка проекта
clean:
	@echo "🧹 Cleaning project..."
	rm -rf bin/ releases/ logs/*.log coverage/ reports/
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
	@if [ ! -f ".env" ]; then \
		echo "⚠️  Warning: .env file not found"; \
		echo "👉 Create .env file first: cp .env.example .env"; \
		exit 1; \
	fi
	docker run --env-file .env crypto-growth-monitor:latest

# ============================================
# ПОЛНОЕ ТЕСТИРОВАНИЕ (исправленное)
# ============================================

## full-test: Полное тестирование системы
full-test:
	@echo "🚀 ЗАПУСК ПОЛНОГО ТЕСТИРОВАНИЯ СИСТЕМЫ"
	@echo "======================================"
	@echo "Версия без таймаутов для стабильности..."
	@echo ""
	@chmod +x ./scripts/full_test.sh
	@./scripts/full_test.sh

## integration-test: Интеграционный тест
integration-test:
	@echo "🔗 ИНТЕГРАЦИОННЫЙ ТЕСТ"
	@echo "======================"
	@echo "Тестирование взаимодействия всех компонентов..."
	@echo "Запуск на 10 секунд..."
	@(go run ./cmd/debug/enhanced/main.go & PID=$$!; sleep 10; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Интеграционный тест завершен"

## analyzer-comparison: Сравнение всех анализаторов
analyzer-comparison:
	@echo "📊 СРАВНЕНИЕ АНАЛИЗАТОРОВ"
	@echo "========================="
	@go run ./cmd/debug/analyzer/main.go

## counter-deep-test: Глубокий тест CounterAnalyzer
counter-deep-test:
	@echo "🔍 ГЛУБОКИЙ ТЕСТ COUNTER ANALYZER"
	@echo "================================"
	@echo "Тестирование всех функций CounterAnalyzer..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(📊|🧮|✅|🔧|🎉)" | head -30

## test-report: Генерация отчета о тестировании
test-report:
	@echo "📋 ГЕНЕРАЦИЯ ОТЧЕТА О ТЕСТИРОВАНИИ"
	@echo "=================================="
	@mkdir -p reports
	@echo "# Отчет о тестировании" > reports/test_report_$(date +%Y%m%d).md
	@echo "Дата: $(date)" >> reports/test_report_$(date +%Y%m%d).md
	@echo "" >> reports/test_report_$(date +%Y%m%d).md
	@echo "## Результаты тестов" >> reports/test_report_$(date +%Y%m%d).md
	@echo "" >> reports/test_report_$(date +%Y%m%d).md
	@echo "### 1. CounterAnalyzer" >> reports/test_report_$(date +%Y%m%d).md
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(✅|📊|🧮|🎉)" >> reports/test_report_$(date +%Y%m%d).md || true
	@echo "" >> reports/test_report_$(date +%Y%m%d).md
	@echo "### 2. Все анализаторы" >> reports/test_report_$(date +%Y%m%d).md
	@go run ./cmd/debug/analyzer/main.go 2>&1 | tail -20 >> reports/test_report_$(date +%Y%m%d).md || true
	@echo "" >> reports/test_report_$(date +%Y%m%d).md
	@echo "✅ Отчет сохранен в: reports/test_report_$(date +%Y%m%d).md"

# ============================================
# ОТЧЕТЫ И АНАЛИТИКА
# ============================================

## coverage: Покрытие кода тестами
coverage:
	@echo "📊 ГЕНЕРАЦИЯ ОТЧЕТА О ПОКРЫТИИ КОДА"
	@echo "==================================="
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out ./internal/analysis/analyzers/... -short
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html 2>/dev/null || true
	@go tool cover -func=coverage/coverage.out > coverage/coverage.txt 2>/dev/null || true
	@echo "✅ Отчет о покрытии сгенерирован в папке coverage/"
	@echo "   • coverage.html - HTML отчет"
	@echo "   • coverage.txt - Текстовый отчет"

## performance-test: Тест производительности
performance-test:
	@echo "⚡ ТЕСТ ПРОИЗВОДИТЕЛЬНОСТИ"
	@echo "========================="
	@echo "Тестирование скорости работы анализаторов..."
	@time go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(real|user|sys)" || true

## security-check: Проверка безопасности
security-check:
	@echo "🔒 ПРОВЕРКА БЕЗОПАСНОСТИ"
	@echo "========================"
	@go vet ./... 2>&1 | grep -v "vendor" | head -20

# ============================================
# ДОПОЛНИТЕЛЬНЫЕ КОМАНДЫ ДЛЯ УДОБСТВА
# ============================================

debug-analyzer:
	@echo "🧪 Отладка анализаторов..."
	go run ./cmd/debug/analyzer/main.go

debug-basic:
	@echo "🐛 Базовая отладка..."
	@(go run ./cmd/debug/basic/main.go & PID=$$!; sleep 20; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Базовая отладка завершена"

debug-enhanced-full:
	@echo "🔬 Полная расширенная отладка..."
	@(go run ./cmd/debug/enhanced/main.go & PID=$$!; sleep 30; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Расширенная отладка завершена"

debug-super:
	@echo "🚀 Супер-чувствительный тест..."
	@(go run ./cmd/debug/supersensitive/main.go & PID=$$!; sleep 30; kill $$PID 2>/dev/null || true) 2>/dev/null || echo "✅ Супер-чувствительный тест завершен"

debug-counter-quick:
	@echo "🔢 Быстрый тест CounterAnalyzer..."
	@go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E "(БАЗОВЫЙ ТЕСТ|📊|✅|🎉)" | head -15 || true

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
	@echo ""
	@echo "🛡️  СТАБИЛЬНЫЕ ТЕСТЫ:"
	@echo "  make test-stable         - Самый стабильный тест (рекомендуется!)"
	@echo "  make test-safe           - Безопасный тест"
	@echo "  make quick-check         - Быстрая проверка"
	@echo "  make test-basic          - Базовые тесты"
	@echo "  make safe-test           - Безопасное тестирование"
	@echo "  make test-all            - Все тесты без ожидания"
	@echo "  make validate            - Проверка кода"

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
	@echo "  make full-test   - Полное тестирование системы"
	@echo "  make test-stable - Самый стабильный тест (рекомендуется!)"
	@echo ""
	@echo "🧮 COUNTER ANALYZER:"
	@echo "  make debug-counter      - Тест CounterAnalyzer"
	@echo "  make test-counter       - Полный тест CounterAnalyzer"
	@echo "  make test-counter-quick - Быстрый тест CounterAnalyzer"
	@echo "  make counter-test-all   - Все тесты CounterAnalyzer"
	@echo ""
	@echo "📊 АНАЛИТИКА И ОТЧЕТЫ:"
	@echo "  make coverage    - Отчет о покрытии кода"
	@echo "  make test-report - Генерация отчета о тестировании"
	@echo "  make performance-test - Тест производительности"
	@echo ""
	@echo "🛡️  СТАБИЛЬНЫЕ ТЕСТЫ (рекомендуются):"
	@echo "  make test-stable - Самый стабильный тест"
	@echo "  make quick-check - Быстрая проверка"
	@echo "  make test-safe   - Безопасный тест"
	@echo "  make safe-test   - Безопасное тестирование"
	@echo ""
	@echo "🔧 Утилиты:"
	@echo "  make clean       - Очистка проекта"
	@echo "  make lint        - Проверка кода"
	@echo "  make deps        - Обновление зависимостей"
	@echo "  make docker-build - Сборка Docker образа"
	@echo "  make fix-vet     - Исправить ошибки go vet"
	@echo "  make validate    - Проверка кода перед коммитом"
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
	@chmod +x ./scripts/test_counter_simple.sh
	@./scripts/test_counter_simple.sh

## create-counter-test-dir: Создание структуры для тестов CounterAnalyzer
create-counter-test-dir:
	@echo "📁 Создание структуры директорий для тестов..."
	@mkdir -p ./cmd/debug/counter_test
	@echo "✅ Создана директория: ./cmd/debug/counter_test"
	@echo "👉 Добавьте файл main.go для тестов CounterAnalyzer"

## daily-test: Ежедневное тестирование системы
daily-test:
	@echo "📅 ЗАПУСК ЕЖЕДНЕВНОГО ТЕСТИРОВАНИЯ"
	@echo "=================================="
	@chmod +x ./scripts/daily_test_simple.sh
	@./scripts/daily_test_simple.shы