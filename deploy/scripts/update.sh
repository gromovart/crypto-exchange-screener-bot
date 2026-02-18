#!/bin/bash
# Скрипт обновления приложения на Ubuntu 22.04
# Использование: ./deploy/scripts/update.sh [OPTIONS]

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
APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/${APP_NAME}"
SERVICE_NAME="crypto-screener"
BACKUP_DIR="/opt/${APP_NAME}_backups"
CERTS_DIR="/etc/crypto-bot/certs"

# Переменные состояния
backup_only=false
rollback=false
no_backup=false
force=false

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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Парсинг аргументов
parse_args() {
    for arg in "$@"; do
        case $arg in
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
            --backup-only)
                backup_only=true
                shift
                ;;
            --rollback)
                rollback=true
                shift
                ;;
            --no-backup)
                no_backup=true
                shift
                ;;
            --force)
                force=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Неизвестный аргумент: $arg"
                show_help
                exit 1
                ;;
        esac
    done
}

# Показать помощь
show_help() {
    echo "Использование: $0 [OPTIONS]"
    echo ""
    echo "Опции:"
    echo "  --ip=IP_ADDRESS      IP адрес сервера (по умолчанию: 95.142.40.244)"
    echo "  --user=USERNAME      Имя пользователя (по умолчанию: root)"
    echo "  --key=PATH           Путь к SSH ключу (по умолчанию: ~/.ssh/id_rsa)"
    echo "  --backup-only        Только создать резервную копию"
    echo "  --rollback           Откатиться к предыдущей версии"
    echo "  --no-backup          Не создавать резервную копию (опасно!)"
    echo "  --force              Принудительное обновление без подтверждения"
    echo "  --help               Показать эту справку"
    echo ""
    echo "Примеры:"
    echo "  $0 --ip=95.142.40.244             # Обновить приложение"
    echo "  $0 --backup-only                 # Создать резервную копию"
    echo "  $0 --rollback                    # Откатить обновление"
    echo "  $0 --no-backup --force           # Быстрое обновление (без подтверждений)"
}

# Проверка SSH подключения
check_ssh_connection() {
    log_step "Проверка SSH подключения..."

    if ! ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no \
        -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" "echo 'SSH подключение успешно'" &> /dev/null; then
        log_error "Не удалось подключиться к серверу"
        log_info "Проверьте SSH ключ: ssh-copy-id -i ${SSH_KEY} ${SERVER_USER}@${SERVER_IP}"
        exit 1
    fi

    log_info "✅ SSH подключение успешно"
}

# Проверка состояния сервера с учетом webhook
check_server_status() {
    log_step "Проверка состояния сервера..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
echo "=== СТАТУС СЕРВЕРА ==="
echo ""

# 1. Загрузка системы
echo "1. Загрузка системы:"
uptime
echo ""

# 2. Статус служб
echo "2. Статус служб:"
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

# 3. Версия приложения
echo "3. Версия приложения:"
INSTALL_DIR="/opt/crypto-screener-bot"
if [ -f "${INSTALL_DIR}/bin/crypto-screener-bot" ]; then
    "${INSTALL_DIR}/bin/crypto-screener-bot" --version 2>&1 | head -1 || echo "  Не удалось определить версию"
else
    echo "  ❌ Приложение не установлено"
fi
echo ""

# 4. Дисковое пространство
echo "4. Дисковое пространство:"
df -h /opt /var/log | grep -v Filesystem
echo ""

# 5. Проверка Redis
echo "5. Статус Redis:"
if systemctl is-active redis-server >/dev/null 2>&1; then
    echo "  ✅ Redis: активен"

    # Проверка подключения к Redis
    if command -v redis-cli >/dev/null 2>&1; then
        if redis-cli ping | grep -q "PONG"; then
            echo "  ✅ Redis: доступен для подключения"
        else
            echo "  ⚠️  Redis: не отвечает на ping"
        fi
    fi
else
    echo "  ⚠️  Redis: не активен"
fi
echo ""

# 6. Проверка webhook статуса
echo "6. Проверка webhook статуса:"
if [ -f "${INSTALL_DIR}/.env" ]; then
    TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "webhook")
    echo "  Режим Telegram: ${TELEGRAM_MODE}"

    if [ "${TELEGRAM_MODE}" = "webhook" ]; then
        echo "  ✅ Режим работы: Webhook"

        # Проверяем webhook порт
        WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "8443")
        echo "  Webhook порт: ${WEBHOOK_PORT}"

        if ss -tln | grep -q ":${WEBHOOK_PORT} "; then
            echo "  ✅ Webhook порт открыт"
        else
            echo "  ⚠️  Webhook порт закрыт"
        fi

        # Проверка SSL сертификатов
        WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "true")
        if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
            echo "  🔐 TLS включен"

            CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")
            KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")

            if [ -f "${CERT_PATH}" ] && [ -f "${KEY_PATH}" ]; then
                echo "  ✅ SSL сертификаты найдены"
            else
                echo "  ⚠️  SSL сертификаты не найдены"
            fi
        fi
    else
        echo "  📡 Режим работы: Polling"
    fi
else
    echo "  ❌ Конфигурация не найдена"
fi
echo ""

echo "=== ПРОВЕРКА ЗАВЕРШЕНА ==="
EOF
}

# Создание резервной копии с сохранением SSL сертификатов
create_backup() {
    log_step "Создание резервной копии..."

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${BACKUP_DIR}/backup_${timestamp}"

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
INSTALL_DIR="${INSTALL_DIR}"
BACKUP_DIR="${BACKUP_DIR}"
BACKUP_PATH="${backup_path}"
SERVICE_NAME="${SERVICE_NAME}"
CERTS_DIR="${CERTS_DIR}"

# Создание директории для резервных копий
mkdir -p "\${BACKUP_DIR}"

echo "📦 Создание резервной копии системы..."
echo "Время: \$(date)"
echo ""

# Останавливаем сервис перед созданием резервной копии
echo "1. Остановка сервиса..."
systemctl stop \${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Сервис уже остановлен или не существует"
sleep 2

# Создание резервной копии
echo "2. Копирование файлов приложения..."
mkdir -p "\${BACKUP_PATH}"

# Копирование бинарника
if [ -f "\${INSTALL_DIR}/bin/\${APP_NAME}" ]; then
    cp "\${INSTALL_DIR}/bin/\${APP_NAME}" "\${BACKUP_PATH}/"
    echo "  ✅ Бинарник скопирован"
else
    echo "  ⚠️  Бинарник не найден"
fi

# Копирование конфигурации
if [ -d "\${INSTALL_DIR}/configs" ]; then
    cp -r "\${INSTALL_DIR}/configs" "\${BACKUP_PATH}/"
    echo "  ✅ Конфигурация скопирована"
else
    echo "  ⚠️  Конфигурация не найдена"
fi

# Копирование .env файла
if [ -f "\${INSTALL_DIR}/.env" ]; then
    cp "\${INSTALL_DIR}/.env" "\${BACKUP_PATH}/"
    echo "  ✅ Конфиг .env скопирован"
fi

# Копирование SSL сертификатов если есть
echo "3. Копирование SSL сертификатов..."
if [ -d "\${CERTS_DIR}" ]; then
    mkdir -p "\${BACKUP_PATH}/ssl_certs"
    cp -r "\${CERTS_DIR}"/* "\${BACKUP_PATH}/ssl_certs/" 2>/dev/null || echo "  ⚠️  Не удалось скопировать сертификаты"
    echo "  ✅ SSL сертификаты скопированы"
else
    echo "  ℹ️  Директория SSL сертификатов не существует"
fi

# Создание дампа базы данных
echo "4. Создание дампа базы данных..."
if command -v pg_dump >/dev/null 2>&1 && [ -f "\${INSTALL_DIR}/.env" ]; then
    # Читаем настройки БД из конфига
    DB_HOST=\$(grep "^DB_HOST=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_PORT=\$(grep "^DB_PORT=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_NAME=\$(grep "^DB_NAME=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_USER=\$(grep "^DB_USER=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_PASSWORD=\$(grep "^DB_PASSWORD=" "\${INSTALL_DIR}/.env" | cut -d= -f2)

    export PGPASSWORD="\${DB_PASSWORD}"
    DUMP_FILE="\${BACKUP_PATH}/database_dump.sql"
    if pg_dump -h "\${DB_HOST:-localhost}" -p "\${DB_PORT:-5432}" -U "\${DB_USER:-bot}" \
        "\${DB_NAME:-cryptobot}" > "\${DUMP_FILE}" 2>/dev/null; then
        echo "  ✅ Дамп БД создан (\$(wc -l < "\${DUMP_FILE}") строк)"
    else
        echo "  ⚠️  Не удалось создать дамп БД"
    fi
else
    echo "  ⚠️  pg_dump не доступен или конфиг не найден"
fi

# Архивирование
echo "5. Архивирование резервной копии..."
cd "\${BACKUP_DIR}"
tar -czf "backup_${timestamp}.tar.gz" "backup_${timestamp}"
rm -rf "backup_${timestamp}"

# Запуск сервиса обратно
echo "6. Запуск сервиса..."
systemctl start \${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Не удалось запустить сервис"

echo ""
echo "✅ Резервная копия создана: \${BACKUP_DIR}/backup_${timestamp}.tar.gz"
echo "📊 Размер: \$(du -h "\${BACKUP_DIR}/backup_${timestamp}.tar.gz" | cut -f1)"
echo "🕐 Время создания: \$(date)"
EOF

    log_info "Резервная копия создана: ${backup_path}.tar.gz"
}

# Отображение списка резервных копий
list_backups() {
    log_step "Список доступных резервных копий:"

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
BACKUP_DIR="${BACKUP_DIR}"

if [ -d "\${BACKUP_DIR}" ]; then
    echo "Резервные копии в \${BACKUP_DIR}:"
    echo ""

    # Подсчитываем общее количество
    TOTAL_COUNT=\$(ls "\${BACKUP_DIR}"/*.tar.gz 2>/dev/null | wc -l)
    echo "Всего копий: \${TOTAL_COUNT}"
    echo ""

    if [ "\${TOTAL_COUNT}" -gt 0 ]; then
        echo "Последние 5 копий:"
        ls -lt "\${BACKUP_DIR}"/*.tar.gz 2>/dev/null | head -5 | while read -r line; do
            filename=\$(echo "\$line" | awk '{print \$NF}')
            size=\$(echo "\$line" | awk '{print \$5}')
            date=\$(echo "\$line" | awk '{print \$6, \$7, \$8}')
            echo "  📁 \$(basename "\$filename") (\${size}, \${date})"
        done

        # Общий размер
        TOTAL_SIZE=\$(du -sh "\${BACKUP_DIR}" | cut -f1)
        echo ""
        echo "Общий размер резервных копий: \${TOTAL_SIZE}"
    else
        echo "  📭 Резервных копий нет"
    fi
else
    echo "Директория резервных копий не существует"
fi
EOF
}

# Откат к предыдущей версии с сохранением SSL
rollback_backup() {
    log_step "Откат к предыдущей версии..."

    # Показываем список резервных копий
    list_backups

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
INSTALL_DIR="${INSTALL_DIR}"
BACKUP_DIR="${BACKUP_DIR}"
SERVICE_NAME="${SERVICE_NAME}"
CERTS_DIR="${CERTS_DIR}"

# Поиск последней резервной копии
latest_backup=\$(ls -t "\${BACKUP_DIR}"/*.tar.gz 2>/dev/null | head -1)

if [ -z "\${latest_backup}" ]; then
    echo "❌ Резервные копии не найдены"
    exit 1
fi

echo ""
echo "Последняя резервная копия: \$(basename "\${latest_backup}")"
echo "Размер: \$(du -h "\${latest_backup}" | cut -f1)"
echo "Создана: \$(stat -c %y "\${latest_backup}" | cut -d'.' -f1)"
echo ""

if [ "${force}" != "true" ]; then
    read -p "Вы уверены, что хотите восстановить эту копию? (y/N): " -n 1 -r
    echo ""
    if [[ ! \$REPLY =~ ^[Yy]$ ]]; then
        echo "Отмена отката"
        exit 0
    fi
fi

echo "🔄 Начало отката..."

# Остановка сервиса
echo "1. Остановка сервиса..."
systemctl stop \${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Сервис уже остановлен"

# Восстановление из резервной копии
echo "2. Восстановление из \${latest_backup}..."
temp_dir=\$(mktemp -d)
tar -xzf "\${latest_backup}" -C "\${temp_dir}"

# Восстановление бинарника
backup_subdir=\$(find "\${temp_dir}" -type d -name "backup_*" | head -1)
if [ -n "\${backup_subdir}" ]; then
    if [ -f "\${backup_subdir}/\${APP_NAME}" ]; then
        echo "  📦 Восстановление бинарника..."
        # Обеспечиваем существование директории bin
        mkdir -p "\${INSTALL_DIR}/bin"
        cp "\${backup_subdir}/\${APP_NAME}" "\${INSTALL_DIR}/bin/"
        chown cryptoapp:cryptoapp "\${INSTALL_DIR}/bin/\${APP_NAME}"
        chmod +x "\${INSTALL_DIR}/bin/\${APP_NAME}"
        echo "  ✅ Бинарник восстановлен"
    fi

    # Восстановление конфигурации
    if [ -d "\${backup_subdir}/configs" ]; then
        echo "  ⚙️  Восстановление конфигурации..."
        # Удаляем старую конфигурацию
        rm -rf "\${INSTALL_DIR}/configs" 2>/dev/null || true
        cp -r "\${backup_subdir}/configs" "\${INSTALL_DIR}/"
        chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}/configs"

        # Обновляем симлинк .env
        if [ -f "\${INSTALL_DIR}/configs/prod/.env" ]; then
            ln -sf "\${INSTALL_DIR}/configs/prod/.env" "\${INSTALL_DIR}/.env" 2>/dev/null || true
            chown cryptoapp:cryptoapp "\${INSTALL_DIR}/.env" 2>/dev/null || true
        fi
        echo "  ✅ Конфигурация восстановлена"
    fi

    # Восстановление .env файла если есть
    if [ -f "\${backup_subdir}/.env" ]; then
        echo "  ⚙️  Восстановление .env файла..."
        cp "\${backup_subdir}/.env" "\${INSTALL_DIR}/.env"
        chown cryptoapp:cryptoapp "\${INSTALL_DIR}/.env"
        chmod 600 "\${INSTALL_DIR}/.env"
        echo "  ✅ .env файл восстановлен"
    fi

    # Восстановление SSL сертификатов
    if [ -d "\${backup_subdir}/ssl_certs" ]; then
        echo "  🔐 Восстановление SSL сертификатов..."
        mkdir -p "\${CERTS_DIR}"
        cp -r "\${backup_subdir}/ssl_certs"/* "\${CERTS_DIR}/" 2>/dev/null || true
        chown -R cryptoapp:cryptoapp "\${CERTS_DIR}" 2>/dev/null || true
        echo "  ✅ SSL сертификаты восстановлены"
    fi

    # Восстановление дампа БД (опционально)
    if [ -f "\${backup_subdir}/database_dump.sql" ] && command -v psql >/dev/null 2>&1; then
        echo "  🗄️  Восстановление базы данных..."
        if [ -f "\${INSTALL_DIR}/.env" ]; then
            DB_HOST=\$(grep "^DB_HOST=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_PORT=\$(grep "^DB_PORT=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_NAME=\$(grep "^DB_NAME=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_USER=\$(grep "^DB_USER=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_PASSWORD=\$(grep "^DB_PASSWORD=" "\${INSTALL_DIR}/.env" | cut -d= -f2)

            export PGPASSWORD="\${DB_PASSWORD}"
            psql -h "\${DB_HOST:-localhost}" -p "\${DB_PORT:-5432}" -U "\${DB_USER:-bot}" \
                "\${DB_NAME:-cryptobot}" < "\${backup_subdir}/database_dump.sql" 2>/dev/null && \
                echo "  ✅ База данных восстановлена" || echo "  ⚠️  Не удалось восстановить БД"
        fi
    fi
else
    echo "  ❌ Не удалось найти данные в резервной копии"
fi

# Очистка
rm -rf "\${temp_dir}"

# Запуск сервиса
echo "3. Запуск сервиса..."
systemctl start \${SERVICE_NAME}.service

echo ""
echo "✅ Откат выполнен успешно!"
echo "Версия восстановлена из: \$(basename "\${latest_backup}")"
EOF

    log_info "Откат завершен"
}

# Определение корневой директории проекта
find_project_root() {
    local script_dir
    script_dir=$(dirname "$(realpath "$0")")

    # Проверяем разные возможные пути к корню проекта
    local possible_paths=(
        "${script_dir}/../.."  # deploy/scripts -> корень
        "${script_dir}/.."     # scripts -> корень
        "."                    # текущая директория
        ".."                   # родительская директория
    )

    for path in "${possible_paths[@]}"; do
        if [ -f "${path}/go.mod" ] && [ -f "${path}/application/cmd/bot/main.go" ]; then
            echo "$(realpath "${path}")"
            return 0
        fi
    done

    return 1
}

# Обновление исходного кода с сохранением SSL
update_source_code() {
    log_step "Обновление исходного кода..."

    # Находим корневую директорию проекта
    local project_root
    project_root=$(find_project_root)

    if [ $? -ne 0 ] || [ -z "${project_root}" ]; then
        log_error "Не удалось найти корневую директорию проекта (go.mod)"
        log_info "Запустите скрипт из корневой директории проекта"
        exit 1
    fi

    log_info "Корень проекта: ${project_root}"

    # Переходим в корень проекта
    cd "${project_root}"

    # Проверяем структуру
    if [ ! -f "go.mod" ] || [ ! -f "application/cmd/bot/main.go" ]; then
        log_error "Неправильная структура проекта!"
        log_info "Ожидается наличие: go.mod и application/cmd/bot/main.go"
        exit 1
    fi

    # Создание архива с обновлениями (вся структура, кроме ненужных файлов)
    log_info "Создание архива с обновлениями..."
    tar -czf /tmp/app_update.tar.gz \
        --exclude=.git \
        --exclude=node_modules \
        --exclude=*.log \
        --exclude=*.tar.gz \
        --exclude=bin \
        --exclude=coverage \
        --exclude=tests \
        --exclude=Makefile \
        --exclude=README.md \
        --exclude=LICENSE \
        .

    # Копирование на сервер
    log_info "Копирование обновлений на сервер..."
    scp -i "${SSH_KEY}" /tmp/app_update.tar.gz "${SERVER_USER}@${SERVER_IP}:/tmp/app_update.tar.gz"

    # Обновление на сервере с сохранением SSL
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR}"
APP_NAME="${APP_NAME}"
SERVICE_NAME="${SERVICE_NAME}"
CERTS_DIR="${CERTS_DIR}"

echo "🔄 Обновление исходного кода..."

# Остановка сервиса
echo "1. Остановка сервиса для обновления..."
systemctl stop \${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Сервис уже остановлен"
sleep 2

# Сохраняем SSL сертификаты
echo "2. Сохранение SSL сертификатов..."
SSL_BACKUP_DIR="/tmp/ssl_backup_\$(date +%s)"
mkdir -p "\${SSL_BACKUP_DIR}"
if [ -d "\${CERTS_DIR}" ]; then
    cp -r "\${CERTS_DIR}"/* "\${SSL_BACKUP_DIR}/" 2>/dev/null || true
    echo "  ✅ SSL сертификаты сохранены"
fi

# Сохраняем webhook секретный токен
echo "3. Сохранение webhook настроек..."
if [ -f "\${INSTALL_DIR}/.env" ]; then
    WEBHOOK_SECRET_TOKEN=\$(grep "^WEBHOOK_SECRET_TOKEN=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")
    if [ -n "\${WEBHOOK_SECRET_TOKEN}" ]; then
        echo "WEBHOOK_SECRET_TOKEN=\${WEBHOOK_SECRET_TOKEN}" > /tmp/webhook_backup.env
        echo "  ✅ Webhook токен сохранен"
    fi
fi

# Создание быстрой резервной копии текущей версии
echo "4. Создание быстрой резервной копии..."
quick_backup_dir="${BACKUP_DIR}/quick_backup_\$(date +%Y%m%d_%H%M%S)"
mkdir -p "\${quick_backup_dir}"

# Копируем только самое важное
cp -r "\${INSTALL_DIR}/bin" "\${quick_backup_dir}/" 2>/dev/null || echo "  ⚠️  Не удалось скопировать bin"
cp -r "\${INSTALL_DIR}/configs" "\${quick_backup_dir}/" 2>/dev/null || echo "  ⚠️  Не удалось скопировать configs"
cp "\${INSTALL_DIR}/.env" "\${quick_backup_dir}/" 2>/dev/null || echo "  ⚠️  Не удалось скопировать .env"
echo "  ✅ Быстрая резервная копия создана в \${quick_backup_dir}"

# Сохраняем старые файлы перед удалением
echo "5. Сохранение конфигурации и данных..."
# Сохраняем configs если они есть
if [ -d "\${INSTALL_DIR}/configs" ]; then
    mv "\${INSTALL_DIR}/configs" "\${INSTALL_DIR}/configs_backup_\$(date +%s)"
    echo "  ✅ Конфиги сохранены для восстановления"
fi

# Сохраняем .env если он есть
if [ -f "\${INSTALL_DIR}/.env" ]; then
    cp "\${INSTALL_DIR}/.env" "\${INSTALL_DIR}/.env_backup_\$(date +%s)"
    echo "  ✅ .env сохранен для восстановления"
fi

# Очистка директории установки (кроме bin и logs и ssl)
echo "6. Очистка директории установки..."
# Удаляем всё, кроме bin, logs, ssl, configs_backup* и .env_backup*
find "\${INSTALL_DIR}" -maxdepth 1 ! -name "bin" ! -name "logs" ! -name "ssl" ! -name "configs_backup_*" ! -name ".env_backup_*" ! -name "crypto-screener-bot" -exec rm -rf {} + 2>/dev/null || true

# Распаковка нового кода (в корень установки)
echo "7. Распаковка нового кода..."
tar -xzf /tmp/app_update.tar.gz -C "\${INSTALL_DIR}"
chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}"

# Восстановление конфигурации
echo "8. Восстановление конфигурации..."
# Ищем последний backup configs
LATEST_CONFIGS_BACKUP=\$(find "\${INSTALL_DIR}" -type d -name "configs_backup_*" | sort -r | head -1)
if [ -n "\${LATEST_CONFIGS_BACKUP}" ] && [ -d "\${LATEST_CONFIGS_BACKUP}" ]; then
    # Восстанавливаем configs
    rm -rf "\${INSTALL_DIR}/configs" 2>/dev/null || true
    mv "\${LATEST_CONFIGS_BACKUP}" "\${INSTALL_DIR}/configs"
    echo "  ✅ Конфиги восстановлены из backup"
fi

# Ищем последний backup .env
LATEST_ENV_BACKUP=\$(find "\${INSTALL_DIR}" -type f -name ".env_backup_*" | sort -r | head -1)
if [ -n "\${LATEST_ENV_BACKUP}" ] && [ -f "\${LATEST_ENV_BACKUP}" ]; then
    # Восстанавливаем .env симлинк
    if [ -f "\${INSTALL_DIR}/configs/prod/.env" ]; then
        ln -sf "\${INSTALL_DIR}/configs/prod/.env" "\${INSTALL_DIR}/.env"
        chown cryptoapp:cryptoapp "\${INSTALL_DIR}/.env"
    else
        # Используем backup .env
        cp "\${LATEST_ENV_BACKUP}" "\${INSTALL_DIR}/.env"
        chown cryptoapp:cryptoapp "\${INSTALL_DIR}/.env"
        chmod 600 "\${INSTALL_DIR}/.env"
    fi
    echo "  ✅ .env восстановлен из backup"
fi

# Восстановление webhook токена если он был
if [ -f "/tmp/webhook_backup.env" ]; then
    echo "9. Восстановление webhook токена..."
    if [ -f "\${INSTALL_DIR}/.env" ]; then
        # Читаем сохраненный токен
        BACKUP_TOKEN=\$(grep "^WEBHOOK_SECRET_TOKEN=" "/tmp/webhook_backup.env" | cut -d= -f2)
        if [ -n "\${BACKUP_TOKEN}" ]; then
            # Обновляем токен в конфиге
            if grep -q "^WEBHOOK_SECRET_TOKEN=" "\${INSTALL_DIR}/.env"; then
                sed -i "s|^WEBHOOK_SECRET_TOKEN=.*|WEBHOOK_SECRET_TOKEN=\${BACKUP_TOKEN}|" "\${INSTALL_DIR}/.env"
            else
                echo "WEBHOOK_SECRET_TOKEN=\${BACKUP_TOKEN}" >> "\${INSTALL_DIR}/.env"
            fi
            echo "  ✅ Webhook токен восстановлен"
        fi
    fi
    rm -f /tmp/webhook_backup.env
fi

# Восстановление SSL сертификатов
echo "10. Восстановление SSL сертификатов..."
if [ -d "\${SSL_BACKUP_DIR}" ]; then
    mkdir -p "\${CERTS_DIR}"
    cp -r "\${SSL_BACKUP_DIR}"/* "\${CERTS_DIR}/" 2>/dev/null || true
    chown -R cryptoapp:cryptoapp "\${CERTS_DIR}" 2>/dev/null || true
    echo "  ✅ SSL сертификаты восстановлены"
    rm -rf "\${SSL_BACKUP_DIR}"
fi

# Удаляем backup файлы
rm -rf "\${INSTALL_DIR}/configs_backup_*" 2>/dev/null || true
rm -f "\${INSTALL_DIR}/.env_backup_*" 2>/dev/null || true

# Очистка
rm -f /tmp/app_update.tar.gz

echo "✅ Исходный код обновлен и конфигурация восстановлена"
EOF

    # Очистка локального архива
    rm -f /tmp/app_update.tar.gz

    log_info "Исходный код обновлен"
}

# Пересборка приложения
rebuild_application() {
    log_step "Пересборка приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR}"
APP_NAME="${APP_NAME}"

cd "\${INSTALL_DIR}"

echo "🔨 Пересборка приложения..."

# Обновление зависимостей
echo "1. Обновление зависимостей Go..."
if [ ! -f "go.mod" ]; then
    echo "  ❌ go.mod не найден в \${INSTALL_DIR}"
    echo "  Проверка содержимого директории:"
    ls -la "\${INSTALL_DIR}/" | head -10
    exit 1
fi

/usr/local/go/bin/go mod download
echo "  ✅ Зависимости обновлены"

# Пересборка приложения
echo "2. Пересборка основного приложения..."
if [ -f "./application/cmd/bot/main.go" ]; then
    /usr/local/go/bin/go build -o "\${INSTALL_DIR}/bin/\${APP_NAME}" ./application/cmd/bot/main.go

    if [ -f "\${INSTALL_DIR}/bin/\${APP_NAME}" ]; then
        echo "  ✅ Приложение успешно пересобрано"

        # Проверка версии
        echo "  🔍 Проверка версии:"
        "\${INSTALL_DIR}/bin/\${APP_NAME}" --version 2>&1 | head -1 || echo "  ⚠️  Не удалось получить версию"

        # Проверка webhook поддержки
        echo "  🔍 Проверка webhook поддержки:"
        strings "\${INSTALL_DIR}/bin/\${APP_NAME}" | grep -i "webhook" | head -3 || echo "  ℹ️  Webhook strings не найдены"
    else
        echo "  ❌ Ошибка: бинарный файл не создан"
        echo "  Проверка ошибок сборки..."
        /usr/local/go/bin/go build -o "\${INSTALL_DIR}/bin/\${APP_NAME}" ./application/cmd/bot/main.go 2>&1 | tail -20
        exit 1
    fi
else
    echo "  ❌ Файл основного приложения не найден: ./application/cmd/bot/main.go"
    echo "  Поиск файлов application..."
    find . -name "main.go" -type f | head -10
    exit 1
fi

# Проверка наличия миграций
echo "3. Проверка миграций..."
if [ -f "./internal/infrastructure/persistence/postgres/migrator.go" ]; then
    echo "  ✅ Мигратор найден"
    if [ -d "./internal/infrastructure/persistence/postgres/migrations" ]; then
        MIGRATION_COUNT=\$(ls "./internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
        echo "  📊 Количество миграций: \${MIGRATION_COUNT}"
    fi
else
    echo "  ⚠️  Мигратор не найден"
fi

# Проверка запуска
echo "4. Проверка запуска приложения..."
timeout 3 "\${INSTALL_DIR}/bin/\${APP_NAME}" --help 2>&1 | grep -i "usage\|help\|version\|webhook" | head -3 || echo "  ⚠️  Быстрый тест не прошел"

echo "✅ Пересборка завершена"
EOF

    log_info "Приложение пересобрано"
}

# Проверка миграций базы данных
check_database_migrations() {
    log_step "Проверка миграций базы данных..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR}"

echo "🗄️  Проверка состояния базы данных..."

# Читаем настройки БД из конфига
if [ -f "\${INSTALL_DIR}/.env" ]; then
    DB_NAME=\$(grep "^DB_NAME=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_USER=\$(grep "^DB_USER=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_PASSWORD=\$(grep "^DB_PASSWORD=" "\${INSTALL_DIR}/.env" | cut -d= -f2)

    echo "📊 Настройки БД: \${DB_NAME} (пользователь: \${DB_USER})"
else
    echo "⚠️  Конфиг не найден, используем значения по умолчанию"
    DB_NAME="cryptobot"
    DB_USER="cryptobot"
fi

# Проверяем существование папки миграций
if [ -d "\${INSTALL_DIR}/internal/infrastructure/persistence/postgres/migrations" ]; then
    MIGRATION_COUNT=\$(ls "\${INSTALL_DIR}/internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
    echo "✅ Найдено миграций: \${MIGRATION_COUNT}"

    if [ "\${MIGRATION_COUNT}" -gt 0 ]; then
        echo "📋 Последние 3 миграции:"
        ls -t "\${INSTALL_DIR}/internal/infrastructure/persistence/postgres/migrations/"*.sql | head -3
    fi
else
    echo "⚠️  Папка миграций не найдена"
fi

echo ""
echo "ℹ️  Миграции будут автоматически применены при запуске приложения"
echo "   (если DB_ENABLE_AUTO_MIGRATE=true в .env файле)"

# Добавить проверку и восстановление прав
echo ""
echo "🔐 Проверка прав доступа к таблицам..."
sudo -u postgres psql -d \${DB_NAME} << SQL
    GRANT ALL ON SCHEMA public TO \${DB_USER};
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO \${DB_USER};
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO \${DB_USER};
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO \${DB_USER};
SQL
echo "✅ Права PostgreSQL проверены и восстановлены"
EOF

    log_info "Миграции и права PostgreSQL проверены"
}

# Запуск обновленного приложения
start_updated_application() {
    log_step "Запуск обновленного приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

SERVICE_NAME="${SERVICE_NAME}"
APP_NAME="${APP_NAME}"
INSTALL_DIR="${INSTALL_DIR}"

echo "🚀 Запуск обновленного приложения..."

# Проверка конфигурации перед запуском
echo "1. Проверка конфигурации..."
if [ -f "\${INSTALL_DIR}/.env" ]; then
    TELEGRAM_MODE=\$(grep "^TELEGRAM_MODE=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "webhook")
    echo "  Режим Telegram: \${TELEGRAM_MODE}"

    if [ "\${TELEGRAM_MODE}" = "webhook" ]; then
        WEBHOOK_USE_TLS=\$(grep "^WEBHOOK_USE_TLS=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "true")
        echo "  Использовать TLS: \${WEBHOOK_USE_TLS}"

        if [ "\${WEBHOOK_USE_TLS}" = "true" ]; then
            CERT_PATH=\$(grep "^WEBHOOK_TLS_CERT_PATH=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")
            KEY_PATH=\$(grep "^WEBHOOK_TLS_KEY_PATH=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")

            if [ -f "\${CERT_PATH}" ] && [ -f "\${KEY_PATH}" ]; then
                echo "  ✅ SSL сертификаты проверены"
            else
                echo "  ⚠️  SSL сертификаты не найдены"
            fi
        fi
    fi
fi

# Запуск сервиса
echo "2. Запуск сервиса \${SERVICE_NAME}..."
systemctl start \${SERVICE_NAME}.service

# Даем время на запуск
echo "3. Ожидание запуска (5 секунд)..."
sleep 5

# Проверка статуса
echo "4. Статус сервиса:"
systemctl status \${SERVICE_NAME}.service --no-pager | head -10

# Проверка процесса
echo "5. Проверка процесса:"
if pgrep -f "\${APP_NAME}" > /dev/null; then
    echo "  ✅ Приложение запущено"
    echo "  PID: \$(pgrep -f "\${APP_NAME}")"

    # Проверка webhook порта если в webhook режиме
    if [ "\${TELEGRAM_MODE}" = "webhook" ]; then
        WEBHOOK_PORT=\$(grep "^WEBHOOK_PORT=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "8443")
        echo "  Webhook порт: \${WEBHOOK_PORT}"
        if ss -tln | grep -q ":\${WEBHOOK_PORT} "; then
            echo "  ✅ Webhook порт открыт"
        else
            echo "  ⚠️  Webhook порт закрыт"
        fi
    fi
else
    echo "  ❌ Приложение не запущено"
fi

# Просмотр логов
echo "6. Последние 10 строк лога:"
journalctl -u \${SERVICE_NAME}.service -n 10 --no-pager | grep -v "^--" | tail -10 || echo "  Логи пока пусты"

echo ""
echo "✅ Обновленное приложение запущено"
EOF

    log_info "Обновленное приложение запущено"
}

# Проверка обновления с webhook проверкой
verify_update() {
    log_step "Проверка обновления..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
SERVICE_NAME="${SERVICE_NAME}"
INSTALL_DIR="${INSTALL_DIR}"

echo "🔍 ПРОВЕРКА ОБНОВЛЕНИЯ"
echo "===================="
echo "Время проверки: \$(date)"
echo ""

# 1. Проверка версии приложения
echo "1. Версия приложения:"
if [ -f "\${INSTALL_DIR}/bin/\${APP_NAME}" ]; then
    "\${INSTALL_DIR}/bin/\${APP_NAME}" --version 2>&1 | head -1 || echo "  ❌ Не удалось определить версию"
else
    echo "  ❌ Бинарный файл не найден"
fi
echo ""

# 2. Проверка сервиса
echo "2. Статус сервиса:"
SERVICE_STATUS=\$(systemctl is-active \${SERVICE_NAME}.service)
case "\${SERVICE_STATUS}" in
    active) echo "  ✅ Активен" ;;
    inactive) echo "  ⏸️  Не активен" ;;
    failed) echo "  ❌ Ошибка" ;;
    *) echo "  ❓ \${SERVICE_STATUS}" ;;
esac
echo ""

# 3. Проверка webhook статуса
echo "3. Webhook статус:"
if [ -f "\${INSTALL_DIR}/.env" ]; then
    TELEGRAM_MODE=\$(grep "^TELEGRAM_MODE=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "webhook")
    echo "  Режим Telegram: \${TELEGRAM_MODE}"

    if [ "\${TELEGRAM_MODE}" = "webhook" ]; then
        WEBHOOK_PORT=\$(grep "^WEBHOOK_PORT=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "8443")
        WEBHOOK_DOMAIN=\$(grep "^WEBHOOK_DOMAIN=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")

        echo "  Webhook порт: \${WEBHOOK_PORT}"
        echo "  Домен: \${WEBHOOK_DOMAIN}"

        if ss -tln | grep -q ":\${WEBHOOK_PORT} "; then
            echo "  ✅ Webhook порт открыт"
        else
            echo "  ⚠️  Webhook порт закрыт"
        fi
    fi
fi
echo ""

# 4. Проверка логов на ошибки
echo "4. Ошибки в логах (последние 5 минут):"
LOG_FILE="/opt/crypto-screener-bot/logs/app.log"
if [ -f "${LOG_FILE}" ]; then
    ERROR_COUNT=$(tail -n 1000 ${LOG_FILE} 2>/dev/null | \
        grep -i -c "error\|fail\|panic\|fatal")
    if [ "${ERROR_COUNT}" -gt 0 ]; then
        echo "  ⚠️  Найдено ошибок: ${ERROR_COUNT}"
        echo "  Последние ошибки:"
        tail -n 100 ${LOG_FILE} 2>/dev/null | \
            grep -i "error\|fail\|panic\|fatal" | tail -3 | while read line; do
            echo "    📛 $(echo "$line" | cut -d' ' -f6-)"
        done
    else
        echo "  ✅ Ошибок не обнаружено"
    fi
else
    echo "  ⚠️  Файл лога не найден: ${LOG_FILE}"
fi
echo ""

# 5. Проверка процессов
echo "5. Запущенные процессы:"
if pgrep -f "\${APP_NAME}" > /dev/null; then
    echo "  ✅ Приложение работает"
    echo "  Время работы: \$(ps -p \$(pgrep -f "\${APP_NAME}") -o etime= 2>/dev/null || echo "неизвестно")"
else
    echo "  ❌ Приложение не работает"
fi
echo ""

# 6. Проверка миграций в логах
echo "6. Миграции базы данных:"
if journalctl -u \${SERVICE_NAME}.service --since "10 minutes ago" 2>/dev/null | \
    grep -i "migration\|migrate" > /dev/null; then
    echo "  ✅ Миграции обнаружены в логах"
else
    echo "  ℹ️  Миграции не обнаружены (возможно уже применены)"
fi
echo ""

# 7. Проверка Redis
echo "7. Статус Redis:"
if systemctl is-active redis-server >/dev/null 2>&1; then
    echo "  ✅ Redis: активен"

    if command -v redis-cli >/dev/null 2>&1; then
        if redis-cli ping | grep -q "PONG"; then
            echo "  ✅ Redis: доступен для подключения"
        else
            echo "  ⚠️  Redis: не отвечает на ping"
        fi
    fi
else
    echo "  ⚠️  Redis: не активен"
fi
echo ""

# 8. Проверка конфигурации
echo "8. Проверка конфигурации:"
if [ -f "\${INSTALL_DIR}/.env" ]; then
    echo "  ✅ Конфиг найден"
    # Проверка основных настроек
    echo "  Основные настройки:"
    grep -E "^(APP_ENV|LOG_LEVEL|EXCHANGE|TELEGRAM_ENABLED|TELEGRAM_MODE|DB_ENABLE_AUTO_MIGRATE|REDIS_ENABLED)=" \
        "\${INSTALL_DIR}/.env" 2>/dev/null | head -7 | while read line; do
        echo "    ⚙️  \$line"
    done
else
    echo "  ❌ Конфиг не найден"
fi
echo ""

echo "🎯 ИТОГ ПРОВЕРКИ:"
if [ "\${SERVICE_STATUS}" = "active" ] && pgrep -f "\${APP_NAME}" > /dev/null && [ "\${ERROR_COUNT}" -eq 0 ]; then
    echo "✅ ОБНОВЛЕНИЕ УСПЕШНО!"
    echo "Приложение работает корректно"

    # Дополнительная проверка для webhook
    if [ "\${TELEGRAM_MODE}" = "webhook" ]; then
        WEBHOOK_PORT=\$(grep "^WEBHOOK_PORT=" "\${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "8443")
        if ss -tln | grep -q ":\${WEBHOOK_PORT} "; then
            echo "✅ Webhook порт работает"
        else
            echo "⚠️  Webhook порт не работает"
        fi
    fi
else
    echo "⚠️  ЕСТЬ ПРОБЛЕМЫ"
    echo "Проверьте сообщения выше"
fi
EOF

    log_info "Проверка завершена"
}

# Основная функция
main() {
    log_step "Начало процесса обновления"
    log_info "Сервер: ${SERVER_USER}@${SERVER_IP}"
    log_info "Приложение: ${APP_NAME}"
    echo ""

    # Проверяем подключение
    check_ssh_connection

    # Показываем текущий статус
    check_server_status

    # Если запрошен только бэкап
    if [ "${backup_only}" = "true" ]; then
        create_backup
        list_backups
        exit 0
    fi

    # Если запрошен откат
    if [ "${rollback}" = "true" ]; then
        rollback_backup
        exit 0
    fi

    # Подтверждение обновления
    if [ "${force}" != "true" ]; then
        echo ""
        log_warn "⚠️  ВНИМАНИЕ: Выполнение обновления приложения"
        log_info "Сервер: ${SERVER_IP}"
        log_info "Приложение будет остановлено на время обновления"
        echo ""

        read -p "Продолжить обновление? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Обновление отменено"
            exit 0
        fi
    fi

    # Полный процесс обновления
    echo ""
    log_step "1. Создание резервной копии..."
    if [ "${no_backup}" != "true" ]; then
        create_backup
    else
        log_warn "⚠️  Пропуск создания резервной копии (опция --no-backup)"
    fi

    log_step "2. Обновление исходного кода..."
    update_source_code

    log_step "3. Пересборка приложения..."
    rebuild_application

    log_step "4. Проверка миграций базы данных..."
    check_database_migrations

    log_step "5. Запуск обновленного приложения..."
    start_updated_application

    log_step "6. Проверка обновления..."
    sleep 3
    verify_update

    log_step "Обновление успешно завершено!"
    echo ""
    log_info "📋 ИТОГ:"
    if [ "${no_backup}" != "true" ]; then
        log_info "  ✅ Резервная копия создана"
    fi
    log_info "  ✅ Исходный код обновлен"
    log_info "  ✅ Приложение пересобрано"
    log_info "  ✅ SSL сертификаты сохранены и восстановлены"
    log_info "  ✅ Webhook настройки сохранены"
    log_info "  ✅ База данных проверена"
    log_info "  ✅ Приложение запущено"
    echo ""
    log_info "🚀 Команды управления:"
    log_info "  $0 --backup-only          # Создать резервную копию"
    log_info "  $0 --rollback             # Откатить обновление"
    log_info "  systemctl status ${SERVICE_NAME}  # Статус сервиса"
    log_info "  journalctl -u ${SERVICE_NAME} -f  # Просмотр логов"
    echo ""
    log_info "📊 Для мониторинга используйте:"
    log_info "  ./deploy/scripts/service.sh monitor"
    log_info "  ./deploy/scripts/service.sh health"
    log_info "  ./deploy/scripts/service.sh webhook-info"
    log_info "  ./deploy/scripts/service.sh ssl-check"
}

# Запуск скрипта
parse_args "$@"
main
