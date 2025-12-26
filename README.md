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

## 🏗️ Архитектура

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

### Уровень 1: Data Acquisition (Получение данных)
- **PriceFetcher** - интерфейс получения данных
- **BybitPriceFetcher** - реализация для Bybit API
- **HistoricalDataFetcher** - для бэктестинга

### Уровень 2: Storage (Хранение)
- **PriceStorage** - интерфейс хранения
- **InMemoryPriceStorage** - оперативное хранение в памяти
- **TimeSeriesStorage** - хранилище временных рядов

### Уровень 3: Analysis Engine (Двигатель анализа)
- **AnalysisCoordinator** - координатор анализа
- **TrendAnalyzer** - анализ трендов
  - SimpleTrendAnalyzer (базовый)
  - ContinuousGrowthAnalyzer (непрерывный рост)
  - PatternAnalyzer (распознавание паттернов)
- **SignalDetector** - детектор сигналов
  - GrowthSignalDetector (сигналы роста)
  - FallSignalDetector (сигналы падения)
  - VolumeSignalDetector (сигналы по объему)
- **FilterChain** - цепочка фильтров
  - ConfidenceFilter (по уверенности)
  - VolumeFilter (по объему)
  - RateLimitFilter (по частоте)
  - SymbolFilter (по символам)

### Уровень 4: Notification System (Уведомления)
- **NotificationCoordinator** - координатор уведомлений
- **TelegramNotifier** - уведомления в Telegram
- **ConsoleNotifier** - вывод в консоль
- **WebhookNotifier** - вебхук уведомления
- **LogNotifier** - логирование в файл

### Уровень 5: Orchestration (Оркестрация)
- **AppOrchestrator** - главный оркестратор
- **Scheduler** - планировщик задач
- **HealthMonitor** - мониторинг здоровья
- **ConfigManager** - управление конфигурацией
- **MetricsCollector** - сбор метрик

## 🚀 Быстрый старт

### 1. Предварительные требования
- Go 1.21 или выше
- API ключи от Bybit или Binance
- (Опционально) Telegram Bot Token для уведомлений

### 2. Установка и настройка

```bash
# Клонирование репозитория
git clone <repository-url>
cd crypto-exchange-screener-bot

# Настройка окружения
make setup
# Будет создан .env файл из .env.example

# Отредактируйте .env файл:
nano .env
# Добавьте ваши API ключи:
# BYBIT_API_KEY=your_api_key_here
# BYBIT_SECRET_KEY=your_secret_key_here
```

### 3. Запуск

#### Режим разработки (из исходников):
```bash
make run
```

#### Продакшен режим (сборка + запуск):
```bash
make build        # Сборка бинарника
make run-prod     # Запуск собранной версии
```

#### Установка в систему:
```bash
make install      # Установка в системные пути
growth-monitor --help  # Теперь доступна как системная команда
```

### 4. Тестирование системы

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

## ⚙️ Конфигурация

Основной конфигурационный файл `.env`:

```env
# Обязательные настройки
BYBIT_API_KEY=your_api_key
BYBIT_SECRET_KEY=your_secret_key

# Настройки анализа
SYMBOL_FILTER=BTC,ETH,BNB,SOL,XRP
MIN_VOLUME_FILTER=100000
GROWTH_ANALYZER_MIN_GROWTH=2.0
FALL_ANALYZER_MIN_FALL=2.0

# Telegram уведомления
TELEGRAM_ENABLED=true
TG_API_KEY=your_telegram_bot_token
TG_CHAT_ID=your_chat_id

# Логирование
LOG_LEVEL=info
LOG_FILE=logs/growth.log
```

Полный список всех настроек смотрите в `.env.example`.

## 🛠️ Доступные команды

### Основные команды
```bash
make help         # Показать все команды
make setup        # Настройка окружения
make build        # Сборка продакшен версии
make run          # Запуск в режиме разработки
make run-prod     # Запуск собранной версии
make install      # Установка в систему
make release      # Сборка для всех платформ
```

### Тестирование и отладка
```bash
make test         # Запуск unit тестов
make lint         # Проверка кода
make clean        # Очистка проекта
make deps         # Обновление зависимостей

make debug        # Базовая отладка
make debug-all    # Все тесты сразу
make analyzer-test # Тест анализаторов
```

### Docker
```bash
make docker-build # Сборка Docker образа
make docker-run   # Запуск в Docker
```

## 📊 Анализаторы

Бот включает несколько интеллектуальных анализаторов:

### GrowthAnalyzer
- Обнаружение роста цены
- Настраиваемый порог минимального роста (0.01% - 10%)
- Учет уверенности сигнала (10% - 100%)
- Анализ непрерывности тренда

### FallAnalyzer
- Обнаружение падения цены
- Настраиваемый порог минимального падения
- Локальное и максимальное падение
- Подтверждение объемами

### ContinuousAnalyzer
- Обнаружение непрерывных трендов
- Анализ последовательных движений
- Определение силы тренда

### VolumeAnalyzer
- Анализ объемов торгов
- Обнаружение аномальных объемов
- Подтверждение сигналов объемами

## 🔔 Уведомления

### Форматы уведомлений
- **📱 Telegram** - мгновенные уведомления в мессенджер
- **🖥️ Console** - цветной вывод в консоли
- **🌐 Webhook** - HTTP уведомления на ваш сервер
- **📝 Log file** - детальное логирование в файл

### Пример уведомления:
```
📈 РОСТ: BTCUSDT +2.5% за 15 минут
📊 Уверенность: 75%
💰 Объем: 1.2M USDT
⏰ Время: 14:30:15
```

## 🐳 Docker развертывание

### 1. Сборка образа
```bash
make docker-build
```

### 2. Запуск контейнера
```bash
docker run \
  --name crypto-monitor \
  --env-file .env \
  -v $(pwd)/logs:/logs \
  crypto-growth-monitor:latest
```

### 3. Docker Compose (рекомендуется)
```yaml
# docker-compose.yml
version: '3.8'

services:
  crypto-monitor:
    build: .
    container_name: crypto-monitor
    env_file:
      - .env
    volumes:
      - ./logs:/logs
    restart: unless-stopped
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## 📈 Мониторинг и метрики

Бот предоставляет метрики для мониторинга:

```bash
# Статус системы (каждые 5 минут)
[STATUS] Uptime: 2h15m, Symbols: 45, Signals: 12

# Метрики производительности
- Время обработки символа: < 50ms
- Память: < 100MB
- Частота обновления: каждые 10 секунд
```

## 🔧 Расширенная конфигурация

### Настройка символов
```env
# Мониторинг конкретных символов
SYMBOL_FILTER=BTC,ETH,BNB,SOL,XRP,ADA,DOGE

# Исключение символов
EXCLUDE_SYMBOLS=STGUSDT,ICPUSDT

# Все символы с фильтром по объему
SYMBOL_FILTER=all
MIN_VOLUME_FILTER=500000
```

### Настройка анализа
```env
# Периоды анализа (в минутах)
ANALYSIS_PERIODS=5,15,30,60

# Пороги для сигналов
GROWTH_ANALYZER_MIN_GROWTH=1.5
GROWTH_ANALYZER_MIN_CONFIDENCE=60
FALL_ANALYZER_MIN_FALL=1.5
FALL_ANALYZER_MIN_CONFIDENCE=60

# Фильтрация сигналов
SIGNAL_FILTERS_ENABLED=true
MAX_SIGNALS_PER_MIN=5
MIN_CONFIDENCE=50
```

## 🐛 Отладка и устранение неполадок

### Проверка подключения к API
```bash
make debug-diagnostic
```

### Тестирование анализаторов
```bash
make analyzer-test
```

### Включение детального логирования
```env
LOG_LEVEL=debug
EVENT_BUS_ENABLE_LOGGING=true
```

### Просмотр логов
```bash
tail -f logs/growth.log
```

## 📝 Логирование

Уровни логирования:
- `debug` - все события (для отладки)
- `info` - важные события и сигналы (рекомендуется)
- `warn` - только предупреждения и ошибки
- `error` - только ошибки

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для новой функции (`git checkout -b feature/amazing-feature`)
3. Зафиксируйте изменения (`git commit -m 'Add amazing feature'`)
4. Запушьте в ветку (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. См. файл `LICENSE` для подробностей.

## 🙏 Благодарности

- [Bybit API](https://bybit-exchange.github.io/docs/) за предоставление данных
- [Binance API](https://binance-docs.github.io/apidocs/) за альтернативный источник
- Сообществу Go за отличные библиотеки

## 📞 Поддержка

Если у вас есть вопросы или проблемы:
1. Проверьте раздел "Отладка и устранение неполадок"
2. Создайте Issue в GitHub
3. Проверьте примеры конфигурации в `.env.example`

---

**Happy trading!** 🚀

*Примечание: Криптовалютные рынки высоковолатильны. Используйте этот инструмент для анализа и принятия решений на свой страх и риск.*


internal/
├── core/                    # Основная бизнес-логика
│   ├── signals/
│   │   ├── detectors/       # growth_analyzer, fall_analyzer, etc
│   │   ├── filters/
│   │   └── pipeline/
│   ├── market/
│   │   ├── data_fetcher/
│   │   └── storage/
│   └── events/
│
├── delivery/                # Доставка (внешние интерфейсы)
│   ├── telegram/
│   ├── rest/                # если будет API
│   └── websocket/
│
└── infrastructure/
    ├── config/
    ├── logger/
    ├── persistence/
    └── cache/


pkg/
├── exchanges/               # Общие типы для бирж
├── signals/                 # Общие типы сигналов
├── utils/
│   ├── math/
│   ├── time/
│   └── validation/
└── logger/                  # Оставить здесь



configs/
├── dev/
│   ├── config.yaml
│   └── docker-compose.yml
├── prod/
│   ├── config.yaml
│   └── docker-compose.yml
└── local/
    └── config.yaml

deploy/
├── docker/
│   ├── Dockerfile.bot
│   ├── Dockerfile.debug
│   └── docker-compose.full.yml
├── k8s/                    # если используете Kubernetes
└── scripts/
    ├── deploy.sh
    └── monitoring/



internal/
├── api/
│   ├── exchanges/
│   │   ├── binance/
│   │   ├── bybit/
│   │   └── interface.go
│   └── market_data/
│       ├── client.go
│       └── aggregator.go


cmd/
├── main/
│   ├── main.go (основной бот)
│   └── main_test.go
├── debug/
│   ├── analyzer/
│   ├── counter_test/
│   └── ...
└── tools/
    ├── migration/
    ├── seed/
    └── diagnostics/


docs/
├── api/
│   ├── endpoints.md
│   └── webhooks.md
├── architecture/
│   ├── diagrams/
│   └── decisions/
├── guides/
│   ├── development.md
│   ├── deployment.md
│   └── testing.md
└── signals/
    ├── algorithms.md
    └── filters.md



internal/
├── ports/           # Интерфейсы
│   ├── primary/    # Для входящих запросов
│   └── secondary/  # Для внешних зависимостей
├── adapters/       # Реализации портов
└── core/           # Бизнес-логика




