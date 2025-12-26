#!/bin/bash
echo "🤖 НАСТРОЙКА TELEGRAM БОТА ДЛЯ ТЕСТИРОВАНИЯ"
echo "=========================================="
echo ""

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"

echo "🎯 Настройка окружения: $ENV"
echo "📁 Файл конфигурации: $ENV_FILE"
echo ""

# Проверка существования директории окружения
if [ ! -d "configs/$ENV" ]; then
    echo "Создаю директорию окружения..."
    mkdir -p "configs/$ENV"
fi

# Проверка существования .env файла
if [ ! -f "$ENV_FILE" ]; then
    echo "⚠️  Файл конфигурации не найден"
    echo "Создаю из примера..."

    if [ -f "configs/example/.env" ]; then
        cp configs/example/.env "$ENV_FILE"
        echo "✅ Создан файл $ENV_FILE (из example)"
    elif [ -f ".env.example" ]; then
        cp .env.example "$ENV_FILE"
        echo "✅ Создан файл $ENV_FILE (из .env.example)"
    else
        echo "❌ Не удалось найти файл-шаблон"
        echo "   Создайте configs/example/.env или .env.example"
        exit 1
    fi
fi

echo ""
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
echo "📋 ШАГ 3: НАСТРОЙКА $ENV_FILE"
echo "------------------------------"

# Создаем резервную копию
BACKUP_FILE="$ENV_FILE.backup.$(date +%Y%m%d_%H%M%S)"
cp "$ENV_FILE" "$BACKUP_FILE"
echo "✅ Создана резервная копия: $BACKUP_FILE"

# Обновляем .env файл
if [ -f "$ENV_FILE" ]; then
    # Удаляем старые настройки Telegram
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' '/TELEGRAM_ENABLED/d' "$ENV_FILE"
        sed -i '' '/TG_API_KEY/d' "$ENV_FILE"
        sed -i '' '/TG_CHAT_ID/d' "$ENV_FILE"
        sed -i '' '/COUNTER_ANALYZER_ENABLED/d' "$ENV_FILE"
        sed -i '' '/COUNTER_NOTIFICATION_ENABLED/d' "$ENV_FILE"
    else
        # Linux
        sed -i '/TELEGRAM_ENABLED/d' "$ENV_FILE"
        sed -i '/TG_API_KEY/d' "$ENV_FILE"
        sed -i '/TG_CHAT_ID/d' "$ENV_FILE"
        sed -i '/COUNTER_ANALYZER_ENABLED/d' "$ENV_FILE"
        sed -i '/COUNTER_NOTIFICATION_ENABLED/d' "$ENV_FILE"
    fi

    # Добавляем новые настройки
    echo "" >> "$ENV_FILE"
    echo "# Telegram Bot Settings" >> "$ENV_FILE"
    echo "TELEGRAM_ENABLED=true" >> "$ENV_FILE"
    echo "TG_API_KEY=$BOT_TOKEN" >> "$ENV_FILE"
    echo "TG_CHAT_ID=$CHAT_ID" >> "$ENV_FILE"
    echo "TELEGRAM_NOTIFY_GROWTH=true" >> "$ENV_FILE"
    echo "TELEGRAM_NOTIFY_FALL=true" >> "$ENV_FILE"
    echo "TELEGRAM_GROWTH_THRESHOLD=0.5" >> "$ENV_FILE"
    echo "TELEGRAM_FALL_THRESHOLD=0.5" >> "$ENV_FILE"
    echo "MESSAGE_FORMAT=compact" >> "$ENV_FILE"
    echo "" >> "$ENV_FILE"
    echo "# Counter Analyzer Settings" >> "$ENV_FILE"
    echo "COUNTER_ANALYZER_ENABLED=true" >> "$ENV_FILE"
    echo "COUNTER_NOTIFICATION_ENABLED=true" >> "$ENV_FILE"
    echo "COUNTER_BASE_PERIOD_MINUTES=1" >> "$ENV_FILE"
    echo "COUNTER_ANALYSIS_PERIOD=15m" >> "$ENV_FILE"
    echo "COUNTER_CHART_PROVIDER=coinglass" >> "$ENV_FILE"

    echo "✅ Настройки добавлены в $ENV_FILE"
else
    echo "❌ Файл $ENV_FILE не найден"
    exit 1
fi

echo ""
echo "📋 ШАГ 4: ПРОВЕРКА НАСТРОЕК"
echo "--------------------------"
echo "Текущие настройки Telegram в $ENV_FILE:"
grep -E "(TELEGRAM|TG_|COUNTER_)" "$ENV_FILE"

echo ""
echo "📋 ШАГ 5: ЗАПУСК ТЕСТА"
echo "---------------------"
echo "Для запуска теста выполните:"
echo "  make real-telegram-test ENV=$ENV"
echo ""
echo "Или напрямую:"
echo "  go run ./application/cmd/debug/real_telegram_test/main.go --config=$ENV_FILE --debug"
echo ""
echo "🎯 Готово! Теперь можно тестировать бота."

# Пример использования:
# ./scripts/setup_telegram_test.sh dev      # Настройка Telegram для dev
# ./scripts/setup_telegram_test.sh prod     # Настройка Telegram для prod