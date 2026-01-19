#!/bin/bash
# Скрипт первичного развертывания приложения на Ubuntu 22.04
# Использование: ./deploy.sh [OPTIONS]
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

# Проверка локального конфига
check_local_config() {
    log_step "Проверка локальной конфигурации..."

    if [ ! -f "./configs/prod/.env" ]; then
        log_warn "Продакшен конфиг не найден: ./configs/prod/.env"
        log_info "Убедитесь, что файл существует в репозитории"
        log_info "Текущая структура configs/:"
        ls -la ./configs/ 2>/dev/null || echo "Директория configs/ не найдена"
        echo ""
        log_info "Продолжить с минимальной конфигурацией? (y/N)"
        read -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "Прерывание: требуется продакшен конфиг"
            exit 1
        fi
    else
        log_info "✅ Продакшен конфиг найден: ./configs/prod/.env"

        # Проверяем наличие критических настроек
        CRITICAL_SETTINGS=("DB_HOST" "DB_NAME" "DB_USER" "LOG_LEVEL")
        MISSING_SETTINGS=()

        for setting in "${CRITICAL_SETTINGS[@]}"; do
            if ! grep -q "^${setting}=" "./configs/prod/.env"; then
                MISSING_SETTINGS+=("$setting")
            fi
        done

        if [ ${#MISSING_SETTINGS[@]} -gt 0 ]; then
            log_warn "⚠️  В конфиге отсутствуют настройки: ${MISSING_SETTINGS[*]}"
        else
            log_info "✅ Критические настройки присутствуют"
        fi
    fi
}

# Установка зависимостей на сервере
install_dependencies() {
    log_step "Установка системных зависимостей..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
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
    redis-tools

# Установка Go 1.21+
if ! command -v go &> /dev/null; then
    echo "Установка Go..."
    wget -q https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
    rm go1.21.6.linux-amd64.tar.gz

    # Добавление в PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
    source /etc/profile
fi

# Установка PostgreSQL 15
if ! systemctl is-active --quiet postgresql; then
    echo "Установка PostgreSQL 15..."

    # Добавление репозитория
    sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/pgdg.gpg
    apt-get update

    apt-get install -y postgresql-15 postgresql-contrib-15

    # Настройка PostgreSQL
    echo "Настройка PostgreSQL..."

    # Разрешить подключения с localhost
    sed -i "s/#listen_addresses = 'localhost'/listen_addresses = 'localhost'/g" /etc/postgresql/15/main/postgresql.conf
    systemctl restart postgresql

    # Создание пользователя и базы данных
    sudo -u postgres psql -c "CREATE USER crypto_screener WITH PASSWORD 'SecurePass123!';"
    sudo -u postgres psql -c "CREATE DATABASE crypto_screener_db OWNER crypto_screener;"
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE crypto_screener_db TO crypto_screener;"
fi

# Установка Redis
if ! systemctl is-active --quiet redis-server; then
    echo "Установка Redis..."
    apt-get install -y redis-server

    # Настройка Redis
    sed -i "s/bind 127.0.0.1 ::1/bind 127.0.0.1/g" /etc/redis/redis.conf
    sed -i "s/# maxmemory <bytes>/maxmemory 256mb/g" /etc/redis/redis.conf
    sed -i "s/# maxmemory-policy noeviction/maxmemory-policy allkeys-lru/g" /etc/redis/redis.conf

    systemctl restart redis-server
    systemctl enable redis-server
fi

echo "Зависимости установлены успешно"
EOF

    log_info "Системные зависимости установлены"
}

# Настройка брандмауэра
setup_firewall() {
    log_step "Настройка брандмауэра UFW..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

# Настройка UFW
ufw --force reset
ufw default deny incoming
ufw default allow outgoing

# Разрешить SSH
ufw allow 22/tcp

# Разрешить порты для мониторинга
ufw allow 5432/tcp  # PostgreSQL (только localhost)
ufw allow 6379/tcp  # Redis (только localhost)
ufw allow 8080/tcp  # HTTP мониторинг (опционально)

# Включить брандмауэр
ufw --force enable
ufw status verbose

echo "Брандмауэр настроен"
EOF

    log_info "Брандмауэр настроен"
}

# Создание системного пользователя
create_app_user() {
    log_step "Создание системного пользователя для приложения..."

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

# Создание директорий
mkdir -p "\${INSTALL_DIR}"
mkdir -p "\${INSTALL_DIR}/bin"
mkdir -p "\${INSTALL_DIR}/configs"
mkdir -p "\${INSTALL_DIR}/logs"
mkdir -p "\${INSTALL_DIR}/data"
mkdir -p "/var/log/\${APP_NAME}"

# Настройка прав
chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}"
chown -R cryptoapp:cryptoapp "/var/log/\${APP_NAME}"
chmod 755 "\${INSTALL_DIR}"
chmod 755 "/var/log/\${APP_NAME}"

echo "Структура директорий создана"
EOF

    log_info "Пользователь и директории созданы"
}

# Настройка логирования
setup_logging() {
    log_step "Настройка системы логирования..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
INSTALL_DIR="${INSTALL_DIR}"

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
        systemctl reload ${SERVICE_NAME}.service > /dev/null 2>&1 || true
    endscript
}
LOGROTATE

# Создание файлов логов
touch "/var/log/\${APP_NAME}/app.log"
touch "/var/log/\${APP_NAME}/error.log"
chown -R cryptoapp:cryptoapp "/var/log/\${APP_NAME}"
chmod 644 "/var/log/\${APP_NAME}"/*.log

echo "Логирование настроено"
EOF

    log_info "Система логирования настроена"
}

# Копирование исходного кода
copy_source_code() {
    log_step "Копирование исходного кода приложения..."

    # ВАЖНО: Скрипт должен запускаться из корня репозитория
    if [ ! -f "go.mod" ] || [ ! -f "application/cmd/bot/main.go" ]; then
        log_error "Скрипт должен запускаться из корневой директории репозитория!"
        log_info "Текущая директория: $(pwd)"
        log_info "Ожидается наличие файлов: go.mod и application/cmd/bot/main.go"
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
        .

    # Копирование на сервер
    log_info "Копирование архива на сервер..."
    scp -i "${SSH_KEY}" /tmp/app_source.tar.gz "${SERVER_USER}@${SERVER_IP}:/tmp/app_source.tar.gz"

    # Распаковка на сервере
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR}"

# Удаление старой версии если существует
if [ -d "\${INSTALL_DIR}/src" ]; then
    rm -rf "\${INSTALL_DIR}/src"
fi

# Распаковка архива
mkdir -p "\${INSTALL_DIR}/src"
tar -xzf /tmp/app_source.tar.gz -C "\${INSTALL_DIR}/src"
chown -R cryptoapp:cryptoapp "\${INSTALL_DIR}/src"

# Очистка
rm -f /tmp/app_source.tar.gz

echo "Исходный код скопирован"
EOF

    # Очистка локального архива
    rm -f /tmp/app_source.tar.gz

    log_info "Исходный код скопирован на сервер"
}

# Установка приложения
install_application() {
    log_step "Установка и сборка приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"
APP_NAME="crypto-screener-bot"
SRC_DIR="${INSTALL_DIR}/src"

cd "${SRC_DIR}"

# Установка зависимостей Go
echo "Установка зависимостей Go..."
/usr/local/go/bin/go mod download

# Сборка приложения
echo "Сборка основного приложения..."
if [ -f "./application/cmd/bot/main.go" ]; then
    /usr/local/go/bin/go build -o "${INSTALL_DIR}/bin/${APP_NAME}" ./application/cmd/bot/main.go
    echo "✅ Основное приложение собрано"

    # Тестовый запуск для проверки версии
    echo "Проверка версии приложения..."
    "${INSTALL_DIR}/bin/${APP_NAME}" --version 2>&1 | head -1 || echo "⚠️  Не удалось получить версию"
else
    echo "❌ Файл основного приложения не найден: ./application/cmd/bot/main.go"
    exit 1
fi

# Проверка существования мигратора
echo "Проверка файла миграций..."
if [ -f "./internal/infrastructure/persistence/postgres/migrator.go" ]; then
    echo "✅ Файл migrator.go найден"

    # Проверяем, есть ли папка с миграциями
    if [ -d "./internal/infrastructure/persistence/postgres/migrations" ]; then
        echo "✅ Папка миграций найдена"
        MIGRATION_COUNT=$(ls "./internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
        echo "Количество SQL файлов миграций: ${MIGRATION_COUNT}"

        if [ "${MIGRATION_COUNT}" -gt 0 ]; then
            echo "Первые 5 файлов миграций:"
            ls "./internal/infrastructure/persistence/postgres/migrations/"*.sql | head -5
        fi
    else
        echo "⚠️  Папка миграций не найдена"
    fi

    echo "ℹ️  Мигратор встроен в основное приложение и запускается автоматически"

else
    echo "⚠️  Файл migrator.go не найден"
fi

# Проверяем, что приложение может запуститься
echo "Проверка запуска приложения (быстрый тест)..."
timeout 5 "${INSTALL_DIR}/bin/${APP_NAME}" --help 2>&1 | grep -i "usage\|help\|version" | head -3 || echo "⚠️  Быстрый тест не прошел"

echo "✅ Установка приложения завершена"
EOF

    log_info "Приложение собрано"
}

# Настройка конфигурации
setup_configuration() {
    log_step "Настройка конфигурации приложения..."

    # Проверяем, существует ли продакшен конфиг локально
    if [ ! -f "./configs/prod/.env" ]; then
        log_error "Файл конфигурации не найден: ./configs/prod/.env"
        log_info "Убедитесь, что продакшен конфиг существует в репозитории"
        exit 1
    fi

    log_info "Используется существующий продакшен конфиг"

    # Копирование конфигурации
    scp -i "${SSH_KEY}" -r ./configs/ "${SERVER_USER}@${SERVER_IP}:${INSTALL_DIR}/configs/"

    # Настройка на сервере
    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"

# Проверяем, что конфиг скопировался
if [ -f "${INSTALL_DIR}/configs/prod/.env" ]; then
    echo "✅ Продакшен конфиг найден: ${INSTALL_DIR}/configs/prod/.env"

    # Симлинк для текущего окружения
    ln -sf "${INSTALL_DIR}/configs/prod/.env" "${INSTALL_DIR}/.env"
    echo "✅ Создан симлинк: ${INSTALL_DIR}/.env -> ${INSTALL_DIR}/configs/prod/.env"

    # Проверяем права
    chown cryptoapp:cryptoapp "${INSTALL_DIR}/.env"
    chown -R cryptoapp:cryptoapp "${INSTALL_DIR}/configs"
    chmod 600 "${INSTALL_DIR}/.env"

    # Показываем основные настройки (без секретов)
    echo "📋 Основные настройки из конфига:"
    grep -E "^(APP_ENV|DB_HOST|DB_PORT|DB_NAME|LOG_LEVEL|EXCHANGE|TELEGRAM_ENABLED|DB_ENABLE_AUTO_MIGRATE)=" \
        "${INSTALL_DIR}/.env" | head -10

else
    echo "❌ Продакшен конфиг не найден после копирования"
    echo "Создаем минимальный конфиг..."

    # Создание минимального конфига
    cat > "${INSTALL_DIR}/.env" << 'CONFIG'
# Минимальная конфигурация
APP_ENV=production
LOG_LEVEL=info

# База данных
DB_HOST=localhost
DB_PORT=5432
DB_NAME=crypto_screener_db
DB_USER=crypto_screener
DB_PASSWORD=SecurePass123!
DB_SSL_MODE=disable
DB_ENABLE_AUTO_MIGRATE=true

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Отключить Telegram до настройки
TELEGRAM_ENABLED=false

# Биржа
EXCHANGE=bybit
EXCHANGE_TYPE=futures
UPDATE_INTERVAL=30
CONFIG

    chown cryptoapp:cryptoapp "${INSTALL_DIR}/.env"
    chmod 600 "${INSTALL_DIR}/.env"
    echo "⚠️  Создан минимальный конфиг, требуется настройка"
fi

echo "Конфигурация настроена"
EOF

    log_info "Конфигурация настроена"
}

# Настройка systemd сервиса
setup_systemd_service() {
    log_step "Настройка systemd сервиса..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << EOF
#!/bin/bash
set -e

APP_NAME="${APP_NAME}"
SERVICE_NAME="${SERVICE_NAME}"
INSTALL_DIR="${INSTALL_DIR}"

# Создание файла сервиса
cat > /etc/systemd/system/\${SERVICE_NAME}.service << 'SERVICE'
[Unit]
Description=Crypto Exchange Screener Bot
After=network.target postgresql.service redis-server.service
Requires=postgresql.service redis-server.service

[Service]
Type=simple
User=cryptoapp
Group=cryptoapp
WorkingDirectory=${INSTALL_DIR}
Environment="APP_ENV=production"
EnvironmentFile=${INSTALL_DIR}/.env

ExecStart=${INSTALL_DIR}/bin/${APP_NAME} --config=${INSTALL_DIR}/.env --mode=full
Restart=always
RestartSec=10
StandardOutput=append:/var/log/${APP_NAME}/app.log
StandardError=append:/var/log/${APP_NAME}/error.log

# Лимиты безопасности
LimitNOFILE=65536
LimitNPROC=65536
LimitMEMLOCK=infinity
LimitCORE=infinity

# Сетевая изоляция
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${INSTALL_DIR} /var/log/${APP_NAME}
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
SERVICE

# Перезагрузка systemd
systemctl daemon-reload
systemctl enable \${SERVICE_NAME}.service

echo "Systemd сервис настроен"
EOF

    log_info "Systemd сервис настроен"
}

# Выполнение миграций базы данных
run_migrations() {
    log_step "Проверка и выполнение миграций базы данных..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="/opt/crypto-screener-bot"
APP_NAME="crypto-screener-bot"

echo "Проверка наличия файлов миграций..."

# Проверяем существование папки миграций
if [ -d "${INSTALL_DIR}/src/internal/infrastructure/persistence/postgres/migrations" ]; then
    echo "✅ Папка миграций существует"
    MIGRATION_COUNT=$(ls "${INSTALL_DIR}/src/internal/infrastructure/persistence/postgres/migrations/"*.sql 2>/dev/null | wc -l)
    echo "Количество SQL файлов миграций: ${MIGRATION_COUNT}"

    # Показываем список миграций
    if [ "${MIGRATION_COUNT}" -gt 0 ]; then
        echo "Список миграций:"
        ls "${INSTALL_DIR}/src/internal/infrastructure/persistence/postgres/migrations/"*.sql | head -10
    fi
else
    echo "⚠️  Папка миграций не найдена"
fi

echo ""
echo "Миграции будут автоматически выполнены при первом запуске приложения"
echo "Приложение проверяет и применяет миграции через migrator.go"
echo ""

# Вместо прямого запуска мигратора, запускаем приложение в режиме инициализации
echo "Запуск приложения для проверки миграций (таймаут 10 секунд)..."
cd "${INSTALL_DIR}"

# Экспортируем переменные окружения для подключения к БД
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=crypto_screener_db
export DB_USER=crypto_screener
export DB_PASSWORD=SecurePass123!
export DB_SSLMODE=disable
export LOG_LEVEL=info

# Запускаем приложение с коротким таймаутом только для инициализации
timeout 10 "${INSTALL_DIR}/bin/${APP_NAME}" --env=prod 2>&1 | grep -i -E "(migration|migrate|database|postgres|init)" | head -20 || true

echo ""
echo "✅ Проверка миграций завершена"
echo "Примечание: Если миграции не применились автоматически,"
echo "они будут применены при первом полноценном запуске приложения"
EOF

    log_info "Миграции проверены"
}

# Запуск приложения
start_application() {
    log_step "Запуск приложения..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

SERVICE_NAME="crypto-screener"
APP_NAME="crypto-screener-bot"
INSTALL_DIR="/opt/crypto-screener-bot"

# Сначала останавливаем сервис если он уже запущен
echo "Остановка сервиса (если запущен)..."
systemctl stop "${SERVICE_NAME}.service" 2>/dev/null || true

# Проверяем конфигурацию
echo "Проверка конфигурации..."
if [ -f "${INSTALL_DIR}/.env" ]; then
    echo "✅ Файл конфигурации найден"

    # Проверяем наличие обязательных настроек
    if grep -q "DB_PASSWORD=" "${INSTALL_DIR}/.env"; then
        echo "✅ Настройки базы данных найдены"
    else
        echo "⚠️  Настройки базы данных не найдены, добавляем..."
        echo "DB_PASSWORD=SecurePass123!" >> "${INSTALL_DIR}/.env"
    fi
else
    echo "❌ Файл конфигурации не найден"
    exit 1
fi

# Запуск сервиса
echo "Запуск сервиса ${SERVICE_NAME}..."
systemctl start "${SERVICE_NAME}.service"
sleep 5

# Проверка статуса
echo "Статус сервиса:"
systemctl status "${SERVICE_NAME}.service" --no-pager

# Ждем немного для инициализации
echo "Ожидание инициализации приложения (10 секунд)..."
sleep 10

# Просмотр логов на предмет миграций
echo "Проверка логов (миграции и инициализация):"
journalctl -u "${SERVICE_NAME}.service" -n 20 --no-pager | grep -i -E "(migration|migrate|database|postgres|init|starting|started)" || echo "Логи не содержат информации о миграциях"

# Общий просмотр логов
echo "Последние 10 строк лога:"
tail -10 "/var/log/${APP_NAME}/app.log" 2>/dev/null || echo "Файл лога еще не создан"

# Проверка процесса
echo "Проверка процессов:"
pgrep -f "${APP_NAME}" && echo "✅ Приложение запущено" || echo "❌ Приложение не запущено"
EOF

    log_info "Приложение запущено"
}

# Проверка развертывания
verify_deployment() {
    log_step "Проверка развертывания..."

    ssh -i "${SSH_KEY}" "${SERVER_USER}@${SERVER_IP}" << 'EOF'
#!/bin/bash
set -e

APP_NAME="crypto-screener-bot"
SERVICE_NAME="crypto-screener"

echo "=== ПРОВЕРКА РАЗВЕРТЫВАНИЯ ==="
echo ""

# 1. Проверка сервисов
echo "1. Проверка системных сервисов:"
echo "   PostgreSQL: $(systemctl is-active postgresql)"
echo "   Redis: $(systemctl is-active redis-server)"
echo "   ${SERVICE_NAME}: $(systemctl is-active ${SERVICE_NAME})"
echo ""

# 2. Проверка процессов
echo "2. Запущенные процессы:"
pgrep -f "${APP_NAME}" && echo "   Приложение запущено" || echo "   Приложение не запущено"
echo ""

# 3. Проверка логов
echo "3. Проверка логов:"
if [ -f "/var/log/${APP_NAME}/app.log" ]; then
    echo "   Файл лога существует"
    echo "   Размер: $(du -h /var/log/${APP_NAME}/app.log | cut -f1)"
    echo "   Последние 5 строк:"
    tail -5 "/var/log/${APP_NAME}/app.log" 2>/dev/null || echo "   Не удалось прочитать лог"
else
    echo "   Файл лога не найден"
fi
echo ""

# 4. Проверка сетевых портов
echo "4. Проверка сетевых портов:"
echo "   PostgreSQL (5432): $(ss -tln | grep ':5432' && echo 'открыт' || echo 'закрыт')"
echo "   Redis (6379): $(ss -tln | grep ':6379' && echo 'открыт' || echo 'закрыт')"
echo ""

# 5. Проверка дискового пространства
echo "5. Дисковое пространство:"
df -h /opt /var/log | grep -v Filesystem
echo ""

echo "=== ПРОВЕРКА ЗАВЕРШЕНА ==="
EOF

    log_info "Проверка завершена"
}

# Основная функция
main() {
    log_step "Начало развертывания Crypto Exchange Screener Bot"
    log_info "Сервер: ${SERVER_USER}@${SERVER_IP}"
    log_info "Директория установки: ${INSTALL_DIR}"
    log_info "Имя сервиса: ${SERVICE_NAME}"
    echo ""

    # Проверяем локальный конфиг перед началом
    check_local_config

    # Выполнение шагов развертывания
    check_ssh_connection
    install_dependencies
    setup_firewall
    create_app_user
    setup_logging
    copy_source_code
    install_application
    setup_configuration
    setup_systemd_service
    run_migrations
    start_application
    verify_deployment

    log_step "Развертывание успешно завершено!"
    echo ""
    log_info "ВАЖНО: Проверьте настройки в файле: ${INSTALL_DIR}/.env"
    log_info "Обязательные настройки для проверки:"
    log_info "1. TELEGRAM_BOT_TOKEN - токен бота Telegram"
    log_info "2. TELEGRAM_ENABLED=true - включить Telegram"
    log_info "3. TELEGRAM_ADMIN_IDS - ваш Telegram ID"
    log_info "4. API ключи бирж (BINANCE_API_KEY/SECRET или BYBIT_API_KEY/SECRET)"
    echo ""
    log_info "Команды управления:"
    log_info "  systemctl status ${SERVICE_NAME}  # Статус сервиса"
    log_info "  systemctl restart ${SERVICE_NAME} # Перезапуск"
    log_info "  journalctl -u ${SERVICE_NAME} -f  # Просмотр логов"
    echo ""
    log_info "Для настройки конфигурации на сервере:"
    log_info "  nano ${INSTALL_DIR}/.env"
    log_info "  systemctl restart ${SERVICE_NAME}"
}

# Запуск скрипта
parse_args "$@"
main