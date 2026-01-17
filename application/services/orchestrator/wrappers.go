// application/services/orchestrator/wrappers.go
package orchestrator

import (
	"crypto-exchange-screener-bot/application/pipeline"
	fetcher "crypto-exchange-screener-bot/internal/adapters/market"
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/internal/core/domain/signals/engine"
	telegrambot "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	redis "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"fmt"
	"sync"
	"time"
)

// ==================== УНИВЕРСАЛЬНАЯ ОБЕРТКА ====================

// UniversalServiceWrapper универсальная обертка для любого сервиса
type UniversalServiceWrapper struct {
	name         string
	service      interface{}
	defaultStart bool // Запускать ли при Start() если нет метода
	defaultStop  bool // Останавливать ли при Stop() если нет метода
	started      bool
	startTime    time.Time
	mu           sync.RWMutex
}

// NewUniversalServiceWrapper создает универсальную обертку
func NewUniversalServiceWrapper(name string, service interface{}, defaultStart, defaultStop bool) *UniversalServiceWrapper {
	return &UniversalServiceWrapper{
		name:         name,
		service:      service,
		defaultStart: defaultStart,
		defaultStop:  defaultStop,
		started:      false,
		startTime:    time.Time{},
	}
}

func (w *UniversalServiceWrapper) Name() string {
	// Пытаемся получить имя из сервиса
	if named, ok := w.service.(interface{ Name() string }); ok {
		return named.Name()
	}
	return w.name
}

func (w *UniversalServiceWrapper) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return nil
	}

	// Проверяем что сервис не nil
	if w.service == nil {
		return fmt.Errorf("сервис %s не инициализирован", w.name)
	}

	// Пытаемся вызвать Start() если есть
	if starter, ok := w.service.(interface{ Start() error }); ok {
		err := starter.Start()
		if err == nil {
			w.started = true
			w.startTime = time.Now()
		}
		return err
	}

	// Если нет метода Start, но defaultStart=true
	if w.defaultStart {
		w.started = true
		w.startTime = time.Now()
		return nil
	}

	// Если нет метода Start и defaultStart=false - это НЕ ошибка
	// Просто пропускаем запуск, так как сервис не требует запуска
	w.started = true // Отмечаем как "запущенный" для отслеживания состояния
	w.startTime = time.Now()
	return nil
}

func (w *UniversalServiceWrapper) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return nil
	}

	// Пытаемся вызвать Stop() если есть
	if stopper, ok := w.service.(interface{ Stop() error }); ok {
		err := stopper.Stop()
		if err == nil {
			w.started = false
		}
		return err
	}

	// Если нет метода Stop, но defaultStop=true
	if w.defaultStop {
		w.started = false
		return nil
	}

	// Если нет метода Stop и defaultStop=false - это НЕ ошибка
	// Просто пропускаем остановку
	w.started = false
	return nil
}

func (w *UniversalServiceWrapper) State() ServiceState {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Пытаемся получить State() если есть
	if stater, ok := w.service.(interface{ State() ServiceState }); ok {
		return stater.State()
	}

	// Пытаемся получить State() как string если есть
	if stater, ok := w.service.(interface{ State() string }); ok {
		stateStr := stater.State()
		switch stateStr {
		case "running":
			return StateRunning
		case "stopped":
			return StateStopped
		case "error":
			return StateError
		default:
			return StateStopped
		}
	}

	// Используем наше отслеживание состояния
	if w.started {
		return StateRunning
	}
	return StateStopped
}

func (w *UniversalServiceWrapper) HealthCheck() bool {
	// Пытаемся вызвать HealthCheck() если есть
	if checker, ok := w.service.(interface{ HealthCheck() bool }); ok {
		return checker.HealthCheck()
	}

	// Пытаемся проверить IsRunning() если есть
	if runner, ok := w.service.(interface{ IsRunning() bool }); ok {
		return runner.IsRunning()
	}

	// Базовая проверка
	return w.service != nil
}

func (w *UniversalServiceWrapper) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Пытаемся получить IsRunning() если есть
	if runner, ok := w.service.(interface{ IsRunning() bool }); ok {
		return runner.IsRunning()
	}

	return w.started
}

func (w *UniversalServiceWrapper) GetExtendedStatus() map[string]interface{} {
	status := map[string]interface{}{
		"name":         w.Name(),
		"running":      w.IsRunning(),
		"state":        w.State(),
		"healthy":      w.HealthCheck(),
		"has_service":  w.service != nil,
		"wrapper_type": "UniversalServiceWrapper",
	}

	if w.started && !w.startTime.IsZero() {
		status["start_time"] = w.startTime
		status["uptime"] = time.Since(w.startTime).String()
	}

	// Пытаемся получить статус из сервиса
	if statuser, ok := w.service.(interface{ GetStatus() map[string]interface{} }); ok {
		extended := statuser.GetStatus()
		for k, v := range extended {
			status[k] = v
		}
	}

	// Пытаемся получить статистику из сервиса
	if stater, ok := w.service.(interface{ GetStats() map[string]interface{} }); ok {
		stats := stater.GetStats()
		for k, v := range stats {
			status[k] = v
		}
	}

	return status
}

// ==================== СПЕЦИАЛИЗИРОВАННЫЕ ОБЕРТКИ ====================

// PriceFetcherWrapper обертка для PriceFetcher
type PriceFetcherWrapper struct {
	fetcher.PriceFetcher
}

func (w *PriceFetcherWrapper) Name() string { return "PriceFetcher" }
func (w *PriceFetcherWrapper) Start() error {
	if w.PriceFetcher == nil {
		return fmt.Errorf("PriceFetcher не инициализирован")
	}
	// PriceFetcher.Start требует параметр interval - используем конфигурационный интервал
	return w.PriceFetcher.Start(10 * time.Second) // Берем из конфигурации
}
func (w *PriceFetcherWrapper) Stop() error {
	if w.PriceFetcher == nil {
		return nil
	}
	w.PriceFetcher.Stop()
	return nil
}
func (w *PriceFetcherWrapper) State() ServiceState {
	if w.PriceFetcher == nil {
		return StateStopped
	}
	// Проверяем, запущен ли фетчер
	return StateRunning
}
func (w *PriceFetcherWrapper) HealthCheck() bool {
	return w.PriceFetcher != nil
}
func (w *PriceFetcherWrapper) GetStats() map[string]interface{} {
	if w.PriceFetcher == nil {
		return map[string]interface{}{"error": "не инициализирован"}
	}
	return w.PriceFetcher.GetStats()
}

// DatabaseServiceWrapper обертка для DatabaseService
type DatabaseServiceWrapper struct {
	*database.DatabaseService
}

func (w *DatabaseServiceWrapper) Name() string { return "DatabaseService" }
func (w *DatabaseServiceWrapper) Start() error {
	if w.DatabaseService == nil {
		return fmt.Errorf("DatabaseService не инициализирован")
	}
	return w.DatabaseService.Start()
}
func (w *DatabaseServiceWrapper) Stop() error {
	if w.DatabaseService == nil {
		return nil
	}
	return w.DatabaseService.Stop()
}
func (w *DatabaseServiceWrapper) State() ServiceState {
	if w.DatabaseService == nil {
		return StateStopped
	}
	switch w.DatabaseService.State() {
	case database.StateRunning:
		return StateRunning
	case database.StateError:
		return StateError
	case database.StateStopped:
		return StateStopped
	default:
		return StateStopped
	}
}
func (w *DatabaseServiceWrapper) HealthCheck() bool {
	if w.DatabaseService == nil {
		return false
	}
	return w.DatabaseService.HealthCheck()
}

// RedisServiceWrapper обертка для RedisService
type RedisServiceWrapper struct {
	*redis.RedisService
}

func (w *RedisServiceWrapper) Name() string { return "RedisService" }
func (w *RedisServiceWrapper) Start() error {
	if w.RedisService == nil {
		return fmt.Errorf("RedisService не инициализирован")
	}
	return w.RedisService.Start()
}
func (w *RedisServiceWrapper) Stop() error {
	if w.RedisService == nil {
		return nil
	}
	return w.RedisService.Stop()
}
func (w *RedisServiceWrapper) State() ServiceState {
	if w.RedisService == nil {
		return StateStopped
	}
	switch w.RedisService.State() {
	case redis.StateRunning:
		return StateRunning
	case redis.StateError:
		return StateError
	case redis.StateStopped:
		return StateStopped
	default:
		return StateStopped
	}
}
func (w *RedisServiceWrapper) HealthCheck() bool {
	if w.RedisService == nil {
		return false
	}
	return w.RedisService.HealthCheck()
}

// EventBusWrapper обертка для EventBus
type EventBusWrapper struct {
	*events.EventBus
}

func (w *EventBusWrapper) Name() string {
	if w.EventBus == nil {
		return "EventBus"
	}
	return w.EventBus.Name()
}
func (w *EventBusWrapper) Start() error {
	if w.EventBus == nil {
		return fmt.Errorf("EventBus не инициализирован")
	}
	w.EventBus.Start()
	return nil
}
func (w *EventBusWrapper) Stop() error {
	if w.EventBus == nil {
		return nil
	}
	w.EventBus.Stop()
	return nil
}
func (w *EventBusWrapper) State() ServiceState {
	if w.EventBus == nil {
		return StateStopped
	}
	if w.EventBus.IsRunning() {
		return StateRunning
	}
	return StateStopped
}
func (w *EventBusWrapper) HealthCheck() bool {
	if w.EventBus == nil {
		return false
	}
	return w.EventBus.HealthCheck()
}

// ==================== ПРОСТЫЕ ОБЕРТКИ ====================

// PriceStorageWrapper обертка для PriceStorage
type PriceStorageWrapper struct {
	storage.PriceStorage
}

func (w *PriceStorageWrapper) Name() string { return "PriceStorage" }
func (w *PriceStorageWrapper) Start() error { return nil }
func (w *PriceStorageWrapper) Stop() error  { return nil }
func (w *PriceStorageWrapper) State() ServiceState {
	if w.PriceStorage == nil {
		return StateStopped
	}
	return StateRunning
}
func (w *PriceStorageWrapper) HealthCheck() bool {
	return w.PriceStorage != nil
}

// CandleSystemWrapper обертка для CandleSystem
type CandleSystemWrapper struct {
	*candle.CandleSystem
}

func (w *CandleSystemWrapper) Name() string { return "CandleSystem" }
func (w *CandleSystemWrapper) Start() error {
	if w.CandleSystem == nil {
		return nil
	}
	return w.CandleSystem.Start()
}
func (w *CandleSystemWrapper) Stop() error {
	if w.CandleSystem == nil {
		return nil
	}
	return w.CandleSystem.Stop()
}
func (w *CandleSystemWrapper) State() ServiceState {
	if w.CandleSystem == nil {
		return StateStopped
	}
	return StateRunning
}
func (w *CandleSystemWrapper) HealthCheck() bool {
	return w.CandleSystem != nil
}

// AnalysisEngineWrapper обертка для AnalysisEngine
type AnalysisEngineWrapper struct {
	*engine.AnalysisEngine
}

func (w *AnalysisEngineWrapper) Name() string { return "AnalysisEngine" }
func (w *AnalysisEngineWrapper) Start() error {
	if w.AnalysisEngine == nil {
		return fmt.Errorf("AnalysisEngine не инициализирован")
	}
	return w.AnalysisEngine.Start()
}
func (w *AnalysisEngineWrapper) Stop() error {
	if w.AnalysisEngine == nil {
		return nil
	}
	w.AnalysisEngine.Stop()
	return nil
}
func (w *AnalysisEngineWrapper) State() ServiceState {
	if w.AnalysisEngine == nil {
		return StateStopped
	}
	return StateRunning
}
func (w *AnalysisEngineWrapper) HealthCheck() bool {
	return w.AnalysisEngine != nil
}

// SignalPipelineWrapper обертка для SignalPipeline
type SignalPipelineWrapper struct {
	*pipeline.SignalPipeline
}

func (w *SignalPipelineWrapper) Name() string { return "SignalPipeline" }
func (w *SignalPipelineWrapper) Start() error {
	if w.SignalPipeline == nil {
		return fmt.Errorf("SignalPipeline не инициализирован")
	}
	w.SignalPipeline.Start()
	return nil
}
func (w *SignalPipelineWrapper) Stop() error {
	if w.SignalPipeline == nil {
		return nil
	}
	return nil
}
func (w *SignalPipelineWrapper) State() ServiceState {
	if w.SignalPipeline == nil {
		return StateStopped
	}
	return StateRunning
}
func (w *SignalPipelineWrapper) HealthCheck() bool {
	return w.SignalPipeline != nil
}

// NotificationServiceWrapper обертка для CompositeNotificationService
type NotificationServiceWrapper struct {
	*notifier.CompositeNotificationService
}

func (w *NotificationServiceWrapper) Name() string { return "NotificationService" }
func (w *NotificationServiceWrapper) Start() error { return nil }
func (w *NotificationServiceWrapper) Stop() error  { return nil }
func (w *NotificationServiceWrapper) State() ServiceState {
	if w.CompositeNotificationService == nil {
		return StateStopped
	}
	return StateRunning
}
func (w *NotificationServiceWrapper) HealthCheck() bool {
	return w.CompositeNotificationService != nil
}

// TelegramBotWrapper обертка для TelegramBot
type TelegramBotWrapper struct {
	*telegrambot.TelegramBot
}

func (w *TelegramBotWrapper) Name() string { return "TelegramBot" }
func (w *TelegramBotWrapper) Start() error {
	if w.TelegramBot == nil {
		return nil
	}
	return w.TelegramBot.StartPolling()
}
func (w *TelegramBotWrapper) Stop() error {
	if w.TelegramBot == nil {
		return nil
	}
	return nil
}
func (w *TelegramBotWrapper) State() ServiceState {
	if w.TelegramBot == nil {
		return StateStopped
	}
	return StateRunning
}
func (w *TelegramBotWrapper) HealthCheck() bool {
	return w.TelegramBot != nil
}
