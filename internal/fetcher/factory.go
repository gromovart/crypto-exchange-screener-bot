// internal/fetcher/factory.go
package fetcher

import (
	"crypto-exchange-screener-bot/internal/api"
	binance "crypto-exchange-screener-bot/internal/api/exchanges/binance"
	bybit "crypto-exchange-screener-bot/internal/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
)

// Factory - фабрика для создания PriceFetcher
type Factory struct{}

// NewPriceFetcherFromConfig создает PriceFetcher из конфигурации
func (f *Factory) NewPriceFetcherFromConfig(
	apiClient api.ExchangeClient,
	storage storage.PriceStorage,
	eventBus *events.EventBus,
	cfg *config.Config,
) PriceFetcher {

	// Определяем тип фетчера на основе конфигурации
	switch cfg.FuturesCategory {
	case "binance_spot", "binance_futures", "binance":
		// Если клиент Binance
		if binanceClient, ok := apiClient.(*binance.BinanceClient); ok {
			return NewBinancePriceFetcher(binanceClient, storage, eventBus)
		}
	default:
		// По умолчанию используем Bybit
		if bybitClient, ok := apiClient.(*bybit.BybitClient); ok {
			return NewPriceFetcher(bybitClient, storage, eventBus)
		}
	}

	// Fallback: используем Bybit как дефолт
	if bybitClient, ok := apiClient.(*bybit.BybitClient); ok {
		return NewPriceFetcher(bybitClient, storage, eventBus)
	}

	// Если не удалось определить тип, возвращаем nil
	return nil
}
