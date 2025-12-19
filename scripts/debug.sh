#!/bin/bash

# Скрипт отладки для Crypto Exchange Screener Bot

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔧 Отладка Crypto Exchange Screener Bot${NC}"
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
if [ ! -f ".env" ]; then
    echo -e "${RED}❌ Файл .env не найден${NC}"
    echo "Создайте .env файл на основе .env.example"
    exit 1
fi
echo -e "${GREEN}✅ .env файл найден${NC}"

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
go build -o build/debug_bot ./cmd/bot/debug_main.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Сборка завершена${NC}"
else
    echo -e "${RED}❌ Ошибка сборки${NC}"
    exit 1
fi

# Создание лог директории
mkdir -p logs

# Запуск в режиме отладки
echo -e "${YELLOW}5. Запуск в режиме отладки...${NC}"
echo -e "${BLUE}========================================${NC}"
echo

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="logs/debug_${TIMESTAMP}.log"

# Запуск с записью лога
./build/debug_bot 2>&1 | tee "$LOG_FILE"

echo
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ Отладка завершена${NC}"
echo -e "${BLUE}Лог сохранен в: $LOG_FILE${NC}"