#debug_run.sh
#!/bin/bash

echo "🔧 Запуск в режиме отладки..."
echo "==============================="

# Устанавливаем переменные окружения для тестирования
export USE_TESTNET=true
export ALERT_THRESHOLD=0.1
export UPDATE_INTERVAL=5
export TRACKED_INTERVALS=1,5,15
export HTTP_ENABLED=false

# Запускаем с подробным выводом
go run cmd/bot/main.go 2>&1 | tee debug.log