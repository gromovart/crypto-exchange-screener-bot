#!/bin/bash

echo "🤖 Тестирование Telegram интеграции..."
echo ""

# Проверяем переменные окружения
if [ -z "$TG_API_KEY" ]; then
    echo "❌ TG_API_KEY не установлен!"
    echo "Получите API ключ у @BotFather и установите:"
    echo "export TG_API_KEY='ваш_ключ'"
    exit 1
fi

if [ -z "$TG_CHAT_ID" ]; then
    echo "❌ TG_CHAT_ID не установлен!"
    echo "Получите Chat ID у @userinfobot и установите:"
    echo "export TG_CHAT_ID='ваш_chat_id'"
    exit 1
fi

echo "🔧 Настройки Telegram:"
echo "   API Key: ${TG_API_KEY:0:10}..."
echo "   Chat ID: $TG_CHAT_ID"
echo ""

# Создаем тестовый .env файл
cat > .env.telegram << EOF
USE_TESTNET=false
BYBIT_API_KEY=$BYBIT_API_KEY
BYBIT_SECRET_KEY=$BYBIT_SECRET_KEY
FUTURES_CATEGORY=linear
SYMBOL_FILTER=BTC,ETH
MAX_SYMBOLS_TO_MONITOR=10
MIN_VOLUME_FILTER=100000
GROWTH_PERIODS=5
GROWTH_THRESHOLD=5.0  # Высокий порог для теста
FALL_THRESHOLD=5.0
CHECK_CONTINUITY=false
SIGNAL_FILTERS_ENABLED=false
UPDATE_INTERVAL=30
HTTP_ENABLED=false
TG_API_KEY=$TG_API_KEY
TG_CHAT_ID=$TG_CHAT_ID
TELEGRAM_ENABLED=true
TELEGRAM_NOTIFY_GROWTH=true
TELEGRAM_NOTIFY_FALL=true
MESSAGE_FORMAT=detailed
EOF

echo "📋 Запуск теста..."
echo "Бот будет работать 60 секунд и отправит тестовое сообщение"
echo ""

cp .env.telegram .env
timeout 60 go run cmd/bot/main.go

# Очистка
rm -f .env.telegram .env
echo ""
echo "✅ Тест завершен. Проверьте Telegram чат на наличие сообщений."