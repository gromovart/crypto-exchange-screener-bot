# crypto-exchange-screener-bot
Бот для мониторинга изменения цен на биржах криптовалюты

┌─────────────────────────────────────────────┐
│             Основной цикл опроса            │
├─────────────────────────────────────────────┤
│ 1. Получить текущие цены всех пар (тикеры)  │
│    - Endpoint: /v5/market/tickers           │
│    - Период: каждые 10 секунд               │
├─────────────────────────────────────────────┤
│ 2. Для символов с изменениями > threshold:  │
│    - Получить свечные данные (K-line)       │
│    - Endpoint: /v5/market/kline             │
│    - Параметры: interval, limit=100         │
├─────────────────────────────────────────────┤
│ 3. Рассчитать процентное изменение:         │
│    Δ% = (Цена_текущая - Цена_старая) /      │
│           Цена_старая * 100%                │
├─────────────────────────────────────────────┤
│ 4. Если |Δ%| ≥ порог сигнала:              │
│    - Определить направление (pump/dump)     │
│    - Сгенерировать сигнал                   │
│    - Вывести в терминал                     │
└─────────────────────────────────────────────┘


Цепочка взаимодействия

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


Уровень 1: Data Acquisition (Получение данных)

PriceFetcher (интерфейс)
├── BybitPriceFetcher (реализация для Bybit)
└── HistoricalDataFetcher (для бэктестинга)

Уровень 2: Storage (Хранение)

PriceStorage (интерфейс)
├── InMemoryPriceStorage (оперативное хранение)
├── TimeSeriesStorage (хранилище временных рядов)
└── CacheLayer (кэширование частых запросов)



Уровень 3: Analysis Engine (Двигатель анализа)


AnalysisCoordinator (координатор анализа)
├── TrendAnalyzer (анализ трендов)
│   ├── SimpleTrendAnalyzer (базовый)
│   ├── ContinuousGrowthAnalyzer (непрерывный рост)
│   └── PatternAnalyzer (распознавание паттернов)
├── SignalDetector (детектор сигналов)
│   ├── GrowthSignalDetector (сигналы роста)
│   ├── FallSignalDetector (сигналы падения)
│   └── VolumeSignalDetector (сигналы по объему)
└── FilterChain (цепочка фильтров)
    ├── ConfidenceFilter (по уверенности)
    ├── VolumeFilter (по объему)
    ├── RateLimitFilter (по частоте)
    └── SymbolFilter (по символам)



Уровень 4: Notification System (Уведомления)


NotificationCoordinator
├── TelegramNotifier
├── ConsoleNotifier
├── WebhookNotifier
└── LogNotifier


Уровень 5: Orchestration (Оркестрация)

AppOrchestrator
├── Scheduler (планировщик задач)
├── HealthMonitor (мониторинг здоровья)
├── ConfigManager (управление конфигурацией)
└── MetricsCollector (сбор метрик)