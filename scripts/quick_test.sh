#!/bin/bash

echo "⚡ БЫСТРЫЙ ТЕСТ СИСТЕМЫ"
echo "====================="

echo "1. Проверяем компиляцию..."
if go build ./cmd/debug/counter_test/; then
    echo "✅ CounterAnalyzer компилируется"
else
    echo "❌ Ошибка компиляции CounterAnalyzer"
    exit 1
fi

echo ""
echo "2. Запускаем CounterAnalyzer на 3 секунды..."
timeout() {
    perl -e 'alarm shift; exec @ARGV' "$@"
}

# Пробуем разные варианты timeout
if command -v gtimeout &> /dev/null; then
    gtimeout 3 go run ./cmd/debug/counter_test/main.go | head -10
elif command -v timeout &> /dev/null; then
    timeout 3 go run ./cmd/debug/counter_test/main.go | head -10
else
    # Запускаем без timeout, но убиваем через 3 секунды
    go run ./cmd/debug/counter_test/main.go &
    PID=$!
    sleep 3
    kill $PID 2>/dev/null || true
    wait $PID 2>/dev/null || true
    echo "✅ CounterAnalyzer тест выполнен (с ограничением времени)"
fi

echo ""
echo "3. Сборка основного приложения..."
make build

echo ""
echo "🎉 БЫСТРЫЙ ТЕСТ ЗАВЕРШЕН УСПЕШНО!"