// application/layer_manager/factory.go
package layer_manager

import (
	"crypto-exchange-screener-bot/application/layer_manager/layers"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
)

// LayerFactory фабрика для создания слоев
type LayerFactory struct {
	config *config.Config
}

// NewLayerFactory создает фабрику слоев
func NewLayerFactory(cfg *config.Config) *LayerFactory {
	return &LayerFactory{
		config: cfg,
	}
}

// CreateLayers создает все слои
func (lf *LayerFactory) CreateLayers() (*layers.LayerRegistry, error) {
	logger.Info("🏗️  Создание слоев через LayerFactory...")

	// 1. Создаем реестр слоев
	registry := layers.NewLayerRegistry()

	// 2. Создаем слой инфраструктуры
	logger.Debug("Создание InfrastructureLayer...")
	infraLayer := layers.NewInfrastructureLayer(lf.config)
	if err := registry.Register(infraLayer); err != nil {
		return nil, fmt.Errorf("не удалось зарегистрировать InfrastructureLayer: %w", err)
	}

	// 3. Инициализируем слой инфраструктуры
	// logger.Debug("Инициализация InfrastructureLayer...")
	// if err := infraLayer.Initialize(); err != nil {
	// 	return nil, fmt.Errorf("не удалось инициализировать InfrastructureLayer: %w", err)
	// }

	// УБИРАЕМ запуск здесь - только инициализация
	// 4. Запускаем слой инфраструктуры
	// logger.Debug("Запуск InfrastructureLayer...")
	// if err := infraLayer.Start(); err != nil {
	// 	return nil, fmt.Errorf("не удалось запустить InfrastructureLayer: %w", err)
	// }

	// 5. Создаем слой ядра (зависит от инфраструктуры)
	logger.Debug("Создание CoreLayer...")
	coreLayer := layers.NewCoreLayer(lf.config, infraLayer)
	if err := registry.Register(coreLayer); err != nil {
		return nil, fmt.Errorf("не удалось зарегистрировать CoreLayer: %w", err)
	}

	// // 6. Инициализируем слой ядра
	// logger.Debug("Инициализация CoreLayer...")
	// if err := coreLayer.Initialize(); err != nil {
	// 	return nil, fmt.Errorf("не удалось инициализировать CoreLayer: %w", err)
	// }

	// 7. Создаем слой доставки (зависит от ядра)
	logger.Debug("Создание DeliveryLayer...")
	deliveryLayer := layers.NewDeliveryLayer(lf.config, coreLayer)
	if err := registry.Register(deliveryLayer); err != nil {
		return nil, fmt.Errorf("не удалось зарегистрировать DeliveryLayer: %w", err)
	}

	// 8. Инициализируем слой доставки
	// logger.Debug("Инициализация DeliveryLayer...")
	// if err := deliveryLayer.Initialize(); err != nil {
	// 	return nil, fmt.Errorf("не удалось инициализировать DeliveryLayer: %w", err)
	// }

	// 9. Настраиваем зависимости между слоями
	// logger.Debug("Валидация зависимостей слоев...")
	// if err := registry.ValidateDependencies(); err != nil {
	// 	return nil, fmt.Errorf("ошибка зависимостей слоев: %w", err)
	// }

	logger.Info("✅ Все слои созданы и инициализированы")
	return registry, nil
}
