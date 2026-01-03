// internal/adapters/market/bybit_fetcher.go
package market

import (
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
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
	client   *bybit.BybitClient
	storage  storage.PriceStorage
	eventBus *events.EventBus
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
	config   *config.Config

	// Кэш для Open Interest
	oiCache   map[string]float64
	oiCacheMu sync.RWMutex

	// Настройки OI
	oiEnabled        bool
	oiUpdateInterval time.Duration
	lastOIUpdate     time.Time
	oiRetryCount     int
}

// NewPriceFetcher создает новый PriceFetcher
func NewPriceFetcher(apiClient *bybit.BybitClient, storage storage.PriceStorage, eventBus *events.EventBus) *BybitPriceFetcher {
	return &BybitPriceFetcher{
		client:   apiClient,
		storage:  storage,
		eventBus: eventBus,
		stopChan: make(chan struct{}),
		running:  false,
		oiCache:  make(map[string]float64),

		// Настройки OI
		oiEnabled:        true,
		oiUpdateInterval: 5 * time.Minute, // Обновлять OI каждые 5 минут
		lastOIUpdate:     time.Now(),
		oiRetryCount:     0,
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

	go func() {
		defer f.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Первоначальный запрос
		if err := f.fetchPrices(); err != nil {
			logger.Warn("Ошибка первоначального получения цен: %v", err)
		}

		// Запускаем отдельный горутин для получения OI
		f.wg.Add(1)
		go f.fetchOpenInterestLoop(interval * 3) // Получаем OI реже

		for {
			select {
			case <-ticker.C:
				if err := f.fetchPrices(); err != nil {
					logger.Warn("Ошибка получения цен: %v", err)
				}
			case <-f.stopChan:
				return
			}
		}
	}()

	logger.Info("✅ PriceFetcher запущен")
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

// fetchOpenInterestLoop циклически получает Open Interest
func (f *BybitPriceFetcher) fetchOpenInterestLoop(interval time.Duration) {
	defer f.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Первоначальный запрос
	if err := f.fetchOpenInterest(); err != nil {
		logger.Warn("Ошибка первоначального получения OI: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := f.fetchOpenInterest(); err != nil {
				logger.Warn("Ошибка получения OI: %v", err)
			}
		case <-f.stopChan:
			return
		}
	}
}

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

// getOpenInterestForSymbol получает OI для конкретного символа
func (f *BybitPriceFetcher) getOpenInterestForSymbol(symbol string) float64 {
	f.oiCacheMu.RLock()
	oi, exists := f.oiCache[symbol]
	f.oiCacheMu.RUnlock()

	if exists && oi > 0 {
		return oi
	}

	// Если нет в кэше, ПРОБУЕМ ПОЛУЧИТЬ С API
	oi, err := f.client.GetOpenInterest(symbol)
	if err != nil {
		logger.Debug("⚠️ Не удалось получить OI для %s: %v", symbol, err)
		// Используем эвристику
		return f.calculateEstimatedOIFromStorage(symbol)
	}

	// Кэшируем
	f.oiCacheMu.Lock()
	f.oiCache[symbol] = oi
	f.oiCacheMu.Unlock()

	if oi > 0 {
		logger.Debug("📊 BybitFetcher: получен OI для %s: %.0f", symbol, oi)
	}

	return oi
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
	// Базовый OI - 5% от объема
	baseOI := snapshot.VolumeUSD * 0.05

	// Корректируем для разных типов символов
	symbolUpper := strings.ToUpper(symbol)

	switch {
	case strings.Contains(symbolUpper, "BTC"):
		// BTC имеет высокий OI
		baseOI *= 1.5
	case strings.Contains(symbolUpper, "ETH"):
		baseOI *= 1.3
	case strings.Contains(symbolUpper, "SOL") || strings.Contains(symbolUpper, "BNB"):
		baseOI *= 1.2
	case strings.Contains(symbolUpper, "STABLE") || strings.Contains(symbolUpper, "USDT"):
		// Стабильные монеты имеют низкий OI
		baseOI *= 0.3
	case snapshot.Price < 0.01:
		// Очень дешевые монеты
		baseOI *= 0.5
	}

	// Ограничиваем разумными значениями
	if baseOI > 10_000_000_000 { // 10B
		baseOI = 10_000_000_000
	}
	if baseOI < 10_000 { // Минимум 10K
		baseOI = 10_000
	}

	return baseOI
}

func (f *BybitPriceFetcher) fetchPrices() error {
	logger.Debug("🔄 BybitFetcher: начало получения цен...")

	// Получаем тикеры
	tickers, err := f.client.GetTickers(f.client.Category())
	if err != nil {
		logger.Error("❌ BybitFetcher: ошибка получения тикеров: %v", err)
		return fmt.Errorf("failed to get tickers: %w", err)
	}

	logger.Debug("📊 BybitFetcher: получено %d тикеров", len(tickers.Result.List))

	now := time.Now()
	updatedCount := 0
	errorCount := 0

	// Собираем все цены в массив
	var priceDataList []types.PriceData

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

		// Проверка на аномальные значения
		if volumeUSD > 1000000000 && price < 0.1 { // Объем > 1B при цене < $0.1
			logger.Warn("⚠️ Подозрительный объем для %s: цена=$%f, объем=$%.0f",
				ticker.Symbol, price, volumeUSD)
			// Используем скорректированный объем
			volumeUSD = volumeUSD / 1000 // Уменьшаем в 1000 раз
		}

		// Если turnover недоступен, используем расчетный объем
		if volumeUSD == 0 && price > 0 && volumeBase > 0 {
			volumeUSD = price * volumeBase
			logger.Debug("📝 BybitFetcher: расчетный VolumeUSD для %s: %f", ticker.Symbol, volumeUSD)
		}

		// 🔴 КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Используем метод getOpenInterestForSymbol
		openInterest := f.getOpenInterestForSymbol(ticker.Symbol)

		// Логируем OI если есть
		if openInterest > 0 {
			logger.Debug("📊 BybitFetcher: %s OI=%.0f", ticker.Symbol, openInterest)
		}

		// Также получаем фандинг для фьючерсов
		fundingRate := 0.0

		if ticker.FundingRate != "" {
			fundingRate, _ = parseFloat(ticker.FundingRate)
			logger.Debug("💰 BybitFetcher: %s фандинг = %.4f%%", ticker.Symbol, fundingRate*100)
		}

		// Change24h
		change24h, _ := parseFloat(ticker.Price24hPcnt)

		// Получаем High24h и Low24h из тикер-данных
		high24h := price
		low24h := price

		// Временная логика: если цена растет, устанавливаем high24h выше
		if change24h > 0 {
			high24h = price * (1 + change24h/100)
			low24h = price * (1 - change24h/200)
		} else if change24h < 0 {
			high24h = price * (1 - change24h/200)
			low24h = price * (1 + change24h/100)
		}

		logger.Debug("💰 BybitFetcher: сохранение %s: price=%f, volume24h=%f, OI=%f",
			ticker.Symbol, price, volumeUSD, openInterest)

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

	logger.Info("✅ BybitFetcher: успешно сохранено %d цен, ошибок: %d", updatedCount, errorCount)

	// Публикуем одно событие со всеми ценами
	if updatedCount > 0 && f.eventBus != nil {
		event := events.Event{
			Type:      events.EventPriceUpdated,
			Source:    "bybit_price_fetcher",
			Data:      priceDataList,
			Timestamp: now,
		}

		err := f.eventBus.Publish(event)
		if err != nil {
			logger.Error("❌ BybitFetcher: ошибка публикации события: %v", err)
		} else {
			logger.Info("📨 BybitFetcher: опубликовано событие с %d ценами", updatedCount)
		}
	}

	return nil
}

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

	return map[string]interface{}{
		"running":            f.running,
		"type":               "bybit",
		"oi_cache_size":      oiCount,
		"oi_last_update":     oiLastUpdate.Format("2006-01-02 15:04:05"),
		"oi_update_interval": f.oiUpdateInterval.String(),
		"oi_retry_count":     f.oiRetryCount,
	}
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
