#!/bin/bash
# Скрипт первичного развертывания приложения на Ubuntu 22.04
# Использование: ./deploy/scripts/deploy.sh [OPTIONS]
# Опции:
#   --ip=95.142.40.244    IP адрес сервера
#   --user=root          Пользователь для подключения
#   --key=~/.ssh/id_rsa  SSH ключ


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

# Переменные для настроек БД (будут заполнены из .env)
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="cryptobot"
DB_USER="bot"
DB_PASSWORD="SecurePass123!"
DB_ENABLE_AUTO_MIGRATE="true"
REDIS_HOST="localhost"
REDIS_PORT="6379"
REDIS_PASSWORD=""  # Добавляем переменную для пароля Redis
REDIS_ENABLED="true"  # Добавляем флаг включения Redis

# Webhook параметры по умолчанию
WEBHOOK_DOMAIN="bot.gromovart.ru"
WEBHOOK_PORT="8443"
WEBHOOK_USE_TLS="true"
WEBHOOK_SECRET_TOKEN=""
TELEGRAM_MODE="webhook"  # По умолчанию webhook режим

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
    echo "  --help               Показать эту справку"
    echo ""
    echo "Примеры:"
    echo "  $0 --ip=95.142.40.244 --user=root"
    echo "  $0 --ip=192.168.1.100 --user=ubuntu --key=~/.ssh/my_key"
}

# Чтение настроек из .env файла
read_env_config() {
    log_step "Чтение настроек из конфигурации..."

    # Находим корень проекта
    local project_root
    project_root=$(find_project_root)
    if [ $? -ne 0 ] || [ -z "${project_root}" ]; then
        log_error "Не удалось найти корневую директорию проекта"
        exit 1
    fi

    local env_file="${project_root}/configs/prod/.env"

    if [ -f "${env_file}" ]; then
        log_info "✅ Чтение настроек из: ${env_file}"

        # Читаем настройки БД
        if grep -q "^DB_HOST=" "${env_file}"; then
            DB_HOST=$(grep "^DB_HOST=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   DB_HOST: ${DB_HOST}"
        fi

        if grep -q "^DB_PORT=" "${env_file}"; then
            DB_PORT=$(grep "^DB_PORT=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   DB_PORT: ${DB_PORT}"
        fi

        if grep -q "^DB_NAME=" "${env_file}"; then
            DB_NAME=$(grep "^DB_NAME=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   DB_NAME: ${DB_NAME}"
        fi

        if grep -q "^DB_USER=" "${env_file}"; then
            DB_USER=$(grep "^DB_USER=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   DB_USER: ${DB_USER}"
        fi

        if grep -q "^DB_PASSWORD=" "${env_file}"; then
            DB_PASSWORD=$(grep "^DB_PASSWORD=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   DB_PASSWORD: [скрыто]"
        fi

        if grep -q "^DB_ENABLE_AUTO_MIGRATE=" "${env_file}"; then
            DB_ENABLE_AUTO_MIGRATE=$(grep "^DB_ENABLE_AUTO_MIGRATE=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   DB_ENABLE_AUTO_MIGRATE: ${DB_ENABLE_AUTO_MIGRATE}"
        fi

        # Читаем настройки Redis
        if grep -q "^REDIS_HOST=" "${env_file}"; then
            REDIS_HOST=$(grep "^REDIS_HOST=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   REDIS_HOST: ${REDIS_HOST}"
        fi

        if grep -q "^REDIS_PORT=" "${env_file}"; then
            REDIS_PORT=$(grep "^REDIS_PORT=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   REDIS_PORT: ${REDIS_PORT}"
        fi

        if grep -q "^REDIS_PASSWORD=" "${env_file}"; then
            REDIS_PASSWORD=$(grep "^REDIS_PASSWORD=" "${env_file}" | cut -d= -f2- | xargs)
            if [ -n "${REDIS_PASSWORD}" ]; then
                log_info "   REDIS_PASSWORD: [скрыто]"
            fi
        fi

        if grep -q "^REDIS_ENABLED=" "${env_file}"; then
            REDIS_ENABLED=$(grep "^REDIS_ENABLED=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   REDIS_ENABLED: ${REDIS_ENABLED}"
        fi

        # Читаем настройки Webhook
        if grep -q "^TELEGRAM_MODE=" "${env_file}"; then
            TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   TELEGRAM_MODE: ${TELEGRAM_MODE}"
        fi

        if grep -q "^WEBHOOK_DOMAIN=" "${env_file}"; then
            WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   WEBHOOK_DOMAIN: ${WEBHOOK_DOMAIN}"
        fi

        if grep -q "^WEBHOOK_PORT=" "${env_file}"; then
            WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   WEBHOOK_PORT: ${WEBHOOK_PORT}"
        fi

        if grep -q "^WEBHOOK_USE_TLS=" "${env_file}"; then
            WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${env_file}" | cut -d= -f2- | xargs)
            log_info "   WEBHOOK_USE_TLS: ${WEBHOOK_USE_TLS}"
        fi

        if grep -q "^WEBHOOK_SECRET_TOKEN=" "${env_file}"; then
            WEBHOOK_SECRET_TOKEN=$(grep "^WEBHOOK_SECRET_TOKEN=" "${env_file}" | cut -d= -f2- | xargs)
            if [ -n "${WEBHOOK_SECRET_TOKEN}" ]; then
                log_info "   WEBHOOK_SECRET_TOKEN: [скрыто]"
            fi
        fi

        log_info "✅ Настройки прочитаны из конфига"
    else
        log_warn "⚠️  Конфиг не найден, будут использованы значения по умолчанию"
        log_info "   DB: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
        log_info "   Redis: ${REDIS_HOST}:${REDIS_PORT}"
        log_info "   Webhook: ${WEBHOOK_DOMAIN}:${WEBHOOK_PORT} (TLS: ${WEBHOOK_USE_TLS})"
        log_info "   Telegram режим: ${TELEGRAM_MODE}"
    fi
}

# Создание SSH ключа
create_ssh_key() {
    log_step "Создание нового SSH ключа..."

    local new_key="${HOME}/.ssh/id_rsa_crypto"

    if [ -f "${new_key}" ]; then
        log_warn "Ключ уже существует: ${new_key}"
        SSH_KEY="${new_key}"
        return
    fi

    ssh-keygen -t rsa -b 4096 -f "${new_key}" -N "" -q

    if [ $? -eq 0 ]; then
        log_info "✅ SSH ключ создан: ${new_key}"
        SSH_KEY="${new_key}"

        echo ""
        log_info "Нужно скопировать публичный ключ на сервер."
        log_info "Выполните команду и введите пароль сервера:"
        echo ""
        echo "ssh-copy-id -i ${new_key}.pub ${SERVER_USER}@${SERVER_IP}"
        echo ""
        read -p "Скопировать ключ сейчас? (y/N): " -n 1 -r
        echo ""

        if [[ $REPLY =~ ^[Yy]$ ]]; then
            ssh-copy-id -i "${new_key}.pub" "${SERVER_USER}@${SERVER_IP}"
            if [ $? -eq 0 ]; then
                log_info "✅ Ключ скопирован на сервер"
                return 0
            else
                log_error "Не удалось скопировать ключ"
                log_info "Скопируйте вручную:"
                echo "cat ${new_key}.pub | ssh ${SERVER_USER}@${SERVER_IP} 'mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys'"
                return 1
            fi
        fi
    else
        log_error "Не удалось создать SSH ключ"
        return 1
    fi
}

# Проверка SSH подключения
check_ssh_connection() {
    log_step "Проверка SSH подключения к серверу..."

    # Проверяем базовую доступность
    if ! ping -c 1 -W 1 "${SERVER_IP}" &> /dev/null; then
        log_error "Сервер не отвечает на ping"
        exit 1
    fi

    log_info "✅ Сервер доступен по ping"

    # Проверяем SSH порт
    if ! nc -z -w 2 "${SERVER_IP}" 22 &> /dev/null; then
        log_error "SSH порт (22) закрыт"
        exit 1
    fi

    log_info "✅ SSH порт открыт"

    # Проверяем SSH ключ
    if [ ! -f "${SSH_KEY}" ]; then
        log_warn "SSH ключ не найден: ${SSH_KEY}"
        log_info "Создаем новый ключ..."
        create_ssh_key
    fi

    # Проверяем права на ключ
    if [ -f "${SSH_KEY}" ]; then
        KEY_PERMS=$(stat -f "%A" "${SSH_KEY}" 2>/dev/null || stat -c "%a" "${SSH_KEY}")
        if [ "$KEY_PERMS" != "600" ]; then
            log_warn "Исправляем права SSH ключа..."
            chmod 600 "${SSH_KEY}"
        fi
    fi

    # Пробуем подключиться с ключом
    log_info "Тестирование SSH подключения с ключом..."

    if ssh -o BatchMode=yes \
           -o ConnectTimeout=5 \
           -i "${SSH_KEY}" \
           "${SERVER_USER}@${SERVER_IP}" "echo 'SSH ключ работает'" &> /dev/null; then
        log_info "✅ SSH ключ авторизован на сервере"
        return 0
    else
        log_warn "SSH ключ не авторизован на сервере"
        echo ""
        log_info "Нужно скопировать публичный ключ на сервер."
        log_info "Выполните команду и введите пароль сервера:"
        echo ""
        echo "ssh-copy-id -i ${SSH_KEY}.pub ${SERVER_USER}@${SERVER_IP}"
        echo ""

        read -p "Попробовать скопировать ключ сейчас? (y/N): " -n 1 -r
        echo ""

        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if [ ! -f "${SSH_KEY}.pub" ]; then
                log_error "Публичный ключ не найден: ${SSH_KEY}.pub"
                log_info "Создайте публичный ключ:"
                echo "ssh-keygen -y -f ${SSH_KEY} > ${SSH_KEY}.pub"
                exit 1
            fi

            ssh-copy-id -i "${SSH_KEY}.pub" "${SERVER_USER}@${SERVER_IP}"
            if [ $? -eq 0 ]; then
                log_info "✅ Ключ скопирован на сервер"

                # Проверяем снова
                if ssh -o BatchMode=yes \
                       -o ConnectTimeout=5 \
                       -i "${SSH_KEY}" \
                       "${SERVER_USER}@${SERVER_IP}" "echo 'SSH ключ работает'" &> /dev/null; then
                    log_info "✅ SSH подключение успешно установлено"
                    return 0
                fi
            else
                log_error "Не удалось скопировать ключ"
            fi
        fi

        log_error "SSH подключение с ключом не работает"
        log_info "Используйте скрипт диагностики для проверки:"
        log_info "  ./deploy/scripts/check-connection.sh"
        log_info ""
        log_info "Или настройте SSH ключ вручную:"
        log_info "  1. Сгенерируйте новый ключ: ssh-keygen -t rsa"
        log_info "  2. Скопируйте на сервер: ssh-copy-id -i ~/.ssh/id_rsa.pub root@${SERVER_IP}"
        log_info "  3. Запустите развертывание снова"
        exit 1
    fi
}

# Находим корневую директорию проекта
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

    log_error "Не удалось найти корневую директорию проекта"
    log_info "Запустите скрипт из директории проекта или укажите правильный пути"
    return 1
}

# Проверка локального конфига
check_local_config() {
    log_step "Проверка локальной конфигурации..."

    # Находим корень проекта
    local project_root
    project_root=$(find_project_root)
    if [ $? -ne 0 ] || [ -z "${project_root}" ]; then
        log_error "Не удалось найти корневую директорию проекта"
        exit 1
    fi

    # Проверяем конфиг
    local config_path="${project_root}/configs/prod/.env"

    if [ -f "${config_path}" ]; then
        log_info "✅ Продакшен конфиг найден: ${config_path}"
    else
        log_warn "Продакшен конфиг не найден: ${config_path}"
        log_info "Убедитесь, что файл существует в репозитории"
        log_info "Текущая структура configs/:"
        ls -la "${project_root}/configs/" 2>/dev/null || echo "Директория configs/ не найдена"
        echo ""
        log_info "Продолжить с минимальной конфигурацией? (y/N)"
        read -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "Прерывание: требуется продакшен конфиг"
            exit 1
        fi
    fi
}

# Установка зависимостей на сервере
install_dependencies() {
    log_step "Установка системных зависимостей..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

# Обновление системы
apt-get update
apt-get upgrade -y

# Установка базовых утилит
apt-get install -y \
    curl \
    wget \
    git \
    htop \
    nano \
    net-tools \
    build-essential \
    software-properties-common \
    ufw \
    fail2ban \
    logrotate \
    postgresql-client \
    redis-tools \
    openssl

# Установка Go 1.21+
if ! command -v go &> /dev/null; then
    echo "Установка Go..."
    wget -q https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    rm go1.21.6.linux-amd64.tar.gz

    # Добавление в PATH
    echo 'export PATH=\$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export PATH=\$PATH:/usr/local/go/bin' >> /root/.bashrc
    source /etc/profile
fi

echo "✅ Системные зависимости установлены"
EOF

    log_info "Системные зависимости установлены"
}

# Настройка PostgreSQL с использованием данных из конфига
setup_postgresql() {
    log_step "Настройка PostgreSQL..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

DB_HOST="${DB_HOST}"
DB_PORT="${DB_PORT}"
DB_NAME="${DB_NAME}"
DB_USER="${DB_USER}"
DB_PASSWORD="${DB_PASSWORD}"

echo "Настройка PostgreSQL с параметрами из конфига:"
echo "  Хост: \${DB_HOST}"
echo "  Порт: \${DB_PORT}"
echo "  База: \${DB_NAME}"
echo "  Пользователь: \${DB_USER}"

# Установка PostgreSQL 15
if ! systemctl is-active --quiet postgresql; then
    echo "Установка PostgreSQL 15..."

    # Добавление репозитория
    sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt \$(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/pgdg.gpg
    apt-get update

    apt-get install -y postgresql-15 postgresql-contrib-15

    # Настройка PostgreSQL
    echo "Настройка PostgreSQL..."

    # Разрешить подключения с localhost
    sed -i "s/#listen_addresses = 'localhost'/listen_addresses = 'localhost'/g" /etc/postgresql/15/main/postgresql.conf
    systemctl restart postgresql

    # Создание пользователя и базы данных из конфига
    echo "Создание пользователя и базы данных..."
    sudo -u postgres psql -c "CREATE USER \${DB_USER} WITH PASSWORD '\${DB_PASSWORD}';" || echo "Пользователь уже существует"
    sudo -u postgres psql -c "CREATE DATABASE \${DB_NAME} OWNER \${DB_USER};" || echo "База данных уже существует"
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE \${DB_NAME} TO \${DB_USER};"

    # Если база уже существует, даем права
    sudo -u postgres psql -c "ALTER DATABASE \${DB_NAME} OWNER TO \${DB_USER};" 2>/dev/null || true
    sudo -u postgres psql -d \${DB_NAME} -c "GRANT ALL ON SCHEMA public TO \${DB_USER};" 2>/dev/null || true

    echo "✅ PostgreSQL настроен с пользователем \${DB_USER} и базой \${DB_NAME}"
else
    echo "✅ PostgreSQL уже установлен"

    # Проверяем и создаем пользователя/базу если нужно
    echo "Проверка пользователя и базы данных..."
    if ! sudo -u postgres psql -c "\du" | grep -q "\${DB_USER}"; then
        echo "Создание пользователя \${DB_USER}..."
        sudo -u postgres psql -c "CREATE USER \${DB_USER} WITH PASSWORD '\${DB_PASSWORD}';"
    fi

    if ! sudo -u postgres psql -c "\l" | grep -q "\${DB_NAME}"; then
        echo "Создание базы данных \${DB_NAME}..."
        sudo -u postgres psql -c "CREATE DATABASE \${DB_NAME} OWNER \${DB_USER};"
        sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE \${DB_NAME} TO \${DB_USER};"
    else
        echo "База \${DB_NAME} уже существует"
        sudo -u postgres psql -c "ALTER DATABASE \${DB_NAME} OWNER TO \${DB_USER};" 2>/dev/null || true
        sudo -u postgres psql -d \${DB_NAME} -c "GRANT ALL ON SCHEMA public TO \${DB_USER};" 2>/dev/null || true
    fi
fi
EOF

    log_info "PostgreSQL настроен"
}

# Настройка Redis с использованием данных из конфига
setup_redis() {
    log_step "Настройка Redis..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

REDIS_HOST="${REDIS_HOST}"
REDIS_PORT="${REDIS_PORT}"
REDIS_PASSWORD="${REDIS_PASSWORD}"
REDIS_ENABLED="${REDIS_ENABLED}"

echo "Настройка Redis с параметрами из конфига:"
echo "  Хост: \${REDIS_HOST}"
echo "  Порт: \${REDIS_PORT}"
echo "  Пароль: \$(if [ -n "\${REDIS_PASSWORD}" ]; then echo '[установлен]'; else echo '[нет]'; fi)"
echo "  Включен: \${REDIS_ENABLED}"

# Если Redis отключен в конфиге, просто выходим
if [ "\${REDIS_ENABLED}" = "false" ]; then
    echo "⚠️  Redis отключен в конфиге (REDIS_ENABLED=false)"
    echo "Redis не будет установлен и настроен"
    exit 0
fi

# Установка Redis только если он не установлен
if ! systemctl is-active --quiet redis-server; then
    echo "Установка Redis..."
    apt-get install -y redis-server

    # Настройка Redis с использованием значений из конфига
    echo "Настройка Redis с параметрами из конфига..."

    # Если указан нестандартный порт, настраиваем его
    if [ "\${REDIS_PORT}" != "6379" ]; then
        echo "Настройка Redis на порт \${REDIS_PORT}..."
        sed -i "s/port 6379/port \${REDIS_PORT}/g" /etc/redis/redis.conf
    fi

    # Настройка привязки к хосту из конфига
    if [ "\${REDIS_HOST}" != "localhost" ] && [ "\${REDIS_HOST}" != "127.0.0.1" ]; then
        echo "Настройка Redis для работы с хостом: \${REDIS_HOST}"

        # Если хост - IP адрес, добавляем его в bind
        if echo "\${REDIS_HOST}" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
            sed -i "s/bind 127.0.0.1 ::1/bind 127.0.0.1 \${REDIS_HOST}/g" /etc/redis/redis.conf
        fi
    else
        # По умолчанию только localhost
        sed -i "s/bind 127.0.0.1 ::1/bind 127.0.0.1/g" /etc/redis/redis.conf
    fi

    # Настройка пароля если указан
    if [ -n "\${REDIS_PASSWORD}" ]; then
        echo "Настройка пароля для Redis..."

        # Комментируем существующий requirepass если есть
        sed -i "s/^requirepass/#requirepass/g" /etc/redis/redis.conf

        # Добавляем новый requirepass
        echo "requirepass \${REDIS_PASSWORD}" >> /etc/redis/redis.conf

        # Также настраиваем в настройках клиента
        echo "masterauth \${REDIS_PASSWORD}" >> /etc/redis/redis.conf
    fi

    # Настройка памяти
    sed -i "s/# maxmemory <bytes>/maxmemory 256mb/g" /etc/redis/redis.conf
    sed -i "s/# maxmemory-policy noeviction/maxmemory-policy allkeys-lru/g" /etc/redis/redis.conf

    # Отключаем защищенный режим если хост не localhost
    if [ "\${REDIS_HOST}" != "localhost" ] && [ "\${REDIS_HOST}" != "127.0.0.1" ]; then
        echo "Отключение защищенного режима для удаленного доступа..."
        sed -i "s/protected-mode yes/protected-mode no/g" /etc/redis/redis.conf
    fi

    systemctl restart redis-server
    systemctl enable redis-server

    echo "✅ Redis установлен и настроен с параметрами из конфига"
else
    echo "✅ Redis уже установлен"

    # Проверяем настройки и при необходимости обновляем их
    echo "Проверка настроек Redis..."

    # Проверяем порт
    CURRENT_PORT=\$(grep "^port" /etc/redis/redis.conf | head -1 | cut -d' ' -f2)
    if [ "\${CURRENT_PORT}" != "\${REDIS_PORT}" ]; then
        echo "Обновление порта Redis с \${CURRENT_PORT} на \${REDIS_PORT}..."
        sed -i "s/port \${CURRENT_PORT}/port \${REDIS_PORT}/g" /etc/redis/redis.conf
    fi

    # Обновляем пароль если указан
    if [ -n "\${REDIS_PASSWORD}" ]; then
        echo "Обновление пароля Redis..."

        # Удаляем старые настройки пароля
        sed -i "/^requirepass/d" /etc/redis/redis.conf
        sed -i "/^masterauth/d" /etc/redis/redis.conf

        # Добавляем новые
        echo "requirepass \${REDIS_PASSWORD}" >> /etc/redis/redis.conf
        echo "masterauth \${REDIS_PASSWORD}" >> /etc/redis/redis.conf
    fi

    # Перезапускаем Redis если были изменения
    systemctl restart redis-server
    echo "✅ Настройки Redis обновлены"
fi

# Проверяем доступность Redis
echo "Проверка доступности Redis..."
if redis-cli -h "\${REDIS_HOST}" -p "\${REDIS_PORT}" \$(if [ -n "\${REDIS_PASSWORD}" ]; then echo "-a \${REDIS_PASSWORD}"; fi) ping | grep -q "PONG"; then
    echo "✅ Redis доступен на \${REDIS_HOST}:\${REDIS_PORT}"
else
    echo "⚠️  Redis не отвечает по адресу \${REDIS_HOST}:\${REDIS_PORT}"
    echo "Проверьте настройки и перезапустите сервис: systemctl restart redis-server"
fi
EOF

    log_info "Redis настроен"
}

# Настройка SSL сертификатов для webhook - ПОЛНОСТЬЮ ПЕРЕПИСАНА
setup_ssl_certificates() {
    log_step "Проверка и настройка SSL сертификатов для webhook (домен: ${WEBHOOK_DOMAIN})..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

DOMAIN="${WEBHOOK_DOMAIN}"
IP="${SERVER_IP}"
INSTALL_DIR="/opt/crypto-screener-bot"
CERTS_DIR="/etc/crypto-bot/certs"

echo "🔍 Проверка SSL сертификатов для домена: \${DOMAIN}"
echo ""

# Создаем директории для сертификатов
echo "Создание директорий для сертификатов..."
mkdir -p "\${CERTS_DIR}"
mkdir -p "\${INSTALL_DIR}/ssl"

# Проверяем существующие сертификаты
echo "1. Проверка существующих сертификатов..."
CERT_VALID=false

# Проверка 1: Let's Encrypt сертификаты
if [ -d "/etc/letsencrypt/live/\${DOMAIN}" ]; then
    echo "   ✅ Let's Encrypt сертификаты найдены"

    if [ -f "/etc/letsencrypt/live/\${DOMAIN}/fullchain.pem" ] && \
       [ -f "/etc/letsencrypt/live/\${DOMAIN}/privkey.pem" ]; then
        echo "   ✅ Let's Encrypt файлы сертификатов найдены"

        # Копируем Let's Encrypt сертификаты
        echo "   📋 Копирование Let's Encrypt сертификатов..."
        cp "/etc/letsencrypt/live/\${DOMAIN}/fullchain.pem" "\${CERTS_DIR}/cert.pem"
        cp "/etc/letsencrypt/live/\${DOMAIN}/privkey.pem" "\${CERTS_DIR}/key.pem"

        # Копируем в директорию приложения для удобства
        cp "/etc/letsencrypt/live/\${DOMAIN}/fullchain.pem" "\${INSTALL_DIR}/ssl/fullchain.pem"
        cp "/etc/letsencrypt/live/\${DOMAIN}/privkey.pem" "\${INSTALL_DIR}/ssl/privkey.pem"

        CERT_VALID=true
        CERT_SOURCE="Let's Encrypt"
    fi
fi

# Проверка 2: Существующие сертификаты в CERTS_DIR
if [ "\${CERT_VALID}" = "false" ] && \
   [ -f "\${CERTS_DIR}/cert.pem" ] && \
   [ -f "\${CERTS_DIR}/key.pem" ]; then
    echo "   ✅ Существующие сертификаты найдены в \${CERTS_DIR}"

    # Проверяем валидность существующего сертификата
    if openssl x509 -in "\${CERTS_DIR}/cert.pem" -noout -checkend 86400 >/dev/null 2>&1; then
        echo "   ✅ Сертификат валиден (действителен минимум 24 часа)"

        # Копируем в директорию приложения
        cp "\${CERTS_DIR}/cert.pem" "\${INSTALL_DIR}/ssl/fullchain.pem"
        cp "\${CERTS_DIR}/key.pem" "\${INSTALL_DIR}/ssl/privkey.pem"

        CERT_VALID=true
        CERT_SOURCE="существующие"
    else
        echo "   ⚠️  Сертификат просрочен или невалиден"
    fi
fi

# Если нет валидных сертификатов, создаем новые
if [ "\${CERT_VALID}" = "false" ]; then
    echo "2. Создание новых самоподписанных сертификатов..."

    # Переходим в директорию сертификатов
    cd "\${CERTS_DIR}"

    # Удаляем старые сертификаты если есть
    rm -f cert.pem key.pem cert.cnf 2>/dev/null || true

    # Создаем конфиг для сертификата
    cat > cert.cnf << CONFIG
[req]
default_bits = 2048
prompt = no
default_md = sha256
x509_extensions = v3_req
distinguished_name = dn

[dn]
C = RU
ST = Moscow
L = Moscow
O = CryptoBot
CN = \${DOMAIN}

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = \${DOMAIN}
IP.1 = \${IP}
CONFIG

    # Генерируем новый сертификат
    echo "   Генерация самоподписанного сертификата..."
    openssl req -x509 -newkey rsa:2048 \
        -keyout key.pem \
        -out cert.pem \
        -days 365 \
        -nodes \
        -config cert.cnf

    # Копируем в директорию приложения
    cp cert.pem "\${INSTALL_DIR}/ssl/fullchain.pem"
    cp key.pem "\${INSTALL_DIR}/ssl/privkey.pem"

    CERT_SOURCE="самоподписанные"

    # Очищаем временный файл
    rm -f cert.cnf
fi

# Настраиваем права доступа
echo "3. Настройка прав доступа..."
chmod 644 "\${CERTS_DIR}/cert.pem" 2>/dev/null || true
chmod 600 "\${CERTS_DIR}/key.pem" 2>/dev/null || true
chmod 644 "\${INSTALL_DIR}/ssl/fullchain.pem" 2>/dev/null || true
chmod 600 "\${INSTALL_DIR}/ssl/privkey.pem" 2>/dev/null || true

chown -R cryptoapp:cryptoapp "\${CERTS_DIR}" 2>/dev/null || chown -R root:root "\${CERTS_DIR}"
chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}/ssl" 2>/dev/null || true

echo ""
echo "✅ SSL сертификаты настроены"
echo "   Источник: \${CERT_SOURCE}"
echo "   Пути:"
echo "     - \${CERTS_DIR}/cert.pem (основной)"
echo "     - \${INSTALL_DIR}/ssl/fullchain.pem (для приложения)"
echo "     - \${INSTALL_DIR}/ssl/privkey.pem (для приложения)"
echo ""

# Проверяем сертификат
if [ -f "\${CERTS_DIR}/cert.pem" ]; then
    echo "🔍 Проверка сертификата:"
    openssl x509 -in "\${CERTS_DIR}/cert.pem" -noout -subject -dates 2>/dev/null | sed 's/^/   /'

    NOT_AFTER=\$(openssl x509 -in "\${CERTS_DIR}/cert.pem" -noout -enddate 2>/dev/null | cut -d= -f2)
    echo "   Срок действия до: \${NOT_AFTER}"
fi
EOF

    log_info "SSL сертификаты настроены"
}

# Настройка брандмауэра с поддержкой webhook портов
setup_firewall() {
    log_step "Настройка брандмауэра UFW для webhook..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

WEBHOOK_PORT="${WEBHOOK_PORT}"

echo "Настройка брандмауэра UFW с webhook портом: \${WEBHOOK_PORT}"

# Настройка UFW
ufw --force reset 2>/dev/null || true
ufw default deny incoming
ufw default allow outgoing

# Разрешить SSH (обязательно!)
ufw allow 22/tcp comment 'SSH'

# Разрешить webhook порт если используется TLS
if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
    echo "Разрешение HTTPS webhook порта: \${WEBHOOK_PORT}"
    ufw allow \${WEBHOOK_PORT}/tcp comment "Telegram webhook HTTPS"
else
    echo "Разрешение HTTP webhook порта: \${WEBHOOK_PORT}"
    ufw allow \${WEBHOOK_PORT}/tcp comment "Telegram webhook HTTP"
fi

# Разрешить стандартный HTTP порт для отладки
ufw allow 8080/tcp comment "HTTP debug port"

# Включить брандмауэр
echo "y" | ufw enable

echo "✅ Брандмауэр настроен с портами:"
ufw status verbose

# Проверяем открытые порты
echo "Проверка открытых портов:"
ss -tln | grep -E ':(22|${WEBHOOK_PORT}|8080)' | sort
EOF

    log_info "Брандмауэр настроен с поддержкой webhook"
}

# Создание правильной структуры директорий
create_directory_structure() {
    log_step "Создание правильной структуры директорий..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
INSTALL_DIR="${INSTALL_DIR}"

# Создание пользователя если не существует
if ! id "cryptoapp" &>/dev/null; then
    useradd -m -s /bin/bash -r cryptoapp
    echo "Пользователь cryptoapp создан"
fi

# Удаление старой структуры если существует
if [ -d "\${INSTALL_DIR}" ]; then
    echo "Удаление старой структуры директорий..."
    rm -rf "\${INSTALL_DIR}"
fi

# Создание новой правильной структуры
echo "Создание структуры директорий..."
mkdir -p "\${INSTALL_DIR}"
mkdir -p "\${INSTALL_DIR}/bin"
mkdir -p "\${INSTALL_DIR}/ssl"
mkdir -p "\${INSTALL_DIR}/logs"
mkdir -p "/var/log/\${APP_NAME}"

# Настройка прав
chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}"
chown -R cryptoapp:cryptoapp "/var/log/\${APP_NAME}"
chmod 755 "\${INSTALL_DIR}"
chmod 755 "/var/log/\${APP_NAME}"
chmod 700 "\${INSTALL_DIR}/ssl"  # Строгие права для SSL

echo "✅ Структура директорий создана:"
echo "   \${INSTALL_DIR}/"
echo "   ├── bin/"
echo "   ├── ssl/"
echo "   ├── logs/"
echo "   /var/log/\${APP_NAME}/"
EOF

    log_info "Структура директорий создана"
}

# Настройка логирования
setup_logging() {
    log_step "Настройка системы логирования..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"

# Конфигурация logrotate
cat > /etc/logrotate.d/\${APP_NAME} << 'LOGROTATE'
/var/log/${APP_NAME}/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 cryptoapp cryptoapp
    sharedscripts
    postrotate
        systemctl reload ${SERVICE_NAME}.service > /dev/null 2>/dev/null || true
    endscript
}
LOGROTATE

# Создание файлов логов
touch "/var/log/\${APP_NAME}/app.log"
touch "/var/log/\${APP_NAME}/error.log"
touch "/var/log/\${APP_NAME}/webhook.log"
chown -R cryptoapp:cryptoapp "/var/log/\${APP_NAME}"
chmod 644 "/var/log/\${APP_NAME}"/*.log

echo "✅ Логирование настроено"
EOF

    log_info "Система логирования настроена"
}

# Копирование исходного кода
copy_source_code() {
    log_step "Копирование исходного кода приложения..."

    # Определяем корневую директорию проекта
    local project_root
    project_root=$(find_project_root)
    if [ $? -ne 0 ] || [ -z "${project_root}" ]; then
        log_error "Не удалось найти корневую директорию проекта"
        exit 1
    fi

    log_info "Корневая директория проекта: ${project_root}"

    # Переходим в корень проекта
    cd "${project_root}"

    # Проверяем структуру проекта
    if [ ! -f "go.mod" ] || [ ! -d "application" ]; then
        log_error "Неправильная структура проекта!"
        log_info "Ожидается наличие: go.mod и application/"
        exit 1
    fi

    # Создание архива с исходным кодом
    log_info "Создание архива с исходным кодом..."
    tar -czf /tmp/app_source.tar.gz \
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
    log_info "Копирование архива на сервер..."
    scp -i "${SSH_KEY}" /tmp/app_source.tar.gz "${SERVER_USER}@${SERVER_IP}:/tmp/app_source.tar.gz"

    # Распаковка на сервере
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"

echo "Распаковка исходного кода..."

# Распаковка в корень установки
echo "1. Распаковка исходного кода..."
tar -xzf /tmp/app_source.tar.gz -C "${INSTALL_DIR}"

# Создаем необходимые директории если их нет
echo "2. Создание необходимых директорий..."
mkdir -p "${INSTALL_DIR}/bin"
mkdir -p "${INSTALL_DIR}/ssl"
mkdir -p "${INSTALL_DIR}/logs"
mkdir -p "${INSTALL_DIR}/configs/prod" 2>/dev/null || true

# Настройка прав
echo "3. Настройка прав доступа..."
chown -R cryptoapp:cryptoapp "${INSTALL_DIR}"
chmod 755 "${INSTALL_DIR}"
chmod 755 "${INSTALL_DIR}/bin"
chmod 700 "${INSTALL_DIR}/ssl"

# Очистка временных файлов
echo "4. Очистка временных файлов..."
rm -f /tmp/app_source.tar.gz

echo "✅ Исходный код скопирован"
echo "Структура директории ${INSTALL_DIR}:"
ls -la "${INSTALL_DIR}/" | head -10
EOF

    # Очистка локального архива
    rm -f /tmp/app_source.tar.gz

    log_info "Исходный код скопирован на сервер"
}

# Сборка приложения
build_application() {
    log_step "Сборка приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"
APP_NAME="crypto-screener-bot"

cd "${INSTALL_DIR}"

echo "Текущая директория: $(pwd)"
echo "Содержимое:"
ls -la

# Проверка структуры
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod не найден!"
    echo "Содержимое директории:"
    ls -la
    exit 1
fi

# Установка зависимостей Go
echo "Установка зависимостей Go..."
/usr/local/go/bin/go mod download

# Сборка основного приложения
echo "Сборка основного приложения..."
if [ -f "./application/cmd/bot/main.go" ]; then
    echo "✅ Найден main.go: ./application/cmd/bot/main.go"
    /usr/local/go/bin/go build -o "${INSTALL_DIR}/bin/${APP_NAME}" ./application/cmd/bot/main.go

    if [ $? -eq 0 ]; then
        echo "✅ Основное приложение собрано"

        # Проверка бинарника
        echo "Проверка версии приложения..."
        "${INSTALL_DIR}/bin/${APP_NAME}" --version 2>&1 | head -1 || echo "⚠️  Не удалось получить версию"

        # Настройка прав бинарника
        chown cryptoapp:cryptoapp "${INSTALL_DIR}/bin/${APP_NAME}"
        chmod +x "${INSTALL_DIR}/bin/${APP_NAME}"

        # Проверка webhook режима в коде
        echo "Проверка webhook поддержки в бинарнике..."
        strings "${INSTALL_DIR}/bin/${APP_NAME}" | grep -i "webhook" | head -5 || echo "   Webhook strings not found"
    else
        echo "❌ Ошибка сборки приложения"
        exit 1
    fi
else
    echo "❌ Файл основного приложения не найден: ./application/cmd/bot/main.go"
    echo "Поиск файлов application..."
    find . -name "main.go" -type f | head -10
    exit 1
fi

# Проверка наличия мигратора
echo "Проверка миграций..."
if [ -f "./internal/infrastructure/persistence/postgres/migrator.go" ]; then
    echo "✅ Мигратор найден"
else
    echo "⚠️  Мигратор не найден"
fi

# Проверка SQL файлов миграций
if [ -d "./internal/infrastructure/persistence/postgres/migrations" ]; then
    MIGRATION_COUNT=$(ls "./internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
    echo "✅ Найдено SQL файлов миграций: ${MIGRATION_COUNT}"

    if [ "${MIGRATION_COUNT}" -gt 0 ]; then
        echo "Первые 5 файлов миграций:"
        ls "./internal/infrastructure/persistence/postgres/migrations/"*.sql | head -5
    fi
else
    echo "⚠️  Директория миграций не найдена"
fi

echo "✅ Сборка приложения завершена"
EOF

    log_info "Приложение собрано"
}

# Обновляем конфигурацию приложения с правильными путями к сертификатам
setup_configuration() {
    log_step "Настройка конфигурации приложения..."

    # Определяем корневую директорию проекта
    local project_root
    project_root=$(find_project_root)
    if [ $? -ne 0 ] || [ -z "${project_root}" ]; then
        log_error "Не удалось найти корневую директорию проекта"
        exit 1
    fi

    local config_path="${project_root}/configs/prod/.env"

    if [ -f "${config_path}" ]; then
        log_info "✅ Найден конфиг: ${config_path}"
        scp -i "${SSH_KEY}" "${config_path}" "${SERVER_USER}@${SERVER_IP}:/tmp/prod.env"
    else
        log_warn "Конфиг не найден, используем .env.example"
        local example_path="${project_root}/.env.example"
        if [ -f "${example_path}" ]; then
            scp -i "${SSH_KEY}" "${example_path}" "${SERVER_USER}@${SERVER_IP}:/tmp/prod.env"
        else
            log_error "Не найден ни конфиг, ни пример конфига"
            exit 1
        fi
    fi

    # Настраиваем конфиг на сервере
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR}"
CERTS_DIR="/etc/crypto-bot/certs"

echo "Настройка конфигурации приложения..."

# Создаем структуру директорий
mkdir -p "\${INSTALL_DIR}/configs/prod"

# Копируем конфиг
if [ -f "/tmp/prod.env" ]; then
    cp "/tmp/prod.env" "\${INSTALL_DIR}/configs/prod/.env"

    # Создаем симлинк для обратной совместимости
    ln -sf "\${INSTALL_DIR}/configs/prod/.env" "\${INSTALL_DIR}/.env"

    # ОБНОВЛЯЕМ ПУТИ К СЕРТИФИКАТАМ
    echo "Обновление путей к SSL сертификатам..."

    # Обновляем пути в конфиге
    sed -i "s|^WEBHOOK_TLS_CERT_PATH=.*|WEBHOOK_TLS_CERT_PATH=\${CERTS_DIR}/cert.pem|" "\${INSTALL_DIR}/.env"
    sed -i "s|^WEBHOOK_TLS_KEY_PATH=.*|WEBHOOK_TLS_KEY_PATH=\${CERTS_DIR}/key.pem|" "\${INSTALL_DIR}/.env"

    # Проверяем и обновляем другие настройки если их нет
    if ! grep -q "^TELEGRAM_MODE=" "\${INSTALL_DIR}/.env"; then
        echo "TELEGRAM_MODE=${TELEGRAM_MODE}" >> "\${INSTALL_DIR}/.env"
    fi

    if ! grep -q "^WEBHOOK_DOMAIN=" "\${INSTALL_DIR}/.env"; then
        echo "WEBHOOK_DOMAIN=${WEBHOOK_DOMAIN}" >> "\${INSTALL_DIR}/.env"
    fi

    if ! grep -q "^WEBHOOK_PORT=" "\${INSTALL_DIR}/.env"; then
        echo "WEBHOOK_PORT=${WEBHOOK_PORT}" >> "\${INSTALL_DIR}/.env"
    fi

    if ! grep -q "^WEBHOOK_USE_TLS=" "\${INSTALL_DIR}/.env"; then
        echo "WEBHOOK_USE_TLS=${WEBHOOK_USE_TLS}" >> "\${INSTALL_DIR}/.env"
    fi

    # Генерируем секретный токен если его нет
    if ! grep -q "^WEBHOOK_SECRET_TOKEN=" "\${INSTALL_DIR}/.env" || \
       [ -z "\$(grep '^WEBHOOK_SECRET_TOKEN=' "\${INSTALL_DIR}/.env" | cut -d= -f2)" ]; then
        SECRET_TOKEN=\$(openssl rand -hex 16)
        sed -i '/^WEBHOOK_SECRET_TOKEN=/d' "\${INSTALL_DIR}/.env"
        echo "WEBHOOK_SECRET_TOKEN=\${SECRET_TOKEN}" >> "\${INSTALL_DIR}/.env"
        echo "   Сгенерирован новый WEBHOOK_SECRET_TOKEN: \${SECRET_TOKEN}"
    fi

    # Проверяем и добавляем обязательные настройки если их нет
    if ! grep -q "^DB_ENABLE_AUTO_MIGRATE=" "\${INSTALL_DIR}/.env"; then
        echo "DB_ENABLE_AUTO_MIGRATE=true" >> "\${INSTALL_DIR}/.env"
    fi

    if ! grep -q "^REDIS_ENABLED=" "\${INSTALL_DIR}/.env"; then
        echo "REDIS_ENABLED=true" >> "\${INSTALL_DIR}/.env"
    fi

    # Настройка прав
    chown cryptoapp:cryptoapp "\${INSTALL_DIR}/.env"
    chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}/configs"
    chmod 600 "\${INSTALL_DIR}/.env"
    chmod 600 "\${INSTALL_DIR}/configs/prod/.env"

    # Очистка
    rm -f /tmp/prod.env

    echo "✅ Конфигурация настроена"
    echo "📋 Основные настройки:"
    grep -E "^(APP_ENV|TELEGRAM_MODE|WEBHOOK_DOMAIN|WEBHOOK_PORT|WEBHOOK_USE_TLS|WEBHOOK_TLS_CERT_PATH|WEBHOOK_TLS_KEY_PATH|WEBHOOK_SECRET_TOKEN|DB_HOST|DB_PORT|DB_NAME|DB_USER|LOG_LEVEL|EXCHANGE|TELEGRAM_ENABLED|DB_ENABLE_AUTO_MIGRATE|REDIS_HOST|REDIS_PORT|REDIS_PASSWORD|REDIS_ENABLED)=" \
        "\${INSTALL_DIR}/.env" | head -25
else
    echo "❌ Конфиг не найден после копирования"
    exit 1
fi
EOF

    log_info "Конфигурация настроена"
}

# Настройка systemd сервиса
setup_systemd_service() {
    log_step "Настройка systemd сервиса..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
SERVICE_NAME="crypto-screener"
INSTALL_DIR="/opt/${APP_NAME}"

echo "🔧 Настройка systemd сервиса ${SERVICE_NAME}..."

# Создание пользователя если не существует
if ! id "cryptoapp" &>/dev/null; then
    echo "👤 Создание пользователя cryptoapp..."
    useradd -m -s /bin/bash -r cryptoapp
    echo "✅ Пользователь cryptoapp создан"
fi

# Проверка существования бинарника
BINARY_PATH="${INSTALL_DIR}/bin/${APP_NAME}"
if [ ! -f "${BINARY_PATH}" ]; then
    echo "❌ КРИТИЧЕСКАЯ ОШИБКА: Бинарник не найден: ${BINARY_PATH}"
    echo "   Проверка содержимого:"
    ls -la "${INSTALL_DIR}/" 2>/dev/null | head -10
    echo "   Попытка найти бинарник:"
    find "${INSTALL_DIR}" -name "*crypto*" -type f -executable 2>/dev/null || echo "   Исполняемые файлы не найдены"
    exit 1
fi

echo "✅ Бинарник найден: ${BINARY_PATH}"
echo "   Размер: $(du -h "${BINARY_PATH}" | cut -f1)"
echo "   Права: $(ls -la "${BINARY_PATH}" | awk '{print $1 " " $3 ":" $4}')"

# Остановка и удаление старого сервиса если существует
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    echo "🔄 Остановка и удаление старого сервиса..."
    systemctl stop "${SERVICE_NAME}.service" 2>/dev/null || echo "   ⚠️  Сервис не был запущен"
    systemctl disable "${SERVICE_NAME}.service" 2>/dev/null || echo "   ⚠️  Сервис не был включен"
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    rm -rf "/etc/systemd/system/${SERVICE_NAME}.service.d" 2>/dev/null || true
    echo "✅ Старый сервис удален"
fi

# Создание нового правильного сервиса
echo "📄 Создание нового systemd сервиса..."
cat > /etc/systemd/system/${SERVICE_NAME}.service << SERVICE
[Unit]
Description=Crypto Exchange Screener Bot (Webhook Mode)
After=network.target postgresql.service redis-server.service
Requires=postgresql.service redis-server.service

[Service]
Type=simple
User=cryptoapp
Group=cryptoapp
WorkingDirectory=${INSTALL_DIR}
Environment="APP_ENV=production"
EnvironmentFile=${INSTALL_DIR}/.env

# ИСПРАВЛЕННЫЙ ПУТЬ: bin/crypto-screener-bot
ExecStart=${INSTALL_DIR}/bin/${APP_NAME}
Restart=always
RestartSec=10
StandardOutput=append:/var/log/${APP_NAME}/app.log
StandardError=append:/var/log/${APP_NAME}/error.log

# Лимиты безопасности
LimitNOFILE=65536
LimitNPROC=65536

# Сетевая изоляция
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${INSTALL_DIR} /var/log/${APP_NAME} /etc/crypto-bot
NoNewPrivileges=true

# Настройки для webhook режима
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
SERVICE

echo "✅ Файл сервиса создан: /etc/systemd/system/${SERVICE_NAME}.service"

# Настройка прав доступа
echo "🔐 Настройка прав доступа..."
chown -R cryptoapp:cryptoapp "${INSTALL_DIR}"
chmod +x "${BINARY_PATH}"
echo "✅ Права на бинарник установлены"

# Создание директории логов если нет
echo "📁 Создание директории логов..."
mkdir -p "/var/log/${APP_NAME}"
chown -R cryptoapp:cryptoapp "/var/log/${APP_NAME}"
chmod 755 "/var/log/${APP_NAME}"
echo "✅ Директория логов создана: /var/log/${APP_NAME}"

# Перезагрузка systemd
echo "🔄 Перезагрузка systemd..."
systemctl daemon-reload
systemctl enable ${SERVICE_NAME}.service

echo "✅ Systemd сервис настроен"

# Проверка конфигурации
echo "🔍 Проверка конфигурации сервиса:"
if systemctl cat ${SERVICE_NAME}.service > /dev/null 2>&1; then
    echo "✅ Сервис загружен в systemd"

    # Показать ExecStart строку для проверки
    EXEC_LINE=$(systemctl cat ${SERVICE_NAME}.service | grep "^ExecStart=")
    echo "   ExecStart: ${EXEC_LINE}"

    if echo "${EXEC_LINE}" | grep -q "bin/${APP_NAME}"; then
        echo "   ✅ Путь к бинарнику правильный"
    else
        echo "   ❌ ПУТЬ НЕПРАВИЛЬНЫЙ! Исправьте вручную"
        echo "   Ожидалось: ${INSTALL_DIR}/bin/${APP_NAME}"
    fi
else
    echo "❌ Ошибка: сервис не загружен в systemd"
    exit 1
fi

echo ""
echo "🎯 Systemd сервис готов к использованию"
echo "   Команда запуска: systemctl start ${SERVICE_NAME}"
echo "   Команда статуса: systemctl status ${SERVICE_NAME}"
echo "   Просмотр логов: journalctl -u ${SERVICE_NAME} -f"
EOF

    log_info "Systemd сервис настроен"
}

# Проверка миграций базы данных
check_migrations() {
    log_step "Проверка миграций базы данных..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"

echo "Проверка миграций..."

# Проверяем наличие миграций в исходном коде
if [ -d "${INSTALL_DIR}/internal/infrastructure/persistence/postgres/migrations" ]; then
    echo "✅ Директория миграций найдена"
    MIGRATION_COUNT=$(find "${INSTALL_DIR}/internal/infrastructure/persistence/postgres/migrations" -name "*.sql" 2>/dev/null | wc -l)
    echo "Количество SQL файлов миграций: ${MIGRATION_COUNT}"

    if [ "${MIGRATION_COUNT}" -gt 0 ]; then
        echo "Список миграций:"
        find "${INSTALL_DIR}/internal/infrastructure/persistence/postgres/migrations" -name "*.sql" | head -10
    fi
else
    echo "⚠️  Директория миграций не найдена"
fi

echo ""
echo "ℹ️  Миграции будут автоматически выполнены при запуске приложения"
echo "Приложение проверит DB_ENABLE_AUTO_MIGRATE=true и выполнит миграции"
echo ""
EOF

    log_info "Миграции проверены"
}

# Запуск приложения
start_application() {
    log_step "Запуск приложения в webhook режиме..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

SERVICE_NAME="crypto-screener"
INSTALL_DIR="/opt/crypto-screener-bot"

echo "🚀 Запуск приложения ${SERVICE_NAME} в webhook режиме..."

# Останавливаем сервис если запущен
echo "⏹️  Остановка сервиса (если запущен)..."
systemctl stop "${SERVICE_NAME}.service" 2>/dev/null || echo "   ⚠️  Сервис не был запущен"
sleep 2

# Проверяем конфигурацию webhook
echo "🔍 Проверка webhook конфигурации..."
if [ -f "${INSTALL_DIR}/.env" ]; then
    echo "✅ Файл конфигурации найден"

    # Проверяем режим Telegram
    TELEGRAM_MODE=$(grep "^TELEGRAM_MODE=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "webhook")
    echo "   Режим Telegram: ${TELEGRAM_MODE}"

    # Проверяем webhook настройки
    if [ "${TELEGRAM_MODE}" = "webhook" ]; then
        echo "   Режим работы: Webhook"

        # Проверяем обязательные настройки
        WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "8443")
        WEBHOOK_USE_TLS=$(grep "^WEBHOOK_USE_TLS=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "true")

        echo "   Webhook порт: ${WEBHOOK_PORT}"
        echo "   Использовать TLS: ${WEBHOOK_USE_TLS}"

        if [ "${WEBHOOK_USE_TLS}" = "true" ]; then
            CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "")
            KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "")

            if [ -f "${CERT_PATH}" ] && [ -f "${KEY_PATH}" ]; then
                echo "   ✅ Сертификаты найдены:"
                echo "      cert: ${CERT_PATH}"
                echo "      key: ${KEY_PATH}"
            else
                echo "   ⚠️  Сертификаты не найдены по указанным путям"
                echo "   Проверьте настройки или создайте сертификаты:"
                echo "   /etc/crypto-bot/certs/"
            fi
        fi

        # Проверяем секретный токен
        SECRET_TOKEN=$(grep "^WEBHOOK_SECRET_TOKEN=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "")
        if [ -n "${SECRET_TOKEN}" ]; then
            echo "   ✅ Секретный токен установлен"
        else
            echo "   ⚠️  Секретный токен не установлен"
        fi
    else
        echo "   Режим работы: Polling"
    fi

    # Проверяем основные настройки
    if grep -q "DB_ENABLE_AUTO_MIGRATE=" "${INSTALL_DIR}/.env"; then
        AUTO_MIGRATE=$(grep "DB_ENABLE_AUTO_MIGRATE=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        echo "   Автоматические миграции: ${AUTO_MIGRATE}"
    else
        echo "⚠️  DB_ENABLE_AUTO_MIGRATE не настроен, добавляем..."
        echo "DB_ENABLE_AUTO_MIGRATE=true" >> "${INSTALL_DIR}/.env"
    fi
else
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

# Проверяем бинарник перед запуском
echo "🔍 Проверка бинарника..."
BINARY_PATH="${INSTALL_DIR}/bin/crypto-screener-bot"
if [ ! -f "${BINARY_PATH}" ]; then
    echo "❌ Бинарник не найден: ${BINARY_PATH}"
    echo "Попытка сборки..."
    cd "${INSTALL_DIR}"
    if [ -f "go.mod" ] && [ -f "application/cmd/bot/main.go" ]; then
        /usr/local/go/bin/go build -o "${BINARY_PATH}" ./application/cmd/bot/main.go
        chown cryptoapp:cryptoapp "${BINARY_PATH}"
        chmod +x "${BINARY_PATH}"
        echo "✅ Бинарник собран"
    else
        echo "❌ Не удалось собрать бинарник"
        exit 1
    fi
fi

# Проверяем права на бинарник
if [ ! -x "${BINARY_PATH}" ]; then
    echo "⚠️  Бинарник не исполняемый, исправляем..."
    chmod +x "${BINARY_PATH}"
fi

# Запуск сервиса
echo "🚀 Запуск сервиса ${SERVICE_NAME}..."
systemctl start "${SERVICE_NAME}.service"
sleep 3

# Проверка статуса
echo "📊 Статус сервиса:"
systemctl status "${SERVICE_NAME}.service" --no-pager | head -20

# Ждем инициализацию
echo "⏳ Ожидание инициализации (15 секунд)..."
sleep 15

# Проверка процесса
echo "🔍 Проверка процессов:"
if pgrep -f "crypto-screener-bot" > /dev/null; then
    echo "✅ Приложение запущено"
    PID=$(pgrep -f "crypto-screener-bot")
    echo "   PID: ${PID}"
    echo "   Uptime: $(ps -o etime= -p ${PID} 2>/dev/null || echo "неизвестно")"

    # Проверяем webhook порт если в webhook режиме
    if [ "${TELEGRAM_MODE}" = "webhook" ]; then
        echo "🔍 Проверка webhook порта ${WEBHOOK_PORT}..."
        if ss -tln | grep -q ":${WEBHOOK_PORT} "; then
            echo "✅ Webhook порт ${WEBHOOK_PORT} открыт"
        else
            echo "⚠️  Webhook порт ${WEBHOOK_PORT} не открыт"
            echo "Проверьте логи: journalctl -u ${SERVICE_NAME}.service -n 50"
        fi
    fi
else
    echo "❌ Приложение не запущено"
    echo "Проверьте логи: journalctl -u ${SERVICE_NAME}.service -n 50"
    exit 1
fi

echo ""
echo "✅ Приложение успешно запущено!"
echo ""
echo "ℹ️  Для настройки webhook в Telegram выполните:"
echo "   curl -X POST 'https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook' \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"url\": \"https://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}/webhook\","
echo "          \"secret_token\": \"${SECRET_TOKEN}\"}'"
EOF

    log_info "Приложение запущено"
}

# Проверка развертывания с webhook
verify_deployment() {
    log_step "Проверка развертывания с webhook..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
SERVICE_NAME="crypto-screener"
INSTALL_DIR="/opt/crypto-screener-bot"

echo "=== ПРОВЕРКА РАЗВЕРТЫВАНИЯ С WEBHOOK ==="
echo ""

# 1. Проверка структуры директорий
echo "1. Структура директорий:"
echo "   ${INSTALL_DIR}/"
ls -la "${INSTALL_DIR}/" | head -20
echo ""

# 2. Проверка сервисов
echo "2. Проверка системных сервисов:"
echo "   PostgreSQL: $(systemctl is-active postgresql 2>/dev/null || echo 'не установлен')"
echo "   Redis: $(systemctl is-active redis-server 2>/dev/null || echo 'не установлен')"
echo "   ${SERVICE_NAME}: $(systemctl is-active ${SERVICE_NAME} 2>/dev/null || echo 'не установлен')"
echo ""

# 3. Проверка процессов
echo "3. Запущенные процессы:"
if pgrep -f "${APP_NAME}" > /dev/null; then
    echo "   ✅ Приложение запущено"
    ps -f -p $(pgrep -f "${APP_NAME}")
else
    echo "   ❌ Приложение не запущено"
fi
echo ""

# 4. Проверка логов
echo "4. Проверка логов:"
if [ -f "/var/log/${APP_NAME}/app.log" ]; then
    echo "   ✅ Файл лога существует"
    echo "   Размер: $(du -h /var/log/${APP_NAME}/app.log | cut -f1)"
    echo "   Последние 5 строк:"
    tail -5 "/var/log/${APP_NAME}/app.log" 2>/dev/null | sed 's/^/   /'
else
    echo "   ❌ Файл лога не найден"
fi
echo ""

# 5. Проверка портов
echo "5. Проверка сетевых портов:"
echo "   SSH (22): $(ss -tln | grep ':22' >/dev/null && echo 'открыт' || echo 'закрыт')"

# Проверяем webhook порт из конфига
if [ -f "${INSTALL_DIR}/.env" ]; then
    WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2 || echo "8443")
    echo "   Webhook (${WEBHOOK_PORT}): $(ss -tln | grep ":${WEBHOOK_PORT} " >/dev/null && echo 'открыт' || echo 'закрыт')"
else
    echo "   Webhook (8443): $(ss -tln | grep ':8443' >/dev/null && echo 'открыт' || echo 'закрыт')"
fi

echo "   PostgreSQL (5432): $(ss -tln | grep ':5432' >/dev/null && echo 'открыт' || echo 'закрыт')"
echo "   Redis (6379): $(ss -tln | grep ':6379' >/dev/null && echo 'открыт' || echo 'закрыт')"
echo ""

# 6. Проверка БД и Redis
echo "6. Проверка базы данных и Redis:"
if command -v psql >/dev/null 2>&1; then
    if [ -f "${INSTALL_DIR}/.env" ]; then
        # Проверка БД
        DB_HOST=$(grep "^DB_HOST=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_PORT=$(grep "^DB_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_NAME=$(grep "^DB_NAME=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_USER=$(grep "^DB_USER=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        DB_PASSWORD=$(grep "^DB_PASSWORD=" "${INSTALL_DIR}/.env" | cut -d= -f2)

        export PGPASSWORD="${DB_PASSWORD}"
        if psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-bot}" \
            "${DB_NAME:-cryptobot}" -c "SELECT 1" >/dev/null 2>&1; then
            echo "   ✅ База данных доступна"
            TABLE_COUNT=$(psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-bot}" \
                "${DB_NAME:-cryptobot}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')
            echo "   Количество таблиц: ${TABLE_COUNT:-0}"
        else
            echo "   ❌ База данных недоступна"
        fi

        # Проверка Redis
        REDIS_HOST=$(grep "^REDIS_HOST=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        REDIS_PORT=$(grep "^REDIS_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2)
        REDIS_PASSWORD=$(grep "^REDIS_PASSWORD=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")
        REDIS_ENABLED=$(grep "^REDIS_ENABLED=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "true")

        if [ "${REDIS_ENABLED}" = "true" ]; then
            echo "   Redis (${REDIS_HOST:-localhost}:${REDIS_PORT:-6379}):"

            if command -v redis-cli >/dev/null 2>&1; then
                REDIS_CMD="redis-cli -h '${REDIS_HOST:-localhost}' -p '${REDIS_PORT:-6379}'"
                if [ -n "${REDIS_PASSWORD}" ]; then
                    REDIS_CMD="${REDIS_CMD} -a '${REDIS_PASSWORD}'"
                fi

                if eval "${REDIS_CMD} ping 2>/dev/null" | grep -q "PONG"; then
                    echo "   ✅ Redis доступен"
                else
                    echo "   ❌ Redis недоступен"
                fi
            else
                echo "   ⚠️  redis-cli не установлен"
            fi
        else
            echo "   Redis: отключен (REDIS_ENABLED=false)"
        fi
    else
        echo "   ⚠️  Конфиг не найден"
    fi
else
    echo "   ⚠️  psql не установлен"
fi
echo ""

# 7. Проверка конфигурации webhook
echo "7. Проверка конфигурации webhook:"
if [ -f "${INSTALL_DIR}/.env" ]; then
    echo "   ✅ Конфиг найден"
    echo "   Основные настройки webhook:"
    grep -E "^(TELEGRAM_MODE|WEBHOOK_DOMAIN|WEBHOOK_PORT|WEBHOOK_USE_TLS|WEBHOOK_TLS_CERT_PATH|WEBHOOK_TLS_KEY_PATH|WEBHOOK_SECRET_TOKEN)=" "${INSTALL_DIR}/.env" | sed 's/^/   /'

    # Проверка сертификатов
    echo "   Проверка сертификатов:"

    # Получаем пути из конфига
    CERT_PATH=$(grep "^WEBHOOK_TLS_CERT_PATH=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")
    KEY_PATH=$(grep "^WEBHOOK_TLS_KEY_PATH=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")

    echo "   Путь к сертификату из конфига: ${CERT_PATH}"
    echo "   Путь к ключу из конфига: ${KEY_PATH}"

    if [ -n "${CERT_PATH}" ] && [ -f "${CERT_PATH}" ]; then
        echo "   ✅ Сертификат найден: ${CERT_PATH}"
        echo "      Размер: $(stat -c%s "${CERT_PATH}" 2>/dev/null || echo "неизвестно") bytes"
        echo "      Срок действия: $(openssl x509 -in "${CERT_PATH}" -noout -enddate 2>/dev/null | cut -d= -f2 || echo "неизвестно")"

        # Проверяем домен в сертификате
        if openssl x509 -in "${CERT_PATH}" -noout -text 2>/dev/null | grep -q "${WEBHOOK_DOMAIN}"; then
            echo "      ✅ Содержит домен: ${WEBHOOK_DOMAIN}"
        else
            echo "      ⚠️  Не содержит домен ${WEBHOOK_DOMAIN}"
            echo "      Информация о Subject:"
            openssl x509 -in "${CERT_PATH}" -noout -subject 2>/dev/null | sed 's/^/         /'
        fi
    else
        echo "   ❌ Сертификат не найден по пути из конфига"

        # Проверяем альтернативные пути
        ALT_CERT_PATHS=(
            "/etc/crypto-bot/certs/cert.pem"
            "/opt/crypto-screener-bot/ssl/fullchain.pem"
        )

        for alt_path in "${ALT_CERT_PATHS[@]}"; do
            if [ -f "${alt_path}" ]; then
                echo "   ✅ Сертификат найден по альтернативному пути: ${alt_path}"
                CERT_PATH="${alt_path}"
                break
            fi
        done
    fi

    if [ -n "${KEY_PATH}" ] && [ -f "${KEY_PATH}" ]; then
        echo "   ✅ Ключ найден: ${KEY_PATH}"
        echo "      Размер: $(stat -c%s "${KEY_PATH}" 2>/dev/null || echo "неизвестно") bytes"
    else
        echo "   ❌ Ключ не найден по пути из конфига"

        # Проверяем альтернативные пути
        ALT_KEY_PATHS=(
            "/etc/crypto-bot/certs/key.pem"
            "/opt/crypto-screener-bot/ssl/privkey.pem"
        )

        for alt_path in "${ALT_KEY_PATHS[@]}"; do
            if [ -f "${alt_path}" ]; then
                echo "   ✅ Ключ найден по альтернативному пути: ${alt_path}"
                KEY_PATH="${alt_path}"
                break
            fi
        done
    fi
else
    echo "   ❌ Конфиг не найден"
fi
echo ""

# 8. Инструкция по настройке webhook в Telegram
echo "8. ИНСТРУКЦИЯ ПО НАСТРОЙКЕ WEBHOOK В TELEGRAM:"
echo ""
if [ -f "${INSTALL_DIR}/.env" ]; then
    SECRET_TOKEN=$(grep "^WEBHOOK_SECRET_TOKEN=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "")
    WEBHOOK_PORT=$(grep "^WEBHOOK_PORT=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "8443")
    WEBHOOK_DOMAIN=$(grep "^WEBHOOK_DOMAIN=" "${INSTALL_DIR}/.env" | cut -d= -f2 2>/dev/null || echo "bot.gromovart.ru")

    echo "   Выполните команду для настройки webhook в Telegram API:"
    echo ""
    echo "   curl -X POST 'https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook' \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d '{"
    echo "       \"url\": \"https://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}/webhook\","
    echo "       \"secret_token\": \"${SECRET_TOKEN}\""
    echo "     }'"
    echo ""
    echo "   Для проверки webhook:"
    echo "   curl -X POST 'https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo'"
    echo ""
    echo "   Для удаления webhook:"
    echo "   curl -X POST 'https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook'"
else
    echo "   ⚠️  Не удалось получить настройки для инструкции"
fi
echo ""

echo "=== ПРОВЕРКА ЗАВЕРШЕНА ==="
EOF

    log_info "Проверка завершена"
}

# Основная функция
main() {
    log_step "Начало развертывания Crypto Exchange Screener Bot с Webhook"
    log_info "Сервер: ${SERVER_USER}@${SERVER_IP}"
    log_info "Директория установки: ${INSTALL_DIR}"
    log_info "Имя сервиса: ${SERVICE_NAME}"
    log_info "Telegram режим: ${TELEGRAM_MODE}"
    log_info "Webhook: ${WEBHOOK_DOMAIN}:${WEBHOOK_PORT} (TLS: ${WEBHOOK_USE_TLS})"
    echo ""

    # Читаем настройки из .env файла
    read_env_config

    # Проверяем локальный конфиг перед началом
    check_local_config

    # Выполнение шагов развертывания
    check_ssh_connection
    install_dependencies
    setup_postgresql
    setup_redis
    setup_ssl_certificates  # ОБНОВЛЕННАЯ ФУНКЦИЯ
    setup_firewall
    create_directory_structure
    setup_logging
    copy_source_code
    build_application
    setup_configuration     # ОБНОВЛЕННАЯ ФУНКЦИЯ
    setup_systemd_service
    check_migrations
    start_application
    verify_deployment

    log_step "✅ Развертывание успешно завершено!"
    echo ""
    log_info "📋 Использованы настройки:"
    log_info "  PostgreSQL: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
    log_info "  Redis: ${REDIS_HOST}:${REDIS_PORT} (включен: ${REDIS_ENABLED})"
    log_info "  Webhook: ${WEBHOOK_DOMAIN}:${WEBHOOK_PORT} (TLS: ${WEBHOOK_USE_TLS})"
    log_info "  Telegram режим: ${TELEGRAM_MODE}"
    echo ""
    log_info "ВАЖНО: Проверьте настройки в файле: ${INSTALL_DIR}/.env"
    log_info "Обязательные настройки для проверки:"
    log_info "1. TG_API_KEY - токен Telegram бота"
    log_info "2. TG_CHAT_ID - ваш Chat ID"
    log_info "3. Биржевые API ключи (BYBIT_API_KEY/SECRET или BINANCE_API_KEY/SECRET)"
    echo ""
    log_info "Для настройки webhook в Telegram выполните команду из инструкции выше"
    log_info "или проверьте вывод секции 'ИНСТРУКЦИЯ ПО НАСТРОЙКЕ WEBHOOK'"
    echo ""
    log_info "Команды управления:"
    log_info "  systemctl status ${SERVICE_NAME}  # Статус сервиса"
    log_info "  systemctl restart ${SERVICE_NAME} # Перезапуск"
    log_info "  journalctl -u ${SERVICE_NAME} -f  # Просмотр логов"
    log_info "  ss -tln | grep ':${WEBHOOK_PORT}'  # Проверка webhook порта"
    echo ""
    log_info "Для настройки конфигурации на сервере:"
    log_info "  nano ${INSTALL_DIR}/.env"
    log_info "  systemctl restart ${SERVICE_NAME}"
    echo ""
    log_info "📝 Webhook URL для Telegram:"
    log_info "  https://${WEBHOOK_DOMAIN}:${WEBHOOK_PORT}/webhook"
}

# Запуск скрипта
parse_args "$@"
main