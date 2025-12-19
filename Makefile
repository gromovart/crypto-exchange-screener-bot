# Makefile

.PHONY: debug debug-enhanced debug-diagnostic analyzer-test debug-super-sensitive debug-all run

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

debug-all:
	@echo "🚀 Полный набор тестов..."
	@echo ""
	@echo "1. Тест анализаторов..."
	@$(MAKE) analyzer-test
	@echo ""
	@echo "2. Диагностика системы..."
	@$(MAKE) debug-diagnostic
	@echo ""
	@echo "3. Расширенная отладка..."
	@$(MAKE) debug-enhanced
	@echo ""
	@echo "4. Супер-чувствительный тест..."
	@$(MAKE) debug-super-sensitive

run:
	@echo "🚀 Запуск основного бота..."
	go run ./cmd/bot/main.go

# Дополнительные команды для удобства
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

list-debug:
	@echo "📁 Доступные отладочные программы:"
	@echo "  make analyzer-test     - Тест анализаторов"
	@echo "  make debug             - Базовая отладка"
	@echo "  make debug-diagnostic  - Глубокая диагностика"
	@echo "  make debug-enhanced    - Расширенная отладка"
	@echo "  make debug-super-sensitive - Супер-чувствительный"
	@echo "  make debug-all         - Все тесты сразу"
	@echo "  make run               - Запуск основного бота"