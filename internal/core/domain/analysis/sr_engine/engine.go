// internal/core/domain/analysis/sr_engine/engine.go
package sr_engine

import (
	sr_zones "crypto-exchange-screener-bot/internal/core/domain/analysis/sr_zones"
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	sr_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/sr_storage"
	candleStorage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	event_bus "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"sync"
	"time"
)

// OrderBookFetcher — интерфейс для получения стакана из Bybit.
type OrderBookFetcher interface {
	GetOrderBook(symbol string, depth int) (*bybit.OrderBookV5, error)
}

// CandleHistoryProvider — интерфейс для получения истории свечей.
type CandleHistoryProvider interface {
	GetHistory(symbol, period string, limit int) ([]candleStorage.CandleInterface, error)
}

// Volume24hProvider — интерфейс для получения суточного объёма торгов в USD.
// Используется при расчёте динамического порога стены ордеров.
type Volume24hProvider interface {
	GetVolume24hUSD(symbol string) float64
}

// Engine — движок расчёта S/R зон.
// Подписывается на EventCandleClosed и пересчитывает зоны при каждом закрытии свечи.
type Engine struct {
	candleStorage  CandleHistoryProvider
	srStorage      *sr_storage.SRZoneStorage
	orderBook      OrderBookFetcher
	volume24h      Volume24hProvider
	eventBus       *event_bus.EventBus
	calculator     *sr_zones.Calculator

	// Кэш стакана: symbol → (book, expiry)
	obCacheMu sync.RWMutex
	obCache   map[string]obCacheEntry

	subscriber types.EventSubscriber
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

type obCacheEntry struct {
	book   *bybit.OrderBookV5
	expiry time.Time
}

const (
	candleHistoryDepth = 200
	orderBookDepth     = 200
	obCacheTTL         = 60 * time.Second
)

// NewEngine создаёт новый движок S/R зон.
func NewEngine(
	candleStorage CandleHistoryProvider,
	srStorage *sr_storage.SRZoneStorage,
	orderBook OrderBookFetcher,
	volume24h Volume24hProvider,
	eventBus *event_bus.EventBus,
) *Engine {
	return &Engine{
		candleStorage: candleStorage,
		srStorage:     srStorage,
		orderBook:     orderBook,
		volume24h:     volume24h,
		eventBus:      eventBus,
		calculator:    sr_zones.NewCalculator(),
		obCache:       make(map[string]obCacheEntry),
		stopCh:        make(chan struct{}),
	}
}

// Start запускает движок — подписывается на EventCandleClosed.
func (e *Engine) Start() {
	e.subscriber = event_bus.NewBaseSubscriber(
		"sr_zone_engine",
		[]types.EventType{types.EventCandleClosed},
		func(event types.Event) error {
			data, ok := event.Data.(types.CandleClosedData)
			if !ok {
				return nil
			}
			// Обрабатываем в горутине чтобы не блокировать EventBus
			e.wg.Add(1)
			go func() {
				defer e.wg.Done()
				e.recalculate(data.Symbol, data.Period)
			}()
			return nil
		},
	)

	e.eventBus.Subscribe(types.EventCandleClosed, e.subscriber)
	logger.Info("✅ SRZoneEngine запущен, подписан на EventCandleClosed")
}

// Stop останавливает движок.
func (e *Engine) Stop() {
	if e.subscriber != nil {
		e.eventBus.Unsubscribe(types.EventCandleClosed, e.subscriber)
	}
	close(e.stopCh)
	e.wg.Wait()
	logger.Info("🛑 SRZoneEngine остановлен")
}

// recalculate пересчитывает зоны для пары symbol/period.
func (e *Engine) recalculate(symbol, period string) {
	candles, err := e.candleStorage.GetHistory(symbol, period, candleHistoryDepth)
	if err != nil || len(candles) < 10 {
		return
	}

	zones := e.calculator.FindZones(symbol, period, candles)
	if len(zones) == 0 {
		return
	}

	// Получаем стакан (с кэшем)
	book := e.getOrderBookCached(symbol)

	// Конвертируем bybit.OrderBookV5 → *sr_zones.OrderBook и обогащаем зоны
	if book != nil {
		srBook := convertOrderBook(book)
		vol24h := e.volume24h.GetVolume24hUSD(symbol)
		zones = sr_zones.EnrichWithOrderBook(zones, srBook, vol24h)
	}

	if err := e.srStorage.SaveZones(symbol, period, zones); err != nil {
		logger.Warn("⚠️ SRZoneEngine: ошибка сохранения зон %s/%s: %v", symbol, period, err)
		return
	}

	logger.Debug("📐 SRZoneEngine: %s/%s → %d зон сохранено", symbol, period, len(zones))
}

// getOrderBookCached возвращает стакан из кэша или запрашивает у биржи.
func (e *Engine) getOrderBookCached(symbol string) *bybit.OrderBookV5 {
	e.obCacheMu.RLock()
	if entry, ok := e.obCache[symbol]; ok && time.Now().Before(entry.expiry) {
		e.obCacheMu.RUnlock()
		return entry.book
	}
	e.obCacheMu.RUnlock()

	book, err := e.orderBook.GetOrderBook(symbol, orderBookDepth)
	if err != nil {
		logger.Debug("⚠️ SRZoneEngine: не удалось получить стакан %s: %v", symbol, err)
		return nil
	}

	e.obCacheMu.Lock()
	e.obCache[symbol] = obCacheEntry{book: book, expiry: time.Now().Add(obCacheTTL)}
	e.obCacheMu.Unlock()

	return book
}

// convertOrderBook конвертирует bybit.OrderBookV5 в sr_zones.OrderBook.
func convertOrderBook(b *bybit.OrderBookV5) *sr_zones.OrderBook {
	book := &sr_zones.OrderBook{Symbol: b.Symbol}
	for _, l := range b.Bids {
		book.Bids = append(book.Bids, sr_zones.OrderLevel{Price: l.Price, Size: l.Size})
	}
	for _, l := range b.Asks {
		book.Asks = append(book.Asks, sr_zones.OrderLevel{Price: l.Price, Size: l.Size})
	}
	return book
}
