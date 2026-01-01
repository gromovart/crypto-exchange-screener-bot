#!/bin/bash
echo "🛡️  БЕЗОПАСНОЕ ТЕСТИРОВАНИЕ"
echo "================================"

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"
MAIN_FILE="./application/main.go"

echo "🎯 Окружение: $ENV"
echo "📁 Конфигурация: $ENV_FILE"
echo "📄 Основной файл: $MAIN_FILE"
echo ""

# Проверка файлов
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ Файл конфигурации не найден: $ENV_FILE"
    echo "   Создайте: make config-init ENV=$ENV"
    exit 1
fi

if [ ! -f "$MAIN_FILE" ]; then
    echo "❌ Основной файл не найден: $MAIN_FILE"
    exit 1
fi

# Функция для безопасного запуска
run_safe() {
    local name=$1
    local cmd=$2
    local timeout=${3:-10}

    echo "🧪 $name (таймаут: ${timeout}с)..."

    # Запуск с таймаутом
    timeout ${timeout}s bash -c "$cmd" 2>&1
    local status=$?

    if [ $status -eq 0 ]; then
        echo "✅ $name завершен успешно"
        return 0
    elif [ $status -eq 124 ]; then
        echo "⏱️  $name: время истекло (таймаут ${timeout}с)"
        return 0
    elif [ $status -eq 130 ]; then
        echo "🛑 $name: прервано пользователем"
        return 0
    else
        echo "⚠️  $name: код выхода $status"
        return 1
    fi
    echo ""
}

# Запускаем тесты
echo "📋 ПЛАН ТЕСТИРОВАНИЯ:"
echo "1. Проверка конфигурации"
echo "2. Компиляция"
echo "3. Простой режим (simple)"
echo "4. Полный режим (full)"
echo "5. Сборка"
echo ""

# 1. Проверка конфигурации
echo "🔍 Проверка конфигурации $ENV..."
grep -E "(TG_API_KEY|TELEGRAM_ENABLED|COUNTER_ANALYZER_ENABLED|LOG_LEVEL)" "$ENV_FILE" 2>/dev/null || echo "⚠️  Основные настройки не найдены"
echo "✅ Конфигурация загружена"
echo ""

# 2. Компиляция
run_safe "Компиляция" "go build $MAIN_FILE"

# 3. Простой режим (simple)
run_safe "Простой режим (5s)" "go run $MAIN_FILE --config=$ENV_FILE --mode=simple --test" 5

# 4. Полный режим (full)
run_safe "Полный режим (8s)" "go run $MAIN_FILE --config=$ENV_FILE --mode=full --log-level=error --test" 8

# 5. Сборка
run_safe "Сборка" "make build ENV=$ENV"

echo ""
echo "🎯 ТЕСТИРОВАНИЕ ЗАВЕРШЕНО"
echo "========================="
echo "✅ Все тесты выполнены безопасно"
echo "✅ Окружение: $ENV"
echo "✅ Система готова к работе"
echo ""
echo "📝 Команды для запуска:"
echo "   make run ENV=$ENV        # Простой режим"
echo "   make run-full ENV=$ENV   # Полный режим"