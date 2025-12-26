#!/bin/bash
echo "🤖 ЗАПУСК TELEGRAM БОТА В ЛОКАЛЬНОМ РЕЖИМЕ"
echo "========================================"
echo ""

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"

echo "🎯 Окружение: $ENV"
echo "📁 Конфигурация: $ENV_FILE"
echo ""

# Проверка наличия .env файла
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ Файл конфигурации не найден: $ENV_FILE"
    echo "   Создайте: make config-init ENV=$ENV"
    echo "   Отредактируйте $ENV_FILE и добавьте TG_API_KEY и TG_CHAT_ID"
    exit 1
fi

# Проверка настроек Telegram
TOKEN=$(grep "TG_API_KEY=" "$ENV_FILE" | cut -d= -f2)
CHAT_ID=$(grep "TG_CHAT_ID=" "$ENV_FILE" | cut -d= -f2)

if [ -z "$TOKEN" ] || [ "$TOKEN" = "your_telegram_bot_token_here" ]; then
    echo "❌ TG_API_KEY не настроен в $ENV_FILE"
    echo "   Получите токен у @BotFather в Telegram"
    exit 1
fi

if [ -z "$CHAT_ID" ] || [ "$CHAT_ID" = "your_telegram_chat_id_here" ]; then
    echo "❌ TG_CHAT_ID не настроен в $ENV_FILE"
    echo "   Узнайте ваш Chat ID через @userinfobot в Telegram"
    exit 1
fi

echo "✅ Конфигурация Telegram проверена:"
echo "   Bot Token: ${TOKEN:0:10}...${TOKEN: -10}"
echo "   Chat ID: $CHAT_ID"
echo ""

# Создаем временный .env файл для локального запуска
TEMP_ENV_FILE="$ENV_FILE.local"
cp "$ENV_FILE" "$TEMP_ENV_FILE"
echo "" >> "$TEMP_ENV_FILE"
echo "# Локальный режим (добавлено скриптом)" >> "$TEMP_ENV_FILE"
echo "HTTP_ENABLED=false" >> "$TEMP_ENV_FILE"
echo "TEST_MODE=false" >> "$TEMP_ENV_FILE"
echo "POLLING_INTERVAL=1s" >> "$TEMP_ENV_FILE"

echo "🔧 Настройка локального режима..."
echo "   Отключаем HTTP порт для использования polling"
echo "   Для работы меню используйте команду /start в Telegram"
echo ""

echo "🚀 Запуск бота..."
echo "📌 Откройте Telegram и найдите своего бота"
echo "📌 Отправьте команду /start"
echo "📌 Используйте меню кнопок для управления"
echo ""
echo "🔄 Бот будет опрашивать Telegram API каждую секунду"
echo "🛑 Для остановки нажмите Ctrl+C"
echo ""

# Запускаем бота
go run ./application/cmd/bot/main.go --config="$TEMP_ENV_FILE" --log-level=debug

# Очистка
rm -f "$TEMP_ENV_FILE"
echo ""
echo "✅ Бот остановлен"

# Пример использования:
# ./scripts/run_bot_local.sh dev      # Локальный запуск dev бота
# ./scripts/run_bot_local.sh prod     # Локальный запуск prod бота