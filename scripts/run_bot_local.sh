#!/bin/bash

echo "🤖 ЗАПУСК TELEGRAM БОТА В ЛОКАЛЬНОМ РЕЖИМЕ"
echo "========================================"
echo ""

# Проверка наличия .env файла
if [ ! -f ".env" ]; then
    echo "❌ Файл .env не найден!"
    echo "   Создайте: cp .env.example .env"
    echo "   Отредактируйте .env и добавьте TG_API_KEY и TG_CHAT_ID"
    exit 1
fi

# Проверка настроек Telegram
TOKEN=$(grep "TG_API_KEY=" .env | cut -d= -f2)
CHAT_ID=$(grep "TG_CHAT_ID=" .env | cut -d= -f2)

if [ -z "$TOKEN" ] || [ "$TOKEN" = "your_telegram_bot_token_here" ]; then
    echo "❌ TG_API_KEY не настроен в .env файле"
    echo "   Получите токен у @BotFather в Telegram"
    exit 1
fi

if [ -z "$CHAT_ID" ] || [ "$CHAT_ID" = "your_telegram_chat_id_here" ]; then
    echo "❌ TG_CHAT_ID не настроен в .env файле"
    echo "   Узнайте ваш Chat ID через @userinfobot в Telegram"
    exit 1
fi

echo "✅ Конфигурация Telegram проверена:"
echo "   Bot Token: ${TOKEN:0:10}...${TOKEN: -10}"
echo "   Chat ID: $CHAT_ID"
echo ""

# Отключаем HTTP порт для локального запуска (используем polling)
echo "🔧 Настройка локального режима..."
echo "   Отключаем HTTP порт для использования polling"
echo "   Для работы меню используйте команду /start в Telegram"
echo ""

# Создаем временный .env файл для локального запуска
cp .env .env.local
echo "HTTP_ENABLED=false" >> .env.local
echo "TEST_MODE=false" >> .env.local

echo "🚀 Запуск бота..."
echo "📌 Откройте Telegram и найдите своего бота"
echo "📌 Отправьте команду /start"
echo "📌 Используйте меню кнопок для управления"
echo ""
echo "🔄 Бот будет опрашивать Telegram API каждую секунду"
echo "🛑 Для остановки нажмите Ctrl+C"
echo ""

# Запускаем бота
go run cmd/bot/main.go --config=.env.local --log-level=debug

# Очистка
rm -f .env.local
echo ""
echo "✅ Бот остановлен"