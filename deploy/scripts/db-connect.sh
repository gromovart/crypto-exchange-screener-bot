#!/bin/bash
# Скрипт для безопасного подключения к БД через SSH туннель
# Использование: ./deploy/scripts/db-connect.sh [COMMAND] [OPTIONS]
# Команды:
#   start     - Создать SSH туннель
#   stop      - Остановить SSH туннель
#   status    - Проверить статус туннеля
#   config    - Показать конфигурацию для DataGrip
#   test      - Проверить подключение через туннель

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(realpath "$SCRIPT_DIR/../..")"
CONFIG_FILE="$PROJECT_ROOT/configs/prod/.env"
TUNNEL_PID_FILE="/tmp/crypto_db_tunnel.pid"
LOG_FILE="/tmp/crypto_db_tunnel.log"

# Параметры по умолчанию (будут переопределены из .env)
SERVER_IP="95.142.40.244"
SERVER_USER="root"
SSH_KEY="${HOME}/.ssh/id_rsa"
LOCAL_PORT="15432"  # Локальный порт для туннеля
REMOTE_PORT="5432"  # Порт PostgreSQL на сервере

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

# Проверка занят ли порт (кросс-платформенная)
check_port_in_use() {
    local port=$1

    # Проверяем ОС
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if lsof -ti:"$port" > /dev/null 2>&1; then
            return 0  # Порт занят
        else
            return 1  # Порт свободен
        fi
    else
        # Linux и другие
        if ss -tln | grep -q ":$port "; then
            return 0  # Порт занят
        else
            return 1  # Порт свободен
        fi
    fi
}

# Проверка, слушает ли порт
is_port_listening() {
    local port=$1

    if check_port_in_use "$port"; then
        return 0  # Порт слушает
    else
        return 1  # Порт не слушает
    fi
}

# Получить процесс использующий порт
get_process_on_port() {
    local port=$1

    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        lsof -ti:"$port" 2>/dev/null || echo ""
    else
        # Linux
        ss -tlnp | grep ":$port " | awk '{print $7}' | cut -d= -f2 | cut -d, -f1 || echo ""
    fi
}

# Чтение конфигурации из .env файла
read_config() {
    if [ -f "$CONFIG_FILE" ]; then
        log_info "Чтение конфигурации из: $CONFIG_FILE"

        # Базовые параметры подключения
        if grep -q "^SSH_HOST=" "$CONFIG_FILE"; then
            SERVER_IP=$(grep "^SSH_HOST=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        if grep -q "^SSH_USER=" "$CONFIG_FILE"; then
            SERVER_USER=$(grep "^SSH_USER=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        if grep -q "^SSH_KEY=" "$CONFIG_FILE"; then
            SSH_KEY=$(grep "^SSH_KEY=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        # Параметры БД из конфига
        if grep -q "^DB_HOST=" "$CONFIG_FILE"; then
            DB_HOST=$(grep "^DB_HOST=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        if grep -q "^DB_PORT=" "$CONFIG_FILE"; then
            REMOTE_PORT=$(grep "^DB_PORT=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        if grep -q "^DB_NAME=" "$CONFIG_FILE"; then
            DB_NAME=$(grep "^DB_NAME=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        if grep -q "^DB_USER=" "$CONFIG_FILE"; then
            DB_USER=$(grep "^DB_USER=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi

        if grep -q "^DB_PASSWORD=" "$CONFIG_FILE"; then
            DB_PASSWORD=$(grep "^DB_PASSWORD=" "$CONFIG_FILE" | cut -d= -f2- | xargs)
        fi
    else
        log_warn "Файл конфигурации не найден: $CONFIG_FILE"
        log_info "Используются параметры по умолчанию"
    fi

    # Если DB_HOST не localhost, используем его как SERVER_IP для туннеля
    if [ -n "$DB_HOST" ] && [ "$DB_HOST" != "localhost" ] && [ "$DB_HOST" != "127.0.0.1" ]; then
        log_info "DB_HOST указывает на удаленный сервер: $DB_HOST"
        log_info "Использую $DB_HOST для SSH туннеля"
        SERVER_IP="$DB_HOST"
    fi
}

# Проверка SSH ключа
check_ssh_key() {
    if [ ! -f "$SSH_KEY" ]; then
        log_error "SSH ключ не найден: $SSH_KEY"
        log_info "Доступные ключи:"
        ls -la ~/.ssh/id_rsa* 2>/dev/null | grep -v ".pub"
        log_info "Укажите ключ через параметр --key или настройте в .env"
        exit 1
    fi

    # Проверка прав на ключ
    KEY_PERMS=$(stat -f "%A" "$SSH_KEY" 2>/dev/null || stat -c "%a" "$SSH_KEY")
    if [ "$KEY_PERMS" != "600" ]; then
        log_warn "Исправляю права SSH ключа..."
        chmod 600 "$SSH_KEY"
    fi

    # Проверка подключения
    log_info "Проверка SSH подключения к $SERVER_USER@$SERVER_IP..."
    if ssh -o BatchMode=yes \
           -o ConnectTimeout=5 \
           -i "$SSH_KEY" \
           "$SERVER_USER@$SERVER_IP" "echo 'SSH OK'" &> /dev/null; then
        log_info "✅ SSH подключение работает"
        return 0
    else
        log_error "SSH подключение не работает"
        log_info "Проверьте:"
        log_info "  1. Доступность сервера: ping $SERVER_IP"
        log_info "  2. Правильность SSH ключа"
        log_info "  3. Наличие пользователя $SERVER_USER на сервере"
        exit 1
    fi
}

# Проверка, запущен ли уже туннель
check_tunnel_status() {
    if [ -f "$TUNNEL_PID_FILE" ]; then
        PID=$(cat "$TUNNEL_PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            log_info "✅ SSH туннель запущен (PID: $PID)"

            # Проверяем, слушает ли порт
            if is_port_listening "$LOCAL_PORT"; then
                log_info "✅ Локальный порт $LOCAL_PORT слушает"
                return 0
            else
                log_warn "Порт $LOCAL_PORT не слушает, но процесс есть"
                return 1
            fi
        else
            log_warn "Процесс туннеля не найден, очистка PID файла"
            rm -f "$TUNNEL_PID_FILE"
            return 1
        fi
    else
        log_info "SSH туннель не запущен"
        return 1
    fi
}

# Запуск SSH туннеля
start_tunnel() {
    log_step "Запуск SSH туннеля к БД..."

    # Проверяем, не запущен ли уже
    if check_tunnel_status; then
        log_info "Туннель уже запущен, используйте 'stop' чтобы остановить"
        return 0
    fi

    # Проверяем, свободен ли локальный порт
    if is_port_listening "$LOCAL_PORT"; then
        log_error "Локальный порт $LOCAL_PORT уже занят"
        log_info "Используйте другой порт:"
        log_info "  export LOCAL_PORT=15433"
        log_info "  или укажите в .env: LOCAL_PORT=15433"
        exit 1
    fi

    log_info "Создание туннеля:"
    log_info "  Локально: localhost:$LOCAL_PORT"
    log_info "  Удаленно: $SERVER_IP:$REMOTE_PORT"
    log_info "  SSH: $SERVER_USER@$SERVER_IP"

    # Запускаем SSH туннель в фоне
    ssh -N -L "$LOCAL_PORT:localhost:$REMOTE_PORT" \
        -i "$SSH_KEY" \
        "$SERVER_USER@$SERVER_IP" \
        -o ExitOnForwardFailure=yes \
        -o ServerAliveInterval=60 \
        -o ServerAliveCountMax=3 \
        &> "$LOG_FILE" &

    TUNNEL_PID=$!

    # Сохраняем PID
    echo "$TUNNEL_PID" > "$TUNNEL_PID_FILE"

    # Ждем инициализации
    sleep 3

    # Проверяем запуск
    if check_tunnel_status; then
        log_info "✅ SSH туннель успешно запущен"
        log_info "   PID: $TUNNEL_PID"
        log_info "   Лог: $LOG_FILE"

        # Показываем команду для подключения
        echo ""
        log_info "Теперь можно подключиться к БД через:"
        log_info "  Host: localhost"
        log_info "  Port: $LOCAL_PORT"
        log_info "  Database: ${DB_NAME:-cryptobot}"
        log_info "  User: ${DB_USER:-bot}"
        log_info "  Password: ${DB_PASSWORD:-SecurePass123!}"
    else
        log_error "Не удалось запустить SSH туннель"
        log_info "Проверьте лог: tail -20 $LOG_FILE"
        tail -20 "$LOG_FILE" 2>/dev/null || true
        exit 1
    fi
}

# Остановка SSH туннеля
stop_tunnel() {
    log_step "Остановка SSH туннеля..."

    if [ -f "$TUNNEL_PID_FILE" ]; then
        PID=$(cat "$TUNNEL_PID_FILE")

        if ps -p "$PID" > /dev/null 2>&1; then
            log_info "Остановка процесса туннеля (PID: $PID)..."
            kill "$PID"

            # Ждем завершения
            sleep 2

            if ps -p "$PID" > /dev/null 2>&1; then
                log_warn "Процесс не остановился, принудительное завершение..."
                kill -9 "$PID"
            fi
        fi

        rm -f "$TUNNEL_PID_FILE"
        log_info "✅ SSH туннель остановлен"
    else
        log_info "SSH туннель не был запущен"
    fi

    # Проверяем, освободился ли порт
    if is_port_listening "$LOCAL_PORT"; then
        log_warn "Порт $LOCAL_PORT все еще занят"
        log_info "Найдите и завершите процесс:"

        if [[ "$OSTYPE" == "darwin"* ]]; then
            log_info "  lsof -ti:$LOCAL_PORT | xargs kill -9"
            PROCESS_IDS=$(lsof -ti:"$LOCAL_PORT" 2>/dev/null || true)
            if [ -n "$PROCESS_IDS" ]; then
                log_info "  Запущенные процессы на порту $LOCAL_PORT: $PROCESS_IDS"
            fi
        else
            log_info "  sudo ss -tlnp | grep :$LOCAL_PORT"
        fi
    fi
}

# Показать конфигурацию для DataGrip
show_config() {
    log_step "Конфигурация для DataGrip/IntelliJ IDEA:"
    echo ""

    echo "1. Создайте новое подключение PostgreSQL:"
    echo "   Database → New → Data Source → PostgreSQL"
    echo ""

    echo "2. Основные настройки:"
    echo "   Host: localhost"
    echo "   Port: $LOCAL_PORT"
    echo "   Database: ${DB_NAME:-cryptobot}"
    echo "   User: ${DB_USER:-bot}"
    echo "   Password: ${DB_PASSWORD:-SecurePass123!}"
    echo ""

    echo "3. SSH/SSL вкладка (ВАЖНО!):"
    echo "   ☑ Use SSH tunnel"
    echo "   Host: $SERVER_IP"
    echo "   Port: 22"
    echo "   User: $SERVER_USER"
    echo "   Authentication type: Key pair"
    echo "   Private key: $SSH_KEY"
    echo ""

    echo "4. Тестовые команды:"
    echo "   Подключиться через psql:"
    echo "   PGPASSWORD='${DB_PASSWORD:-SecurePass123!}' psql -h localhost -p $LOCAL_PORT -U ${DB_USER:-bot} -d ${DB_NAME:-cryptobot}"
    echo ""

    echo "5. Файл конфигурации:"
    echo "   $CONFIG_FILE"
}

# Тестирование подключения через туннель
test_connection() {
    log_step "Тестирование подключения к БД..."

    # Проверяем туннель
    if ! check_tunnel_status; then
        log_error "SSH туннель не запущен"
        log_info "Запустите сначала: ./deploy/scripts/db-connect.sh start"
        exit 1
    fi

    # Проверяем, установлен ли psql
    if ! command -v psql &> /dev/null; then
        log_error "psql не установлен"
        log_info "Установите:"
        log_info "  macOS: brew install postgresql"
        log_info "  Ubuntu: sudo apt-get install postgresql-client"
        log_info "  Fedora: sudo dnf install postgresql"
        exit 1
    fi

    # Пробуем подключиться
    log_info "Подключение к БД через туннель..."

    export PGPASSWORD="${DB_PASSWORD:-SecurePass123!}"

    if psql -h localhost \
            -p "$LOCAL_PORT" \
            -U "${DB_USER:-bot}" \
            -d "${DB_NAME:-cryptobot}" \
            -c "SELECT version();" 2>/dev/null; then
        log_info "✅ Подключение к БД успешно!"

        # Показываем информацию о БД
        echo ""
        log_info "Информация о БД:"
        psql -h localhost \
             -p "$LOCAL_PORT" \
             -U "${DB_USER:-bot}" \
             -d "${DB_NAME:-cryptobot}" \
             -c "SELECT current_database(), current_user, inet_server_addr(), inet_server_port();" 2>/dev/null

        echo ""
        log_info "Таблицы в БД:"
        psql -h localhost \
             -p "$LOCAL_PORT" \
             -U "${DB_USER:-bot}" \
             -d "${DB_NAME:-cryptobot}" \
             -c "SELECT schemaname, tablename FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY tablename;" 2>/dev/null | head -20
    else
        log_error "Не удалось подключиться к БД"
        log_info "Проверьте:"
        log_info "  1. Запущен ли SSH туннель"
        log_info "  2. Правильность параметров БД"
        log_info "  3. Существует ли база данных ${DB_NAME:-cryptobot}"
        exit 1
    fi
}

# Показать справку
show_help() {
    echo "Использование: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Команды:"
    echo "  start           - Запустить SSH туннель к БД"
    echo "  stop            - Остановить SSH туннель"
    echo "  status          - Показать статус туннеля"
    echo "  config          - Показать конфигурацию для DataGrip"
    echo "  test            - Протестировать подключение к БД"
    echo "  help            - Показать эту справку"
    echo ""
    echo "Опции:"
    echo "  --local-port=NUM    Локальный порт (по умолчанию: 15432)"
    echo "  --key=PATH          Путь к SSH ключу"
    echo ""
    echo "Примеры:"
    echo "  $0 start                     # Запустить туннель"
    echo "  $0 start --local-port=15433  # Запустить на порту 15433"
    echo "  $0 status                    # Проверить статус"
    echo "  $0 config                    # Конфигурация для DataGrip"
    echo "  $0 test                      # Тест подключения"
    echo "  $0 stop                      # Остановить туннель"
    echo ""
    echo "Интеграция с Makefile:"
    echo "  make db-tunnel-start        # Запустить через Makefile"
    echo "  make db-tunnel-stop         # Остановить через Makefile"
    echo "  make db-tunnel-status       # Статус через Makefile"
    echo ""
}

# Основная функция
main() {
    # Читаем конфигурацию
    read_config

    # Парсим аргументы
    COMMAND="$1"
    shift

    # Парсим опции
    while [ $# -gt 0 ]; do
        case "$1" in
            --local-port=*)
                LOCAL_PORT="${1#*=}"
                shift
                ;;
            --key=*)
                SSH_KEY="${1#*=}"
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                shift
                ;;
        esac
    done

    # Выполняем команду
    case "$COMMAND" in
        start)
            check_ssh_key
            start_tunnel
            ;;
        stop)
            stop_tunnel
            ;;
        status)
            check_tunnel_status
            ;;
        config)
            show_config
            ;;
        test)
            test_connection
            ;;
        help|"")
            show_help
            ;;
        *)
            log_error "Неизвестная команда: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

# Запуск
main "$@"