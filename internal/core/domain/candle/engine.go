// internal/core/domain/candle/engine.go
package candle

import (
	"fmt"
	"math"
	"sync"
	"time"

	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// CandleEngine - движок построения свечей
type CandleEngine struct {
	storage  storage.CandleStorageInterface
	config   storage.CandleConfig
	eventBus *events.EventBus

	// Каналы для обработки
	priceUpdates chan storage.PriceData
	stopCh       chan struct{}
	wg           sync.WaitGroup

	// Статистика
	buildErrors   int
	buildSuccess  int
	totalBuilds   int
	closedCandles int
	lastStatsLog  time.Time
	statsInterval time.Duration
	statsMu       sync.RWMutex

	// Подписчик на события
	priceSubscriber types.EventSubscriber
}

// NewCandleEngine создает новый движок свечей
func NewCandleEngine(
	candleStorage storage.CandleStorageInterface,
	config storage.CandleConfig,
	eventBus *events.EventBus, // НОВЫЙ параметр
) *CandleEngine {
	engine := &CandleEngine{
		storage:       candleStorage,
		config:        config,
		eventBus:      eventBus, // НОВОЕ
		priceUpdates:  make(chan storage.PriceData, 50000),
		stopCh:        make(chan struct{}),
		lastStatsLog:  time.Now(),
		statsInterval: 60 * time.Second,
	}

	// Создаем подписчика на события цен
	engine.createPriceSubscriber()

	return engine
}

// Start запускает движок
func (ce *CandleEngine) Start() error {
	logger.Info("🚀 Запуск CandleEngine...")

	// Подписываемся на события цен
	if ce.eventBus != nil && ce.priceSubscriber != nil {
		ce.eventBus.Subscribe(types.EventPriceUpdated, ce.priceSubscriber)
		logger.Info("✅ CandleEngine подписался на EventPriceUpdated")
	}

	// Запускаем обработчики
	ce.wg.Add(1)
	go ce.processPriceUpdates()

	// Запускаем очистку если настроено
	if ce.config.CleanupInterval > 0 {
		ce.wg.Add(1)
		go ce.cleanupRoutine()
	}

	logger.Info("✅ CandleEngine запущен")
	return nil
}

// Stop останавливает движок
func (ce *CandleEngine) Stop() error {
	logger.Info("🛑 Остановка CandleEngine...")

	// Отписываемся от событий
	if ce.eventBus != nil && ce.priceSubscriber != nil {
		ce.eventBus.Unsubscribe(types.EventPriceUpdated, ce.priceSubscriber)
		logger.Info("✅ CandleEngine отписался от EventPriceUpdated")
	}

	close(ce.stopCh)
	ce.wg.Wait()

	logger.Info("✅ CandleEngine остановлен")
	return nil
}

// createPriceSubscriber создает подписчика на события цен
func (ce *CandleEngine) createPriceSubscriber() {
	ce.priceSubscriber = events.NewBaseSubscriber(
		"candle_engine",
		[]types.EventType{types.EventPriceUpdated},
		ce.handlePriceEvent,
	)
}

// handlePriceEvent обрабатывает события цен из EventBus
func (ce *CandleEngine) handlePriceEvent(event types.Event) error {
	logger.Debug("🕯️ CandleEngine получил событие цены: %s", event.Type)

	switch event.Type {
	case types.EventPriceUpdated:
		if priceData, ok := event.Data.(storage.PriceData); ok {
			// Отправляем цену в канал для асинхронной обработки
			select {
			case ce.priceUpdates <- priceData:
				// Успешно добавлено в очередь
			default:
				ce.statsMu.Lock()
				ce.buildErrors++
				ce.statsMu.Unlock()

				logger.Warn("⚠️ Очередь цен CandleEngine переполнена, пропускаем цену %s",
					priceData.Symbol)
			}
		}
	}

	return nil
}

// processPriceUpdates обрабатывает обновления цен
func (ce *CandleEngine) processPriceUpdates() {
	defer ce.wg.Done()

	logger.Debug("🔄 CandleEngine: запущен обработчик цен")

	for {
		select {
		case priceData := <-ce.priceUpdates:
			ce.processPriceData(priceData)
		case <-ce.stopCh:
			logger.Debug("🔄 CandleEngine: остановка обработчика цен")
			return
		}
	}
}

// processPriceData обрабатывает одну цену
func (ce *CandleEngine) processPriceData(priceData storage.PriceData) {
	startTime := time.Now()
	symbol := priceData.Symbol

	// Для каждого поддерживаемого периода
	for _, period := range ce.config.SupportedPeriods {
		buildResult := ce.buildCandleForPeriod(symbol, period, priceData)
		ce.recordBuildResult(buildResult)
	}

	duration := time.Since(startTime)
	if duration > 10*time.Millisecond {
		logger.Debug("⏱️ Обработка цены %s заняла %v", symbol, duration)
	}
}

// buildCandleForPeriod строит/обновляет свечу для периода
func (ce *CandleEngine) buildCandleForPeriod(symbol, period string,
	priceData storage.PriceData) BuildResult {

	startTime := time.Now()

	// Получаем или создаем свечу
	candle, err := ce.getOrCreateCandle(symbol, period, priceData)
	if err != nil {
		return BuildResult{
			Error:    err,
			Duration: time.Since(startTime),
		}
	}

	// Проверяем, не нужно ли закрыть свечу
	if ce.shouldCloseCandle(candle, period) {
		// Логируем информацию о закрытии
		elapsed := time.Now().Sub(candle.StartTime)
		expectedDuration := ce.getExpectedDuration(period)
		completionPercent := float64(elapsed) / float64(expectedDuration) * 100

		if completionPercent >= 95.0 {
			changePercent := ((candle.Close - candle.Open) / candle.Open) * 100
			// Вариант B: Только значимые изменения
			if math.Abs(changePercent) > 0.05 { // > 0.05%
				logger.Debug("📊 CandleEngine: значимое закрытие %s %s: %.2f%%",
					symbol, period, changePercent)
			}
		}

		ce.closeCandle(candle)
		candle = ce.createNewCandle(symbol, period, priceData)
		ce.storage.SaveActiveCandle(candle)

		return BuildResult{
			Candle:   candle,
			IsNew:    true,
			Duration: time.Since(startTime),
		}
	}

	// Обновляем существующую свечу
	ce.updateCandle(candle, priceData)
	ce.storage.SaveActiveCandle(candle)

	// Логируем обновление если свеча почти завершена
	elapsed := time.Now().Sub(candle.StartTime)
	expectedDuration := ce.getExpectedDuration(period)
	completionPercent := float64(elapsed) / float64(expectedDuration) * 100

	if completionPercent >= 80.0 && completionPercent < 95.0 {
		logger.Debug("⏳ CandleEngine: обновляем почти завершенную свечу %s %s (%.0f%% завершено)",
			symbol, period, completionPercent)
	}

	return BuildResult{
		Candle:   candle,
		IsNew:    false,
		Duration: time.Since(startTime),
	}
}

// getOrCreateCandle получает или создает свечу
func (ce *CandleEngine) getOrCreateCandle(symbol, period string,
	priceData storage.PriceData) (*storage.Candle, error) {

	// Пробуем получить активную свечу
	if candleInterface, exists := ce.storage.GetActiveCandle(symbol, period); exists {
		// Конвертируем интерфейс в *Candle
		if candle, ok := candleInterface.(*storage.Candle); ok {
			return candle, nil
		}
		// Если это не *Candle, создаем новый из интерфейса
		return ce.convertCandleInterface(candleInterface), nil
	}

	// Создаем новую свечу
	return ce.createNewCandle(symbol, period, priceData), nil
}

// convertCandleInterface конвертирует интерфейс в *Candle
func (ce *CandleEngine) convertCandleInterface(candleInterface storage.CandleInterface) *storage.Candle {
	return &storage.Candle{
		Symbol:       candleInterface.GetSymbol(),
		Period:       candleInterface.GetPeriod(),
		Open:         candleInterface.GetOpen(),
		High:         candleInterface.GetHigh(),
		Low:          candleInterface.GetLow(),
		Close:        candleInterface.GetClose(),
		Volume:       candleInterface.GetVolume(),
		VolumeUSD:    candleInterface.GetVolumeUSD(),
		Trades:       candleInterface.GetTrades(),
		StartTime:    candleInterface.GetStartTime(),
		EndTime:      candleInterface.GetEndTime(),
		IsClosedFlag: candleInterface.IsClosed(),
		IsRealFlag:   candleInterface.IsReal(),
	}
}

// createNewCandle создает новую свечу
func (ce *CandleEngine) createNewCandle(symbol, period string,
	priceData storage.PriceData) *storage.Candle {

	now := time.Now()
	price := priceData.Price

	// Определяем время начала и окончания свечи
	startTime := ce.calculateCandleStartTime(now, period)
	endTime := ce.calculateCandleEndTime(startTime, period)

	return &storage.Candle{
		Symbol:       symbol,
		Period:       period,
		Open:         price,
		High:         price,
		Low:          price,
		Close:        price,
		Volume:       priceData.Volume24h,
		VolumeUSD:    priceData.VolumeUSD,
		Trades:       1,
		StartTime:    startTime,
		EndTime:      endTime,
		IsClosedFlag: false,
		IsRealFlag:   price > 0,
	}
}

// calculateCandleStartTime вычисляет время начала свечи
func (ce *CandleEngine) calculateCandleStartTime(currentTime time.Time, period string) time.Time {
	switch period {
	case "5m":
		// Округляем до ближайших 5 минут
		minutes := currentTime.Minute() / 5 * 5
		return time.Date(
			currentTime.Year(), currentTime.Month(), currentTime.Day(),
			currentTime.Hour(), minutes, 0, 0, currentTime.Location(),
		)
	case "15m":
		minutes := currentTime.Minute() / 15 * 15
		return time.Date(
			currentTime.Year(), currentTime.Month(), currentTime.Day(),
			currentTime.Hour(), minutes, 0, 0, currentTime.Location(),
		)
	case "30m":
		minutes := currentTime.Minute() / 30 * 30
		return time.Date(
			currentTime.Year(), currentTime.Month(), currentTime.Day(),
			currentTime.Hour(), minutes, 0, 0, currentTime.Location(),
		)
	case "1h":
		return time.Date(
			currentTime.Year(), currentTime.Month(), currentTime.Day(),
			currentTime.Hour(), 0, 0, 0, currentTime.Location(),
		)
	case "4h":
		hour := currentTime.Hour() / 4 * 4
		return time.Date(
			currentTime.Year(), currentTime.Month(), currentTime.Day(),
			hour, 0, 0, 0, currentTime.Location(),
		)
	case "1d":
		return time.Date(
			currentTime.Year(), currentTime.Month(), currentTime.Day(),
			0, 0, 0, 0, currentTime.Location(),
		)
	default:
		return currentTime
	}
}

// calculateCandleEndTime вычисляет время окончания свечи
func (ce *CandleEngine) calculateCandleEndTime(startTime time.Time, period string) time.Time {
	switch period {
	case "5m":
		return startTime.Add(5 * time.Minute)
	case "15m":
		return startTime.Add(15 * time.Minute)
	case "30m":
		return startTime.Add(30 * time.Minute)
	case "1h":
		return startTime.Add(1 * time.Hour)
	case "4h":
		return startTime.Add(4 * time.Hour)
	case "1d":
		return startTime.Add(24 * time.Hour)
	default:
		return startTime.Add(15 * time.Minute)
	}
}

// shouldCloseCandle проверяет, нужно ли закрыть свечу
func (ce *CandleEngine) shouldCloseCandle(candle *storage.Candle, period string) bool {
	// 1. Если свеча уже закрыта - да
	if candle.IsClosedFlag {
		return true
	}

	now := time.Now()

	// 2. Если текущее время после времени окончания свечи - закрываем
	if now.After(candle.EndTime) {
		logger.Debug("🕐 CandleEngine: закрываем свечу %s %s (время окончания: %s, сейчас: %s)",
			candle.Symbol, period,
			candle.EndTime.Format("15:04:05"), now.Format("15:04:05"))
		return true
	}

	// 3. Если свеча слишком старая (больше чем 2 периода) - закрываем как бракованную
	elapsed := now.Sub(candle.StartTime)
	expectedDuration := ce.getExpectedDuration(period)

	if elapsed > expectedDuration*2 {
		logger.Warn("⚠️ CandleEngine: закрываем бракованную свечу %s %s (слишком старая: %v, ожидалось: %v)",
			candle.Symbol, period, elapsed.Round(time.Second), expectedDuration)
		return true
	}

	// 4. Для тестового режима: если свеча ненастоящая и старая
	if !candle.IsRealFlag && elapsed > 2*time.Minute {
		return true
	}

	// 5. НОВОЕ: если прошло >95% времени свечи, закрываем её заранее
	// Это гарантирует что свеча будет закрыта даже если цена пришла чуть раньше
	completionPercent := float64(elapsed) / float64(expectedDuration) * 100

	if completionPercent >= 95.0 {
		logger.Debug("⚡ CandleEngine: досрочно закрываем свечу %s %s (%.0f%% завершено)",
			candle.Symbol, period, completionPercent)
		return true
	}

	return false
}

// getExpectedDuration возвращает ожидаемую длительность периода
func (ce *CandleEngine) getExpectedDuration(period string) time.Duration {
	switch period {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

// updateCandle обновляет свечу новой ценой
func (ce *CandleEngine) updateCandle(candle *storage.Candle, priceData storage.PriceData) {
	price := priceData.Price

	// Обновляем high/low
	if price > candle.High {
		candle.High = price
	}
	if price < candle.Low {
		candle.Low = price
	}

	// Обновляем close
	candle.Close = price

	// Обновляем объем
	candle.Volume += priceData.Volume24h
	candle.VolumeUSD += priceData.VolumeUSD
	candle.Trades++
}

// closeCandle закрывает свечу - УБРАНО ЛОГИРОВАНИЕ
func (ce *CandleEngine) closeCandle(candle *storage.Candle) {
	candle.EndTime = time.Now()
	candle.IsClosedFlag = true
	ce.storage.CloseAndArchiveCandle(candle)
	// Убрано логирование каждой закрытой свечи
}

// recordBuildResult записывает результат построения и логирует статистику раз в statsInterval
func (ce *CandleEngine) recordBuildResult(result BuildResult) {
	ce.statsMu.Lock()
	defer ce.statsMu.Unlock()

	ce.totalBuilds++

	if result.Error != nil {
		ce.buildErrors++
		logger.Debug("❌ CandleEngine: ошибка построения свечи: %v", result.Error)
	} else {
		ce.buildSuccess++
		// Если свеча новая (была закрыта старая и создана новая)
		if result.IsNew {
			ce.closedCandles++
			// Логируем закрытие свечи только для статистики
			if ce.closedCandles%100 == 0 { // Каждую 100-ю свечу
				logger.Info("📊 CandleEngine: закрыто %d свечей", ce.closedCandles)
			}
		}
	}

	// Логировать агрегированную статистику раз в statsInterval
	now := time.Now()
	if now.Sub(ce.lastStatsLog) >= ce.statsInterval {
		var successRate float64
		if ce.totalBuilds > 0 {
			successRate = float64(ce.buildSuccess) / float64(ce.totalBuilds) * 100
		}

		logger.Info("📊 Статистика CandleEngine за %v: обработано %d свечей, закрыто %d, успешно: %.2f%%",
			ce.statsInterval, ce.totalBuilds, ce.closedCandles, successRate)

		// Сбросить счетчики для следующего интервала
		ce.totalBuilds = 0
		ce.buildSuccess = 0
		ce.buildErrors = 0
		ce.closedCandles = 0
		ce.lastStatsLog = now
	}
}

// cleanupRoutine очищает старые данные
func (ce *CandleEngine) cleanupRoutine() {
	defer ce.wg.Done()

	ticker := time.NewTicker(ce.config.CleanupInterval)
	defer ticker.Stop()

	logger.Debug("🧹 CandleEngine: запущена очистка (интервал: %v)", ce.config.CleanupInterval)

	for {
		select {
		case <-ticker.C:
			removed := ce.storage.CleanupOldCandles(24 * time.Hour)
			if removed > 0 {
				logger.Debug("🧹 CandleEngine: очищено %d старых свечей", removed)
			}
		case <-ce.stopCh:
			logger.Debug("🧹 CandleEngine: остановка очистки")
			return
		}
	}
}

// GetStats возвращает статистику движка
func (ce *CandleEngine) GetStats() map[string]interface{} {
	ce.statsMu.RLock()
	defer ce.statsMu.RUnlock()

	var storageStats interface{}
	if stats := ce.storage.GetStats(); stats != nil {
		storageStats = stats
	}

	var successRate float64
	if ce.totalBuilds > 0 {
		successRate = float64(ce.buildSuccess) / float64(ce.totalBuilds) * 100
	}

	return map[string]interface{}{
		"storage_stats": storageStats,
		"engine_stats": map[string]interface{}{
			"total_builds":   ce.totalBuilds,
			"build_success":  ce.buildSuccess,
			"build_errors":   ce.buildErrors,
			"closed_candles": ce.closedCandles, // Добавлено в статистику
			"queue_size":     len(ce.priceUpdates),
			"success_rate":   successRate,
			"stats_interval": ce.statsInterval.String(),
		},
	}
}

// subscribeToEvents подписывается на события через EventBus
func (ce *CandleEngine) subscribeToEvents() error {
	if ce.eventBus == nil {
		return fmt.Errorf("EventBus не инициализирован")
	}

	if ce.priceSubscriber == nil {
		return fmt.Errorf("подписчик на события не создан")
	}

	// Подписываемся на событие обновления цен
	ce.eventBus.Subscribe(types.EventPriceUpdated, ce.priceSubscriber)
	logger.Info("✅ CandleEngine подписался на EventPriceUpdated через EventBus")

	return nil
}

// unsubscribeFromEvents отписывается от событий EventBus
func (ce *CandleEngine) unsubscribeFromEvents() error {
	if ce.eventBus == nil || ce.priceSubscriber == nil {
		return nil // Ничего не делаем если EventBus не инициализирован
	}

	// Отписываемся от события обновления цен
	ce.eventBus.Unsubscribe(types.EventPriceUpdated, ce.priceSubscriber)
	logger.Info("✅ CandleEngine отписался от EventPriceUpdated")

	return nil
}
