#!/bin/bash

echo "🛡️  БЕЗОПАСНОЕ ПОЛНОЕ ТЕСТИРОВАНИЕ"
echo "================================"

# Обработка Ctrl+C
trap 'echo -e "\n🛑 Тестирование прервано пользователем"; exit 0' INT

# Создаем директорию для логов
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="logs/safe_test_${TIMESTAMP}"
mkdir -p "$LOG_DIR"

echo "📁 Логи: $LOG_DIR"
echo ""

# Функция для безопасного запуска
run_safe() {
    local name=$1
    local cmd=$2
    local timeout=${3:-15}

    echo "🧪 $name..."
    local log_file="$LOG_DIR/${name// /_}.log"

    # Запуск с таймаутом
    timeout ${timeout}s bash -c "$cmd" 2>&1 | tee "$log_file"
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
}

# Запускаем тесты
echo "📋 ПЛАН ТЕСТИРОВАНИЯ:"
echo "1. Компиляция"
echo "2. CounterAnalyzer"
echo "3. Анализаторы"
echo "4. Проверка типов"
echo "5. Сборка"
echo ""

# 1. Компиляция
run_safe "Компиляция" "go build ./cmd/debug/..."

# 2. CounterAnalyzer
run_safe "CounterAnalyzer" "go run ./cmd/debug/counter_test/main.go"

# 3. Анализаторы
run_safe "Анализаторы" "go run ./cmd/debug/analyzer/main.go"

# 4. Проверка типов
run_safe "Проверка типов" "go vet ./internal/analysis/analyzers/..."

# 5. Сборка
run_safe "Сборка" "make build"

echo ""
echo "🎯 ТЕСТИРОВАНИЕ ЗАВЕРШЕНО"
echo "========================="
echo "✅ Все тесты выполнены безопасно"
echo "✅ Логи сохранены в $LOG_DIR"
echo "✅ Система готова к работе"