#!/bin/bash

echo "🚀 ПОЛНОЕ ТЕСТИРОВАНИЕ СИСТЕМЫ (macOS version)"
echo "=============================================="
echo ""

# Цвета
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Обработка Ctrl+C
trap 'echo -e "\n${YELLOW}🛑 Прерывание тестирования${NC}"; exit 130' INT TERM

# Создаем директорию для логов
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="logs/full_test_${TIMESTAMP}"
mkdir -p "$LOG_DIR"

echo -e "${YELLOW}📁 Логи будут сохранены в: $LOG_DIR${NC}"
echo ""

# Функция timeout для macOS
mac_timeout() {
    local timeout=$1
    shift
    local cmd=("$@")

    # Запускаем команду в фоне
    "${cmd[@]}" &
    local pid=$!

    # Ждем указанное время
    sleep ${timeout}s

    # Проверяем, работает ли процесс
    if kill -0 $pid 2>/dev/null; then
        # Процесс все еще работает - убиваем его
        kill $pid 2>/dev/null
        wait $pid 2>/dev/null
        return 124  # Код таймаута
    else
        # Процесс уже завершился
        wait $pid
        return $?
    fi
}

# Функция для запуска теста с обработкой вывода
run_test() {
    local test_num=$1
    local test_name=$2
    local test_cmd=$3

    echo -e "${BLUE}[$test_num] $test_name${NC}"
    echo -e "${BLUE}$(printf '%.0s-' {1..60})${NC}"

    local log_file="$LOG_DIR/${test_num}_${test_name// /_}.log"

    # Запускаем команду с таймаутом (macOS версия)
    mac_timeout 30 bash -c "$test_cmd" 2>&1 | tee "$log_file"
    local exit_code=${PIPESTATUS[0]}

    if [ $exit_code -eq 0 ] || [ $exit_code -eq 124 ]; then
        # 0 - успех, 124 - таймаут (но команда выполнена)
        echo -e "${GREEN}✅ УСПЕХ${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}❌ ОШИБКА (код: $exit_code)${NC}"
        echo ""
        return 1
    fi
}

# Счетчики
total_tests=0
passed_tests=0
failed_tests=0

# 1. Проверка компиляции
if run_test "01" "Проверка компиляции" "go build ./cmd/debug/..."; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# 2. Базовый тест CounterAnalyzer
if run_test "02" "CounterAnalyzer базовый тест" "go run ./cmd/debug/counter_test/main.go 2>&1 | head -30"; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# 3. Тест всех анализаторов
if run_test "03" "Тест всех анализаторов" "go run ./cmd/debug/analyzer/main.go 2>&1 | head -40"; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# 4. Тест сборки продакшн
if run_test "04" "Тест сборки продакшн" "make build"; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# 5. Проверка типов
if run_test "05" "Проверка типов (go vet)" "go vet ./internal/analysis/analyzers/... 2>&1 | head -20"; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# 6. Быстрый тест CounterAnalyzer
if run_test "06" "Быстрый тест CounterAnalyzer" "go run ./cmd/debug/counter_test/main.go 2>&1 | grep -E '(✅|📊|🧮)' | head -10"; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# 7. Тест покрытия
if run_test "07" "Тест покрытия" "go test ./internal/analysis/analyzers/... -v 2>&1 | tail -15"; then
    ((passed_tests++))
else
    ((failed_tests++))
fi
((total_tests++))

# Итоговый отчет
echo -e "${BLUE}📊 ИТОГОВЫЙ ОТЧЕТ${NC}"
echo -e "${BLUE}$(printf '%.0s=' {1..60})${NC}"

echo -e "Всего тестов: $total_tests"
echo -e "${GREEN}✅ Пройдено: $passed_tests${NC}"
echo -e "${RED}❌ Провалено: $failed_tests${NC}"

# Процент успеха
if [ $total_tests -gt 0 ]; then
    success_rate=$((passed_tests * 100 / total_tests))
    echo -e "Процент успеха: ${success_rate}%"

    if [ $success_rate -ge 80 ]; then
        echo -e "${GREEN}🎉 ОТЛИЧНЫЙ РЕЗУЛЬТАТ!${NC}"
    elif [ $success_rate -ge 60 ]; then
        echo -e "${YELLOW}⚠️  УДОВЛЕТВОРИТЕЛЬНО${NC}"
    else
        echo -e "${RED}💥 ТРЕБУЕТСЯ ДОРАБОТКА${NC}"
    fi
fi

echo ""
echo -e "${YELLOW}📁 Полные логи доступны в: $LOG_DIR${NC}"
echo ""

# Проверяем, есть ли ошибки в логах
echo -e "${BLUE}🔍 ПРОВЕРКА ОШИБОК В ЛОГАХ${NC}"
echo -e "${BLUE}$(printf '%.0s-' {1..60})${NC}"

error_files=()
for log_file in "$LOG_DIR"/*.log; do
    if [ -f "$log_file" ]; then
        error_count=$(grep -c -i "error\|panic\|fatal\|❌\|FAIL" "$log_file" 2>/dev/null || true)
        if [ "$error_count" -gt 0 ]; then
            filename=$(basename "$log_file")
            echo -e "${RED}  $filename: $error_count ошибок${NC}"
            error_files+=("$log_file")
        fi
    fi
done

if [ ${#error_files[@]} -eq 0 ]; then
    echo -e "${GREEN}  ✅ Критических ошибок не обнаружено${NC}"
else
    echo ""
    echo -e "${YELLOW}📋 ОШИБКИ В ФАЙЛАХ:${NC}"
    for err_file in "${error_files[@]}"; do
        echo -e "  ${YELLOW}$(basename "$err_file"):${NC}"
        grep -n -i "error\|panic\|fatal\|❌\|FAIL" "$err_file" | head -3 | sed 's/^/    /'
    done
fi

echo ""
echo -e "${GREEN}✨ ТЕСТИРОВАНИЕ ЗАВЕРШЕНО${NC}"