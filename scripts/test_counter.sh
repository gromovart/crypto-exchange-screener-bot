#!/bin/bash

echo "🧪 ЗАПУСК ТЕСТОВ COUNTER ANALYZER"
echo "=========================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода секции
print_section() {
    echo ""
    echo "${BLUE}$1${NC}"
    echo "${BLUE}$(printf '=%.0s' $(seq 1 ${#1}))${NC}"
}

# Функция для проверки успешности
check_success() {
    if [ $? -eq 0 ]; then
        echo "${GREEN}✅ Успешно${NC}"
    else
        echo "${RED}❌ Ошибка${NC}"
    fi
}

# Создаем лог файл
LOG_FILE="logs/counter_test_$(date +%Y%m%d_%H%M%S).log"
mkdir -p logs

print_section "1. БАЗОВЫЙ ТЕСТ COUNTER ANALYZER"
echo "Запуск: go run cmd/debug/analyzer/main.go"
go run cmd/debug/analyzer/main.go 2>&1 | tee -a "$LOG_FILE" | grep -A 30 "ТЕСТ COUNTER ANALYZER"
check_success

print_section "2. ПОЛНЫЙ ТЕСТ COUNTER ANALYZER"
echo "Запуск: go run cmd/debug/counter_test/main.go"
go run cmd/debug/counter_test/main.go 2>&1 | tee -a "$LOG_FILE"
check_success

print_section "3. ИНТЕГРАЦИОННЫЙ ТЕСТ"
echo "Запуск: go run cmd/debug/enhanced/main.go"
go run cmd/debug/enhanced/main.go 2>&1 | tee -a "$LOG_FILE" | grep -B5 -A40 "ТЕСТ 3: COUNTER ANALYZER"
check_success

print_section "4. СТАТИСТИКА ТЕСТОВ"
echo "Проверка лог файла: $LOG_FILE"
if [ -f "$LOG_FILE" ]; then
    echo "📊 Статистика лог файла:"
    echo "   • Общий размер: $(wc -l < "$LOG_FILE") строк"
    echo "   • Ошибки: $(grep -c "❌\|Ошибка\|ERROR" "$LOG_FILE")"
    echo "   • Предупреждения: $(grep -c "⚠️\|Warning\|WARN" "$LOG_FILE")"
    echo "   • Успехи: $(grep -c "✅\|Успешно\|SUCCESS" "$LOG_FILE")"

    # Последние 10 строк лога
    echo "   • Последние записи:"
    tail -10 "$LOG_FILE" | sed 's/^/     /'
else
    echo "${YELLOW}⚠️  Лог файл не найден${NC}"
fi

print_section "РЕЗУЛЬТАТ"
echo "${GREEN}✅ Тесты CounterAnalyzer завершены${NC}"
echo "Логи сохранены в: $LOG_FILE"