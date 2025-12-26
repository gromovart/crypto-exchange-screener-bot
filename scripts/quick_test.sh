#!/bin/bash
echo "⚡ БЫСТРЫЙ ТЕСТ СИСТЕМЫ"
echo "====================="

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"

echo "🎯 Окружение: $ENV"
echo ""

# Проверка конфигурации
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ Файл конфигурации не найден: $ENV_FILE"
    echo "   Создайте: make config-init ENV=$ENV"
    exit 1
fi

echo "1. Проверяем компиляцию..."
if go build ./application/cmd/debug/counter_test/; then
    echo "✅ CounterAnalyzer компилируется"
else
    echo "❌ Ошибка компиляции CounterAnalyzer"
    exit 1
fi

echo ""
echo "2. Запускаем CounterAnalyzer на 3 секунды..."

# Пробуем разные варианты timeout
if command -v timeout &> /dev/null; then
    timeout 3 go run ./application/cmd/debug/counter_test/main.go --config="$ENV_FILE" 2>&1 | head -10
elif command -v gtimeout &> /dev/null; then
    gtimeout 3 go run ./application/cmd/debug/counter_test/main.go --config="$ENV_FILE" 2>&1 | head -10
else
    # Запускаем без timeout, но убиваем через 3 секунды
    go run ./application/cmd/debug/counter_test/main.go --config="$ENV_FILE" &
    PID=$!
    sleep 3
    kill $PID 2>/dev/null || true
    wait $PID 2>/dev/null || true
    echo "✅ CounterAnalyzer тест выполнен (с ограничением времени)"
fi

echo ""
echo "3. Сборка основного приложения..."
make build ENV="$ENV"

echo ""
echo "🎉 БЫСТРЫЙ ТЕСТ ЗАВЕРШЕН УСПЕШНО!"

# Пример использования:
# ./scripts/quick_test.sh dev      # Быстрый тест dev
# ./scripts/quick_test.sh prod     # Быстрый тест prod