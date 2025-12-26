#!/bin/bash
# scripts/setup_telegram_test.sh

echo "🤖 НАСТРОЙКА TELEGRAM БОТА ДЛЯ ТЕСТИРОВАНИЯ"
echo "=========================================="
echo ""

# Проверка существования .env файла
if [ ! -f ".env" ]; then
    echo "⚠️  Файл .env не найден"
    echo "Создаю из примера..."
    cp .env.example .env 2>/dev/null || echo "❌ Не удалось создать .env"
    echo "✅ Создан файл .env"
fi

echo "📋 ШАГ 1: СОЗДАНИЕ TELEGRAM БОТА"
echo "--------------------------------"
echo "1. Откройте Telegram"
echo "2. Найдите @BotFather"
echo "3. Отправьте команду: /newbot"
echo "4. Укажите имя бота (например: Crypto Signal Test)"
echo "5. Укажите username бота (должен оканчиваться на 'bot')"
echo "6. Скопируйте токен вида: 1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
echo ""

read -p "Введите токен бота: " BOT_TOKEN

echo ""
echo "📋 ШАГ 2: ПОЛУЧЕНИЕ CHAT ID"
echo "---------------------------"
echo "1. Найдите @userinfobot в Telegram"
echo "2. Отправьте любую команду (/start)"
echo "3. Скопируйте ваш Chat ID (число)"
echo ""

read -p "Введите ваш Chat ID: " CHAT_ID

echo ""
echo "📋 ШАГ 3: НАСТРОЙКА .env ФАЙЛА"
echo "------------------------------"

# Обновляем .env файл
if [ -f ".env" ]; then
    # Удаляем старые настройки Telegram
    sed -i '' '/TELEGRAM_ENABLED/d' .env 2>/dev/null || sed -i '/TELEGRAM_ENABLED/d' .env
    sed -i '' '/TG_API_KEY/d' .env 2>/dev/null || sed -i '/TG_API_KEY/d' .env
    sed -i '' '/TG_CHAT_ID/d' .env 2>/dev/null || sed -i '/TG_CHAT_ID/d' .env
    sed -i '' '/COUNTER_ANALYZER_ENABLED/d' .env 2>/dev/null || sed -i '/COUNTER_ANALYZER_ENABLED/d' .env
    sed -i '' '/COUNTER_NOTIFICATION_ENABLED/d' .env 2>/dev/null || sed -i '/COUNTER_NOTIFICATION_ENABLED/d' .env

    # Добавляем новые настройки
    echo "" >> .env
    echo "# Telegram Bot Settings" >> .env
    echo "TELEGRAM_ENABLED=true" >> .env
    echo "TG_API_KEY=$BOT_TOKEN" >> .env
    echo "TG_CHAT_ID=$CHAT_ID" >> .env
    echo "TELEGRAM_NOTIFY_GROWTH=true" >> .env
    echo "TELEGRAM_NOTIFY_FALL=true" >> .env
    echo "TELEGRAM_GROWTH_THRESHOLD=0.5" >> .env
    echo "TELEGRAM_FALL_THRESHOLD=0.5" >> .env
    echo "MESSAGE_FORMAT=compact" >> .env
    echo "" >> .env
    echo "# Counter Analyzer Settings" >> .env
    echo "COUNTER_ANALYZER_ENABLED=true" >> .env
    echo "COUNTER_NOTIFICATION_ENABLED=true" >> .env
    echo "COUNTER_BASE_PERIOD_MINUTES=1" >> .env
    echo "COUNTER_ANALYSIS_PERIOD=15m" >> .env
    echo "COUNTER_CHART_PROVIDER=coinglass" >> .env

    echo "✅ Настройки добавлены в .env файл"
else
    echo "❌ Файл .env не найден"
    exit 1
fi

echo ""
echo "📋 ШАГ 4: ПРОВЕРКА НАСТРОЕК"
echo "--------------------------"
echo "Текущие настройки Telegram:"
grep -E "(TELEGRAM|TG_|COUNTER_)" .env

echo ""
echo "📋 ШАГ 5: ЗАПУСК ТЕСТА"
echo "---------------------"
echo "Для запуска теста выполните:"
echo "  make real-telegram-test"
echo ""
echo "Или напрямую:"
echo "  go run ./application/cmd/debug/real_telegram_test/main.go --debug"
echo ""
echo "🎯 Готово! Теперь можно тестировать бота."