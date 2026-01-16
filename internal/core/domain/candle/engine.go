// internal/core/candle/engine.go
package candle

import (
	"sync"
	"time"

	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	"crypto-exchange-screener-bot/pkg/logger"
)

// CandleEngine - движок построения свечей
type CandleEngine struct {
	storage *CandleStorage
	config  CandleConfig

	// Каналы для обработки
	priceUpdates chan storage.PriceData
	stopCh       chan struct{}
	wg           sync.WaitGroup

	// Статистика
	buildErrors  int
	buildSuccess int
	totalBuilds  int
	statsMu      sync.RWMutex
}

// NewCandleEngine создает новый движок свечей
func NewCandleEngine(candleStorage *CandleStorage, config CandleConfig) *CandleEngine {
	return &CandleEngine{
		storage:      candleStorage,
		config:       config,
		priceUpdates: make(chan storage.PriceData, 10000),
		stopCh:       make(chan struct{}),
	}
}

// Start запускает движок
func (ce *CandleEngine) Start() error {
	logger.Info("🚀 Запуск CandleEngine...")

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

	close(ce.stopCh)
	ce.wg.Wait()

	logger.Info("✅ CandleEngine остановлен")
	return nil
}

// OnPriceUpdate вызывается при новой цене
func (ce *CandleEngine) OnPriceUpdate(priceData storage.PriceData) {
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

	return BuildResult{
		Candle:   candle,
		IsNew:    false,
		Duration: time.Since(startTime),
	}
}

// getOrCreateCandle получает или создает свечу
func (ce *CandleEngine) getOrCreateCandle(symbol, period string,
	priceData storage.PriceData) (*Candle, error) {

	// Пробуем получить активную свечу
	if candle, exists := ce.storage.GetActiveCandle(symbol, period); exists {
		return candle, nil
	}

	// Создаем новую свечу
	return ce.createNewCandle(symbol, period, priceData), nil
}

// createNewCandle создает новую свечу
func (ce *CandleEngine) createNewCandle(symbol, period string,
	priceData storage.PriceData) *Candle {

	now := time.Now()
	price := priceData.Price

	// Определяем время начала и окончания свечи
	startTime := ce.calculateCandleStartTime(now, period)
	endTime := ce.calculateCandleEndTime(startTime, period)

	return &Candle{
		Symbol:    symbol,
		Period:    period,
		Open:      price,
		High:      price,
		Low:       price,
		Close:     price,
		Volume:    priceData.Volume24h,
		VolumeUSD: priceData.VolumeUSD,
		Trades:    1,
		StartTime: startTime,
		EndTime:   endTime,
		IsClosed:  false,
		IsReal:    price > 0,
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
func (ce *CandleEngine) shouldCloseCandle(candle *Candle, period string) bool {
	if candle.IsClosed {
		return true
	}

	// Если текущее время после времени окончания свечи
	if time.Now().After(candle.EndTime) {
		return true
	}

	// Для тестирования: закрываем если свеча слишком старая
	if !candle.IsReal && time.Since(candle.StartTime) > 2*time.Minute {
		return true
	}

	return false
}

// updateCandle обновляет свечу новой ценой
func (ce *CandleEngine) updateCandle(candle *Candle, priceData storage.PriceData) {
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

// closeCandle закрывает свечу
func (ce *CandleEngine) closeCandle(candle *Candle) {
	candle.EndTime = time.Now()
	candle.IsClosed = true
	ce.storage.CloseAndArchiveCandle(candle)
}

// recordBuildResult записывает результат построения
func (ce *CandleEngine) recordBuildResult(result BuildResult) {
	ce.statsMu.Lock()
	defer ce.statsMu.Unlock()

	ce.totalBuilds++

	if result.Error != nil {
		ce.buildErrors++
	} else {
		ce.buildSuccess++
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

	storageStats := ce.storage.GetStats()

	return map[string]interface{}{
		"storage_stats": storageStats,
		"engine_stats": map[string]interface{}{
			"total_builds":  ce.totalBuilds,
			"build_success": ce.buildSuccess,
			"build_errors":  ce.buildErrors,
			"queue_size":    len(ce.priceUpdates),
			"success_rate":  float64(ce.buildSuccess) / float64(ce.totalBuilds) * 100,
		},
	}
}
