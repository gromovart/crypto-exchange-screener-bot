# Makefile в корне проекта
.PHONY: bot signals launcher build-all clean help

# Сборка всех программ
build-all:
	@echo "🔨 Сборка всех программ..."
	mkdir -p bin
	go build -o bin/bot cmd/bot/main.go
	go build -o bin/signals cmd/signals/main.go
	go build -o bin/launcher cmd/launcher/main.go
	@echo "✅ Все программы собраны в папке bin/"

# Запуск полного бота
bot:
	@echo "🚀 Запуск полного бота..."
	go run cmd/bot/main.go

# Запуск только сигналов
signals:
	@echo "📈 Запуск режима только сигналов..."
	go run cmd/signals/main.go

# Запуск лаунчера
launcher:
	@echo "🎮 Запуск лаунчера..."
	go run cmd/launcher/main.go $(filter-out $@,$(MAKECMDGOALS))

# Демо режим (тестовые сигналы)
demo:
	@echo "🎭 Запуск демо режима..."
	@echo "══════════════════════════════════════════════════"
	@echo "              ДЕМО РЕЖИМ - ТЕСТ СИГНАЛОВ          "
	@echo "══════════════════════════════════════════════════"
	@echo ""
	@echo "⚫ Bybit - 15 мин - BTCUSDT"
	@echo "🟢 Pump: +2.55%"
	@echo "📡 Signal 24h: 1"
	@echo ""
	@echo "⚫ Bybit - 15 мин - ETHUSDT"
	@echo "🔴 Dump: -1.85%"
	@echo "📡 Signal 24h: 1"
	@echo ""
	@echo "✅ Демо завершено"

# Очистка
clean:
	@echo "🧹 Очистка..."
	rm -rf bin/ logs/
	@echo "✅ Очистка завершена"

# Тест системы
test:
	@echo "🧪 Тестирование системы..."
	go test ./internal/... -v

# Помощь
help:
	@echo "Crypto Exchange Screener Bot - Makefile"
	@echo ""
	@echo "Команды:"
	@echo "  make build-all   - Собрать все программы в bin/"
	@echo "  make bot         - Запустить полный бот"
	@echo "  make signals     - Запустить только мониторинг сигналов"
	@echo "  make launcher    - Запустить лаунчер (используйте с аргументами)"
	@echo "  make demo        - Демо режим с тестовыми сигналами"
	@echo "  make clean       - Очистить бинарные файлы и логи"
	@echo "  make test        - Запустить тесты"
	@echo "  make help        - Эта справка"
	@echo ""
	@echo "Примеры использования лаунчера:"
	@echo "  make launcher full"
	@echo "  make launcher signals"
	@echo "  make launcher help"