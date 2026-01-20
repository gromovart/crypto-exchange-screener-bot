Вы правы! Я слишком сильно сократил файл. Давайте восстановим важные разделы и сделаем README.md более полным, объединив всю необходимую информацию:

```markdown
# 📈 Crypto Exchange Screener Bot

Бот для мониторинга изменения цен на биржах криптовалюты с обнаружением сигналов роста и падения в реальном времени.

## ✨ Особенности

- **📊 Мониторинг в реальном времени** - получение данных с Bybit/Binance API
- **🔍 Умный анализ** - несколько алгоритмов анализа трендов
- **⚡ Быстрая обработка** - параллельная обработка множества символов
- **🔔 Многоуровневые уведомления** - Telegram, консоль, вебхуки
- **🎯 Настраиваемые фильтры** - по объему, уверенности, частоте сигналов
- **📈 Визуализация** - цветной вывод в консоли
- **🐳 Docker поддержка** - готовые контейнеры для продакшена
- **🏗️ Чистая архитектура** - модульная структура по принципам Hexagonal Architecture
- **🔐 Авторизация пользователей** - JWT токены, сессии, подписки
- **🚀 Автоматическое развертывание** - полный набор скриптов для сервера

## 🏗️ Архитектура системы

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   PriceFetcher  │────▶│   PriceStorage  │────▶│  AnalysisEngine │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                            │                         │
                            ▼                         ▼
                    ┌─────────────────┐     ┌─────────────────┐
                    │   EventBus      │◀────│  SignalPipeline │
                    └─────────────────┘     └─────────────────┘
                            │                         │
                            ▼                         ▼
                    ┌─────────────────┐     ┌─────────────────┐
                    │  Subscribers    │     │  Notifications  │
                    │  (Telegram,     │     │  Coordinator    │
                    │   Console, etc) │     └─────────────────┘
                    └─────────────────┘
```

## 📁 Структура проекта

```
📦 crypto-exchange-screener-bot
├── 📂 application/                  # Слой оркестрации приложения
│   ├── 📂 bootstrap/               # Инициализация приложения
│   ├── 📂 cmd/                     # Точки входа
│   │   └── 📂 bot/                 # Основной бот
│   │       └── main.go
│   ├── 📂 layer_manager/           # Управление слоями
│   │   ├── layers/                 # Реализации слоев
│   │   │   ├── core.go             # Бизнес-логика
│   │   │   ├── delivery.go         # Доставка (Telegram)
│   │   │   ├── infrastructure.go   # Инфраструктура
│   │   │   ├── layer.go            # Базовый слой
│   │   │   └── registry.go         # Реестр слоев
│   │   └── manager.go              # Менеджер слоев
│   └── 📂 pipeline/                # Конвейеры обработки
│
├── 📂 internal/                     # Внутренние пакеты
│   ├── 📂 adapters/                # Адаптеры для внешних интерфейсов
│   │   ├── 📂 notification/        # Адаптеры уведомлений
│   │   └── signal_adapters.go      # Адаптеры сигналов
│   │
│   ├── 📂 core/                    # Ядро системы (чистая бизнес-логика)
│   │   ├── 📂 domain/              # Доменные сущности и сервисы
│   │   │   ├── 📂 auth/            # Аутентификация и авторизация
│   │   │   ├── 📂 candle/          # Работа со свечами
│   │   │   ├── 📂 fetchers/        # Получение данных с бирж
│   │   │   ├── 📂 signals/         # Обнаружение и фильтрация сигналов
│   │   │   │   ├── 📂 detectors/   # Анализаторы сигналов
│   │   │   │   │   ├── 📂 continuous_analyzer/    # Анализ непрерывных трендов
│   │   │   │   │   ├── 📂 counter/                 # Счетчик и подтверждение сигналов
│   │   │   │   │   ├── 📂 fall_analyzer/          # Анализатор падения
│   │   │   │   │   ├── 📂 growth_analyzer/        # Анализатор роста
│   │   │   │   │   ├── 📂 open_interest_analyzer/ # Анализ открытого интереса
│   │   │   │   │   └── 📂 volume_analyzer/        # Анализ объемов
│   │   │   │   ├── 📂 engine/      # Движок анализа
│   │   │   │   ├── 📂 filters/     # Фильтры сигналов
│   │   │   │   └── types.go
│   │   │   ├── 📂 subscription/    # Управление подписками
│   │   │   └── 📂 users/           # Управление пользователями
│   │   ├── 📂 errors/              # Доменные ошибки
│   │   └── 📂 package/
│   │
│   ├── 📂 delivery/                # Слой доставки (внешние интерфейсы)
│   │   └── 📂 telegram/            # Telegram бот
│   │       ├── 📂 app/             # Приложение Telegram бота
│   │       │   ├── 📂 bot/         # Основная логика бота
│   │       │   │   ├── 📂 buttons/          # Генерация кнопок
│   │       │   │   ├── 📂 constants/        # Константы
│   │       │   │   ├── 📂 formatters/       # Форматирование сообщений
│   │       │   │   ├── 📂 handlers/         # Обработчики команд и callback-ов
│   │       │   │   │   ├── 📂 callbacks/    # Обработчики callback-кнопок
│   │       │   │   │   ├── 📂 commands/     # Обработчики команд
│   │       │   │   │   └── 📂 router/       # Маршрутизация
│   │       │   │   ├── message_sender/      # Отправка сообщений
│   │       │   │   └── middlewares/         # Middleware (например, авторизация)
│   │       │   ├── 📂 http_client/ # HTTP клиент для Telegram API
│   │       │   └── singleton.go
│   │       ├── 📂 components/      # Компоненты
│   │       ├── 📂 controllers/     # Контроллеры
│   │       ├── 📂 services/        # Сервисы
│   │       └── types.go
│   │
│   ├── 📂 infrastructure/          # Инфраструктурный слой
│   │   ├── 📂 api/                 # API клиенты
│   │   │   └── 📂 exchanges/       # Клиенты бирж (Bybit, Binance)
│   │   ├── 📂 cache/               # Кеширование (Redis)
│   │   ├── 📂 config/              # Конфигурация
│   │   ├── 📂 persistence/         # Хранение данных
│   │   │   ├── 📂 in_memory_storage/ # In-memory хранилище
│   │   │   └── 📂 postgres/        # PostgreSQL хранилище
│   │   │       ├── 📂 database/    # Подключение к БД
│   │   │       ├── 📂 migrations/  # Миграции базы данных
│   │   │       ├── 📂 models/      # Модели данных
│   │   │       └── 📂 repository/  # Репозитории
│   │   └── 📂 transport/           # Транспортный слой
│   │       └── 📂 event_bus/       # Шина событий
│   │
│   ├── 📂 types/                   # Общие типы данных
│   └── 📂 utils/                   # Вспомогательные утилиты
│
├── 📂 pkg/                         # Переиспользуемые пакеты
│   ├── 📂 logger/                  # Логирование
│   └── 📂 utils/                   # Общие утилиты
│
├── 📂 bin/                         # Скомпилированные бинарники
├── 📂 configs/                     # Конфигурационные файлы
│   ├── 📂 dev/                     # Для разработки
│   └── 📂 prod/                    # Для продакшена
│
├── 📂 deploy/                      # Развертывание
│   ├── 📂 scripts/                 # Скрипты развертывания и управления
│   │   ├── deploy.sh               # Основной скрипт развертывания
│   │   ├── service.sh              # Управление службой
│   │   ├── update.sh               # Обновление приложения
│   │   ├── check-connection.sh     # Проверка подключения
│   │   └── README.md               # Документация скриптов
│   ├── 📂 docker/                  # Docker конфигурации
│   │   ├── docker-compose.db.yml   # Композ для базы данных
│   │   ├── dockerfile              # Dockerfile для бота
│   │   └── persistence/            # Персистентное хранилище
│   │       └── postgres/
│   │           └── migrations/     # Миграции БД
│   └── 📂 systemd/                 # Systemd сервисы
│       └── crypto-screener-bot.service
│
├── 📂 tests/                       # Тесты
│   ├── 📂 e2e/                     # End-to-end тесты
│   ├── 📂 fixtures/                # Фикстуры для тестов
│   └── 📂 integration/             # Интеграционные тесты
│       ├── 📂 api/                 # Тесты API
│       └── 📂 telegram/            # Тесты Telegram бота
│
├── 📂 logs/                        # Логи приложения
├── Makefile                        # Управление проектом
├── go.mod                          # Зависимости Go
├── go.sum                          # Хэши зависимостей
└── README.md                       # Документация
```

## 🚀 Быстрый старт

### 1. Предварительные требования
- **Go 1.21** или выше
- **PostgreSQL 14+** (рекомендуется для продакшена)
- **Redis** (рекомендуется для кеширования)
- **API ключи** от Bybit или Binance
- **Telegram Bot Token** для уведомлений

### 2. Установка и настройка

```bash
# Клонирование репозитория
git clone <repository-url>
cd crypto-exchange-screener-bot

# Установка зависимостей
go mod download

# Настройка окружения
cp .env.example .env
nano .env  # Отредактируйте с вашими настройками
```

### 3. Базовый запуск

#### Режим разработки:
```bash
make run
# или
go run application/cmd/bot/main.go
```

#### Продакшен режим:
```bash
make build        # Сборка бинарника
./bin/growth-monitor --config=configs/prod/config.yaml
```

## 🚀 Автоматическое развертывание на сервер

Полный набор скриптов для развертывания на сервере Ubuntu доступен в `deploy/scripts/`.

### Быстрое развертывание:

```bash
# Сделайте скрипты исполняемыми
chmod +x deploy/scripts/*.sh

# Развертывание на сервер
./deploy/scripts/deploy.sh --ip=95.142.40.244 --user=root

# Управление службой после установки
./deploy/scripts/service.sh status --ip=95.142.40.244
./deploy/scripts/service.sh logs --ip=95.142.40.244
```

### Доступные скрипты развертывания:

| Скрипт | Назначение |
|--------|------------|
| `deploy.sh` | Основной скрипт развертывания |
| `service.sh` | Управление службой и мониторинг |
| `update.sh` | Обновление приложения |
| `check-connection.sh` | Проверка подключения к серверу |

Подробную документацию по скриптам смотрите в [deploy/scripts/README.md](deploy/scripts/README.md).

## 🐳 Docker развертывание

### Полная установка с Docker Compose:

```bash
# Использование Docker Compose
docker-compose -f deploy/docker/docker-compose.yml up -d

# Только база данных и Redis
docker-compose -f deploy/docker/docker-compose.db.yml up -d
```

## ⚙️ Конфигурация (.env)

### Основные настройки:

```env
# ========== Telegram ==========
TELEGRAM_ENABLED=true
TELEGRAM_BOT_TOKEN=ваш_токен_бота
TELEGRAM_ADMIN_IDS=ваш_telegram_id
TELEGRAM_WEBHOOK_ENABLED=false
TELEGRAM_WEBHOOK_PORT=8443

# ========== Биржа ==========
EXCHANGE=bybit  # или binance
BYBIT_API_KEY=ваш_api_ключ
BYBIT_API_SECRET=ваш_api_секрет
# ИЛИ
BINANCE_API_KEY=ваш_api_ключ
BINANCE_API_SECRET=ваш_api_секрет

# ========== База данных ==========
DB_ENABLED=true
DB_HOST=localhost
DB_PORT=5432
DB_NAME=crypto_screener_db
DB_USER=crypto_screener
DB_PASSWORD=сложный_пароль
DB_ENABLE_AUTO_MIGRATE=true

# ========== Redis ==========
REDIS_ENABLED=true
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# ========== Настройки анализа ==========
SYMBOL_FILTER=BTCUSDT,ETHUSDT,BNBUSDT,SOLUSDT
MIN_VOLUME_FILTER=100000
GROWTH_ANALYZER_MIN_GROWTH=2.0
GROWTH_ANALYZER_MIN_CONFIDENCE=60
FALL_ANALYZER_MIN_FALL=2.0
FALL_ANALYZER_MIN_CONFIDENCE=60

# ========== Безопасность ==========
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)

# ========== Логирование ==========
LOG_LEVEL=info
LOG_FILE=logs/growth_monitor.log
LOG_MAX_SIZE=100  # MB
LOG_MAX_BACKUPS=10
LOG_MAX_AGE=30    # дней
```

## 📱 Telegram бот

### Структура меню

```
🏠 ГЛАВНОЕ МЕНЮ (Reply Keyboard)
├── ⚙️ Настройки → callback: settings_main
├── 📊 Статус → callback: status
├── 🔔 Уведомления → callback: notifications_menu
├── 📈 Сигналы → callback: signals_menu
├── ⏱️ Периоды → callback: periods_menu
├── 🔄 Сбросить → callback: reset_menu
└── 📋 Помощь → command: /help

⚙️ МЕНЮ НАСТРОЕК (адаптивное)
├── 👤 ДЛЯ НЕАВТОРИЗОВАННЫХ ПОЛЬЗОВАТЕЛЕЙ:
│   ├── 🔑 Войти / Авторизация → callback: auth_login
│   └── 🔙 Назад → callback: menu_main
├── 👤 ДЛЯ АВТОРИЗОВАННЫХ ПОЛЬЗОВАТЕЛЕЙ:
│   ├── 👤 Мой профиль → callback: profile_main
│   ├── 🔔 Уведомления → callback: notifications_menu
│   ├── 📊 Пороги сигналов → callback: thresholds_menu
│   ├── ⏱️ Периоды → callback: periods_menu
│   ├── ⚙️ Сбросить настройки → callback: reset_settings
│   └── 🔙 Назад → callback: menu_main
├── 🔔 МЕНЮ УВЕДОМЛЕНИЙ (одинаковое для всех)
│   ├── ✅ Включить → callback: notify_toggle
│   ├── ❌ Выключить → callback: notify_toggle
│   ├── 📈 Только рост → callback: notify_growth_only
│   ├── 📉 Только падение → callback: notify_fall_only
│   ├── 📊 Все сигналы → callback: notify_both
│   └── 🔙 Назад → callback: settings_main
├── 📊 МЕНЮ ПОРОГОВ (только для авторизованных)
│   ├── 📈 Мин. рост: 1.5% → callback: threshold_growth
│   ├── 📉 Мин. падение: 1.5% → callback: threshold_fall
│   ├── 🕐 Тихие часы: 22-08 → callback: quiet_hours
│   └── 🔙 Назад → callback: settings_main
└── ⏱️ МЕНЮ ПЕРИОДОВ (адаптивное)
    ├── [БАЗОВЫЙ] Текущий: 15мин → callback: period_select
    └── [РАСШИРЕННЫЙ] Предпочтительные: 5m,15m,1h → callback: period_manage
```

### Основные команды:
```
/start - Начало работы с ботом
/help - Помощь и документация
/settings - Настройки приложения
/profile - Профиль пользователя
/signals - Управление сигналами
/notifications - Настройки уведомлений
/periods - Управление периодами анализа
/status - Статус системы
```

## 🔍 Анализаторы сигналов

### 1. GrowthAnalyzer
- **Обнаружение роста цены**
- Настраиваемый порог минимального роста (0.01% - 10%)
- Учет уверенности сигнала (10% - 100%)
- Анализ непрерывности тренда
- Подтверждение объемами

### 2. FallAnalyzer
- **Обнаружение падения цены**
- Настраиваемый порог минимального падения
- Локальное и максимальное падение
- Анализ объема при падении
- Фильтрация ложных сигналов

### 3. ContinuousAnalyzer
- **Обнаружение непрерывных трендов**
- Анализ последовательных движений
- Определение силы тренда
- Оценка устойчивости тренда
- Прогнозирование продолжения

### 4. VolumeAnalyzer
- **Анализ объемов торгов**
- Обнаружение аномальных объемов
- Подтверждение сигналов объемами
- Анализ объемной дивергенции
- Идентификация "больших" игроков

### 5. Counter Analyzer
- **Подтверждение сигналов через счетчики**
- Управление состояниями анализа
- Кеширование промежуточных результатов
- Снижение ложных срабатываний
- Многоуровневое подтверждение

### 6. OpenInterestAnalyzer
- **Анализ открытого интереса**
- Обнаружение дивергенций
- Прогнозирование разворотов тренда
- Анализ позиций трейдеров
- Предсказание волатильности

## 🔧 Архитектурные принципы

### Слоистая архитектура:
1. **Application Layer** - Оркестрация и точки входа
2. **Delivery Layer** - Внешние интерфейсы (Telegram API)
3. **Core Layer** - Чистая бизнес-логика
4. **Infrastructure Layer** - Внешние зависимости (БД, API бирж)

### Dependency Rule:
- Слои зависят только от слоев ниже
- Core слой не зависит от инфраструктуры
- Использование интерфейсов для инверсии зависимостей

### Компоненты:

#### Уровень 1: Data Acquisition (Получение данных)
- **PriceFetcher** - интерфейс получения данных
- **BybitPriceFetcher** - реализация для Bybit API
- **BinancePriceFetcher** - реализация для Binance API

#### Уровень 2: Storage (Хранение)
- **PriceStorage** - интерфейс хранения
- **InMemoryPriceStorage** - оперативное хранение в памяти
- **PostgreSQLStorage** - персистентное хранение в БД

#### Уровень 3: Analysis Engine (Двигатель анализа)
- **AnalysisCoordinator** - координатор анализа
- **TrendAnalyzer** - анализ трендов
- **SignalDetector** - детектор сигналов
- **FilterChain** - цепочка фильтров

#### Уровень 4: Notification System (Уведомления)
- **NotificationCoordinator** - координатор уведомлений
- **TelegramNotifier** - уведомления в Telegram
- **ConsoleNotifier** - вывод в консоль
- **WebhookNotifier** - вебхук уведомления

#### Уровень 5: Orchestration (Оркестрация)
- **AppOrchestrator** - главный оркестратор
- **Scheduler** - планировщик задач
- **HealthMonitor** - мониторинг здоровья
- **ConfigManager** - управление конфигурацией

## 🛠️ Доступные команды

### Основные команды Makefile:
```bash
make help         # Показать все команды
make setup        # Настройка окружения
make build        # Сборка продакшен версии
make run          # Запуск в режиме разработки
make run-prod     # Запуск собранной версии
make test         # Запуск unit тестов
make lint         # Проверка кода
make clean        # Очистка проекта
make deps         # Обновление зависимостей
make release      # Сборка для всех платформ
make install      # Установка в систему
```

### Команды развертывания:
```bash
# Развертывание на сервер
./deploy/scripts/deploy.sh --ip=95.142.40.244 --user=root

# Управление службой
./deploy/scripts/service.sh start     # Запустить
./deploy/scripts/service.sh stop      # Остановить
./deploy/scripts/service.sh restart   # Перезапустить
./deploy/scripts/service.sh status    # Статус
./deploy/scripts/service.sh logs      # Логи (50 строк)
./deploy/scripts/service.sh logs 100  # Логи (100 строк)
./deploy/scripts/service.sh logs-follow  # Логи в реальном времени
./deploy/scripts/service.sh logs-error   # Только ошибки
./deploy/scripts/service.sh monitor   # Мониторинг системы
./deploy/scripts/service.sh health    # Проверка здоровья
./deploy/scripts/service.sh config-show    # Показать конфигурацию
./deploy/scripts/service.sh config-check   # Проверить конфигурацию
./deploy/scripts/service.sh backup    # Резервная копия
./deploy/scripts/service.sh cleanup   # Очистка логов и копий

# Обновление приложения
./deploy/scripts/update.sh --ip=95.142.40.244 --user=root
./deploy/scripts/update.sh --backup-only --ip=95.142.40.244
./deploy/scripts/update.sh --rollback --ip=95.142.40.244

# Проверка подключения
./deploy/scripts/check-connection.sh --ip=95.142.40.244 --user=root
./deploy/scripts/check-connection.sh --full --ip=95.142.40.244
./deploy/scripts/check-connection.sh --generate-key --ip=95.142.40.244
```

### Тестирование и отладка:
```bash
# Полный тест всех компонентов
make debug-all

# Только тест анализаторов
make analyzer-test

# Глубокая диагностика системы
make debug-diagnostic

# Расширенная отладка
make debug-enhanced

# Супер-чувствительный тест
make debug-super-sensitive
```

## 📊 Мониторинг и метрики

### Команды мониторинга:
```bash
# Комплексный мониторинг
./deploy/scripts/service.sh monitor --ip=95.142.40.244

# Проверка здоровья
./deploy/scripts/service.sh health --ip=95.142.40.244

# Быстрый статус
./deploy/scripts/service.sh status --ip=95.142.40.244
```

### Прямой доступ к логам:
```bash
# Логи приложения на сервере
journalctl -u crypto-screener.service -f
tail -f /var/log/crypto-screener-bot/app.log

# Логи базы данных
sudo tail -f /var/log/postgresql/postgresql-14-main.log

# Логи Redis
sudo tail -f /var/log/redis/redis-server.log
```

### Системные метрики:
```bash
# Процессы и ресурсы
htop
nmon

# Дисковое пространство
df -h

# Память
free -h

# Сеть
iftop
nload
```

## 🗄️ Резервное копирование и восстановление

### Создание резервной копии:
```bash
./deploy/scripts/service.sh backup --ip=95.142.40.244
```

Содержимое резервной копии:
- Конфигурация приложения
- Исходный код
- Дамп базы данных PostgreSQL
- Бинарные файлы
- Логи (опционально)

### Восстановление из резервной копии:
```bash
# Через скрипт
./deploy/scripts/update.sh --rollback --ip=95.142.40.244

# Или вручную на сервере
cd /opt/crypto-screener-bot_backups/
tar -xzf backup_20240101_120000.tar.gz
```

### Автоматическая очистка:
```bash
./deploy/scripts/service.sh cleanup --ip=95.142.40.244
```

Удаляет:
- Логи старше 30 дней
- Резервные копии кроме последних 10
- Кэш сборки Go
- Старые журналы systemd

## 🔍 Диагностика проблем

### Ошибка SSH подключения:
```bash
# Проверьте подключение
./deploy/scripts/check-connection.sh --ip=95.142.40.244 --user=root

# Создайте новый SSH ключ
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa_crypto -N ""

# Скопируйте на сервер
ssh-copy-id -i ~/.ssh/id_rsa_crypto.pub root@95.142.40.244

# Используйте новый ключ
./deploy/scripts/deploy.sh --key=~/.ssh/id_rsa_crypto
```

### База данных не подключена:
```bash
# Проверьте PostgreSQL
ssh root@95.142.40.244 "systemctl status postgresql"

# Проверьте подключение к БД
ssh root@95.142.40.244 "sudo -u postgres psql -d crypto_screener_db -c '\conninfo'"

# Проверьте миграции
ssh root@95.142.40.244 "cd /opt/crypto-screener-bot && go run application/cmd/bot/main.go --migrate"
```

### Приложение не запускается:
```bash
# Проверьте логи
./deploy/scripts/service.sh logs 50 --ip=95.142.40.244
./deploy/scripts/service.sh logs-error --ip=95.142.40.244

# Проверьте конфигурацию
./deploy/scripts/service.sh config-check --ip=95.142.40.244

# Проверьте зависимости
ssh root@95.142.40.244 "cd /opt/crypto-screener-bot && go mod verify"
```

### Telegram бот не работает:
1. Убедитесь, что `TELEGRAM_ENABLED=true`
2. Проверьте токен бота в BotFather
3. Убедитесь, что Telegram ID правильный
4. Проверьте, что бот добавлен в чат/канал
5. Проверьте наличие интернета на сервере

## 🧪 Тестирование

```bash
# Запуск всех тестов
make test
go test ./...

# Тесты определенного пакета
go test ./internal/core/domain/signals/...
go test ./internal/delivery/telegram/...

# Интеграционные тесты
go test ./tests/integration/...

# E2E тесты
go test ./tests/e2e/...

# Запуск с покрытием
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Бенчмарки
go test -bench=. -benchmem ./...

# Race детектор
go test -race ./...
```

## 🔐 Безопасность

### Рекомендации по безопасности:

1. **Смена паролей по умолчанию** после установки:
   ```bash
   # PostgreSQL
   ssh root@95.142.40.244 "sudo -u postgres psql -c \"ALTER USER crypto_screener WITH PASSWORD 'НовыйСложныйПароль123!';\""
   ```

2. **Настройка брандмауэра**:
   ```bash
   ssh root@95.142.40.244 << 'EOF'
   ufw default deny incoming
   ufw default allow outgoing
   ufw allow from YOUR_IP to any port 22
   ufw allow from 0.0.0.0/0 to any port 5432 comment 'PostgreSQL'
   ufw allow from 0.0.0.0/0 to any port 6379 comment 'Redis'
   ufw --force enable
   EOF
   ```

3. **Регулярные обновления**:
   ```bash
   ssh root@95.142.40.244 "apt update && apt upgrade -y && apt autoremove -y"
   ```

4. **SSL/TLS для соединений**:
   - Настройка SSL для PostgreSQL
   - Использование TLS для Redis
   - HTTPS для вебхуков

5. **Ротация ключей**:
   - Регулярная смена API ключей бирж
   - Ротация JWT секретов
   - Обновление Telegram токенов

## 📈 Производительность

### Рекомендуемые настройки сервера:
- **CPU**: 2-4 ядра (рекомендуется 4)
- **RAM**: 1-2 GB (рекомендуется 4GB для больших объемов данных)
- **Storage**: SSD, 20GB+ свободного места
- **Network**: Низкая задержка до бирж (< 100ms)

### Оптимизация настроек:

1. **PostgreSQL**:
   ```sql
   ALTER SYSTEM SET shared_buffers = '256MB';
   ALTER SYSTEM SET effective_cache_size = '768MB';
   ALTER SYSTEM SET work_mem = '16MB';
   ALTER SYSTEM SET maintenance_work_mem = '64MB';
   ```

2. **Redis**:
   ```bash
   # В /etc/redis/redis.conf
   maxmemory 512mb
   maxmemory-policy allkeys-lru
   save 900 1
   save 300 10
   save 60 10000
   ```

3. **Systemd лимиты**:
   ```ini
   # В /etc/systemd/system/crypto-screener.service
   [Service]
   LimitNOFILE=65536
   LimitNPROC=65536
   LimitCORE=infinity
   ```

### Метрики производительности:
- Время обработки символа: < 100ms
- Память на символ: ~1MB
- Максимальное количество символов: 100+
- Частота обновления: 10-60 секунд
- Время отклика API: < 500ms

## 📝 Чеклист развертывания

### Перед началом:
- [ ] IP сервера доступен и пингуется
- [ ] SSH ключ сгенерирован и скопирован на сервер
- [ ] Конфиг `configs/prod/.env` проверен и настроен
- [ ] Репозиторий обновлен до последней версии
- [ ] Бот создан в BotFather (токен получен)
- [ ] API ключи бирж созданы и проверены
- [ ] Домен привязан к IP (если нужен вебхук)

### После развертывания:
- [ ] Сервис запущен (`systemctl status crypto-screener`)
- [ ] Логи без критических ошибок (`journalctl -u crypto-screener`)
- [ ] База данных доступна и миграции применены
- [ ] Redis подключен и отвечает
- [ ] Конфигурация корректно загружена
- [ ] Telegram бот работает и отвечает на команды
- [ ] API бирж доступны и данные получаются
- [ ] Сигналы обнаруживаются и обрабатываются
- [ ] Уведомления отправляются в Telegram

### Регулярное обслуживание:
- [ ] Проверять логи на ошибки (ежедневно)
- [ ] Мониторить использование ресурсов (ежедневно)
- [ ] Создавать резервные копии (еженедельно)
- [ ] Обновлять приложение (по мере выхода обновлений)
- [ ] Обновлять систему безопасности (ежемесячно)
- [ ] Проверять доступность API бирж (ежедневно)
- [ ] Валидировать сигналы и точность анализа (еженедельно)
- [ ] Оптимизировать производительность (ежеквартально)

## 🤝 Вклад в проект

1. **Форкните репозиторий** на GitHub
2. **Создайте ветку** для новой функции:
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Зафиксируйте изменения**:
   ```bash
   git commit -m 'Add amazing feature'
   ```
4. **Запушьте в ветку**:
   ```bash
   git push origin feature/amazing-feature
   ```
5. **Откройте Pull Request**

### Руководство по стилю кода:
- Используйте `gofmt` для форматирования
- Следуйте рекомендациям Effective Go
- Пишите комментарии для экспортируемых функций
- Добавляйте тесты для новой функциональности
- Обновляйте документацию при изменениях

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. См. файл `LICENSE` для подробностей.

## 🙏 Благодарности

- [Bybit API](https://bybit-exchange.github.io/docs/) за предоставление данных
- [Binance API](https://binance-docs.github.io/apidocs/) за альтернативный источник
- Сообществу Go за отличные библиотеки и инструменты
- Разработчикам PostgreSQL и Redis за надежные базы данных
- Команде Telegram за прекрасную платформу для ботов

## 📞 Поддержка

Если у вас есть вопросы или проблемы:

1. **Сначала проверьте логи**:
   ```bash
   ./deploy/scripts/service.sh logs 100 --ip=95.142.40.244
   ```

2. **Проверьте конфигурацию**:
   ```bash
   ./deploy/scripts/service.sh config-check --ip=95.142.40.244
   ```

3. **Проверьте здоровье системы**:
   ```bash
   ./deploy/scripts/service.sh health --ip=95.142.40.244
   ```

4. **Используйте мониторинг**:
   ```bash
   ./deploy/scripts/service.sh monitor --ip=95.142.40.244
   ```

5. **Проверьте документацию**:
   - [deploy/scripts/README.md](deploy/scripts/README.md) - документация скриптов
   - Примеры конфигурации в `.env.example`
   - Комментарии в коде

6. **Создайте Issue в GitHub**:
   - Укажите версию приложения
   - Приложите соответствующие логи
   - Опишите шаги для воспроизведения
   - Укажите вашу конфигурацию (без секретов)

---

**Happy trading!** 🚀

*Примечание: Криптовалютные рынки высоковолатильны. Используйте этот инструмент для анализа и принятия решений на свой страх и риск. Авторы не несут ответственности за финансовые потери.*
```