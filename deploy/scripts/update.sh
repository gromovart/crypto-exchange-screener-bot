#!/bin/bash
# Скрипт обновления приложения на Ubuntu 22.04
# Использование: ./update.sh [OPTIONS]

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

# Проверка состояния сервера
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
if [ -f "/opt/crypto-screener-bot/bin/crypto-screener-bot" ]; then
    /opt/crypto-screener-bot/bin/crypto-screener-bot --version 2>&1 | head -1 || echo "  Не удалось определить версию"
else
    echo "  ❌ Приложение не установлено"
fi
echo ""

# 4. Дисковое пространство
echo "4. Дисковое пространство:"
df -h /opt /var/log | grep -v Filesystem
echo ""

echo "=== ПРОВЕРКА ЗАВЕРШЕНА ==="
EOF
}

# Создание резервной копии
create_backup() {
    log_step "Создание резервной копии..."

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${BACKUP_DIR}/backup_${timestamp}"

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
INSTALL_DIR="${INSTALL_DIR}"
BACKUP_PATH="${backup_path}"
SERVICE_NAME="${SERVICE_NAME}"

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

# Копирование исходного кода
if [ -d "\${INSTALL_DIR}/src" ]; then
    cp -r "\${INSTALL_DIR}/src" "\${BACKUP_PATH}/"
    echo "  ✅ Исходный код скопирован"
else
    echo "  ⚠️  Исходный код не найден"
fi

# Создание дампа базы данных
echo "3. Создание дампа базы данных..."
if command -v pg_dump >/dev/null 2>&1 && [ -f "\${INSTALL_DIR}/.env" ]; then
    # Читаем настройки БД из конфига
    DB_HOST=\$(grep "^DB_HOST=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_PORT=\$(grep "^DB_PORT=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_NAME=\$(grep "^DB_NAME=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_USER=\$(grep "^DB_USER=" "\${INSTALL_DIR}/.env" | cut -d= -f2)
    DB_PASSWORD=\$(grep "^DB_PASSWORD=" "\${INSTALL_DIR}/.env" | cut -d= -f2)

    export PGPASSWORD="\${DB_PASSWORD}"
    if pg_dump -h "\${DB_HOST:-localhost}" -p "\${DB_PORT:-5432}" -U "\${DB_USER:-crypto_screener}" \
        "\${DB_NAME:-crypto_screener_db}" > "\${BACKUP_PATH}/database_dump.sql" 2>/dev/null; then
        echo "  ✅ Дамп БД создан"
    else
        echo "  ⚠️  Не удалось создать дамп БД"
    fi
else
    echo "  ⚠️  pg_dump не доступен или конфиг не найден"
fi

# Архивирование
echo "4. Архивирование резервной копии..."
cd "\${BACKUP_DIR}"
tar -czf "backup_\${timestamp}.tar.gz" "backup_\${timestamp}"
rm -rf "backup_\${timestamp}"

# Запуск сервиса обратно
echo "5. Запуск сервиса..."
systemctl start \${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Не удалось запустить сервис"

echo ""
echo "✅ Резервная копия создана: \${BACKUP_DIR}/backup_\${timestamp}.tar.gz"
echo "📊 Размер: \$(du -h "\${BACKUP_DIR}/backup_\${timestamp}.tar.gz" | cut -f1)"
echo "🕐 Время создания: \$(date)"
EOF

    log_info "Резервная копия создана: ${backup_path}.tar.gz"
}

# Отображение списка резервных копий
list_backups() {
    log_step "Список доступных резервных копий:"

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/crypto-screener-bot_backups"

if [ -d "${BACKUP_DIR}" ]; then
    echo "Резервные копии в ${BACKUP_DIR}:"
    echo ""

    # Подсчитываем общее количество
    TOTAL_COUNT=$(ls "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | wc -l)
    echo "Всего копий: ${TOTAL_COUNT}"
    echo ""

    if [ "${TOTAL_COUNT}" -gt 0 ]; then
        echo "Последние 5 копий:"
        ls -lt "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | head -5 | while read -r line; do
            filename=$(echo "$line" | awk '{print $NF}')
            size=$(echo "$line" | awk '{print $5}')
            date=$(echo "$line" | awk '{print $6, $7, $8}')
            echo "  📁 $(basename "$filename") (${size}, ${date})"
        done

        # Общий размер
        TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" | cut -f1)
        echo ""
        echo "Общий размер резервных копий: ${TOTAL_SIZE}"
    else
        echo "  📭 Резервных копий нет"
    fi
else
    echo "Директория резервных копий не существует"
fi
EOF
}

# Откат к предыдущей версии
rollback_backup() {
    log_step "Откат к предыдущей версии..."

    # Показываем список резервных копий
    list_backups

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/crypto-screener-bot"
BACKUP_DIR="/opt/crypto-screener-bot_backups"
SERVICE_NAME="crypto-screener"

# Поиск последней резервной копии
latest_backup=$(ls -t "${BACKUP_DIR}"/*.tar.gz 2>/dev/null | head -1)

if [ -z "${latest_backup}" ]; then
    echo "❌ Резервные копии не найдены"
    exit 1
fi

echo ""
echo "Последняя резервная копия: $(basename "${latest_backup}")"
echo "Размер: $(du -h "${latest_backup}" | cut -f1)"
echo "Создана: $(stat -c %y "${latest_backup}" | cut -d'.' -f1)"
echo ""

if [ "${force:-false}" != "true" ]; then
    read -p "Вы уверены, что хотите восстановить эту копию? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Отмена отката"
        exit 0
    fi
fi

echo "🔄 Начало отката..."

# Остановка сервиса
echo "1. Остановка сервиса..."
systemctl stop ${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Сервис уже остановлен"

# Восстановление из резервной копии
echo "2. Восстановление из ${latest_backup}..."
temp_dir=$(mktemp -d)
tar -xzf "${latest_backup}" -C "${temp_dir}"

# Восстановление бинарника
backup_subdir=$(find "${temp_dir}" -type d -name "backup_*" | head -1)
if [ -n "${backup_subdir}" ]; then
    if [ -f "${backup_subdir}/${APP_NAME}" ]; then
        echo "  📦 Восстановление бинарника..."
        cp "${backup_subdir}/${APP_NAME}" "${INSTALL_DIR}/bin/"
        chown cryptoapp:cryptoapp "${INSTALL_DIR}/bin/${APP_NAME}"
        chmod +x "${INSTALL_DIR}/bin/${APP_NAME}"
        echo "  ✅ Бинарник восстановлен"
    fi

    # Восстановление конфигурации (если существует в бэкапе)
    if [ -d "${backup_subdir}/configs" ]; then
        echo "  ⚙️  Восстановление конфигурации..."
        rm -rf "${INSTALL_DIR}/configs"
        cp -r "${backup_subdir}/configs" "${INSTALL_DIR}/"
        chown -R cryptoapp:cryptoapp "${INSTALL_DIR}/configs"

        # Обновляем симлинк
        ln -sf "${INSTALL_DIR}/configs/prod/.env" "${INSTALL_DIR}/.env" 2>/dev/null || true
        echo "  ✅ Конфигурация восстановлена"
    fi

    # Восстановление дампа БД (опционально)
    if [ -f "${backup_subdir}/database_dump.sql" ] && command -v psql >/dev/null 2>&1; then
        echo "  🗄️  Восстановление базы данных..."
        if [ -f "${INSTALL_DIR}/.env" ]; then
            DB_HOST=$(grep "^DB_HOST=" "${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_PORT=$(grep "^DB_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_NAME=$(grep "^DB_NAME=" "${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_USER=$(grep "^DB_USER=" "${INSTALL_DIR}/.env" | cut -d= -f2)
            DB_PASSWORD=$(grep "^DB_PASSWORD=" "${INSTALL_DIR}/.env" | cut -d= -f2)

            export PGPASSWORD="${DB_PASSWORD}"
            psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-crypto_screener}" \
                "${DB_NAME:-crypto_screener_db}" < "${backup_subdir}/database_dump.sql" 2>/dev/null && \
                echo "  ✅ База данных восстановлена" || echo "  ⚠️  Не удалось восстановить БД"
        fi
    fi
else
    echo "  ❌ Не удалось найти данные в резервной копии"
fi

# Очистка
rm -rf "${temp_dir}"

# Запуск сервиса
echo "3. Запуск сервиса..."
systemctl start ${SERVICE_NAME}.service

echo ""
echo "✅ Откат выполнен успешно!"
echo "Версия восстановлена из: $(basename "${latest_backup}")"
EOF

    log_info "Откат завершен"
}

# Обновление исходного кода
update_source_code() {
    log_step "Обновление исходного кода..."

    # Проверяем, что мы в корне репозитория
    if [ ! -f "go.mod" ] || [ ! -f "application/cmd/bot/main.go" ]; then
        log_error "Скрипт должен запускаться из корневой директории репозитория!"
        exit 1
    fi

    # Создание архива с обновлениями
    log_info "Создание архива с обновлениями..."
    tar -czf /tmp/app_update.tar.gz \
        --exclude=.git \
        --exclude=node_modules \
        --exclude=*.log \
        --exclude=*.tar.gz \
        --exclude=bin \
        --exclude=coverage \
        .

    # Копирование на сервер
    log_info "Копирование обновлений на сервер..."
    scp -i "${SSH_KEY}" /tmp/app_update.tar.gz "${SERVER_USER}@${SERVER_IP}:/tmp/app_update.tar.gz"

    # Обновление на сервере
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR}"
APP_NAME="${APP_NAME}"
SERVICE_NAME="${SERVICE_NAME}"

echo "🔄 Обновление исходного кода..."

# Остановка сервиса
echo "1. Остановка сервиса для обновления..."
systemctl stop \${SERVICE_NAME}.service 2>/dev/null || echo "  ⚠️  Сервис уже остановлен"
sleep 2

# Создание быстрой резервной копии текущей версии
echo "2. Создание быстрой резервной копии..."
quick_backup_dir="\${INSTALL_DIR}_backups/quick_backup_\$(date +%Y%m%d_%H%M%S)"
mkdir -p "\${quick_backup_dir}"
cp -r "\${INSTALL_DIR}/bin" "\${quick_backup_dir}/" 2>/dev/null || echo "  ⚠️  Не удалось скопировать bin"
echo "  ✅ Быстрая резервная копия создана"

# Очистка старого исходного кода
echo "3. Очистка старого исходного кода..."
rm -rf "\${INSTALL_DIR}/src"

# Распаковка нового кода
echo "4. Распаковка нового кода..."
mkdir -p "\${INSTALL_DIR}/src"
tar -xzf /tmp/app_update.tar.gz -C "\${INSTALL_DIR}/src"
chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}/src"

# Очистка
rm -f /tmp/app_update.tar.gz

echo "✅ Исходный код обновлен"
EOF

    # Очистка локального архива
    rm -f /tmp/app_update.tar.gz

    log_info "Исходный код обновлен"
}

# Пересборка приложения
rebuild_application() {
    log_step "Пересборка приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"
APP_NAME="crypto-screener-bot"
SRC_DIR="${INSTALL_DIR}/src"

cd "${SRC_DIR}"

echo "🔨 Пересборка приложения..."

# Обновление зависимостей
echo "1. Обновление зависимостей Go..."
/usr/local/go/bin/go mod download
echo "  ✅ Зависимости обновлены"

# Пересборка приложения
echo "2. Пересборка основного приложения..."
if [ -f "./application/cmd/bot/main.go" ]; then
    /usr/local/go/bin/go build -o "${INSTALL_DIR}/bin/${APP_NAME}" ./application/cmd/bot/main.go

    if [ -f "${INSTALL_DIR}/bin/${APP_NAME}" ]; then
        echo "  ✅ Приложение успешно пересобрано"

        # Проверка версии
        echo "  🔍 Проверка версии:"
        "${INSTALL_DIR}/bin/${APP_NAME}" --version 2>&1 | head -1 || echo "  ⚠️  Не удалось получить версию"
    else
        echo "  ❌ Ошибка: бинарный файл не создан"
        exit 1
    fi
else
    echo "  ❌ Файл основного приложения не найден"
    exit 1
fi

# Проверка наличия миграций
echo "3. Проверка миграций..."
if [ -f "./internal/infrastructure/persistence/postgres/migrator.go" ]; then
    echo "  ✅ Мигратор найден"
    if [ -d "./internal/infrastructure/persistence/postgres/migrations" ]; then
        MIGRATION_COUNT=$(ls "./internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
        echo "  📊 Количество миграций: ${MIGRATION_COUNT}"
    fi
else
    echo "  ⚠️  Мигратор не найден"
fi

# Проверка запуска
echo "4. Проверка запуска приложения..."
timeout 3 "${INSTALL_DIR}/bin/${APP_NAME}" --help 2>&1 | grep -i "usage\|help\|version" | head -2 || echo "  ⚠️  Быстрый тест не прошел"

echo "✅ Пересборка завершена"
EOF

    log_info "Приложение пересобрано"
}

# Проверка миграций базы данных
check_database_migrations() {
    log_step "Проверка миграций базы данных..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"
APP_NAME="crypto-screener-bot"

echo "🗄️  Проверка состояния базы данных..."

# Проверяем существование папки миграций
if [ -d "${INSTALL_DIR}/src/internal/infrastructure/persistence/postgres/migrations" ]; then
    MIGRATION_COUNT=$(ls "${INSTALL_DIR}/src/internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
    echo "✅ Найдено миграций: ${MIGRATION_COUNT}"

    if [ "${MIGRATION_COUNT}" -gt 0 ]; then
        echo "📋 Последние 3 миграции:"
        ls -t "${INSTALL_DIR}/src/internal/infrastructure/persistence/postgres/migrations/"*.sql | head -3
    fi
else
    echo "⚠️  Папка миграций не найдена"
fi

echo ""
echo "ℹ️  Миграции будут автоматически применены при запуске приложения"
echo "   Проверка будет выполнена при следующем запуске сервиса"
EOF

    log_info "Миграции проверены"
}

# Запуск обновленного приложения
start_updated_application() {
    log_step "Запуск обновленного приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

SERVICE_NAME="crypto-screener"
APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/crypto-screener-bot"

echo "🚀 Запуск обновленного приложения..."

# Запуск сервиса
echo "1. Запуск сервиса ${SERVICE_NAME}..."
systemctl start ${SERVICE_NAME}.service

# Даем время на запуск
echo "2. Ожидание запуска (5 секунд)..."
sleep 5

# Проверка статуса
echo "3. Статус сервиса:"
systemctl status ${SERVICE_NAME}.service --no-pager | head -10

# Проверка процесса
echo "4. Проверка процесса:"
if pgrep -f "${APP_NAME}" > /dev/null; then
    echo "  ✅ Приложение запущено"
    echo "  PID: $(pgrep -f "${APP_NAME}")"
else
    echo "  ❌ Приложение не запущено"
fi

# Просмотр логов
echo "5. Последние 10 строк лога:"
journalctl -u ${SERVICE_NAME}.service -n 10 --no-pager | grep -v "^--" | tail -10 || echo "  Логи пока пусты"

echo ""
echo "✅ Обновленное приложение запущено"
EOF

    log_info "Обновленное приложение запущено"
}

# Проверка обновления
verify_update() {
    log_step "Проверка обновления..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
SERVICE_NAME="crypto-screener"
INSTALL_DIR="/opt/crypto-screener-bot"

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

# 3. Проверка логов на ошибки
echo "3. Ошибки в логах (последние 5 минут):"
ERROR_COUNT=\$(journalctl -u \${SERVICE_NAME}.service --since "5 minutes ago" 2>/dev/null | \
    grep -i -c "error\|fail\|panic\|fatal")
if [ "\${ERROR_COUNT}" -gt 0 ]; then
    echo "  ⚠️  Найдено ошибок: \${ERROR_COUNT}"
    echo "  Последние ошибки:"
    journalctl -u \${SERVICE_NAME}.service --since "5 minutes ago" 2>/dev/null | \
        grep -i "error\|fail\|panic\|fatal" | tail -3 | while read line; do
        echo "    📛 \$(echo "\$line" | cut -d' ' -f6-)"
    done
else
    echo "  ✅ Ошибок не обнаружено"
fi
echo ""

# 4. Проверка процессов
echo "4. Запущенные процессы:"
if pgrep -f "\${APP_NAME}" > /dev/null; then
    echo "  ✅ Приложение работает"
    echo "  Время работы: \$(ps -p \$(pgrep -f "\${APP_NAME}") -o etime= 2>/dev/null || echo "неизвестно")"
else
    echo "  ❌ Приложение не работает"
fi
echo ""

# 5. Проверка миграций в логах
echo "5. Миграции базы данных:"
if journalctl -u \${SERVICE_NAME}.service --since "10 minutes ago" 2>/dev/null | \
    grep -i "migration\|migrate" > /dev/null; then
    echo "  ✅ Миграции обнаружены в логах"
else
    echo "  ℹ️  Миграции не обнаружены (возможно уже применены)"
fi
echo ""

echo "🎯 ИТОГ ПРОВЕРКИ:"
if [ "\${SERVICE_STATUS}" = "active" ] && pgrep -f "\${APP_NAME}" > /dev/null && [ "\${ERROR_COUNT}" -eq 0 ]; then
    echo "✅ ОБНОВЛЕНИЕ УСПЕШНО!"
    echo "Приложение работает корректно"
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
    if [ "${backup_only:-false}" = "true" ]; then
        create_backup
        list_backups
        exit 0
    fi

    # Если запрошен откат
    if [ "${rollback:-false}" = "true" ]; then
        rollback_backup
        exit 0
    fi

    # Подтверждение обновления
    if [ "${force:-false}" != "true" ]; then
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
    if [ "${no_backup:-false}" != "true" ]; then
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
    log_info "  ✅ Резервная копия создана (если не отключена)"
    log_info "  ✅ Исходный код обновлен"
    log_info "  ✅ Приложение пересобрано"
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
}

# Запуск скрипта
parse_args "$@"
main