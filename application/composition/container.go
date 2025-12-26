// application/composition/container.go
package composition

import (
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
)

// Container - DI контейнер
type Container struct {
	Config *config.Config

	// Инфраструктура
	BybitClient *bybit.BybitClient

	// Адаптеры
	// MarketFetcher  ports.MarketFetcher
	// SignalDetector ports.SignalDetector
	// Notifier       ports.Notifier
	// PriceStorage   ports.PriceStorage
}

// NewContainer создает и настраивает контейнер
func NewContainer(cfg *config.Config) (*Container, error) {
	c := &Container{
		Config: cfg,
	}

	// 1. Создаем инфраструктурные компоненты
	c.BybitClient = bybit.NewBybitClient(cfg)

	return c, nil
}
