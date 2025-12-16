# test_signal.sh
#!/bin/bash

echo "🧪 Тестирование системы сигналов..."
echo ""

# Создаем временный .env файл с низким порогом
cat > .env.test << 'EOF'
# Signal Monitoring
ALERT_THRESHOLD=0.01  # Очень низкий порог для теста
UPDATE_INTERVAL=5
HTTP_ENABLED=false
USE_TESTNET=true
EOF

# Запускаем бот на 30 секунд
timeout 30 go run cmd/bot/main.go 2>&1 | grep -A3 -B1 "СИГНАЛЫ\|Pump\|Dump"

echo ""
echo "✅ Тест завершен"