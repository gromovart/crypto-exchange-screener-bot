// internal/fetcher/factory.go
package fetcher

import (
	"crypto_exchange_screener_bot/internal/config"
	"crypto_exchange_screener_bot/internal/events"
	"crypto_exchange_screener_bot/internal/storage"
	"crypto_exchange_screener_bot/internal/types/api/binance"
	"crypto_exchange_screener_bot/internal/types/api/bybit"
)

// Factory - фабрика для создания PriceFetcher
type Factory struct{}

// NewPriceFetcherFromConfig создает PriceFetcher из конфигурации
func (f *Factory) NewPriceFetcherFromConfig(
	apiClient bybit.ExchangeClient,
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
