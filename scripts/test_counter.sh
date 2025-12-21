#!/bin/bash

echo "🧪 ТЕСТИРОВАНИЕ COUNTER ANALYZER"
echo "================================"
echo ""

# Создаем директорию для логов
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="logs/counter_test_${TIMESTAMP}"
mkdir -p "$LOG_DIR"

# Цвета
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Функция для вывода секции
print_section() {
    echo ""
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}$(printf '%.0s=' ${1//?/=})${NC}"
}

# 1. Базовый тест
print_section "1. БАЗОВЫЙ ТЕСТ COUNTER ANALYZER"
echo "Запуск: make debug-counter"
if make debug-counter 2>&1 | tee "$LOG_DIR/01_basic_test.log" | grep -q "Тестирование CounterAnalyzer завершено"; then
    echo -e "${GREEN}✅ Базовый тест пройден${NC}"
else
    echo -e "${RED}❌ Базовый тест не пройден${NC}"
fi

# 2. Полный тест
print_section "2. ПОЛНЫЙ ТЕСТ COUNTER ANALYZER"
echo "Запуск: make test-counter"
if make test-counter 2>&1 | tee "$LOG_DIR/02_full_test.log" | tail -5 | grep -q "CounterAnalyzer"; then
    echo -e "${GREEN}✅ Полный тест запущен${NC}"
else
    echo -e "${RED}❌ Полный тест не запущен${NC}"
fi

# 3. Быстрый тест
print_section "3. БЫСТРЫЙ ТЕСТ"
echo "Запуск: make test-counter-quick"
make test-counter-quick 2>&1 | tee "$LOG_DIR/03_quick_test.log" | grep -E "(✅|❌|📊|📈)" || true

# 4. Интеграционный тест
print_section "4. ИНТЕГРАЦИОННЫЙ ТЕСТ"
echo "Проверка работы CounterAnalyzer с другими анализаторами..."
if go run ./cmd/debug/analyzer/main.go 2>&1 | tee "$LOG_DIR/04_integration.log" | grep -q "CounterAnalyzer работает"; then
    echo -e "${GREEN}✅ Интеграционный тест пройден${NC}"
else
    echo -e "${YELLOW}⚠️  CounterAnalyzer не найден в интеграционном тесте${NC}"
fi

# 5. Проверка сборки
print_section "5. ПРОВЕРКА СБОРКИ"
echo "Проверка компиляции CounterAnalyzer..."
if go build -o /tmp/test_counter ./cmd/debug/counter_test 2>&1 | tee "$LOG_DIR/05_build.log"; then
    echo -e "${GREEN}✅ Сборка успешна${NC}"
    rm -f /tmp/test_counter
else
    echo -e "${RED}❌ Ошибка сборки${NC}"
fi

# 6. Статистика тестов
print_section "6. СТАТИСТИКА ТЕСТОВ"
echo "Анализ результатов..."

# Считаем успехи/ошибки
total_tests=5
passed_tests=0

# Проверяем каждый тест
for i in 01 02 03 04 05; do
    log_file="$LOG_DIR/${i}_*.log"
    if ls $log_file 1> /dev/null 2>&1; then
        actual_file=$(ls $log_file)
        if grep -q -i "error\|panic\|fatal\|ошибка" "$actual_file"; then
            echo -e "  Тест $i: ${RED}❌${NC}"
        else
            echo -e "  Тест $i: ${GREEN}✅${NC}"
            ((passed_tests++))
        fi
    fi
done

echo ""
echo -e "${BLUE}📊 РЕЗУЛЬТАТЫ:${NC}"
echo -e "  Пройдено тестов: ${passed_tests}/${total_tests}"
if [ $passed_tests -eq $total_tests ]; then
    echo -e "${GREEN}  🎉 ВСЕ ТЕСТЫ ПРОЙДЕНЫ!${NC}"
elif [ $passed_tests -ge 3 ]; then
    echo -e "${YELLOW}  ⚠️  БОЛЬШИНСТВО ТЕСТОВ ПРОЙДЕНО${NC}"
else
    echo -e "${RED}  💥 МНОГО ОШИБОК!${NC}"
fi

echo ""
echo -e "${YELLOW}📁 Логи сохранены в: $LOG_DIR${NC}"
echo -e "${GREEN}✨ Тестирование CounterAnalyzer завершено${NC}"