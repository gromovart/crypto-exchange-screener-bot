#!/bin/bash
# Скрипт отладки для Crypto Exchange Screener Bot

set -e

# Определяем окружение (по умолчанию dev)
ENV=${1:-dev}
ENV_FILE="configs/$ENV/.env"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Отладка Crypto Exchange Screener Bot${NC}"
echo -e "${YELLOW}Окружение: $ENV${NC}"
echo

# Проверка Go
echo -e "${YELLOW}1. Проверка Go...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go не установлен${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Go: $(go version)${NC}"

# Проверка .env файла
echo -e "${YELLOW}2. Проверка .env файла...${NC}"
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${RED}❌ Файл конфигурации не найден: $ENV_FILE${NC}"
    echo "Создайте: make config-init ENV=$ENV"
    exit 1
fi
echo -e "${GREEN}✅ Конфигурационный файл найден: $ENV_FILE${NC}"

# Проверка зависимостей
echo -e "${YELLOW}3. Проверка зависимостей...${NC}"
go mod tidy
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Зависимости в порядке${NC}"
else
    echo -e "${RED}❌ Ошибка в зависимостях${NC}"
    exit 1
fi

# Сборка
echo -e "${YELLOW}4. Сборка приложения...${NC}"
mkdir -p build
if [ -f "application/cmd/debug/basic/main.go" ]; then
    go build -o build/debug_bot ./application/cmd/debug/basic/main.go
    echo -e "${GREEN}✅ Сборка отладочного приложения завершена${NC}"
else
    echo -e "${YELLOW}⚠️  Отладочное приложение не найдено, собираем основное...${NC}"
    go build -o build/debug_bot ./application/cmd/bot/main.go
fi

# Создание лог директории
mkdir -p logs

# Запуск в режиме отладки
echo -e "${YELLOW}5. Запуск в режиме отладки...${NC}"
echo -e "${BLUE}========================================${NC}"
echo

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="logs/debug_${ENV}_${TIMESTAMP}.log"

echo -e "${YELLOW}Конфигурация: $ENV_FILE${NC}"
echo -e "${YELLOW}Лог файл: $LOG_FILE${NC}"
echo ""

# Запуск с записью лога
./build/debug_bot --config="$ENV_FILE" --log-level=debug 2>&1 | tee "$LOG_FILE"

echo
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ Отладка завершена${NC}"
echo -e "${BLUE}Лог сохранен в: $LOG_FILE${NC}"

# Пример использования:
# ./scripts/debug.sh dev      # Отладка dev окружения
# ./scripts/debug.sh prod     # Отладка prod окружения