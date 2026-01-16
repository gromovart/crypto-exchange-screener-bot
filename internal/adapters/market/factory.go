// internal/adapters/market/factory.go
package market

import (
	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
)

// MarketFetcherFactory - фабрика для создания фетчеров
type MarketFetcherFactory struct {
	config *config.Config
}

// NewMarketFetcherFactory создает новую фабрику
func NewMarketFetcherFactory(cfg *config.Config) *MarketFetcherFactory {
	return &MarketFetcherFactory{
		config: cfg,
	}
}

// CreateBybitFetcher создает Bybit фетчер (обновленный)
func (f *MarketFetcherFactory) CreateBybitFetcher(
	storage storage.PriceStorage,
	eventBus *events.EventBus,
) (*BybitPriceFetcher, error) {
	// Создаем клиент Bybit
	bybitClient := bybit.NewBybitClient(f.config)

	// Тестируем подключение
	if err := bybitClient.TestConnection(); err != nil {
		return nil, err
	}

	// Создаем фетчер без свечной системы (для обратной совместимости)
	fetcher := NewPriceFetcherWithoutCandleSystem(bybitClient, storage, eventBus)
	return fetcher, nil
}

// CreateBybitFetcherWithCandleSystem создает Bybit фетчер со свечной системой
func (f *MarketFetcherFactory) CreateBybitFetcherWithCandleSystem(
	storage storage.PriceStorage,
	eventBus *events.EventBus,
	candleSystem *candle.CandleSystem,
) (*BybitPriceFetcher, error) {
	// Создаем клиент Bybit
	bybitClient := bybit.NewBybitClient(f.config)

	// Тестируем подключение
	if err := bybitClient.TestConnection(); err != nil {
		return nil, err
	}

	// Создаем фетчер со свечной системой
	fetcher := NewPriceFetcher(bybitClient, storage, eventBus, candleSystem)
	return fetcher, nil
}

// CreateBybitFetcherWithClient создает Bybit фетчер с готовым клиентом (обновленный)
func (f *MarketFetcherFactory) CreateBybitFetcherWithClient(
	client *bybit.BybitClient,
	storage storage.PriceStorage,
	eventBus *events.EventBus,
	candleSystem *candle.CandleSystem,
) *BybitPriceFetcher {
	return NewPriceFetcher(client, storage, eventBus, candleSystem)
}

// CreateSimpleBybitFetcher создает простой Bybit фетчер
func (f *MarketFetcherFactory) CreateSimpleBybitFetcher(
	storage storage.PriceStorage,
) (*BybitPriceFetcher, error) {
	// Создаем клиент Bybit
	bybitClient := bybit.NewBybitClient(f.config)

	// Создаем фетчер без EventBus и без свечной системы
	fetcher := NewPriceFetcherWithoutCandleSystem(bybitClient, storage, nil)
	return fetcher, nil
}
