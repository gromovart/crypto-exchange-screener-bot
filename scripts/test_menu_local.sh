#!/bin/bash
echo "📱 ТЕСТИРОВАНИЕ TELEGRAM МЕНЮ В ЛОКАЛЬНОМ РЕЖИМЕ"
echo "=============================================="
echo ""

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"

echo "🎯 Окружение: $ENV"
echo "📁 Конфигурация: $ENV_FILE"
echo ""

# Проверка конфигурации
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ Файл конфигурации не найден: $ENV_FILE"
    echo "   Создайте: make config-init ENV=$ENV"
    echo "   Добавьте TG_API_KEY и TG_CHAT_ID"
    exit 1
fi

echo "1. Проверка конфигурации..."
TOKEN=$(grep "TG_API_KEY=" "$ENV_FILE" | cut -d= -f2)
CHAT_ID=$(grep "TG_CHAT_ID=" "$ENV_FILE" | cut -d= -f2)

if [ -z "$TOKEN" ] || [ "$TOKEN" = "your_telegram_bot_token_here" ]; then
    echo "❌ TG_API_KEY не настроен в $ENV_FILE"
    exit 1
fi

if [ -z "$CHAT_ID" ] || [ "$CHAT_ID" = "your_telegram_chat_id_here" ]; then
    echo "❌ TG_CHAT_ID не настроен в $ENV_FILE"
    exit 1
fi

echo "✅ Конфигурация Telegram проверена"
echo ""

# Создаем временный конфиг для тестового режима
TEMP_ENV_FILE="$ENV_FILE.test_menu"
cp "$ENV_FILE" "$TEMP_ENV_FILE"
echo "" >> "$TEMP_ENV_FILE"
echo "# Тестовый режим меню" >> "$TEMP_ENV_FILE"
echo "HTTP_ENABLED=false" >> "$TEMP_ENV_FILE"

echo "2. Запуск в тестовом режиме..."
echo "   (бот работает, но не отправляет реальные сообщения)"
echo ""

echo "📌 ОТКРОЙТЕ TELEGRAM И:"
echo "   1. Найдите своего бота"
echo "   2. Отправьте /start"
echo "   3. Нажимайте кнопки меню"
echo ""

echo "🔄 Запуск бота..."
echo "🛑 Для остановки нажмите Ctrl+C"
echo ""

# Запуск в тестовом режиме
TEST_MODE=true go run ./application/cmd/bot/main.go --config="$TEMP_ENV_FILE" --log-level=debug

# Очистка
rm -f "$TEMP_ENV_FILE"
echo ""
echo "✅ Тестирование меню завершено"

# Пример использования:
# ./scripts/test_menu_local.sh dev      # Тестирование меню dev
# ./scripts/test_menu_local.sh prod     # Тестирование меню prod