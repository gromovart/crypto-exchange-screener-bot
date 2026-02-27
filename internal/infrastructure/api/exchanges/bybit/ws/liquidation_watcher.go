// internal/infrastructure/api/exchanges/bybit/ws/liquidation_watcher.go
package ws

import (
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	bybitWSURL     = "wss://stream.bybit.com/v5/public/linear"
	pingInterval   = 20 * time.Second
	flushInterval  = 10 * time.Second
	windowDuration = 5 * time.Minute
	maxSymbols     = 200 // Bybit WS лимит топиков на соединение
	maxRetryDelay  = 60 * time.Second
)

// LiquidationCacheSetter узкий интерфейс для записи метрик ликвидаций.
// Реализуется BybitPriceFetcher.
type LiquidationCacheSetter interface {
	// SetLiquidationMetrics записывает агрегированные метрики в кэш
	SetLiquidationMetrics(symbol string, m *bybit.LiquidationMetrics)
	// GetTopSymbols возвращает топ-N символов по объёму (для подписки)
	GetTopSymbols(n int) []string
}

// LiquidationWatcher подписывается на Bybit WebSocket, агрегирует ликвидации
// и периодически обновляет кэш через LiquidationCacheSetter.
type LiquidationWatcher struct {
	cache      LiquidationCacheSetter
	aggregator *SlidingWindowAggregator

	stopCh chan struct{}
	wg     sync.WaitGroup

	// список символов, на которые подписаны
	subscribedSymbols []string
	symbolsMu         sync.RWMutex
}

// NewLiquidationWatcher создает новый наблюдатель ликвидаций
func NewLiquidationWatcher(cache LiquidationCacheSetter) *LiquidationWatcher {
	return &LiquidationWatcher{
		cache:      cache,
		aggregator: NewSlidingWindowAggregator(windowDuration),
		stopCh:     make(chan struct{}),
	}
}

// Start запускает горутины WS-соединения и сброса данных.
// Возвращает ошибку только если символы недоступны.
func (w *LiquidationWatcher) Start() error {
	// Получаем топ-символы для подписки
	symbols := w.cache.GetTopSymbols(maxSymbols)
	if len(symbols) == 0 {
		// Если символы ещё не загружены — стартуем с пустым списком
		// connectLoop попробует получить их позже при переподключении
		logger.Warn("⚠️ LiquidationWatcher: символы ещё не загружены, запускаемся без подписки")
	}

	w.symbolsMu.Lock()
	w.subscribedSymbols = symbols
	w.symbolsMu.Unlock()

	// Горутина WS-соединения с авто-переподключением
	w.wg.Add(1)
	go w.connectLoop()

	// Горутина периодического сброса агрегированных данных в кэш
	w.wg.Add(1)
	go w.flushLoop()

	logger.Info("🌊 LiquidationWatcher: запущен, символов для подписки: %d", len(symbols))
	return nil
}

// Stop останавливает все горутины и ждёт их завершения
func (w *LiquidationWatcher) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	logger.Info("🛑 LiquidationWatcher: остановлен")
}

// connectLoop — WS-соединение с экспоненциальным backoff при переподключении
func (w *LiquidationWatcher) connectLoop() {
	defer w.wg.Done()

	retryDelay := 2 * time.Second

	for {
		select {
		case <-w.stopCh:
			return
		default:
		}

		// Обновляем символы перед каждым подключением
		symbols := w.cache.GetTopSymbols(maxSymbols)
		if len(symbols) > 0 {
			w.symbolsMu.Lock()
			w.subscribedSymbols = symbols
			w.symbolsMu.Unlock()
		} else {
			w.symbolsMu.RLock()
			symbols = w.subscribedSymbols
			w.symbolsMu.RUnlock()
		}

		if len(symbols) == 0 {
			logger.Warn("⚠️ LiquidationWatcher: нет символов, повтор через %v", retryDelay)
			select {
			case <-time.After(retryDelay):
			case <-w.stopCh:
				return
			}
			retryDelay = minDuration(retryDelay*2, maxRetryDelay)
			continue
		}

		logger.Info("🔌 LiquidationWatcher: подключение к Bybit WS (%d символов)", len(symbols))
		err := w.runConnection(symbols)
		if err != nil {
			// Проверяем остановку
			select {
			case <-w.stopCh:
				return
			default:
			}
			logger.Warn("⚠️ LiquidationWatcher: WS-соединение прервано: %v, повтор через %v", err, retryDelay)
			select {
			case <-time.After(retryDelay):
			case <-w.stopCh:
				return
			}
			retryDelay = minDuration(retryDelay*2, maxRetryDelay)
		} else {
			retryDelay = 2 * time.Second // сброс задержки при успешном закрытии
		}
	}
}

// runConnection устанавливает одно WS-соединение, подписывается и читает события
func (w *LiquidationWatcher) runConnection(symbols []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Закрываем ctx при получении stopCh
	go func() {
		select {
		case <-w.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	conn, _, err := websocket.Dial(ctx, bybitWSURL, nil)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}
	defer conn.CloseNow()

	logger.Info("✅ LiquidationWatcher: WS-соединение установлено")

	// Подписываемся батчами (Bybit принимает до 10 args за раз)
	// Используем новый топик allLiquidation.{symbol} (старый liquidation.{symbol} задепрекейтил Bybit)
	topics := make([]string, 0, len(symbols))
	for _, sym := range symbols {
		topics = append(topics, "allLiquidation."+sym)
	}
	if err := w.subscribeTopics(ctx, conn, topics); err != nil {
		return fmt.Errorf("ошибка подписки: %w", err)
	}

	// Пинг-горутина
	pingStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ping := wsPingMsg{Op: "ping"}
				if err := wsjson.Write(ctx, conn, ping); err != nil {
					return
				}
			case <-pingStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	defer close(pingStop)

	// Читаем сообщения
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			select {
			case <-ctx.Done():
				return nil // нормальная остановка
			default:
				return fmt.Errorf("ошибка чтения: %w", err)
			}
		}

		w.handleMessage(raw)
	}
}

// subscribeTopics отправляет сообщения подписки батчами по 10 топиков
func (w *LiquidationWatcher) subscribeTopics(ctx context.Context, conn *websocket.Conn, topics []string) error {
	const batchSize = 10

	for i := 0; i < len(topics); i += batchSize {
		end := i + batchSize
		if end > len(topics) {
			end = len(topics)
		}
		batch := topics[i:end]

		msg := wsSubscribeMsg{
			Op:   "subscribe",
			Args: batch,
		}
		if err := wsjson.Write(ctx, conn, msg); err != nil {
			return err
		}
		// Небольшая пауза между батчами
		time.Sleep(50 * time.Millisecond)
	}

	logger.Info("📡 LiquidationWatcher: подписан на %d топиков", len(topics))
	return nil
}

// handleMessage обрабатывает входящее сообщение
func (w *LiquidationWatcher) handleMessage(raw json.RawMessage) {
	// Сначала пробуем декодировать как системный ответ
	var resp wsResponseMsg
	if err := json.Unmarshal(raw, &resp); err == nil {
		if resp.Op == "pong" {
			logger.Debug("🏓 LiquidationWatcher: получен pong")
			return
		}
		if resp.Op == "subscribe" {
			if resp.Success {
				logger.Debug("✅ LiquidationWatcher: подписка подтверждена")
			} else {
				logger.Warn("⚠️ LiquidationWatcher: ошибка подписки: %s", resp.RetMsg)
			}
			return
		}
	}

	// Пробуем декодировать как ликвидацию (новый формат allLiquidation)
	var msg AllLiquidationMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	// Проверяем что это топик ликвидации
	if !strings.HasPrefix(msg.Topic, "allLiquidation.") {
		return
	}

	// data — массив событий (может быть несколько за один пуш)
	now := time.Now()
	for _, d := range msg.Data {
		if d.Symbol == "" || d.Price == "" || d.Size == "" {
			continue
		}

		price, err := strconv.ParseFloat(d.Price, 64)
		if err != nil || price <= 0 {
			continue
		}

		size, err := strconv.ParseFloat(d.Size, 64)
		if err != nil || size <= 0 {
			continue
		}

		sizeUSD := size * price

		// В новом API S — это сторона ПОЗИЦИИ, а не ордера:
		// "Buy"  = Buy-позиция (лонг) была ликвидирована
		// "Sell" = Sell-позиция (шорт) была ликвидирована
		isLong := d.Side == "Buy"

		event := liqEvent{
			sizeUSD:   sizeUSD,
			isLong:    isLong,
			timestamp: now,
		}

		w.aggregator.Add(d.Symbol, event)

		logger.Debug("💥 LiquidationWatcher: %s %s $%.0f",
			d.Symbol, d.Side, sizeUSD)
	}
}

// flushLoop периодически записывает агрегированные данные в кэш
func (w *LiquidationWatcher) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.flush()
		case <-w.stopCh:
			return
		}
	}
}

// flush записывает все накопленные метрики в кэш
func (w *LiquidationWatcher) flush() {
	w.symbolsMu.RLock()
	symbols := w.subscribedSymbols
	w.symbolsMu.RUnlock()

	written := 0
	for _, sym := range symbols {
		metrics := w.aggregator.GetMetrics(sym)
		if metrics != nil {
			w.cache.SetLiquidationMetrics(sym, metrics)
			written++
			logger.Debug("💾 LiquidationWatcher: %s: $%.0f (LONG $%.0f, SHORT $%.0f)",
				sym, metrics.TotalVolumeUSD, metrics.LongLiqVolume, metrics.ShortLiqVolume)
		}
	}

	if written > 0 {
		logger.Info("🔄 LiquidationWatcher: сброс данных — %d символов с ликвидациями", written)
	}
}

// minDuration возвращает меньшую из двух длительностей
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
