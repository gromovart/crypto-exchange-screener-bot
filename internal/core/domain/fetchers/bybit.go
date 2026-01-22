// internal/core/domain/fetchers/bybit.go
package fetchers

import (
	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/internal/infrastructure/api"
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"strings"
	"sync"
	"time"
)

// BybitPriceFetcher реализация
type BybitPriceFetcher struct {
	client       *bybit.BybitClient
	storage      storage.PriceStorageInterface
	eventBus     *events.EventBus
	candleSystem *candle.CandleSystem // НОВОЕ: Свечная система
	mu           sync.RWMutex
	running      bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
	config       *config.Config

	// Кэш для Open Interest
	oiCache   map[string]float64
	oiCacheMu sync.RWMutex

	// Кэш для ликвидаций
	liqCache   map[string]*bybit.LiquidationMetrics
	liqCacheMu sync.RWMutex

	// Настройки OI
	oiEnabled        bool
	oiUpdateInterval time.Duration
	lastOIUpdate     time.Time
	oiRetryCount     int

	// Настройки ликвидаций
	liqEnabled        bool
	liqUpdateInterval time.Duration
	lastLiqUpdate     time.Time

	// Кэш для дельты объемов
	volumeDeltaCache   map[string]*volumeDeltaCache
	volumeDeltaCacheMu sync.RWMutex
	volumeDeltaTTL     time.Duration

	// Настройки timeout и retry
	maxRetries     int
	retryDelay     time.Duration
	lastFetchError time.Time
	errorCount     int
}

// Структура кэша дельты
type volumeDeltaCache struct {
	data       *bybit.VolumeDelta
	expiration time.Time
	updateTime time.Time
}

// NewPriceFetcher создает новый PriceFetcher (обновленный конструктор)
func NewPriceFetcher(apiClient *bybit.BybitClient, storage storage.PriceStorageInterface,
	eventBus *events.EventBus, candleSystem *candle.CandleSystem) *BybitPriceFetcher { // НОВЫЙ параметр

	return &BybitPriceFetcher{
		client:       apiClient,
		storage:      storage,
		eventBus:     eventBus,
		candleSystem: candleSystem, // НОВОЕ
		stopChan:     make(chan struct{}),
		running:      false,
		oiCache:      make(map[string]float64),
		liqCache:     make(map[string]*bybit.LiquidationMetrics),

		// Инициализация кэша дельты
		volumeDeltaCache: make(map[string]*volumeDeltaCache),
		volumeDeltaTTL:   30 * time.Second,

		// Настройки OI
		oiEnabled:        true,
		oiUpdateInterval: 5 * time.Minute,
		lastOIUpdate:     time.Now(),
		oiRetryCount:     0,

		// Настройки ликвидаций
		liqEnabled:        true,
		liqUpdateInterval: 1 * time.Minute,
		lastLiqUpdate:     time.Now(),

		// Настройки timeout и retry
		maxRetries: 3,
		retryDelay: 2 * time.Second,
		errorCount: 0,
	}
}

func (f *BybitPriceFetcher) Start(interval time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return fmt.Errorf("price fetcher already running")
	}

	f.running = true
	f.wg.Add(1)

	// ДОБАВЛЯЕМ: Логируем
	logger.Info("🚀 BybitFetcher: ЗАПУСК с интервалом %v", interval)

	// Запускаем фоновую очистку кэша дельты
	f.startCacheCleanupLoop()

	go func() {
		defer f.wg.Done()
		logger.Debug("🏃 BybitFetcher: горутина запущена")

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logger.Debug("⏰ BybitFetcher: таймер создан с интервалом %v", interval)

		// Первоначальный запрос
		if err := f.fetchPrices(); err != nil {
			logger.Warn("⚠️ Ошибка первоначального получения цен: %v", err)
		}

		// Запускаем отдельный горутин для получения ликвидаций
		f.wg.Add(1)
		go f.fetchLiquidationsLoop(1 * time.Minute)

		for {
			select {
			case <-ticker.C:
				logger.Debug("⏰ BybitFetcher: сработал таймер в %s",
					time.Now().Format("15:04:05.000"))
				if err := f.fetchPrices(); err != nil {
					logger.Warn("⚠️ Ошибка получения цен: %v", err)
				}
			case <-f.stopChan:
				logger.Debug("🛑 BybitFetcher: получен сигнал остановки")
				return
			}
		}
	}()

	logger.Info("✅ PriceFetcher запущен с поддержкой дельты объемов")
	return nil
}

func (f *BybitPriceFetcher) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return nil
	}

	f.running = false
	close(f.stopChan)
	f.wg.Wait()

	logger.Info("🛑 PriceFetcher остановлен")
	return nil
}

// ==================== МЕТОДЫ ДЛЯ ДЕЛЬТЫ ОБЪЕМОВ ====================

// GetRealTimeVolumeDelta получает дельту объемов за последние 5 минут с кэшированием
func (f *BybitPriceFetcher) GetRealTimeVolumeDelta(symbol string) (*bybit.VolumeDelta, error) {
	// Проверяем кэш
	if cached, found := f.getVolumeDeltaFromCache(symbol); found {
		age := time.Since(cached.updateTime).Round(time.Second)
		logger.Debug("📦 Дельта объемов из кэша для %s (возраст: %v)", symbol, age)
		return cached.data, nil
	}

	// Получаем свежие данные через API
	logger.Debug("🔄 Запрос реальной дельты для %s из API...", symbol)
	volumeDelta, err := f.client.GetRealTimeVolumeDelta(symbol)
	if err != nil {
		logger.Debug("⚠️ Ошибка получения дельты для %s: %v", symbol, err)
		return nil, err
	}

	// Обновляем кэш
	f.setVolumeDeltaToCache(symbol, volumeDelta)

	logger.Debug("✅ Получена свежая дельта для %s: $%.0f (%.1f%%)",
		symbol, volumeDelta.Delta, volumeDelta.DeltaPercent)

	return volumeDelta, nil
}

// GetVolumeDelta получает дельту объемов для символа за указанный период
func (f *BybitPriceFetcher) GetVolumeDelta(symbol string, period time.Duration) (*bybit.VolumeDelta, error) {
	// Для разных периодов используем разные ключи кэша
	cacheKey := fmt.Sprintf("%s_%v", symbol, period)

	// Проверяем кэш
	if cached, found := f.getVolumeDeltaFromCache(cacheKey); found {
		logger.Debug("📦 Дельта из кэша для %s за период %v", symbol, period)
		return cached.data, nil
	}

	// Получаем свежие данные
	volumeDelta, err := f.client.GetVolumeDelta(symbol, period)
	if err != nil {
		logger.Debug("⚠️ Ошибка получения дельты для %s за период %v: %v", symbol, period, err)
		return nil, err
	}

	// Обновляем кэш
	f.setVolumeDeltaToCache(cacheKey, volumeDelta)

	return volumeDelta, nil
}

// CalculateEstimatedVolumeDelta рассчитывает эмулированную дельту (fallback)
func (f *BybitPriceFetcher) CalculateEstimatedVolumeDelta(symbol, direction string, volume24h float64) (*bybit.VolumeDelta, error) {
	// Эмуляция дельты (2% от объема)
	baseDelta := volume24h * 0.02
	basePercent := 10.0

	var delta, deltaPercent float64
	if direction == "growth" {
		delta = baseDelta
		deltaPercent = basePercent
	} else {
		delta = -baseDelta
		deltaPercent = -basePercent
	}

	// Получаем цену из хранилища для расчета объемов
	var price float64
	if snapshot, exists := f.storage.GetCurrentSnapshot(symbol); exists {
		price = snapshot.Price
	} else {
		price = 1.0
	}

	// Рассчитываем примерные объемы покупок/продаж
	// Предполагаем, что за 5 минут происходит ~1% от дневного объема
	totalVolumeUSD := volume24h * 0.01
	buyVolume := totalVolumeUSD * 0.55  // 55% покупок
	sellVolume := totalVolumeUSD * 0.45 // 45% продаж

	if direction == "fall" {
		buyVolume = totalVolumeUSD * 0.45  // 45% покупок при падении
		sellVolume = totalVolumeUSD * 0.55 // 55% продаж при падении
	}

	// Рассчитываем количество сделок (примерно)
	totalTrades := int(totalVolumeUSD / (price * 100)) // Примерно по 100 монет на сделку
	if totalTrades < 10 {
		totalTrades = 10
	} else if totalTrades > 1000 {
		totalTrades = 1000
	}

	return &bybit.VolumeDelta{
		Symbol:       symbol,
		Period:       "5m",
		StartTime:    time.Now().Add(-5 * time.Minute),
		EndTime:      time.Now(),
		BuyVolume:    buyVolume,
		SellVolume:   sellVolume,
		Delta:        delta,
		DeltaPercent: deltaPercent,
		TotalTrades:  totalTrades,
		UpdateTime:   time.Now(),
	}, nil
}

// getVolumeDeltaFromCache получает дельту из кэша
func (f *BybitPriceFetcher) getVolumeDeltaFromCache(key string) (*volumeDeltaCache, bool) {
	f.volumeDeltaCacheMu.RLock()
	defer f.volumeDeltaCacheMu.RUnlock()

	if cache, exists := f.volumeDeltaCache[key]; exists {
		if time.Now().Before(cache.expiration) {
			return cache, true
		}
		// Кэш устарел - удаляем при следующей записи
	}
	return nil, false
}

// setVolumeDeltaToCache сохраняет дельту в кэш
func (f *BybitPriceFetcher) setVolumeDeltaToCache(key string, data *bybit.VolumeDelta) {
	f.volumeDeltaCacheMu.Lock()
	defer f.volumeDeltaCacheMu.Unlock()

	f.volumeDeltaCache[key] = &volumeDeltaCache{
		data:       data,
		expiration: time.Now().Add(f.volumeDeltaTTL),
		updateTime: time.Now(),
	}
}

// cleanupVolumeDeltaCache очищает устаревший кэш
func (f *BybitPriceFetcher) cleanupVolumeDeltaCache() {
	f.volumeDeltaCacheMu.Lock()
	defer f.volumeDeltaCacheMu.Unlock()

	cleared := 0
	now := time.Now()
	for key, cache := range f.volumeDeltaCache {
		if now.After(cache.expiration) {
			delete(f.volumeDeltaCache, key)
			cleared++
		}
	}

	if cleared > 0 {
		logger.Debug("🧹 Очищен кэш дельты: удалено %d устаревших записей", cleared)
	}
}

// startCacheCleanupLoop запускает фоновую очистку кэша
func (f *BybitPriceFetcher) startCacheCleanupLoop() {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				f.cleanupVolumeDeltaCache()
			case <-f.stopChan:
				return
			}
		}
	}()
}

// GetLiquidationMetrics получает метрики ликвидаций для символа
func (f *BybitPriceFetcher) GetLiquidationMetrics(symbol string) (*bybit.LiquidationMetrics, bool) {
	f.liqCacheMu.RLock()
	metrics, exists := f.liqCache[symbol]
	f.liqCacheMu.RUnlock()

	if !exists || time.Since(metrics.UpdateTime) > 10*time.Minute {
		// Если данных нет или они устарели, пытаемся получить свежие
		go func() {
			summary, err := f.client.GetLiquidationsSummary(symbol, 5*time.Minute)
			if err == nil {
				metrics = &bybit.LiquidationMetrics{
					Symbol:         symbol,
					TotalVolumeUSD: summary["total_volume_usd"].(float64),
					LongLiqVolume:  summary["long_liq_volume"].(float64),
					ShortLiqVolume: summary["short_liq_volume"].(float64),
					LongLiqCount:   summary["long_liq_count"].(int),
					ShortLiqCount:  summary["short_liq_count"].(int),
					UpdateTime:     time.Now(),
				}

				f.liqCacheMu.Lock()
				f.liqCache[symbol] = metrics
				f.liqCacheMu.Unlock()
			}
		}()

		if !exists {
			return nil, false
		}
	}

	return metrics, true
}

// ==================== МЕТОДЫ OPEN INTEREST ====================

// fetchOpenInterest получает реальный OI через API
func (f *BybitPriceFetcher) fetchOpenInterest() error {
	// Проверяем, прошло ли достаточно времени с последнего обновления
	if time.Since(f.lastOIUpdate) < f.oiUpdateInterval {
		logger.Debug("⏱️  Пропуск обновления OI, еще не прошло %v", f.oiUpdateInterval)
		return nil
	}

	logger.Info("🔄 BybitFetcher: получение реального Open Interest...")

	// Получаем все символы из хранилища
	symbols := f.storage.GetSymbols()

	if len(symbols) == 0 {
		logger.Info("📭 Нет символов для получения OI")
		return nil
	}

	logger.Debug("📊 Запрашиваем OI для %d символов", len(symbols))

	// Ограничиваем количество символов для запроса (Bybit API может иметь лимиты)
	maxSymbols := 20 // Уменьшили с 50 до 20 для снижения нагрузки
	if len(symbols) > maxSymbols {
		// Берем только топ-символы по объему
		topSymbols, err := f.storage.GetTopSymbolsByVolumeUSD(maxSymbols)
		if err != nil {
			logger.Warn("⚠️ Не удалось получить топ-символы: %v", err)
			// Берем первые maxSymbols
			symbols = symbols[:maxSymbols]
		} else {
			symbols = make([]string, len(topSymbols))
			for i, sv := range topSymbols {
				symbols[i] = sv.Symbol
			}
		}
		logger.Debug("📋 Ограничено до %d символов", len(symbols))
	}

	// Получаем реальный OI через API
	realOI, err := f.client.GetOpenInterestForSymbols(symbols)
	if err != nil {
		// Анализируем ошибку
		if strings.Contains(err.Error(), "intervalTime") || strings.Contains(err.Error(), "10001") {
			logger.Error("❌ BybitFetcher: КРИТИЧЕСКАЯ ОШИБКА - неверный параметр intervalTime")
			logger.Error("⚠️  Проверьте метод GetOpenInterest в BybitClient")
		}

		logger.Warn("⚠️ BybitFetcher: не удалось получить реальный OI: %v", err)
		f.oiRetryCount++

		// Если много неудачных попыток, увеличиваем интервал
		if f.oiRetryCount > 5 {
			f.oiUpdateInterval = 15 * time.Minute
			logger.Warn("⚠️ Увеличено время обновления OI до %v из-за частых ошибок", f.oiUpdateInterval)
		}

		return f.useEstimatedOI(symbols)
	}

	// Сбрасываем счетчик ошибок при успехе
	f.oiRetryCount = 0

	f.oiCacheMu.Lock()
	defer f.oiCacheMu.Unlock()

	updated := 0
	for symbol, oi := range realOI {
		if oi > 0 {
			f.oiCache[symbol] = oi
			updated++
			logger.Debug("📈 Real OI для %s: %.0f", symbol, oi)
		}
	}

	logger.Info("✅ BybitFetcher: обновлен реальный OI для %d/%d символов", updated, len(symbols))

	// Обновляем время последнего обновления
	f.lastOIUpdate = time.Now()

	// Для символов без OI используем эвристику
	if updated < len(symbols) {
		f.estimateMissingOI(symbols, realOI)
	}

	return nil
}

// calculateEstimatedOIFromStorage рассчитывает OI на основе данных из хранилища
func (f *BybitPriceFetcher) calculateEstimatedOIFromStorage(symbol string) float64 {
	if snapshot, exists := f.storage.GetCurrentSnapshot(symbol); exists {
		return f.calculateEstimatedOI(symbol, snapshot)
	}

	// Дефолтное значение
	return 10000
}

// estimateMissingOI оценивает OI для символов без данных
func (f *BybitPriceFetcher) estimateMissingOI(symbols []string, realOI map[string]float64) {
	estimatedCount := 0

	for _, symbol := range symbols {
		if _, hasRealOI := realOI[symbol]; !hasRealOI {
			if snapshot, exists := f.storage.GetCurrentSnapshot(symbol); exists && snapshot.VolumeUSD > 0 {
				// Улучшенная эвристика с учетом типа символа
				estimatedOI := f.calculateEstimatedOI(symbol, snapshot)
				f.oiCache[symbol] = estimatedOI
				estimatedCount++
				logger.Debug("📊 Расчетный OI для %s: %.0f (объем: %.0f)",
					symbol, estimatedOI, snapshot.VolumeUSD)
			}
		}
	}

	if estimatedCount > 0 {
		logger.Info("📊 BybitFetcher: использованы расчетные данные OI для %d символов", estimatedCount)
	}
}

// useEstimatedOI использует расчетный OI если API не доступно
func (f *BybitPriceFetcher) useEstimatedOI(symbols []string) error {
	f.oiCacheMu.Lock()
	defer f.oiCacheMu.Unlock()

	estimatedCount := 0

	for _, symbol := range symbols {
		if _, exists := f.oiCache[symbol]; !exists {
			if snapshot, exists := f.storage.GetCurrentSnapshot(symbol); exists && snapshot.VolumeUSD > 0 {
				// Рассчитываем OI
				estimatedOI := f.calculateEstimatedOI(symbol, snapshot)
				f.oiCache[symbol] = estimatedOI
				estimatedCount++
			}
		}
	}

	logger.Info("⚠️ BybitFetcher: использованы расчетные данные OI для %d символов", estimatedCount)

	// Обновляем время последнего обновления
	f.lastOIUpdate = time.Now()

	return nil
}

// calculateEstimatedOI рассчитывает OI на основе эвристики
func (f *BybitPriceFetcher) calculateEstimatedOI(symbol string, snapshot *storage.PriceSnapshot) float64 {
	logger.Debug("📊 calculateEstimatedOI для %s: VolumeUSD=%.0f, Price=%.8f",
		symbol, snapshot.VolumeUSD, snapshot.Price)

	// Базовый OI - 5% от объема
	baseOI := snapshot.VolumeUSD * 0.05

	logger.Debug("   Базовый OI (5%% от объема): %.0f", baseOI)

	// Корректируем для разных типов символов
	symbolUpper := strings.ToUpper(symbol)

	switch {
	case strings.Contains(symbolUpper, "BTC"):
		// BTC имеет высокий OI
		baseOI *= 1.5
		logger.Debug("   Корректировка для BTC: x1.5 = %.0f", baseOI)
	case strings.Contains(symbolUpper, "ETH"):
		baseOI *= 1.3
		logger.Debug("   Корректировка для ETH: x1.3 = %.0f", baseOI)
	case strings.Contains(symbolUpper, "SOL") || strings.Contains(symbolUpper, "BNB"):
		baseOI *= 1.2
		logger.Debug("   Корректировка для SOL/BNB: x1.2 = %.0f", baseOI)
	case strings.Contains(symbolUpper, "STABLE") || strings.Contains(symbolUpper, "USDT"):
		// Стабильные монеты имеют низкий OI
		baseOI *= 0.3
		logger.Debug("   Корректировка для USDT: x0.3 = %.0f", baseOI)
	case snapshot.Price < 0.01:
		// Очень дешевые монеты
		baseOI *= 0.5
		logger.Debug("   Корректировка для дешевой монеты: x0.5 = %.0f", baseOI)
	}

	// Ограничиваем разумными значениями
	if baseOI > 10_000_000_000 { // 10B
		logger.Warn("⚠️  OI превышает 10B, ограничиваем: %.0f -> 10B", baseOI)
		baseOI = 10_000_000_000
	}
	if baseOI < 10_000 { // Минимум 10K
		baseOI = 10_000
	}

	logger.Debug("   Итоговый OI: %.0f", baseOI)
	return baseOI
}

// ==================== ОСНОВНОЙ МЕТОД ПОЛУЧЕНИЯ ЦЕН ====================

func (f *BybitPriceFetcher) fetchPrices() error {
	startTime := time.Now()
	logger.Info("🔄 BybitFetcher: НАЧАЛО запроса цен в %s", startTime.Format("15:04:05.000"))

	// Добавляем retry логику
	var tickers *api.TickerResponse
	var err error

	for attempt := 1; attempt <= f.maxRetries; attempt++ {
		logger.Debug("🔄 Попытка %d/%d получения тикеров...", attempt, f.maxRetries)

		// Получаем тикеры
		tickers, err = f.client.GetTickers(f.client.Category())

		if err == nil && tickers != nil && tickers.RetCode == 0 && len(tickers.Result.List) > 0 {
			// Успешный запрос
			f.errorCount = 0
			f.lastFetchError = time.Time{} // сбрасываем
			break
		}

		// Проверяем ошибку
		if err != nil {
			logger.Warn("⚠️ Ошибка при получении тикеров (попытка %d/%d): %v", attempt, f.maxRetries, err)
		} else if tickers != nil && tickers.RetCode != 0 {
			logger.Warn("⚠️ API вернуло ошибку %d: %s (попытка %d/%d)",
				tickers.RetCode, tickers.RetMsg, attempt, f.maxRetries)
		} else if tickers == nil || len(tickers.Result.List) == 0 {
			logger.Warn("⚠️ Пустой ответ от API (попытка %d/%d)", attempt, f.maxRetries)
		}

		f.lastFetchError = time.Now()
		f.errorCount++

		// Если это была последняя попытка
		if attempt == f.maxRetries {
			logger.Error("❌ BybitFetcher: все попытки получения тикеров провалились")
			f.handleFetchFailure()
			return fmt.Errorf("failed to get tickers after %d retries: %v", f.maxRetries, err)
		}

		// Ждем перед следующей попыткой
		time.Sleep(f.retryDelay)
	}

	// Проверяем результат
	if tickers == nil || len(tickers.Result.List) == 0 {
		logger.Warn("⚠️ BybitFetcher: получен пустой список тикеров")
		f.handleEmptyTickers()
		return fmt.Errorf("empty tickers response")
	}

	logger.Debug("📊 BybitFetcher: получено %d тикеров, категория: %s",
		len(tickers.Result.List), tickers.Result.Category)

	now := time.Now()
	updatedCount := 0
	errorCount := 0
	oiUpdatedFromTicker := 0

	// Собираем все цены в массив
	var priceDataList []types.PriceData

	// Отладка: лог первых 5 тикеров
	if len(tickers.Result.List) > 0 {
		logger.Debug("🔍 Первые 5 тикеров из ответа API:")
		for i := 0; i < 5 && i < len(tickers.Result.List); i++ {
			ticker := tickers.Result.List[i]
			logger.Debug("   %d. %s: цена=%s, OI=%s, FundingRate='%s'",
				i+1, ticker.Symbol, ticker.LastPrice, ticker.OpenInterest, ticker.FundingRate)
		}
	}

	for i, ticker := range tickers.Result.List {
		// Парсим цену
		price, err := parseFloat(ticker.LastPrice)
		if err != nil {
			logger.Debug("⚠️  BybitFetcher: ошибка парсинга цены для %s: %v", ticker.Symbol, err)
			continue
		}

		// Парсим объем в базовой валюте
		volumeBase, _ := parseFloat(ticker.Volume24h)

		// Парсим объем в USDT (turnover)
		volumeUSD, _ := parseFloat(ticker.Turnover24h)

		// ==================== ИЗМЕНЕНИЕ: ПОЛУЧЕНИЕ OI ИЗ ТИКЕРА ====================
		// Используем OI из тикера вместо отдельного API вызова
		var openInterest float64
		oiFromTicker, oiErr := parseFloat(ticker.OpenInterest)

		if oiErr == nil && oiFromTicker > 0 {
			// OI есть в тикере - используем его
			openInterest = oiFromTicker

			// Обновляем кэш
			f.oiCacheMu.Lock()
			f.oiCache[ticker.Symbol] = openInterest
			f.oiCacheMu.Unlock()

			oiUpdatedFromTicker++
			logger.Debug("📊 BybitFetcher: OI из тикера для %s: %.0f", ticker.Symbol, openInterest)
		} else {
			// OI нет в тикере или ошибка парсинга - используем кэш или расчетное значение
			openInterest = f.getCachedOrEstimatedOI(ticker.Symbol)
			logger.Debug("📊 BybitFetcher: расчетный OI для %s: %.0f", ticker.Symbol, openInterest)
		}
		// ==================== КОНЕЦ ИЗМЕНЕНИЯ ====================

		// Проверка на реалистичность OI (оставляем существующую проверку)
		if openInterest > 0 && volumeUSD > 0 {
			ratio := openInterest / volumeUSD
			if ratio > 10 { // OI не должен быть больше 10x объема
				correctedOI := volumeUSD * 0.05
				logger.Debug("📉 Скорректированный OI для %s: %.0f (было %.0f)",
					ticker.Symbol, correctedOI, openInterest)
				openInterest = correctedOI
			}
		}

		// Также получаем фандинг для фьючерсов
		fundingRate := 0.0
		if ticker.FundingRate != "" {
			fundingRate, _ = parseFloat(ticker.FundingRate)
			if err != nil {
				logger.Debug("⚠️ Ошибка парсинга фандинга для %s: %v", ticker.Symbol, err)
			} else {
				logger.Debug("💰 BybitFetcher: %s фандинг = %.4f%%", ticker.Symbol, fundingRate*100)
			}
		}

		// Change24h
		change24h, _ := parseFloat(ticker.Price24hPcnt)

		// Получаем High24h и Low24h из тикер-данных
		high24h := price
		low24h := price

		// Парсим реальные значения если есть
		if ticker.High24h != "" {
			if h, err := parseFloat(ticker.High24h); err == nil {
				high24h = h
			}
		}
		if ticker.Low24h != "" {
			if l, err := parseFloat(ticker.Low24h); err == nil {
				low24h = l
			}
		}

		// Сохраняем цену со всеми параметрами
		if err := f.storage.StorePrice(
			ticker.Symbol,
			price,
			volumeBase,
			volumeUSD,
			now,
			openInterest,
			fundingRate,
			change24h,
			high24h,
			low24h,
		); err != nil {
			errorCount++
			logger.Error("❌ BybitFetcher: ошибка StorePrice для %s: %v", ticker.Symbol, err)
			continue
		}

		// НОВОЕ: Отправляем цену в свечной движок если он есть
		if f.candleSystem != nil {
			priceData := storage.PriceData{
				Symbol:       ticker.Symbol,
				Price:        price,
				Volume24h:    volumeBase,
				VolumeUSD:    volumeUSD,
				Timestamp:    now,
				OpenInterest: openInterest,
				FundingRate:  fundingRate,
				Change24h:    change24h,
				High24h:      high24h,
				Low24h:       low24h,
			}

			// Отправляем асинхронно чтобы не блокировать основной поток
			go func(pd storage.PriceData) {
				f.candleSystem.OnPriceUpdate(pd)
				logger.Debug("🕯️ Цена отправлена в свечной движок: %s %.6f",
					pd.Symbol, pd.Price)
			}(priceData)
		}

		// Добавляем в массив с полными данными
		priceDataList = append(priceDataList, types.PriceData{
			Symbol:       ticker.Symbol,
			Price:        price,
			Volume24h:    volumeBase,
			VolumeUSD:    volumeUSD,
			Timestamp:    now,
			OpenInterest: openInterest,
			FundingRate:  fundingRate,
			Change24h:    change24h,
			High24h:      high24h,
			Low24h:       low24h,
		})

		updatedCount++

		// Логируем каждый 50-й тикер
		if (i+1)%50 == 0 {
			logger.Debug("📈 BybitFetcher: обработано %d тикеров...", i+1)
		}
	}

	// Логируем статистику OI
	logger.Debug("📊 BybitFetcher: OI получено из тикеров: %d/%d символов",
		oiUpdatedFromTicker, len(tickers.Result.List))

	logger.Info("✅ BybitFetcher: успешно сохранено %d цен за %v, ошибок: %d",
		updatedCount, time.Since(startTime).Round(time.Millisecond), errorCount)

	// Публикуем одно событие со всеми ценами
	if updatedCount > 0 && f.eventBus != nil {
		event := types.Event{
			Type:      types.EventPriceUpdated,
			Source:    "bybit_price_fetcher",
			Data:      priceDataList,
			Timestamp: now,
		}

		err := f.eventBus.Publish(event)
		if err != nil {
			logger.Error("❌ BybitFetcher: ошибка публикации события: %v", err)
		} else {
			logger.Debug("📨 BybitFetcher: опубликовано событие с %d ценами", updatedCount)
		}
	}

	return nil
}

// НОВЫЙ МЕТОД: получает OI из кэша или расчетное значение
func (f *BybitPriceFetcher) getCachedOrEstimatedOI(symbol string) float64 {
	// Сначала проверяем кэш
	f.oiCacheMu.RLock()
	oi, exists := f.oiCache[symbol]
	f.oiCacheMu.RUnlock()

	if exists && oi > 0 {
		return oi
	}

	// Если нет в кэше, используем расчетное значение
	return f.calculateEstimatedOIFromStorage(symbol)
}

// handleFetchFailure обрабатывает ситуацию когда не удалось получить данные
func (f *BybitPriceFetcher) handleFetchFailure() {
	// Увеличиваем интервал между попытками при постоянных ошибках
	if f.errorCount > 10 {
		logger.Warn("⚠️ Много ошибок подряд (%d), переключение в безопасный режим", f.errorCount)
		// Можно временно отключить некоторые функции или уменьшить частоту запросов
	}
}

// handleEmptyTickers обрабатывает пустые тикеры
func (f *BybitPriceFetcher) handleEmptyTickers() {
	// Если тикеры пустые, возможно API вернуло ошибку
	logger.Warn("⚠️ Получен пустой список тикеров, проверьте подключение к интернету")

	// Если есть сохраненные данные, можем продолжать работу с ними
	storedSymbols := f.storage.GetSymbols()
	if len(storedSymbols) > 0 {
		logger.Info("📊 Есть %d сохраненных символов, продолжаем работу", len(storedSymbols))
	}
}

// ==================== МЕТОДЫ ЛИКВИДАЦИЙ ====================

// fetchLiquidationsLoop цикл получения ликвидаций
func (f *BybitPriceFetcher) fetchLiquidationsLoop(interval time.Duration) {
	defer f.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Первоначальный запрос
	if err := f.fetchLiquidations(); err != nil {
		logger.Warn("Ошибка первоначального получения ликвидаций: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := f.fetchLiquidations(); err != nil {
				logger.Warn("Ошибка получения ликвидаций: %v", err)
			}
		case <-f.stopChan:
			return
		}
	}
}

// fetchLiquidations получает данные о ликвидациях
func (f *BybitPriceFetcher) fetchLiquidations() error {
	if !f.liqEnabled {
		return nil
	}

	// Проверяем, прошло ли достаточно времени с последнего обновления
	if time.Since(f.lastLiqUpdate) < f.liqUpdateInterval {
		return nil
	}

	logger.Info("🔄 BybitFetcher: получение данных о ликвидациях...")

	// Получаем символы с наибольшим объемом
	symbols, err := f.storage.GetTopSymbolsByVolumeUSD(10) // Топ-10 символов
	if err != nil {
		logger.Warn("⚠️ Не удалось получить топ-символы: %v", err)
		return err
	}

	for _, symbolVolume := range symbols {
		symbol := symbolVolume.Symbol

		summary, err := f.client.GetLiquidationsSummary(symbol, 5*time.Minute) // За последние 5 минут
		if err != nil {
			logger.Debug("⚠️ Не удалось получить ликвидации для %s: %v", symbol, err)
			continue
		}

		metrics := &bybit.LiquidationMetrics{
			Symbol:         symbol,
			TotalVolumeUSD: summary["total_volume_usd"].(float64),
			LongLiqVolume:  summary["long_liq_volume"].(float64),
			ShortLiqVolume: summary["short_liq_volume"].(float64),
			LongLiqCount:   summary["long_liq_count"].(int),
			ShortLiqCount:  summary["short_liq_count"].(int),
			UpdateTime:     time.Now(),
		}

		f.liqCacheMu.Lock()
		f.liqCache[symbol] = metrics
		f.liqCacheMu.Unlock()

		if metrics.TotalVolumeUSD > 0 {
			logger.Debug("💥 Ликвидации %s: $%.0f (длинные: $%.0f, короткие: $%.0f)",
				symbol, metrics.TotalVolumeUSD, metrics.LongLiqVolume, metrics.ShortLiqVolume)
		}
	}

	f.lastLiqUpdate = time.Now()
	return nil
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

func (f *BybitPriceFetcher) IsRunning() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.running
}

func (f *BybitPriceFetcher) GetStats() map[string]interface{} {
	f.oiCacheMu.RLock()
	oiCount := len(f.oiCache)
	oiLastUpdate := f.lastOIUpdate
	f.oiCacheMu.RUnlock()

	f.volumeDeltaCacheMu.RLock()
	volumeDeltaCount := len(f.volumeDeltaCache)
	f.volumeDeltaCacheMu.RUnlock()

	f.liqCacheMu.RLock()
	liqCount := len(f.liqCache)
	f.liqCacheMu.RUnlock()

	// Добавляем статистику свечной системы
	var candleSystemStats map[string]interface{}
	if f.candleSystem != nil {
		candleSystemStats = f.candleSystem.GetStats()
	} else {
		candleSystemStats = map[string]interface{}{
			"status": "не инициализирована",
		}
	}

	return map[string]interface{}{
		"running":                 f.running,
		"type":                    "bybit",
		"oi_cache_size":           oiCount,
		"oi_last_update":          oiLastUpdate.Format("2006-01-02 15:04:05"),
		"oi_update_interval":      f.oiUpdateInterval.String(),
		"oi_retry_count":          f.oiRetryCount,
		"volume_delta_cache_size": volumeDeltaCount,
		"volume_delta_ttl":        f.volumeDeltaTTL.String(),
		"liq_cache_size":          liqCount,
		"liq_update_interval":     f.liqUpdateInterval.String(),
		"max_retries":             f.maxRetries,
		"error_count":             f.errorCount,
		"last_fetch_error":        f.lastFetchError.Format("2006-01-02 15:04:05"),
		"candle_system":           candleSystemStats,
	}
}

// SetCandleSystem устанавливает свечную систему (дополнительный метод)
func (f *BybitPriceFetcher) SetCandleSystem(candleSystem *candle.CandleSystem) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.candleSystem = candleSystem
	logger.Info("✅ BybitFetcher: свечная система установлена")
}

// GetCandleSystemStats получает статистику свечной системы
func (f *BybitPriceFetcher) GetCandleSystemStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.candleSystem == nil {
		return map[string]interface{}{
			"error": "свечная система не инициализирована",
		}
	}

	return f.candleSystem.GetStats()
}

// Вспомогательная функция для парсинга
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// NewPriceFetcherWithoutCandleSystem создает фетчер без свечной системы (для обратной совместимости)
func NewPriceFetcherWithoutCandleSystem(apiClient *bybit.BybitClient, storage storage.PriceStorageInterface,
	eventBus *events.EventBus) *BybitPriceFetcher {

	return &BybitPriceFetcher{
		client:   apiClient,
		storage:  storage,
		eventBus: eventBus,
		stopChan: make(chan struct{}),
		running:  false,
		oiCache:  make(map[string]float64),
		liqCache: make(map[string]*bybit.LiquidationMetrics),

		// Инициализация кэша дельты
		volumeDeltaCache: make(map[string]*volumeDeltaCache),
		volumeDeltaTTL:   30 * time.Second,

		// Настройки OI
		oiEnabled:        true,
		oiUpdateInterval: 5 * time.Minute,
		lastOIUpdate:     time.Now(),
		oiRetryCount:     0,

		// Настройки ликвидаций
		liqEnabled:        true,
		liqUpdateInterval: 1 * time.Minute,
		lastLiqUpdate:     time.Now(),

		// Настройки timeout и retry
		maxRetries: 3,
		retryDelay: 2 * time.Second,
		errorCount: 0,
	}
}
