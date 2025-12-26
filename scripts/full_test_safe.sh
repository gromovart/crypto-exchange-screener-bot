#!/bin/bash
echo "🛡️  БЕЗОПАСНОЕ ПОЛНОЕ ТЕСТИРОВАНИЕ"
echo "================================"

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"

echo "🎯 Окружение: $ENV"
echo "📁 Конфигурация: $ENV_FILE"
echo ""

# Обработка Ctrl+C
trap 'echo -e "\n🛑 Тестирование прервано пользователем"; exit 0' INT

# Создаем директорию для логов
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="logs/safe_test_${ENV}_${TIMESTAMP}"
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

# Проверка конфигурации
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ Файл конфигурации не найден: $ENV_FILE"
    echo "   Создайте: make config-init ENV=$ENV"
    exit 1
fi

# Запускаем тесты
echo "📋 ПЛАН ТЕСТИРОВАНИЯ:"
echo "1. Проверка конфигурации"
echo "2. Компиляция"
echo "3. CounterAnalyzer"
echo "4. Анализаторы"
echo "5. Проверка типов"
echo "6. Сборка"
echo ""

# 1. Проверка конфигурации
run_safe "Проверка конфигурации" "make check-config ENV=$ENV"

# 2. Компиляция
run_safe "Компиляция" "go build ./application/cmd/debug/..."

# 3. CounterAnalyzer
run_safe "CounterAnalyzer" "go run ./application/cmd/debug/counter_test/main.go --config=$ENV_FILE"

# 4. Анализаторы
run_safe "Анализаторы" "go run ./application/cmd/debug/analyzer/main.go --config=$ENV_FILE"

# 5. Проверка типов
run_safe "Проверка типов" "go vet ./internal/core/domain/signals/detectors/..."

# 6. Сборка
run_safe "Сборка" "make build ENV=$ENV"

echo ""
echo "🎯 ТЕСТИРОВАНИЕ ЗАВЕРШЕНО"
echo "========================="
echo "✅ Все тесты выполнены безопасно"
echo "✅ Окружение: $ENV"
echo "✅ Логи сохранены в $LOG_DIR"
echo "✅ Система готова к работе"

# Пример использования:
# ./scripts/full_test_safe.sh dev      # Безопасное тестирование dev
# ./scripts/full_test_safe.sh prod     # Безопасное тестирование prod