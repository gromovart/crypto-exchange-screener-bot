# Makefile для Crypto Exchange Screener Bot

.PHONY: all bot signals test clean

all: bot

bot:
	go run cmd/bot/main.go

growth:
	go run cmd/signals/main.go

build-bot:
	go build -o bin/bot cmd/bot/main.go

build-growth:
	go build -o bin/growth cmd/signals/main.go

build: build-bot build-growth

test:
	go test ./...

clean:
	rm -rf bin/ logs/*.log

run-debug:
	./debug_run.sh

run-test:
	./test_signal.sh

install:
	go mod download


# Telegram команды
telegram-test:
	@echo "🤖 Тестирование Telegram бота..."
	@./test_telegram.sh

telegram-setup:
	@echo "🔧 Настройка Telegram бота..."
	@echo ""
	@echo "1. Создайте бота через @BotFather"
	@echo "2. Получите API ключ"
	@echo "3. Получите Chat ID через @userinfobot"
	@echo "4. Установите переменные окружения:"
	@echo "   export TG_API_KEY='ваш_ключ'"
	@echo "   export TG_CHAT_ID='ваш_chat_id'"
	@echo "5. Запустите: make telegram-test"

telegram-webhook:
	@echo "🌐 Настройка Telegram webhook..."
	@echo "Убедитесь что у вас есть:"
	@echo "1. Публичный HTTPS домен"
	@echo "2. Открытый порт 8443"
	@echo "3. В .env укажите:"
	@echo "   TELEGRAM_WEBHOOK_URL=https://ваш-домен.com"
	@echo "   TELEGRAM_WEBHOOK_PORT=8443"