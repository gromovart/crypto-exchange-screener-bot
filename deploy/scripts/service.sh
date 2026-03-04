#!/bin/bash
# Скрипт управления службой Crypto Screener Bot
# Использование: ./deploy/scripts/service.sh [COMMAND] [OPTIONS]

set -e  # Выход при ошибке

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Параметры по умолчанию
SERVER_IP="95.142.40.244"
SERVER_USER="root"
SSH_KEY="${HOME}/.ssh/id_rsa"
SERVICE_NAME="crypto-screener"
APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/${APP_NAME}"
LINES=50  # Количество строк по умолчанию

# Функции для вывода
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Показать помощь
show_help() {
    echo "Использование: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Команды:"
    echo "  start               Запустить службу"
    echo "  stop                Остановить службу"
    echo "  restart             Перезапустить службу"
    echo "  status              Показать статус службы"
    echo "  logs [N]            Показать N строк логов app.log (по умолчанию: 50)"
    echo "  logs-error [N]      Показать N строк логов error.log (по умолчанию: 50)"
    echo "  logs-follow         Показать логи app.log в реальном времени"
    echo "  logs-follow-error   Показать логи error.log в реальном времени"
    echo "  logs-webhook        Показать логи webhook.log в реальном времени"
    echo "  monitor             Мониторинг состояния системы"
    echo "  backup              Создать резервную копию"
    echo "  cleanup             Очистка старых логов и резервных копий"
    echo "  config-show         Показать текущую конфигурацию"
    echo "  config-check        Проверить конфигурацию"
    echo "  health              Проверить здоровье системы"
    echo "  restart-app         Перезапуск только приложения (без зависимостей)"
    echo "  webhook-info        Информация о настройках webhook (Telegram)"
    echo "  webhook-setup       Установить/обновить webhook в Telegram"
    echo "  webhook-remove      Удалить webhook из Telegram"
    echo "  webhook-check       Проверить статус webhook в Telegram"
    echo "  max-webhook-info    Информация о настройках webhook (MAX)"
    echo "  ssl-check           Проверить SSL сертификаты (Telegram)"
    echo "  ssl-renew           Обновить SSL сертификаты (Telegram Let's Encrypt)"
    echo "  max-ssl-check       Проверить SSL сертификаты (MAX)"
    echo "  max-ssl-renew       Обновить SSL сертификаты (MAX Let's Encrypt)"
    echo ""
    echo "Опции:"
    echo "  --ip=IP_ADDRESS     IP адрес сервера (по умолчанию: 95.142.40.244)"
    echo "  --user=USERNAME     Имя пользователя (по умолчанию: root)"
    echo "  --key=PATH          Путь к SSH ключу (по умолчанию: ~/.ssh/id_rsa)"
    echo "  --help              Показать эту справку"
    echo ""
    echo "Примеры:"
    echo "  $0 status --ip=95.142.40.244"
    echo "  $0 logs 100                   # 100 строк app.log"
    echo "  $0 logs-error 50              # 50 строк error.log"
    echo "  $0 logs-follow                 # app.log в реальном времени"
    echo "  $0 logs-follow-error           # error.log в реальном времени"
    echo "  $0 monitor"
    echo "  $0 health"
    echo "  $0 webhook-info"
    echo "  $0 ssl-check"
}

# Проверка SSH подключения
check_ssh_connection() {
    if ! ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no \
        -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "echo 'connected'" &> /dev/null; then
        log_error "Не удалось подключиться к серверу"
        echo "Проверьте:"
        echo "1. SSH ключ авторизован: ssh-copy-id -i ${SSH_KEY} ${SERVER_USER}@${SERVER_IP}"
        echo "2. Сервер доступен: ping ${SERVER_IP}"
        echo "3. Используйте диагностику: ./check-connection.sh"
        exit 1
    fi
}

# Управление службой
service_start() {
    log_info "Запуск службы ${SERVICE_NAME}..."
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "systemctl start ${SERVICE_NAME}.service"
    sleep 2
    service_status
}

service_stop() {
    log_info "Остановка службы ${SERVICE_NAME}..."
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "systemctl stop ${SERVICE_NAME}.service"
    sleep 1
    service_status
}

service_restart() {
    log_info "Перезапуск службы ${SERVICE_NAME}..."
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "systemctl restart ${SERVICE_NAME}.service"
    sleep 3
    service_status
}

service_restart_app() {
    log_info "Перезапуск только приложения..."
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/${APP_NAME}"
SERVICE_NAME="crypto-screener"

# Останавливаем приложение
if pgrep -f "${APP_NAME}" > /dev/null; then
    echo "Остановка процесса приложения..."
    pkill -f "${APP_NAME}"
    sleep 2
fi

# Запускаем приложение через systemd
echo "Запуск приложения..."
systemctl restart ${SERVICE_NAME}.service

echo "✅ Приложение перезапущено"
EOF
    sleep 2
    service_status
}

service_status() {
    echo "Статус службы ${SERVICE_NAME}:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "systemctl status ${SERVICE_NAME}.service --no-pager"
}

# ⭐ ИСПРАВЛЕННЫЕ ФУНКЦИИ ДЛЯ ЛОГОВ
service_logs() {
    local lines=${1:-50}
    echo "Последние ${lines} строк app.log:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "tail -n ${lines} /var/log/crypto-screener-bot/app.log"
}

service_logs_error() {
    local lines=${1:-50}
    echo "Последние ${lines} строк error.log:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "tail -n ${lines} /var/log/crypto-screener-bot/error.log"
}

service_logs_follow() {
    echo "Логи app.log в реальном времени (Ctrl+C для выхода):"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "tail -f /var/log/crypto-screener-bot/app.log"
}

service_logs_follow_error() {
    echo "Логи error.log в реальном времени (Ctrl+C для выхода):"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "tail -f /var/log/crypto-screener-bot/error.log"
}

service_logs_follow_webhook() {
    echo "Логи webhook.log в реальном времени (Ctrl+C для выхода):"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "tail -f /var/log/crypto-screener-bot/webhook.log 2>/dev/null || echo 'Webhook лог не найден'"
}

# Остальные функции без изменений...
service_monitor() {
    echo "Мониторинг системы:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
echo "=== СИСТЕМНЫЙ МОНИТОРИНГ ==="
echo "Время: $(date)"
echo ""

# 1. Загрузка системы
echo "1. Загрузка системы:"
uptime
echo ""

# 2. Использование памяти
echo "2. Использование памяти:"
free -h
echo ""

# 3. Использование диска
echo "3. Использование диска:"
df -h /opt /var/log
echo ""

# 4. Статус служб
echo "4. Статус служб:"
services=("crypto-screener" "postgresql" "redis-server")
for service in "${services[@]}"; do
    status=$(systemctl is-active "${service}.service" 2>/dev/null || echo "unknown")
    case "$status" in
        active) echo "  ✅ ${service}: активен" ;;
        inactive) echo "  ⏸️  ${service}: не активен" ;;
        failed) echo "  ❌ ${service}: ошибка" ;;
        *) echo "  ❓ ${service}: ${status}" ;;
    esac
done
echo ""

# 5. Процессы приложения
echo "5. Процессы приложения:"
if pgrep -f "crypto-screener-bot" > /dev/null; then
    echo "  ✅ Приложение работает"
    echo "  PID: $(pgrep -f "crypto-screener-bot")"
    echo "  Uptime: $(ps -o etime= -p $(pgrep -f "crypto-screener-bot") | xargs)"
else
    echo "  ❌ Приложение не работает"
fi
echo ""

# 6. Сетевые порты
echo "6. Сетевые порты:"
echo "  PostgreSQL (5432): $(ss -tln | grep ':5432' > /dev/null && echo '✅ открыт' || echo '❌ закрыт')"
echo "  Redis (6379): $(ss -tln | grep ':6379' > /dev/null && echo '✅ открыт' || echo '❌ закрыт')"

# Проверяем webhook порт из конфига
if [ -f "/opt/crypto-screener-bot/.env" ]; then
    WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "/opt/crypto-screener-bot/.env" | cut -d= -f2 2>/dev/null || echo "8443")
    echo "  Webhook (${WEBHOOK_PORT}): $(ss -tln | grep ":${WEBHOOK_PORT} " > /dev/null && echo '✅ открыт' || echo '❌ закрыт')"
else
    echo "  Webhook (8443): $(ss -tln | grep ':8443' > /dev/null && echo '✅ открыт' || echo '❌ закрыт')"
fi
echo ""

# 7. Логи (последние ошибки)
echo "7. Последние ошибки в логах:"
tail -20 /var/log/crypto-screener-bot/error.log 2>/dev/null | grep -i "error\|fail\|panic" | tail -5 || echo "  ✅ Ошибок не найдено"
echo ""

# 8. Проверка конфигурации webhook
echo "8. Проверка конфигурации webhook:"
CONFIG_FILE="/opt/crypto-screener-bot/.env"
if [ -f "${CONFIG_FILE}" ]; then
    echo "  ✅ Конфиг найден: ${CONFIG_FILE}"

    # Проверяем режим Telegram
    TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "webhook")
    echo "  Режим Telegram: ${TELEGRAM_MODE}"

    if [ "${TELEGRAM_MODE}" = "webhook" ]; then
        echo "  ✅ Режим работы: Webhook"

        # Проверяем webhook настройки
        WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
        WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "8443")
        WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "true")

        echo "  Домен: ${WEBHOOK_DOMAIN}"
        echo "  Порт: ${WEBHOOK_PORT}"
        echo "  TLS: ${WEBHOOK_USE_TLS}"

        if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
            CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
            KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

            if [ -f "${CERT_PATH}" ] && [ -f "${KEY_PATH}" ]; then
                echo "  ✅ Сертификаты найдены"
            else
                echo "  ⚠️  Сертификаты не найдены"
            fi
        fi
    else
        echo "  📡 Режим работы: Polling"
    fi
else
    echo "  ❌ Конфиг не найден"
fi
echo ""

echo "=== МОНИТОРИНГ ЗАВЕРШЕН ==="
EOF
}

service_backup() {
    echo "Создание резервной копии..."
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/${APP_NAME}"
BACKUP_DIR="/opt/${APP_NAME}_backups"
SERVICE_NAME="crypto-screener"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="${BACKUP_DIR}/manual_backup_${TIMESTAMP}"

# Создание директории
mkdir -p "${BACKUP_PATH}"

echo "Создание резервной копии системы..."

# Останавливаем сервис для консистентной резервной копии
echo "Остановка сервиса..."
systemctl stop ${SERVICE_NAME}.service 2>/dev/null || echo "⚠️  Сервис уже остановлен"

# Резервное копирование файлов приложения
echo "Копирование файлов приложения..."
echo "  Копирование бинарника..."
cp -r "${INSTALL_DIR}/bin" "${BACKUP_PATH}/" 2>/dev/null || echo "⚠️  Не удалось скопировать bin"

echo "  Копирование конфигурации..."
cp -r "${INSTALL_DIR}/configs" "${BACKUP_PATH}/" 2>/dev/null || echo "⚠️  Не удалось скопировать configs"

echo "  Копирование .env файла..."
cp "${INSTALL_DIR}/.env" "${BACKUP_PATH}/" 2>/dev/null || echo "⚠️  Не удалось скопировать .env"

# Создание дампа базы данных
echo "Создание дампа базы данных..."
if command -v pg_dump >/dev/null 2>&1; then
    # Читаем настройки БД из конфига
    if [ -f "${INSTALL_DIR}/.env" ]; then
        DB_HOST=$(grep "^DB_HOST=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_PORT=$(grep "^DB_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_NAME=$(grep "^DB_NAME=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_USER=$(grep "^DB_USER=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_PASSWORD=$(grep "^DB_PASSWORD=" "${INSTALL_DIR}/.env" | cut -d= -f2)

        export PGPASSWORD="${DB_PASSWORD}"
        DUMP_FILE="${BACKUP_PATH}/database_dump.sql"
        if pg_dump -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-bot}" \
            "${DB_NAME:-cryptobot}" > "${DUMP_FILE}" 2>/dev/null; then
            echo "✅ Дамп БД создан: $(wc -l < "${DUMP_FILE}") строк"
        else
            echo "⚠️  Не удалось создать дамп БД"
        fi
    else
        echo "⚠️  Конфиг не найден, пропускаем дамп БД"
    fi
else
    echo "⚠️  pg_dump не установлен, пропускаем дамп БД"
fi

# Архивирование
echo "Архивирование резервной копии..."
cd "${BACKUP_DIR}"
tar -czf "manual_backup_${TIMESTAMP}.tar.gz" "manual_backup_${TIMESTAMP}"
rm -rf "manual_backup_${TIMESTAMP}"

# Запуск сервиса обратно
echo "Запуск сервиса..."
systemctl start ${SERVICE_NAME}.service 2>/dev/null || echo "⚠️  Не удалось запустить сервис"

echo ""
echo "✅ Резервная копия создана: ${BACKUP_DIR}/manual_backup_${TIMESTAMP}.tar.gz"
echo "📊 Размер: $(du -h "${BACKUP_DIR}/manual_backup_${TIMESTAMP}.tar.gz" | cut -f1)"
echo ""
echo "📋 Список последних резервных копий:"
ls -la "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | tail -5 || echo "Резервных копий нет"
EOF
}

service_cleanup() {
    echo "Очистка старых файлов..."
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/${APP_NAME}"
LOG_DIR="/var/log/${APP_NAME}"
BACKUP_DIR="/opt/${APP_NAME}_backups"

echo "🧹 ОЧИСТКА СИСТЕМЫ"
echo "================="
echo ""

# 1. Очистка старых логов
echo "1. Очистка старых логов (старше 30 дней):"
if [ -d "${LOG_DIR}" ]; then
    OLD_LOGS=$(find "${LOG_DIR}" -name "*.log" -mtime +30 -type f | wc -l)
    if [ "${OLD_LOGS}" -gt 0 ]; then
        echo "   Найдено файлов для удаления: ${OLD_LOGS}"
        find "${LOG_DIR}" -name "*.log" -mtime +30 -type f -delete
        echo "   ✅ Логи очищены"
    else
        echo "   ✅ Старых логов не найдено"
    fi
else
    echo "   ⚠️  Директория логов не существует"
fi
echo ""

# 2. Очистка старых резервных копий
echo "2. Очистка старых резервных копий (оставить последние 10):"
if [ -d "${BACKUP_DIR}" ]; then
    BACKUP_COUNT=$(ls -1 "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | wc -l)
    echo "   Всего резервных копий: ${BACKUP_COUNT}"

    if [ "${BACKUP_COUNT}" -gt 10 ]; then
        REMOVE_COUNT=$((BACKUP_COUNT - 10))
        echo "   Удаляем старых: ${REMOVE_COUNT}"

        ls -t "${BACKUP_DIR}"/*.tar.gz | tail -${REMOVE_COUNT} | while read -r file; do
            echo "   Удаляем: $(basename "$file")"
            rm -f "$file"
        done
        echo "   ✅ Старые копии удалены"
    else
        echo "   ✅ Копий меньше 10, удаление не требуется"
    fi
else
    echo "   ⚠️  Директория резервных копий не существует"
fi
echo ""

# 3. Очистка временных файлов в директории установки
echo "3. Очистка временных файлов в ${INSTALL_DIR}:"
if [ -d "${INSTALL_DIR}" ]; then
    # Удаляем временные Go файлы
    TEMP_FILES=$(find "${INSTALL_DIR}" -name "*.tmp" -type f 2>/dev/null | wc -l)
    if [ "${TEMP_FILES}" -gt 0 ]; then
        echo "   Найдено временных файлов: ${TEMP_FILES}"
        find "${INSTALL_DIR}" -name "*.tmp" -type f -delete
        echo "   ✅ Временные файлы удалены"
    else
        echo "   ✅ Временных файлов не найдено"
    fi

    # Очистка папки logs внутри установки
    if [ -d "${INSTALL_DIR}/logs" ]; then
        LOGS_IN_APP=$(find "${INSTALL_DIR}/logs" -name "*.log" -type f 2>/dev/null | wc -l)
        if [ "${LOGS_IN_APP}" -gt 0 ]; then
            echo "   Очистка логов в папке приложения: ${LOGS_IN_APP} файлов"
            rm -f "${INSTALL_DIR}/logs"/*.log 2>/dev/null
            echo "   ✅ Логи приложения очищены"
        fi
    fi
fi
echo ""

# 4. Очистка кэша сборки Go
echo "4. Очистка кэша сборки Go:"
if command -v go >/dev/null 2>&1; then
    go clean -cache 2>/dev/null && echo "   ✅ Кэш Go очищен" || echo "   ⚠️  Не удалось очистить кэш Go"
else
    echo "   ⚠️  Go не установлен"
fi
echo ""

# 5. Очистка журналов systemd
echo "5. Очистка старых журналов systemd:"
journalctl --vacuum-time=7d 2>/dev/null && echo "   ✅ Журналы systemd очищены" || echo "   ⚠️  Не удалось очистить журналы"
echo ""

# 6. Проверка свободного места
echo "6. Свободное место на дисках:"
df -h /opt /var/log | grep -v Filesystem | while read line; do
    echo "   💾 $line"
done
echo ""

echo "✅ Очистка завершена"
EOF
}

service_config_show() {
    echo "Текущая конфигурация:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

if [ -f "${CONFIG_FILE}" ]; then
    echo "Файл конфигурации: ${CONFIG_FILE}"
    echo "Размер: $(du -h "${CONFIG_FILE}" | cut -f1)"
    echo "Изменен: $(stat -c %y "${CONFIG_FILE}" | cut -d' ' -f1)"
    echo ""
    echo "Основные настройки:"
    echo "=================="

    # Показываем настройки по категориям

    echo "1. ОСНОВНЫЕ НАСТРОЙКИ:"
    grep -E "^(APP_ENV|APP_NAME|APP_VERSION|LOG_LEVEL)=" "${CONFIG_FILE}" || echo "  (не настроены)"
    echo ""

    echo "2. БАЗА ДАННЫХ:"
    grep -E "^(DB_HOST|DB_PORT|DB_NAME|DB_USER|DB_ENABLE_AUTO_MIGRATE)=" "${CONFIG_FILE}" || echo "  (не настроены)"
    echo ""

    echo "3. REDIS:"
    grep -E "^(REDIS_HOST|REDIS_PORT|REDIS_PASSWORD|REDIS_ENABLED)=" "${CONFIG_FILE}" || echo "  (не настроены)"
    if grep -q "^REDIS_ENABLED=" "${CONFIG_FILE}"; then
        REDIS_ENABLED=$(grep "^REDIS_ENABLED=" "${CONFIG_FILE}" | cut -d= -f2)
        if [ "${REDIS_ENABLED}" = "true" ]; then
            echo "  ✅ Redis включен"

            # Проверка пароля Redis
            if grep -q "^REDIS_PASSWORD=" "${CONFIG_FILE}"; then
                REDIS_PASS=$(grep "^REDIS_PASSWORD=" "${CONFIG_FILE}" | cut -d= -f2)
                if [ -n "${REDIS_PASS}" ]; then
                    echo "  ✅ Redis пароль: настроен"
                else
                    echo "  ⚠️  Redis пароль: не установлен"
                fi
            else
                echo "  ⚠️  Redis пароль: не настроен"
            fi
        else
            echo "  ⚠️  Redis отключен"
        fi
    else
        echo "  ⚠️  REDIS_ENABLED не настроен"
    fi
    echo ""

    echo "4. TELEGRAM И WEBHOOK:"
    TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "webhook")
    echo "  Режим Telegram: ${TELEGRAM_MODE}"

    if [ "${TELEGRAM_MODE}" = "webhook" ]; then
        echo "  ✅ Режим: Webhook"
        grep -E "^(WEBHOOK_DOMAIN|WEBHOOK_PORT|WEBHOOK_PATH|WEBHOOK_USE_TLS|WEBHOOK_TLS_CERT_PATH|WEBHOOK_TLS_KEY_PATH|WEBHOOK_SECRET_TOKEN)=" "${CONFIG_FILE}" || echo "  (не настроены)"
    else
        echo "  📡 Режим: Polling"
    fi

    echo ""
    grep -E "^(TELEGRAM_ENABLED|TELEGRAM_ADMIN_IDS|TELEGRAM_BOT_TOKEN|TG_API_KEY|TG_CHAT_ID)=" "${CONFIG_FILE}" || echo "  (не настроены)"
    if grep -q "TELEGRAM_ENABLED=true" "${CONFIG_FILE}" || grep -q "TG_API_KEY=" "${CONFIG_FILE}"; then
        echo "  ✅ Telegram включен"
    else
        echo "  ⚠️  Telegram отключен"
    fi
    echo ""

    echo "5. БИРЖА:"
    grep -E "^(EXCHANGE|EXCHANGE_TYPE|UPDATE_INTERVAL|MAX_SYMBOLS_TO_MONITOR)=" "${CONFIG_FILE}" || echo "  (не настроены)"
    echo ""

    echo "6. API КЛЮЧИ (проверка наличия):"
    if grep -q "BINANCE_API_KEY=" "${CONFIG_FILE}" || grep -q "BYBIT_API_KEY=" "${CONFIG_FILE}"; then
        echo "  ✅ API ключи настроены"
    else
        echo "  ❌ API ключи не настроены"
    fi
    echo ""

    echo "7. ПРОВЕРКА СЕКРЕТНЫХ КЛЮЧЕЙ:"
    if grep -q "JWT_SECRET=" "${CONFIG_FILE}"; then
        echo "  ✅ JWT секрет настроен"
    else
        echo "  ⚠️  JWT секрет не настроен"
    fi
    if grep -q "ENCRYPTION_KEY=" "${CONFIG_FILE}"; then
        echo "  ✅ Ключ шифрования настроен"
    else
        echo "  ⚠️  Ключ шифрования не настроен"
    fi

    echo ""
    echo "8. ФАЙЛЫ КОНФИГУРАЦИИ В ПРОЕКТЕ:"
    echo "--------------------------------"
    if [ -d "/opt/crypto-screener-bot/configs" ]; then
        echo "Структура configs/:"
        ls -la "/opt/crypto-screener-bot/configs/" 2>/dev/null | head -10 || echo "  Не удалось прочитать директорию"
    else
        echo "⚠️  Директория configs/ не существует"
    fi

else
    echo "❌ Файл конфигурации не найден: ${CONFIG_FILE}"
    echo "Создайте конфиг: cp /opt/crypto-screener-bot/configs/prod/.env /opt/crypto-screener-bot/.env"
fi
EOF
}

service_config_check() {
    echo "Проверка конфигурации:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"
ERRORS=0
WARNINGS=0

echo "🔍 ПРОВЕРКА КОНФИГУРАЦИИ"
echo "======================="
echo ""

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

echo "✅ Файл конфигурации найден: ${CONFIG_FILE}"
echo ""

# Проверка обязательных настроек
echo "1. ОБЯЗАТЕЛЬНЫЕ НАСТРОЙКИ:"
echo "-------------------------"

# База данных
if grep -q "^DB_HOST=" "${CONFIG_FILE}"; then
    echo "  ✅ DB_HOST: настроен"
else
    echo "  ❌ DB_HOST: не настроен"
    ERRORS=$((ERRORS + 1))
fi

if grep -q "^DB_NAME=" "${CONFIG_FILE}"; then
    echo "  ✅ DB_NAME: настроен"
else
    echo "  ❌ DB_NAME: не настроен"
    ERRORS=$((ERRORS + 1))
fi

if grep -q "^DB_USER=" "${CONFIG_FILE}"; then
    echo "  ✅ DB_USER: настроен"
else
    echo "  ❌ DB_USER: не настроен"
    ERRORS=$((ERRORS + 1))
fi

if grep -q "^DB_PASSWORD=" "${CONFIG_FILE}"; then
    DB_PASS=$(grep "^DB_PASSWORD=" "${CONFIG_FILE}" | cut -d= -f2)
    if [ "${DB_PASS}" == "SecurePass123!" ] || [ "${DB_PASS}" == "" ]; then
        echo "  ⚠️  DB_PASSWORD: используется стандартный или пустой пароль"
        WARNINGS=$((WARNINGS + 1))
    else
        echo "  ✅ DB_PASSWORD: настроен"
    fi
else
    echo "  ❌ DB_PASSWORD: не настроен"
    ERRORS=$((ERRORS + 1))
fi

# Проверка DB_ENABLE_AUTO_MIGRATE
if grep -q "^DB_ENABLE_AUTO_MIGRATE=" "${CONFIG_FILE}"; then
    AUTO_MIGRATE=$(grep "^DB_ENABLE_AUTO_MIGRATE=" "${CONFIG_FILE}" | cut -d= -f2)
    if [ "${AUTO_MIGRATE}" == "true" ]; then
        echo "  ✅ DB_ENABLE_AUTO_MIGRATE: включены (миграции выполнятся автоматически)"
    else
        echo "  ⚠️  DB_ENABLE_AUTO_MIGRATE: отключены (миграции не будут выполнены)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "  ⚠️  DB_ENABLE_AUTO_MIGRATE: не настроен"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""

# Проверка Redis настроек
echo "2. REDIS НАСТРОЙКИ:"
echo "------------------"

if grep -q "^REDIS_ENABLED=" "${CONFIG_FILE}"; then
    REDIS_ENABLED=$(grep "^REDIS_ENABLED=" "${CONFIG_FILE}" | cut -d= -f2)
    if [ "${REDIS_ENABLED}" = "true" ]; then
        echo "  ✅ REDIS_ENABLED: включен"

        # Проверка хоста Redis
        if grep -q "^REDIS_HOST=" "${CONFIG_FILE}"; then
            REDIS_HOST=$(grep "^REDIS_HOST=" "${CONFIG_FILE}" | cut -d= -f2)
            echo "  ✅ REDIS_HOST: ${REDIS_HOST}"
        else
            echo "  ⚠️  REDIS_HOST: не настроен, будет использован localhost"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Проверка порта Redis
        if grep -q "^REDIS_PORT=" "${CONFIG_FILE}"; then
            REDIS_PORT=$(grep "^REDIS_PORT=" "${CONFIG_FILE}" | cut -d= -f2)
            echo "  ✅ REDIS_PORT: ${REDIS_PORT}"
        else
            echo "  ⚠️  REDIS_PORT: не настроен, будет использован 6379"
            WARNINGS=$((WARNINGS + 1))
        fi

        # Проверка пароля Redis (необязательно, но рекомендуется)
        if grep -q "^REDIS_PASSWORD=" "${CONFIG_FILE}"; then
            REDIS_PASS=$(grep "^REDIS_PASSWORD=" "${CONFIG_FILE}" | cut -d= -f2)
            if [ -n "${REDIS_PASS}" ]; then
                echo "  ✅ REDIS_PASSWORD: настроен"
            else
                echo "  ⚠️  REDIS_PASSWORD: пустой пароль"
                WARNINGS=$((WARNINGS + 1))
            fi
        else
            echo "  ℹ️  REDIS_PASSWORD: не настроен (Redis без пароля)"
        fi
    else
        echo "  ⚠️  REDIS_ENABLED: отключен"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "  ⚠️  REDIS_ENABLED: не настроен, по умолчанию будет true"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""

# Проверка API ключей
echo "3. API КЛЮЧИ БИРЖ:"
echo "-----------------"

EXCHANGE=$(grep "^EXCHANGE=" "${CONFIG_FILE}" | cut -d= -f2)

if [ "${EXCHANGE}" == "bybit" ]; then
    if grep -q "^BYBIT_API_KEY=" "${CONFIG_FILE}"; then
        API_KEY=$(grep "^BYBIT_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2)
        if [[ "${API_KEY}" == *"your_bybit_api_key"* ]] || [ "${API_KEY}" == "" ]; then
            echo "  ⚠️  BYBIT_API_KEY: не настроен или шаблонный"
            WARNINGS=$((WARNINGS + 1))
        else
            echo "  ✅ BYBIT_API_KEY: настроен"
        fi
    else
        echo "  ⚠️  BYBIT_API_KEY: не настроен"
        WARNINGS=$((WARNINGS + 1))
    fi
elif [ "${EXCHANGE}" == "binance" ]; then
    if grep -q "^BINANCE_API_KEY=" "${CONFIG_FILE}"; then
        API_KEY=$(grep "^BINANCE_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2)
        if [[ "${API_KEY}" == *"your_binance_api_key"* ]] || [ "${API_KEY}" == "" ]; then
            echo "  ⚠️  BINANCE_API_KEY: не настроен или шаблонный"
            WARNINGS=$((WARNINGS + 1))
        else
            echo "  ✅ BINANCE_API_KEY: настроен"
        fi
    else
        echo "  ⚠️  BINANCE_API_KEY: не настроен"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "  ⚠️  EXCHANGE: неизвестная биржа '${EXCHANGE}'"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""

# Проверка Telegram и webhook
echo "4. TELEGRAM И WEBHOOK НАСТРОЙКИ:"
echo "------------------------------"

TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "webhook")
echo "  Режим Telegram: ${TELEGRAM_MODE}"

if [ "${TELEGRAM_MODE}" = "webhook" ]; then
    echo "  ✅ Режим работы: Webhook"

    # Проверка webhook настроек
    if grep -q "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}"; then
        WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2)
        if [ -z "${WEBHOOK_DOMAIN}" ]; then
            echo "  ❌ WEBHOOK_DOMAIN: пустое значение"
            ERRORS=$((ERRORS + 1))
        else
            echo "  ✅ WEBHOOK_DOMAIN: ${WEBHOOK_DOMAIN}"
        fi
    else
        echo "  ❌ WEBHOOK_DOMAIN: не настроен"
        ERRORS=$((ERRORS + 1))
    fi

    if grep -q "^WEBHOOK_SECRET_TOKEN=" "${CONFIG_FILE}"; then
        SECRET_TOKEN=$(grep "^WEBHOOK_SECRET_TOKEN=" "${CONFIG_FILE}" | cut -d= -f2)
        if [ -z "${SECRET_TOKEN}" ]; then
            echo "  ❌ WEBHOOK_SECRET_TOKEN: пустое значение"
            ERRORS=$((ERRORS + 1))
        else
            echo "  ✅ WEBHOOK_SECRET_TOKEN: настроен"
        fi
    else
        echo "  ❌ WEBHOOK_SECRET_TOKEN: не настроен"
        ERRORS=$((ERRORS + 1))
    fi

    # Проверка SSL сертификатов если используется TLS
    WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "true")
    if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
        echo "  ✅ TLS включен"

        CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
        KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

        if [ -f "${CERT_PATH}" ] && [ -f "${KEY_PATH}" ]; then
            echo "  ✅ SSL сертификаты найдены"
        else
            echo "  ⚠️  SSL сертификаты не найдены по указанным путям"
            WARNINGS=$((WARNINGS + 1))
        fi
    else
        echo "  ⚠️  TLS отключен (небезопасно)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "  📡 Режим работы: Polling"
fi

echo ""

# Проверка токена бота
if grep -q "^TG_API_KEY=" "${CONFIG_FILE}"; then
    TG_API_KEY=$(grep "^TG_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2)
    if [[ "${TG_API_KEY}" == *"your_telegram_bot_token"* ]] || [ "${TG_API_KEY}" == "" ]; then
        echo "  ❌ TG_API_KEY: не настроен или шаблонный"
        ERRORS=$((ERRORS + 1))
    else
        echo "  ✅ TG_API_KEY: настроен"
    fi
else
    echo "  ❌ TG_API_KEY: не настроен"
    ERRORS=$((ERRORS + 1))
fi

echo ""

# Проверка безопасности
echo "5. БЕЗОПАСНОСТЬ:"
echo "---------------"

if grep -q "^JWT_SECRET=" "${CONFIG_FILE}"; then
    JWT_SECRET=$(grep "^JWT_SECRET=" "${CONFIG_FILE}" | cut -d= -f2)
    if [[ "${JWT_SECRET}" == *"ваш_секретный_ключ"* ]] || [ "${JWT_SECRET}" == "" ]; then
        echo "  ⚠️  JWT_SECRET: не настроен или шаблонный"
        WARNINGS=$((WARNINGS + 1))
    else
        echo "  ✅ JWT_SECRET: настроен"
    fi
else
    echo "  ⚠️  JWT_SECRET: не настроен"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""

# Итог
echo "📊 ИТОГ ПРОВЕРКИ:"
echo "---------------"
echo "Ошибок: ${ERRORS}"
echo "Предупреждений: ${WARNINGS}"
echo ""

if [ "${ERRORS}" -eq 0 ] && [ "${WARNINGS}" -eq 0 ]; then
    echo "🎉 Конфигурация в полном порядке!"
elif [ "${ERRORS}" -eq 0 ]; then
    echo "⚠️  Конфигурация работает, но есть предупреждения"
else
    echo "❌ В конфигурации есть критические ошибки"
    echo ""
    echo "Рекомендации:"
    echo "1. Отредактируйте конфиг: nano ${CONFIG_FILE}"
    echo "2. Проверьте настройки выше"
    echo "3. Перезапустите сервис: systemctl restart crypto-screener"
fi
EOF
}

service_health() {
    echo "Проверка здоровья системы:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
echo "🏥 ПРОВЕРКА ЗДОРОВЬЯ СИСТЕМЫ"
echo "==========================="
echo "Время: $(date)"
echo ""

HEALTH_OK=true

# 1. Проверка служб
echo "1. 🚀 ПРОВЕРКА СЛУЖБ:"
services=("crypto-screener" "postgresql" "redis-server")
for service in "${services[@]}"; do
    status=$(systemctl is-active "${service}.service" 2>/dev/null || echo "unknown")
    case "$status" in
        active) echo "   ✅ ${service}: активен" ;;
        inactive)
            echo "   ❌ ${service}: не активен"
            HEALTH_OK=false
            ;;
        failed)
            echo "   ❌ ${service}: ошибка"
            HEALTH_OK=false
            ;;
        *)
            echo "   ⚠️  ${service}: статус неизвестен (${status})"
            HEALTH_OK=false
            ;;
    esac
done
echo ""

# 2. Проверка портов
echo "2. 🔌 ПРОВЕРКА ПОРТОВ:"
if ss -tln | grep -q ':5432'; then
    echo "   ✅ PostgreSQL (5432): доступен"
else
    echo "   ❌ PostgreSQL (5432): недоступен"
    HEALTH_OK=false
fi

if ss -tln | grep -q ':6379'; then
    echo "   ✅ Redis (6379): доступен"
else
    echo "   ❌ Redis (6379): недоступен"
    HEALTH_OK=false
fi

# Проверяем webhook порт
if [ -f "/opt/crypto-screener-bot/.env" ]; then
    WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "/opt/crypto-screener-bot/.env" | cut -d= -f2 2>/dev/null || echo "8443")
    if ss -tln | grep -q ":${WEBHOOK_PORT} "; then
        echo "   ✅ Webhook (${WEBHOOK_PORT}): доступен"
    else
        echo "   ⚠️  Webhook (${WEBHOOK_PORT}): недоступен"
        HEALTH_OK=false
    fi
fi
echo ""

# 3. Проверка процессов
echo "3. 🔄 ПРОВЕРКА ПРОЦЕССОВ:"
if pgrep -f "crypto-screener-bot" > /dev/null; then
    echo "   ✅ Приложение: работает"
    echo "   📊 PID: $(pgrep -f "crypto-screener-bot")"
    echo "   ⏱️  Uptime: $(ps -o etime= -p $(pgrep -f "crypto-screener-bot") | xargs)"
else
    echo "   ❌ Приложение: не работает"
    HEALTH_OK=false
fi
echo ""

# 4. Проверка ресурсов
echo "4. 📊 ПРОВЕРКА РЕСУРСОВ:"

# Память
MEM_FREE=$(free -m | awk '/^Mem:/ {print $4}')
if [ "${MEM_FREE}" -lt 100 ]; then
    echo "   ⚠️  Память: мало свободной памяти (${MEM_FREE} MB)"
    HEALTH_OK=false
else
    echo "   ✅ Память: свободно ${MEM_FREE} MB"
fi

# Диск
DISK_USAGE=$(df /opt --output=pcent | tail -1 | tr -d ' %')
if [ "${DISK_USAGE}" -gt 90 ]; then
    echo "   ⚠️  Диск: мало свободного места (используется ${DISK_USAGE}%)"
    HEALTH_OK=false
else
    echo "   ✅ Диск: используется ${DISK_USAGE}%"
fi
echo ""

# 5. Проверка логов на ошибки
echo "5. 📝 ПРОВЕРКА ЛОГОВ (последние 5 минут):"

LOG_FILE="/opt/crypto-screener-bot/logs/app.log"
if [ -f "${LOG_FILE}" ]; then
    # Получаем логи за последние 5 минут (по времени модификации)
    RECENT_ERRORS=$(tail -n 2000 "${LOG_FILE}" 2>/dev/null | grep -i -c "error\|fail\|panic\|fatal" || echo "0")

    if [ "${RECENT_ERRORS}" -gt 0 ]; then
        echo "   ⚠️  Найдено ошибок: ${RECENT_ERRORS}"
        echo "   Последние ошибки:"
        tail -n 500 "${LOG_FILE}" 2>/dev/null | grep -i "error\|fail\|panic\|fatal" | tail -3 | while read line; do
            echo "     📛 $(echo "$line" | cut -d' ' -f6-)"
        done
        HEALTH_OK=false
    else
        echo "   ✅ Ошибок не обнаружено"
    fi
else
    echo "   ⚠️  Файл лога не найден: ${LOG_FILE}"
fi

# Проверяем также error.log для полноты картины
ERROR_LOG="/opt/crypto-screener-bot/logs/error.log"
if [ -f "${ERROR_LOG}" ]; then
    ERROR_COUNT=$(tail -n 500 "${ERROR_LOG}" 2>/dev/null | grep -i -c "error\|fail\|panic\|fatal" || echo "0")
    if [ "${ERROR_COUNT}" -gt 0 ]; then
        echo "   ⚠️  Найдено ошибок в error.log: ${ERROR_COUNT}"
    fi
fi
echo ""

# 6. Проверка подключения к БД и Redis
echo "6. 🗄️  ПРОВЕРКА БАЗЫ ДАННЫХ И REDIS:"

# Проверка БД
if command -v psql >/dev/null 2>&1 && [ -f "/opt/crypto-screener-bot/.env" ]; then
    DB_HOST=$(grep "^DB_HOST=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)
    DB_PORT=$(grep "^DB_PORT=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)
    DB_NAME=$(grep "^DB_NAME=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)
    DB_USER=$(grep "^DB_USER=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)
    DB_PASSWORD=$(grep "^DB_PASSWORD=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)

    export PGPASSWORD="${DB_PASSWORD}"
    if psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-bot}" \
        "${DB_NAME:-cryptobot}" -c "SELECT 1" >/dev/null 2>&1; then
        echo "   ✅ База данных: доступна"
    else
        echo "   ❌ База данных: недоступна"
        HEALTH_OK=false
    fi
else
    echo "   ⚠️  Проверка БД: инструменты не установлены"
fi

# Проверка Redis
if command -v redis-cli >/dev/null 2>&1 && [ -f "/opt/crypto-screener-bot/.env" ]; then
    REDIS_ENABLED=$(grep "^REDIS_ENABLED=" "/opt/crypto-screener-bot/.env" | cut -d= -f2 2>/dev/null || echo "true")

    if [ "${REDIS_ENABLED}" = "true" ]; then
        REDIS_HOST=$(grep "^REDIS_HOST=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)
        REDIS_PORT=$(grep "^REDIS_PORT=" "/opt/crypto-screener-bot/.env" | cut -d= -f2)
        REDIS_PASSWORD=$(grep "^REDIS_PASSWORD=" "/opt/crypto-screener-bot/.env" | cut -d= -f2 2>/dev/null || echo "")

        REDIS_CMD="redis-cli -h '${REDIS_HOST:-localhost}' -p '${REDIS_PORT:-6379}'"
        if [ -n "${REDIS_PASSWORD}" ]; then
            REDIS_CMD="${REDIS_CMD} -a '${REDIS_PASSWORD}'"
        fi

        if eval "${REDIS_CMD} ping 2>/dev/null" | grep -q "PONG"; then
            echo "   ✅ Redis: доступен"
        else
            echo "   ❌ Redis: недоступен"
            HEALTH_OK=false
        fi
    else
        echo "   ℹ️  Redis: отключен в конфиге"
    fi
else
    echo "   ⚠️  Проверка Redis: redis-cli не установлен"
fi
echo ""

# 7. Проверка структуры проекта
echo "7. 📁 ПРОВЕРКА СТРУКТУРЫ ПРОЕКТА:"
INSTALL_DIR="/opt/crypto-screener-bot"
if [ -d "${INSTALL_DIR}" ]; then
    echo "   ✅ Директория проекта существует"

    # Проверка ключевых файлов
    KEY_FILES=(
        "${INSTALL_DIR}/.env"
        "${INSTALL_DIR}/bin/crypto-screener-bot"
        "${INSTALL_DIR}/configs/prod/.env"
    )

    MISSING_FILES=0
    for file in "${KEY_FILES[@]}"; do
        if [ -f "${file}" ]; then
            echo "   ✅ $(basename "${file}"): существует"
        else
            echo "   ❌ $(basename "${file}"): отсутствует"
            MISSING_FILES=$((MISSING_FILES + 1))
            HEALTH_OK=false
        fi
    done

    if [ "${MISSING_FILES}" -eq 0 ]; then
        echo "   ✅ Все ключевые файлы на месте"
    fi
else
    echo "   ❌ Директория проекта не существует"
    HEALTH_OK=false
fi
echo ""

# Итог
echo "🎯 ИТОГ ПРОВЕРКИ:"
echo "================"
if $HEALTH_OK; then
    echo "✅ СИСТЕМА ЗДОРОВА"
    echo "Все компоненты работают корректно"
else
    echo "⚠️  В СИСТЕМЕ ЕСТЬ ПРОБЛЕМЫ"
    echo "Проверьте сообщения выше для диагностики"
fi
echo ""
echo "📋 Рекомендации:"
if ! $HEALTH_OK; then
    echo "1. Проверьте логи: tail -f /var/log/crypto-screener-bot/app.log"
    echo "2. Перезапустите сервис: systemctl restart crypto-screener"
    echo "3. Проверьте конфигурацию: nano /opt/crypto-screener-bot/.env"
fi
echo "4. Мониторинг: ./service.sh monitor"
EOF
}

# Webhook и SSL функции
service_webhook_info() {
    echo "Информация о настройках webhook:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

echo "=== ИНФОРМАЦИЯ О WEBHOOK ==="
echo ""

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

# Получаем настройки
TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "webhook")
WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "8443")
WEBHOOK_PATH=$(grep "^WEBHOOK_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "/webhook")
WEBHOOK_SECRET_TOKEN=$(grep "^WEBHOOK_SECRET_TOKEN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "true")
TG_API_KEY=$(grep "^TG_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

echo "1. 📋 ОСНОВНЫЕ НАСТРОЙКИ:"
echo "   Режим Telegram: ${TELEGRAM_MODE}"
echo "   Домен: ${WEBHOOK_DOMAIN}"
echo "   Порт: ${WEBHOOK_PORT}"
echo "   Путь: ${WEBHOOK_PATH}"
echo "   Использовать TLS: ${WEBHOOK_USE_TLS}"
echo "   Секретный токен: $(if [ -n "${WEBHOOK_SECRET_TOKEN}" ]; then echo 'установлен'; else echo 'не установлен'; fi)"
echo "   Telegram API ключ: $(if [ -n "${TG_API_KEY}" ]; then echo 'установлен'; else echo 'не установлен'; fi)"
echo ""

echo "2. 🌐 WEBHOOK URL:"
if [ -n "${WEBHOOK_DOMAIN}" ]; then
    if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
        echo "   https://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}${WEBHOOK_PATH}"
    else
        echo "   http://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}${WEBHOOK_PATH}"
    fi
else
    echo "   ⚠️  Домен не настроен"
fi
echo ""

echo "3. 🔐 SSL СЕРТИФИКАТЫ:"
if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
    CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
    KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

    echo "   Путь к сертификату: ${CERT_PATH}"
    echo "   Путь к ключу: ${KEY_PATH}"

    if [ -f "${CERT_PATH}" ]; then
        echo "   ✅ Сертификат найден"
        echo "   Срок действия: $(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2 || echo "неизвестно")"
        echo "   Subject: $(openssl x509 -in "${CERT_PATH}" -noout -subject 2>/dev/null | sed 's/subject=//' || echo "неизвестно")"
    else
        echo "   ❌ Сертификат не найден"
    fi

    if [ -f "${KEY_PATH}" ]; then
        echo "   ✅ Ключ найден"
    else
        echo "   ❌ Ключ не найден"
    fi
else
    echo "   ℹ️  TLS отключен, сертификаты не требуются"
fi
echo ""

echo "4. 📝 ИНСТРУКЦИЯ ПО НАСТРОЙКЕ:"
if [ -n "${TG_API_KEY}" ] && [ -n "${WEBHOOK_DOMAIN}" ] && [ -n "${WEBHOOK_SECRET_TOKEN}" ]; then
    echo "   Для настройки webhook в Telegram выполните:"
    echo ""
    echo "   curl -X POST 'https://api.telegram.org/bot${TG_API_KEY}/setWebhook' \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d '{"
    echo "       \"url\": \"https://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}${WEBHOOK_PATH}\","
    echo "       \"secret_token\": \"${WEBHOOK_SECRET_TOKEN}\""
    echo "     }'"
    echo ""
    echo "   Для проверки статуса:"
    echo "   curl -X POST 'https://api.telegram.org/bot${TG_API_KEY}/getWebhookInfo'"
    echo ""
    echo "   Для удаления webhook:"
    echo "   curl -X POST 'https://api.telegram.org/bot${TG_API_KEY}/deleteWebhook'"
else
    echo "   ⚠️  Для показа инструкции необходимо настроить:"
    echo "     - TG_API_KEY (токен бота)"
    echo "     - WEBHOOK_DOMAIN (домен)"
    echo "     - WEBHOOK_SECRET_TOKEN (секретный токен)"
fi
echo ""

echo "=== ИНФОРМАЦИЯ ЗАВЕРШЕНА ==="
EOF
}

service_webhook_setup() {
    echo "Настройка webhook в Telegram:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

# Получаем настройки
TG_API_KEY=$(grep "^TG_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "8443")
WEBHOOK_PATH=$(grep "^WEBHOOK_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "/webhook")
WEBHOOK_SECRET_TOKEN=$(grep "^WEBHOOK_SECRET_TOKEN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

# Проверка обязательных настроек
ERRORS=0
if [ -z "${TG_API_KEY}" ]; then
    echo "❌ TG_API_KEY не настроен"
    ERRORS=1
fi

if [ -z "${WEBHOOK_DOMAIN}" ]; then
    echo "❌ WEBHOOK_DOMAIN не настроен"
    ERRORS=1
fi

if [ -z "${WEBHOOK_SECRET_TOKEN}" ]; then
    echo "❌ WEBHOOK_SECRET_TOKEN не настроен"
    ERRORS=1
fi

if [ ${ERRORS} -gt 0 ]; then
    echo ""
    echo "Настройте недостающие параметры в файле: ${CONFIG_FILE}"
    exit 1
fi

# Формируем URL
WEBHOOK_URL="https://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}${WEBHOOK_PATH}"
echo "🔧 Настройка webhook в Telegram..."
echo "   URL: ${WEBHOOK_URL}"
echo "   Секретный токен: ${WEBHOOK_SECRET_TOKEN:0:8}..."
echo ""

# Устанавливаем webhook
RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${TG_API_KEY}/setWebhook" \
    -H "Content-Type: application/json" \
    -d '{
        "url": "'"${WEBHOOK_URL}"'",
        "secret_token": "'"${WEBHOOK_SECRET_TOKEN}"'"
    }')

if echo "${RESPONSE}" | grep -q '"ok":true'; then
    echo "✅ Webhook успешно настроен"
    echo "Ответ: ${RESPONSE}"

    # Проверяем статус
    echo ""
    echo "🔍 Проверка статуса webhook..."
    CHECK_RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${TG_API_KEY}/getWebhookInfo")
    echo "Статус: ${CHECK_RESPONSE}"
else
    echo "❌ Ошибка настройки webhook"
    echo "Ответ: ${RESPONSE}"
    exit 1
fi
EOF
}

service_webhook_remove() {
    echo "Удаление webhook из Telegram:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

TG_API_KEY=$(grep "^TG_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

if [ -z "${TG_API_KEY}" ]; then
    echo "❌ TG_API_KEY не настроен"
    exit 1
fi

echo "🗑️  Удаление webhook из Telegram..."
RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${TG_API_KEY}/deleteWebhook")

if echo "${RESPONSE}" | grep -q '"ok":true'; then
    echo "✅ Webhook успешно удален"
    echo "Ответ: ${RESPONSE}"

    # Обновляем режим на polling
    echo ""
    echo "🔄 Обновление режима на polling..."
    sed -i "s/^TELEGRAM_MODE=.*/TELEGRAM_MODE=polling/" "${CONFIG_FILE}"
    echo "✅ Режим обновлен на polling"

    # Перезапускаем сервис
    echo "🔄 Перезапуск сервиса..."
    systemctl restart crypto-screener.service
    echo "✅ Сервис перезапущен"
else
    echo "❌ Ошибка удаления webhook"
    echo "Ответ: ${RESPONSE}"
    exit 1
fi
EOF
}

service_webhook_check() {
    echo "Проверка статуса webhook в Telegram:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

TG_API_KEY=$(grep "^TG_API_KEY=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

if [ -z "${TG_API_KEY}" ]; then
    echo "❌ TG_API_KEY не настроен"
    exit 1
fi

echo "🔍 Проверка статуса webhook в Telegram..."
RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${TG_API_KEY}/getWebhookInfo")

echo "Ответ от Telegram API:"
echo "${RESPONSE}" | python3 -m json.tool 2>/dev/null || echo "${RESPONSE}"
EOF
}

service_ssl_check() {
    echo "Проверка SSL сертификатов:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

echo "=== ПРОВЕРКА SSL СЕРТИФИКАТОВ ==="
echo ""

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

# Проверяем настройки TLS
WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "true")

if [ "${WEBHOOK_USE_TLS}" != "true" ]; then
    echo "ℹ️  TLS отключен в настройках (WEBHOOK_USE_TLS=false)"
    echo "SSL сертификаты не требуются"
    exit 0
fi

echo "✅ TLS включен в настройках"
echo ""

# Получаем пути к сертификатам
CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

echo "1. ПРОВЕРКА НАСТРОЕК:"
echo "   Путь к сертификату: ${CERT_PATH}"
echo "   Путь к ключу: ${KEY_PATH}"
echo ""

# Проверяем альтернативные пути
ALT_CERT_PATHS=(
    "${CERT_PATH}"
    "/etc/crypto-bot/certs/cert.pem"
    "/opt/crypto-screener-bot/ssl/fullchain.pem"
    "/etc/letsencrypt/live/bot.gromovart.ru/fullchain.pem"
)

ALT_KEY_PATHS=(
    "${KEY_PATH}"
    "/etc/crypto-bot/certs/key.pem"
    "/opt/crypto-screener-bot/ssl/privkey.pem"
    "/etc/letsencrypt/live/bot.gromovart.ru/privkey.pem"
)

echo "2. ПРОВЕРКА ФАЙЛОВ СЕРТИФИКАТОВ:"
CERT_FOUND=false
KEY_FOUND=false

# Поиск сертификата
for cert_path in "${ALT_CERT_PATHS[@]}"; do
    if [ -f "${cert_path}" ]; then
        echo "   ✅ Сертификат найден: ${cert_path}"
        CERT_FOUND=true
        CERT_PATH="${cert_path}"
        break
    fi
done

if [ "${CERT_FOUND}" = false ]; then
    echo "   ❌ Сертификат не найден ни по одному из путей"
fi

# Поиск ключа
for key_path in "${ALT_KEY_PATHS[@]}"; do
    if [ -f "${key_path}" ]; then
        echo "   ✅ Ключ найден: ${key_path}"
        KEY_FOUND=true
        KEY_PATH="${key_path}"
        break
    fi
done

if [ "${KEY_FOUND}" = false ]; then
    echo "   ❌ Ключ не найден ни по одному из путей"
fi

echo ""

if [ "${CERT_FOUND}" = true ] && [ "${KEY_FOUND}" = true ]; then
    echo "3. 🔍 ПРОВЕРКА ВАЛИДНОСТИ СЕРТИФИКАТА:"

    # Проверка срока действия
    if openssl x509 -in "${CERT_PATH}" -noout -checkend 0 >/dev/null 2>&1; then
        echo "   ✅ Сертификат действителен"

        NOT_BEFORE=$(openssl x509 -in "${CERT_PATH}" -noout -startdate 2>/dev/null | cut -d= -f2)
        NOT_AFTER=$(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2)

        echo "   📅 Действует с: ${NOT_BEFORE}"
        echo "   📅 Действует до: ${NOT_AFTER}"

        # Проверка на 30 дней до истечения
        if openssl x509 -in "${CERT_PATH}" -noout -checkend 2592000 >/dev/null 2>&1; then
            echo "   ✅ Срок действия > 30 дней"
        else
            echo "   ⚠️  Сертификат истекает через < 30 дней"
        fi
    else
        echo "   ❌ Сертификат недействителен или просрочен"
    fi

    # Проверка Subject
    echo ""
    echo "4. 📄 ИНФОРМАЦИЯ О СЕРТИФИКАТЕ:"
    SUBJECT=$(openssl x509 -in "${CERT_PATH}" -noout -subject 2>/dev/null | sed 's/subject=//')
    echo "   Subject: ${SUBJECT}"

    # Проверка SAN (Subject Alternative Names)
    SAN=$(openssl x509 -in "${CERT_PATH}" -noout -text 2>/dev/null | grep -A1 "Subject Alternative Name" | tail -1 | xargs)
    if [ -n "${SAN}" ]; then
        echo "   SAN: ${SAN}"
    fi

    # Проверка домена из конфига
    WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
    if [ -n "${WEBHOOK_DOMAIN}" ]; then
        echo ""
        echo "5. 🔗 ПРОВЕРКА СООТВЕТСТВИЯ ДОМЕНУ:"
        if echo "${SUBJECT} ${SAN}" | grep -q "${WEBHOOK_DOMAIN}"; then
            echo "   ✅ Сертификат содержит домен: ${WEBHOOK_DOMAIN}"
        else
            echo "   ⚠️  Сертификат не содержит домен: ${WEBHOOK_DOMAIN}"
            echo "   Рекомендуется выпустить новый сертификат для этого домена"
        fi
    fi

    # Размер ключа
    echo ""
    echo "6. 🔐 ПРОВЕРКА КЛЮЧА:"
    KEY_SIZE=$(openssl rsa -in "${KEY_PATH}" -noout -text 2>/dev/null | grep "Private-Key:" | awk '{print $2}')
    if [ -n "${KEY_SIZE}" ]; then
        echo "   Размер ключа: ${KEY_SIZE} бит"
        if [ "${KEY_SIZE}" -ge 2048 ]; then
            echo "   ✅ Размер ключа достаточный (>= 2048 бит)"
        else
            echo "   ⚠️  Размер ключа недостаточный (< 2048 бит)"
        fi
    fi
else
    echo "❌ Не найдены необходимые файлы сертификатов"
    echo ""
    echo "Рекомендации:"
    echo "1. Проверьте пути в конфиге: nano ${CONFIG_FILE}"
    echo "2. Создайте самоподписанный сертификат:"
    echo "   mkdir -p /etc/crypto-bot/certs"
    echo "   openssl req -x509 -newkey rsa:2048 -keyout /etc/crypto-bot/certs/key.pem -out /etc/crypto-bot/certs/cert.pem -days 365 -nodes -subj '/CN=bot.gromovart.ru'"
    echo "3. Или установите Let's Encrypt:"
    echo "   apt-get install certbot"
    echo "   certbot certonly --standalone -d bot.gromovart.ru --non-interactive --agree-tos --email admin@example.com"
fi

echo ""
echo "=== ПРОВЕРКА ЗАВЕРШЕНА ==="
EOF
}

service_ssl_renew() {
    echo "Обновление SSL сертификатов Let's Encrypt:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

echo "=== ОБНОВЛЕНИЕ SSL СЕРТИФИКАТОВ LET'S ENCRYPT ==="
echo ""

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

if [ -z "${WEBHOOK_DOMAIN}" ]; then
    echo "❌ WEBHOOK_DOMAIN не настроен"
    exit 1
fi

echo "Домен: ${WEBHOOK_DOMAIN}"
echo ""

# Проверяем установлен ли certbot
if ! command -v certbot >/dev/null 2>&1; then
    echo "❌ certbot не установлен"
    echo ""
    echo "Установите certbot:"
    echo "apt-get update"
    echo "apt-get install -y certbot"
    exit 1
fi

echo "✅ certbot установлен"
echo ""

# Проверяем существующие сертификаты
if [ -d "/etc/letsencrypt/live/${WEBHOOK_DOMAIN}" ]; then
    echo "📋 Существующие сертификаты найдены:"
    echo "   Путь: /etc/letsencrypt/live/${WEBHOOK_DOMAIN}/"

    # Проверяем срок действия
    CERT_PATH="/etc/letsencrypt/live/${WEBHOOK_DOMAIN}/fullchain.pem"
    if [ -f "${CERT_PATH}" ]; then
        NOT_AFTER=$(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2)
        echo "   Срок действия: ${NOT_AFTER}"

        # Проверяем сколько дней осталось
        CURRENT_TIME=$(date +%s)
        NOT_AFTER_TIME=$(date -d "${NOT_AFTER}" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "${NOT_AFTER}" +%s 2>/dev/null || echo 0)
        DAYS_LEFT=$(((NOT_AFTER_TIME - CURRENT_TIME) / 86400))

        echo "   Осталось дней: ${DAYS_LEFT}"

        if [ ${DAYS_LEFT} -gt 30 ]; then
            echo "   ✅ Сертификат еще действителен долгое время"
            echo "   Обновление не требуется"
            exit 0
        elif [ ${DAYS_LEFT} -gt 0 ]; then
            echo "   ⚠️  Сертификат скоро истекает, обновляем..."
        else
            echo "   ❌ Сертификат истек, обновляем..."
        fi
    fi
else
    echo "📋 Существующие сертификаты не найдены"
    echo "   Создаем новые..."
fi

echo ""

# Обновляем или получаем новые сертификаты
echo "🔄 Обновление/получение сертификатов..."
echo "   Остановка сервиса для освобождения порта 80/443..."

# Останавливаем сервис
systemctl stop crypto-screener.service 2>/dev/null || echo "⚠️  Не удалось остановить сервис"

# Обновляем сертификаты
if certbot renew --force-renewal --cert-name "${WEBHOOK_DOMAIN}" --non-interactive --agree-tos 2>/dev/null; then
    echo "✅ Сертификаты успешно обновлены"
else
    echo "⚠️  Не удалось обновить существующие сертификаты"
    echo "   Пробуем получить новые..."

    # Получаем новые сертификаты
    if certbot certonly --standalone -d "${WEBHOOK_DOMAIN}" --non-interactive --agree-tos --email "admin@${WEBHOOK_DOMAIN}" 2>/dev/null; then
        echo "✅ Новые сертификаты успешно получены"
    else
        echo "❌ Не удалось получить сертификаты"
        echo "   Запускаем сервис обратно..."
        systemctl start crypto-screener.service 2>/dev/null || true
        exit 1
    fi
fi

echo ""

# Копируем сертификаты в директорию приложения
echo "📋 Копирование сертификатов..."
CERTS_DIR="/etc/crypto-bot/certs"
mkdir -p "${CERTS_DIR}"

CERT_SOURCE="/etc/letsencrypt/live/${WEBHOOK_DOMAIN}/fullchain.pem"
KEY_SOURCE="/etc/letsencrypt/live/${WEBHOOK_DOMAIN}/privkey.pem"

if [ -f "${CERT_SOURCE}" ] && [ -f "${KEY_SOURCE}" ]; then
    cp "${CERT_SOURCE}" "${CERTS_DIR}/cert.pem"
    cp "${KEY_SOURCE}" "${CERTS_DIR}/key.pem"

    # Также копируем в директорию приложения для удобства
    mkdir -p "/opt/crypto-screener-bot/ssl"
    cp "${CERT_SOURCE}" "/opt/crypto-screener-bot/ssl/fullchain.pem"
    cp "${KEY_SOURCE}" "/opt/crypto-screener-bot/ssl/privkey.pem"

    echo "✅ Сертификаты скопированы в:"
    echo "   ${CERTS_DIR}/cert.pem"
    echo "   ${CERTS_DIR}/key.pem"
    echo "   /opt/crypto-screener-bot/ssl/fullchain.pem"
    echo "   /opt/crypto-screener-bot/ssl/privkey.pem"
else
    echo "❌ Не удалось скопировать сертификаты"
fi

echo ""

# Запускаем сервис обратно
echo "🔄 Запуск сервиса..."
systemctl start crypto-screener.service

echo "✅ Сервис запущен"
echo ""

echo "=== ОБНОВЛЕНИЕ ЗАВЕРШЕНО ==="
EOF
}

# MAX Webhook и SSL функции
service_max_webhook_info() {
    echo "Информация о настройках MAX webhook:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

echo "=== ИНФОРМАЦИЯ О MAX WEBHOOK ==="
echo ""

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

# Получаем настройки
MAX_MODE=$(grep "^MAX_MODE=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "webhook")
MAX_WEBHOOK_DOMAIN=$(grep "^MAX_WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
MAX_WEBHOOK_PORT=$(grep "^MAX_WEBHOOK_PORT=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "8444")
MAX_WEBHOOK_PATH=$(grep "^MAX_WEBHOOK_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "/webhook")
MAX_WEBHOOK_SECRET_TOKEN=$(grep "^MAX_WEBHOOK_SECRET_TOKEN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
MAX_WEBHOOK_USE_TLS=$(grep "^MAX_WEBHOOK_USE_TLS=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "true")
MAX_BOT_TOKEN=$(grep "^MAX_BOT_TOKEN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

echo "1. 📋 ОСНОВНЫЕ НАСТРОЙКИ:"
echo "   Режим MAX: ${MAX_MODE}"
echo "   Домен: ${MAX_WEBHOOK_DOMAIN}"
echo "   Порт: ${MAX_WEBHOOK_PORT}"
echo "   Путь: ${MAX_WEBHOOK_PATH}"
echo "   Использовать TLS: ${MAX_WEBHOOK_USE_TLS}"
echo "   Секретный токен: $(if [ -n "${MAX_WEBHOOK_SECRET_TOKEN}" ]; then echo 'установлен'; else echo 'не установлен'; fi)"
echo "   MAX API ключ: $(if [ -n "${MAX_BOT_TOKEN}" ]; then echo 'установлен'; else echo 'не установлен'; fi)"
echo ""

echo "2. 🌐 WEBHOOK URL:"
if [ -n "${MAX_WEBHOOK_DOMAIN}" ]; then
    if [ "${MAX_WEBHOOK_USE_TLS}" = "true" ]; then
        echo "   https://${MAX_WEBHOOK_DOMAIN}:${MAX_WEBHOOK_PORT}${MAX_WEBHOOK_PATH}"
    else
        echo "   http://${MAX_WEBHOOK_DOMAIN}:${MAX_WEBHOOK_PORT}${MAX_WEBHOOK_PATH}"
    fi
else
    echo "   ⚠️  Домен не настроен"
fi
echo ""

echo "3. 🔐 SSL СЕРТИФИКАТЫ:"
if [ "${MAX_WEBHOOK_USE_TLS}" = "true" ]; then
    CERT_PATH=$(grep "^MAX_WEBHOOK_TLS_CERT_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
    KEY_PATH=$(grep "^MAX_WEBHOOK_TLS_KEY_PATH=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

    echo "   Путь к сертификату: ${CERT_PATH}"
    echo "   Путь к ключу: ${KEY_PATH}"

    if [ -f "${CERT_PATH}" ]; then
        echo "   ✅ Сертификат найден"
        echo "   Срок действия: $(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2 || echo "неизвестно")"
        echo "   Subject: $(openssl x509 -in "${CERT_PATH}" -noout -subject 2>/dev/null | sed 's/subject=//' || echo "неизвестно")"
    else
        echo "   ❌ Сертификат не найден"
    fi

    if [ -f "${KEY_PATH}" ]; then
        echo "   ✅ Ключ найден"
    else
        echo "   ❌ Ключ не найден"
    fi
else
    echo "   ℹ️  TLS отключен, сертификаты не требуются"
fi
echo ""

echo "=== ИНФОРМАЦИЯ ЗАВЕРШЕНА ==="
EOF
}

service_max_ssl_check() {
    echo "Проверка MAX SSL сертификатов:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash

echo "=== ПРОВЕРКА MAX SSL СЕРТИФИКАТОВ ==="
echo ""

MAX_CERTS_DIR="/etc/crypto-bot/max-certs"
APP_SSL_DIR="/opt/crypto-screener-bot/ssl-max"

# Проверка директорий
echo "1. ДИРЕКТОРИИ:"
if [ -d "${MAX_CERTS_DIR}" ]; then
    echo "   ✅ ${MAX_CERTS_DIR} существует"
else
    echo "   ❌ ${MAX_CERTS_DIR} не существует"
fi

if [ -d "${APP_SSL_DIR}" ]; then
    echo "   ✅ ${APP_SSL_DIR} существует"
else
    echo "   ❌ ${APP_SSL_DIR} не существует"
fi
echo ""

# Проверка основных сертификатов
echo "2. ОСНОВНЫЕ СЕРТИФИКАТЫ (${MAX_CERTS_DIR}):"
CERT_PATH="${MAX_CERTS_DIR}/cert.pem"
KEY_PATH="${MAX_CERTS_DIR}/key.pem"

if [ -f "${CERT_PATH}" ]; then
    echo "   ✅ Сертификат найден: ${CERT_PATH}"

    echo "   Subject: $(openssl x509 -in "${CERT_PATH}" -noout -subject 2>/dev/null | sed 's/subject=//')"
    echo "   Issuer: $(openssl x509 -in "${CERT_PATH}" -noout -issuer 2>/dev/null | sed 's/issuer=//')"
    echo "   Valid from: $(openssl x509 -in "${CERT_PATH}" -noout -startdate 2>/dev/null | cut -d= -f2)"
    echo "   Valid until: $(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2)"

    if openssl x509 -in "${CERT_PATH}" -noout -checkend 0 >/dev/null 2>&1; then
        echo "   ✅ Сертификат действителен"

        if openssl x509 -in "${CERT_PATH}" -noout -checkend 2592000 >/dev/null 2>&1; then
            echo "   ✅ Срок действия > 30 дней"
        else
            echo "   ⚠️  Сертификат истекает менее чем через 30 дней"
        fi
    else
        echo "   ❌ Сертификат просрочен"
    fi
else
    echo "   ❌ Сертификат не найден: ${CERT_PATH}"
fi

if [ -f "${KEY_PATH}" ]; then
    echo "   ✅ Ключ найден: ${KEY_PATH}"
else
    echo "   ❌ Ключ не найден: ${KEY_PATH}"
fi
echo ""

# Проверка домена из конфига
CONFIG_FILE="/opt/crypto-screener-bot/.env"
if [ -f "${CONFIG_FILE}" ]; then
    MAX_WEBHOOK_DOMAIN=$(grep "^MAX_WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")
    if [ -n "${MAX_WEBHOOK_DOMAIN}" ]; then
        echo ""
        echo "5. 🔗 ПРОВЕРКА СООТВЕТСТВИЯ ДОМЕНУ:"
        if echo "${SUBJECT} ${SAN}" | grep -q "${MAX_WEBHOOK_DOMAIN}"; then
            echo "   ✅ Сертификат содержит домен: ${MAX_WEBHOOK_DOMAIN}"
        else
            echo "   ⚠️  Сертификат не содержит домен: ${MAX_WEBHOOK_DOMAIN}"
            echo "   Рекомендуется выпустить новый сертификат для этого домена"
        fi
    fi
fi

# Рекомендации
if [ ! -f "${CERT_PATH}" ] || [ ! -f "${KEY_PATH}" ]; then
    echo ""
    echo "📝 РЕКОМЕНДАЦИИ:"
    echo "1. Запустите setup_max_ssl_certificates в deploy.sh"
    echo "2. Или скопируйте существующие сертификаты:"
    echo "   mkdir -p ${MAX_CERTS_DIR}"
    echo "   cp /path/to/cert.pem ${MAX_CERTS_DIR}/cert.pem"
    echo "   cp /path/to/key.pem ${MAX_CERTS_DIR}/key.pem"
    echo "3. Или установите Let's Encrypt:"
    echo "   apt-get install certbot"
    echo "   certbot certonly --standalone -d max-bot.gromovart.ru --non-interactive --agree-tos --email admin@example.com"
fi

echo ""
echo "=== ПРОВЕРКА ЗАВЕРШЕНА ==="
EOF
}

service_max_ssl_renew() {
    echo "Обновление MAX SSL сертификатов Let's Encrypt:"
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
CONFIG_FILE="/opt/crypto-screener-bot/.env"

echo "=== ОБНОВЛЕНИЕ MAX SSL СЕРТИФИКАТОВ LET'S ENCRYPT ==="
echo ""

if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

MAX_WEBHOOK_DOMAIN=$(grep "^MAX_WEBHOOK_DOMAIN=" "${CONFIG_FILE}" | cut -d= -f2 2>/dev/null || echo "")

if [ -z "${MAX_WEBHOOK_DOMAIN}" ]; then
    echo "❌ MAX_WEBHOOK_DOMAIN не настроен"
    exit 1
fi

echo "Домен: ${MAX_WEBHOOK_DOMAIN}"
echo ""

# Проверяем установлен ли certbot
if ! command -v certbot >/dev/null 2>&1; then
    echo "❌ certbot не установлен"
    echo ""
    echo "Установите certbot:"
    echo "apt-get update"
    echo "apt-get install -y certbot"
    exit 1
fi

echo "✅ certbot установлен"
echo ""

# Проверяем существующие сертификаты
if [ -d "/etc/letsencrypt/live/${MAX_WEBHOOK_DOMAIN}" ]; then
    echo "📋 Существующие сертификаты найдены:"
    echo "   Путь: /etc/letsencrypt/live/${MAX_WEBHOOK_DOMAIN}/"

    CERT_PATH="/etc/letsencrypt/live/${MAX_WEBHOOK_DOMAIN}/fullchain.pem"
    if [ -f "${CERT_PATH}" ]; then
        NOT_AFTER=$(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2)
        echo "   Срок действия: ${NOT_AFTER}"

        CURRENT_TIME=$(date +%s)
        NOT_AFTER_TIME=$(date -d "${NOT_AFTER}" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "${NOT_AFTER}" +%s 2>/dev/null || echo 0)
        DAYS_LEFT=$(((NOT_AFTER_TIME - CURRENT_TIME) / 86400))

        echo "   Осталось дней: ${DAYS_LEFT}"

        if [ ${DAYS_LEFT} -gt 30 ]; then
            echo "   ✅ Сертификат еще действителен долгое время"
            echo "   Обновление не требуется"
            exit 0
        fi
    fi
fi

echo ""
echo "🔄 Обновление/получение сертификатов..."
echo "   Остановка сервиса для освобождения порта 80/443..."

systemctl stop crypto-screener.service 2>/dev/null || echo "⚠️  Не удалось остановить сервис"

# Обновляем сертификаты
if certbot renew --force-renewal --cert-name "${MAX_WEBHOOK_DOMAIN}" --non-interactive --agree-tos 2>/dev/null; then
    echo "✅ Сертификаты успешно обновлены"
else
    echo "⚠️  Не удалось обновить существующие сертификаты"
    echo "   Пробуем получить новые..."

    if certbot certonly --standalone -d "${MAX_WEBHOOK_DOMAIN}" --non-interactive --agree-tos --email "admin@${MAX_WEBHOOK_DOMAIN}" 2>/dev/null; then
        echo "✅ Новые сертификаты успешно получены"
    else
        echo "❌ Не удалось получить сертификаты"
        echo "   Запускаем сервис обратно..."
        systemctl start crypto-screener.service 2>/dev/null || true
        exit 1
    fi
fi

echo ""
echo "📋 Копирование сертификатов..."
MAX_CERTS_DIR="/etc/crypto-bot/max-certs"
mkdir -p "${MAX_CERTS_DIR}"

CERT_SOURCE="/etc/letsencrypt/live/${MAX_WEBHOOK_DOMAIN}/fullchain.pem"
KEY_SOURCE="/etc/letsencrypt/live/${MAX_WEBHOOK_DOMAIN}/privkey.pem"

if [ -f "${CERT_SOURCE}" ] && [ -f "${KEY_SOURCE}" ]; then
    cp "${CERT_SOURCE}" "${MAX_CERTS_DIR}/cert.pem"
    cp "${KEY_SOURCE}" "${MAX_CERTS_DIR}/key.pem"

    mkdir -p "/opt/crypto-screener-bot/ssl-max"
    cp "${CERT_SOURCE}" "/opt/crypto-screener-bot/ssl-max/fullchain.pem"
    cp "${KEY_SOURCE}" "/opt/crypto-screener-bot/ssl-max/privkey.pem"

    echo "✅ Сертификаты скопированы в:"
    echo "   ${MAX_CERTS_DIR}/cert.pem"
    echo "   ${MAX_CERTS_DIR}/key.pem"
    echo "   /opt/crypto-screener-bot/ssl-max/fullchain.pem"
    echo "   /opt/crypto-screener-bot/ssl-max/privkey.pem"
else
    echo "❌ Не удалось скопировать сертификаты"
fi

echo ""
echo "🔄 Запуск сервиса..."
systemctl start crypto-screener.service

echo "✅ Сервис запущен"
echo ""

echo "=== ОБНОВЛЕНИЕ ЗАВЕРШЕНО ==="
EOF
}

# Парсинг аргументов
parse_args() {
    command=""

    for arg in "$@"; do
        case $arg in
            start|stop|restart|status|logs|logs-error|logs-follow|logs-follow-error|logs-follow-webhook|monitor|backup|cleanup|config-show|config-check|health|restart-app|webhook-info|webhook-setup|webhook-remove|webhook-check|ssl-check|ssl-renew|max-webhook-info|max-ssl-check|max-ssl-renew)
                command="$arg"
                shift
                ;;
            --ip=*)
                SERVER_IP="${arg#*=}"
                shift
                ;;
            --user=*)
                SERVER_USER="${arg#*=}"
                shift
                ;;
            --key=*)
                SSH_KEY="${arg#*=}"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                # Проверяем, является ли аргумент числом (для количества строк)
                if [[ $arg =~ ^[0-9]+$ ]] && [ "$command" = "logs" || "$command" = "logs-error" ]; then
                    LINES="$arg"
                    shift
                else
                    log_error "Неизвестный аргумент: $arg"
                    show_help
                    exit 1
                fi
                ;;
        esac
    done

    if [ -z "$command" ]; then
        log_error "Не указана команда"
        show_help
        exit 1
    fi
}

# Основная функция
main() {
    parse_args "$@"

    # Проверка подключения
    check_ssh_connection

    # Выполнение команды
    case "$command" in
        start)
            service_start
            ;;
        stop)
            service_stop
            ;;
        restart)
            service_restart
            ;;
        restart-app)
            service_restart_app
            ;;
        status)
            service_status
            ;;
        logs)
            service_logs "$LINES"
            ;;
        logs-error)
            service_logs_error "$LINES"
            ;;
        logs-follow)
            service_logs_follow
            ;;
        logs-follow-error)
            service_logs_follow_error
            ;;
        logs-follow-webhook)
            service_logs_follow_webhook
            ;;
        monitor)
            service_monitor
            ;;
        backup)
            service_backup
            ;;
        cleanup)
            service_cleanup
            ;;
        config-show)
            service_config_show
            ;;
        config-check)
            service_config_check
            ;;
        health)
            service_health
            ;;
        webhook-info)
            service_webhook_info
            ;;
        webhook-setup)
            service_webhook_setup
            ;;
        webhook-remove)
            service_webhook_remove
            ;;
        webhook-check)
            service_webhook_check
            ;;
        ssl-check)
            service_ssl_check
            ;;
        ssl-renew)
            service_ssl_renew
            ;;
        max-webhook-info)
            service_max_webhook_info
            ;;
        max-ssl-check)
            service_max_ssl_check
            ;;
        max-ssl-renew)
            service_max_ssl_renew
            ;;
        *)
            log_error "Неизвестная команда: $command"
            show_help
            exit 1
            ;;
    esac
}

# Запуск скрипта
main "$@"
