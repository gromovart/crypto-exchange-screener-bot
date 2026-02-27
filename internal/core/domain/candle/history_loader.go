// internal/core/domain/candle/history_loader.go
package candle

import (
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/logger"
	"time"
)

const (
	// minCandlesRequired — порог: если свечей меньше, догружаем из Bybit
	minCandlesRequired = 10
	// historicalFetchLimit — сколько свечей запрашивать за один вызов GetKline
	historicalFetchLimit = 200
	// loadRateLimit — пауза между REST-запросами (Bybit public: 120 req/min)
	loadRateLimit = 120 * time.Millisecond
)

// periodToBybitInterval маппинг наших таймфреймов в интервалы Bybit API.
var periodToBybitInterval = map[string]string{
	"1m":  "1",
	"3m":  "3",
	"5m":  "5",
	"15m": "15",
	"30m": "30",
	"1h":  "60",
	"4h":  "240",
	"1d":  "D",
}

// periodDurations — длительность одного бара для корректного расчёта EndTime.
var periodDurations = map[string]time.Duration{
	"1m":  1 * time.Minute,
	"3m":  3 * time.Minute,
	"5m":  5 * time.Minute,
	"15m": 15 * time.Minute,
	"30m": 30 * time.Minute,
	"1h":  1 * time.Hour,
	"4h":  4 * time.Hour,
	"1d":  24 * time.Hour,
}

// HistoricalCandleLoader дозагружает исторические свечи из Bybit REST API
// при старте приложения. Работает в фоновой горутине и не блокирует запуск.
//
// Логика:
//  1. Для каждой пары (symbol, period) проверяет, сколько свечей уже в хранилище.
//  2. Если ≥ minCandlesRequired — пропускает (данные актуальны из предыдущего запуска).
//  3. Иначе — запрашивает historicalFetchLimit свечей через GET /v5/market/kline.
//  4. Сохраняет в Redis через CloseAndArchiveCandle.
type HistoricalCandleLoader struct {
	client  *bybit.BybitClient
	storage storage.CandleStorageInterface
}

// NewHistoricalCandleLoader создаёт загрузчик исторических свечей.
func NewHistoricalCandleLoader(
	client *bybit.BybitClient,
	candleStorage storage.CandleStorageInterface,
) *HistoricalCandleLoader {
	return &HistoricalCandleLoader{
		client:  client,
		storage: candleStorage,
	}
}

// Load запускает дозагрузку в фоновой горутине.
// symbols должны быть отсортированы по приоритету (топ по объёму — первыми):
// самые важные символы загрузятся до того, как SRZoneEngine запустит Warmup (~10 с после старта).
func (l *HistoricalCandleLoader) Load(symbols []string, periods []string) {
	go l.run(symbols, periods)
}

func (l *HistoricalCandleLoader) run(symbols []string, periods []string) {
	logger.Info("📥 HistoricalCandleLoader: запуск для %d символов × %d периодов", len(symbols), len(periods))

	total := len(symbols) * len(periods)
	loaded := 0
	skipped := 0

	for _, sym := range symbols {
		for _, period := range periods {
			interval, ok := periodToBybitInterval[period]
			if !ok {
				skipped++
				continue
			}

			// Проверяем — достаточно ли уже есть свечей
			existing, err := l.storage.GetHistory(sym, period, minCandlesRequired)
			if err == nil && len(existing) >= minCandlesRequired {
				skipped++
				continue
			}

			// Загружаем из Bybit
			klines, err := l.client.GetKline(sym, interval, historicalFetchLimit)
			if err != nil {
				logger.Debug("⚠️ HistoricalCandleLoader: %s/%s: %v", sym, period, err)
				time.Sleep(loadRateLimit)
				continue
			}

			if len(klines) == 0 {
				skipped++
				time.Sleep(loadRateLimit)
				continue
			}

			// Сохраняем каждую свечу
			dur := periodDurations[period]
			if dur == 0 {
				dur = time.Minute
			}

			stored := 0
			for _, kc := range klines {
				startTime := time.UnixMilli(kc.StartTime)
				endTime := startTime.Add(dur)

				c := &storage.Candle{
					Symbol:       sym,
					Period:       period,
					Open:         kc.Open,
					High:         kc.High,
					Low:          kc.Low,
					Close:        kc.Close,
					Volume:       kc.Volume,
					VolumeUSD:    kc.Turnover,
					StartTime:    startTime,
					EndTime:      endTime,
					IsClosedFlag: true,
					IsRealFlag:   true,
				}
				if err := l.storage.CloseAndArchiveCandle(c); err != nil {
					logger.Debug("⚠️ HistoricalCandleLoader: archive %s/%s: %v", sym, period, err)
				} else {
					stored++
				}
			}

			if stored > 0 {
				loaded++
				logger.Debug("📊 HistoricalCandleLoader: %s/%s — сохранено %d свечей", sym, period, stored)
			}

			time.Sleep(loadRateLimit)
		}
	}

	logger.Info("✅ HistoricalCandleLoader: завершено. Загружено пар: %d, пропущено: %d (итого: %d)",
		loaded, skipped, total)
}
