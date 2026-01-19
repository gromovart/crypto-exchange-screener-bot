#!/bin/bash
# Скрипт диагностики SSH подключения

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SERVER_IP="95.142.40.244"
SERVER_USER="root"
SSH_KEY="${HOME}/.ssh/id_rsa"

echo -e "${BLUE}=== ДИАГНОСТИКА SSH ПОДКЛЮЧЕНИЯ ===${NC}"
echo "Сервер: ${SERVER_USER}@${SERVER_IP}"
echo "SSH ключ: ${SSH_KEY}"
echo ""

# 1. Проверка доступности сервера
echo "1. Проверка доступности сервера..."
if ping -c 1 -W 1 "${SERVER_IP}" &> /dev/null; then
    echo -e "${GREEN}✅ Сервер доступен по ping${NC}"
else
    echo -e "${YELLOW}⚠️  Сервер не отвечает на ping${NC}"
    echo "   Проверьте:"
    echo "   - Интернет соединение"
    echo "   - Брандмауэр на вашей машине"
    echo "   - Доступность сервера в сети"
fi

# 2. Проверка SSH порта
echo ""
echo "2. Проверка SSH порта (22)..."
if nc -z -w 1 "${SERVER_IP}" 22 &> /dev/null; then
    echo -e "${GREEN}✅ SSH порт открыт${NC}"
else
    echo -e "${RED}❌ SSH порт закрыт${NC}"
    echo "   Возможные причины:"
    echo "   - SSH сервер не запущен на сервере"
    echo "   - Брандмауэр блокирует порт 22"
    echo "   - Сервер недоступен"
fi

# 3. Проверка SSH ключа
echo ""
echo "3. Проверка SSH ключа..."
if [ -f "${SSH_KEY}" ]; then
    echo -e "${GREEN}✅ SSH ключ найден${NC}"
    echo "   Права: $(stat -f "%A" "${SSH_KEY}")"

    # Проверяем права
    if [[ "$(stat -f "%A" "${SSH_KEY}")" != "600" ]]; then
        echo -e "${YELLOW}⚠️  Неправильные права SSH ключа${NC}"
        echo "   Исправьте: chmod 600 ${SSH_KEY}"
    fi
else
    echo -e "${RED}❌ SSH ключ не найден${NC}"
    echo "   Создайте ключ: ssh-keygen -t rsa -b 4096"
    echo "   или укажите другой ключ: --key=/path/to/key"
fi

# 4. Проверка SSH конфига
echo ""
echo "4. Проверка SSH конфигурации..."
if [ -f "${HOME}/.ssh/config" ]; then
    if grep -q "${SERVER_IP}" "${HOME}/.ssh/config"; then
        echo -e "${GREEN}✅ Сервер найден в SSH конфиге${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Файл SSH конфига не найден${NC}"
fi

# 5. Проверка наличия публичного ключа на сервере
echo ""
echo "5. Проверка авторизации на сервере..."
echo "   (Для этой проверки может потребоваться пароль)"

# Пробуем простую команду с паролем (если потребуется)
ssh -o PreferredAuthentications=publickey \
    -o ConnectTimeout=3 \
    "${SERVER_USER}@${SERVER_IP}" "echo 'SSH ключ сработал'" 2>&1 | grep -v "Warning" | head -5

# 6. Альтернативные проверки
echo ""
echo "6. Альтернативные проверки..."

# Проверка через ssh-keyscan
echo "   Получение отпечатка ключа сервера:"
ssh-keyscan "${SERVER_IP}" 2>/dev/null | head -1 || echo "   Не удалось получить отпечаток"

# 7. Рекомендации
echo ""
echo -e "${BLUE}=== РЕКОМЕНДАЦИИ ===${NC}"

if ! command -v ssh-copy-id &> /dev/null; then
    echo "1. Установите ssh-copy-id: brew install ssh-copy-id (macOS) или apt-get install openssh-client"
fi

echo "2. Если ключ не настроен, скопируйте его на сервер:"
echo "   ssh-copy-id -i ${SSH_KEY} ${SERVER_USER}@${SERVER_IP}"
echo ""
echo "3. Для отладки используйте команду:"
echo "   ssh -vvv -i ${SSH_KEY} ${SERVER_USER}@${SERVER_IP}"
echo ""
echo "4. Если сервер новый, возможно нужно принять ключ хоста:"
echo "   ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP} 'echo Привет'"

# 8. Тест с отключенной проверкой хоста
echo ""
echo "7. Тест с отключенной проверкой ключа хоста..."
ssh -o StrictHostKeyChecking=no \
    -o ConnectTimeout=5 \
    "${SERVER_USER}@${SERVER_IP}" "echo 'Тест успешен'" 2>&1 | grep -E "(Тест успешен|authenticity|Connection refused|Connection timed out)" || true

echo ""
echo -e "${BLUE}=== КОМАНДЫ ДЛЯ УСТРАНЕНИЯ ПРОБЛЕМ ===${NC}"
echo ""
echo "A. Создать и скопировать новый SSH ключ:"
echo "   ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa_crypto -N ''"
echo "   ssh-copy-id -i ~/.ssh/id_rsa_crypto.pub ${SERVER_USER}@${SERVER_IP}"
echo "   ./deploy.sh --ip=${SERVER_IP} --key=~/.ssh/id_rsa_crypto"
echo ""
echo "B. Проверить SSH подключение вручную:"
echo "   ssh -i ${SSH_KEY} ${SERVER_USER}@${SERVER_IP}"
echo ""
echo "C. Если сервер новый, нужно принять ключ:"
echo "   ssh -o StrictHostKeyChecking=no ${SERVER_USER}@${SERVER_IP}"
echo ""
echo "D. Проверить логи SSH на сервере (если есть доступ через другую среду):"
echo "   tail -f /var/log/auth.log"