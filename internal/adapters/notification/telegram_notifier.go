// internal/adapters/notification/telegram_notifier.go
package notification

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	events "crypto-exchange-screener-bot/internal/types"
	"log"
	"time"
)

// TelegramNotifier - единая точка взаимодействия с Telegram через EventBus
type TelegramNotifier struct {
	mainBot  *bot.TelegramBot // Основной чат - ИЗМЕНЕНО тип
	eventBus events.EventBus  // Шина событий для публикации
	enabled  bool
	stats    map[string]interface{}
}

// NewTelegramNotifier создает новый нотификатор с EventBus
func NewTelegramNotifier(cfg *config.Config, eventBus events.EventBus) *TelegramNotifier {
	if cfg == nil || !cfg.TelegramEnabled || cfg.TelegramChatID == "" {
		log.Println("⚠️ TelegramNotifier: Telegram отключен или ChatID не указан")
		return nil
	}

	// Основной бот для торговых сигналов
	// ИЗМЕНЕНО: используем новую функцию GetOrCreateBot из пакета bot
	mainBot := bot.GetOrCreateBot(cfg)
	if mainBot == nil {
		log.Println("⚠️ TelegramNotifier: Не удалось создать основной бот")
		return nil
	}

	return &TelegramNotifier{
		mainBot:  mainBot,
		eventBus: eventBus,
		enabled:  true,
		stats: map[string]interface{}{
			"trading_signals_sent": 0,
			"system_messages_sent": 0,
			"counter_signals_sent": 0,
			"errors":               0,
			"type":                 "telegram_notifier",
		},
	}
}

// SetTelegramBot устанавливает Telegram бота (для обратной совместимости)
func (tn *TelegramNotifier) SetTelegramBot(bot *bot.TelegramBot) { // ИЗМЕНЕНО тип
	tn.mainBot = bot
}

// Send отправляет торговый сигнал через EventBus
func (tn *TelegramNotifier) Send(signal types.TrendSignal) error {
	if !tn.enabled || tn.eventBus == nil {
		return nil
	}

	// Публикуем событие в EventBus
	event := events.Event{
		Type:      events.EventSignalDetected,
		Source:    "telegram_notifier",
		Data:      signal,
		Timestamp: time.Now(),
	}

	err := tn.eventBus.Publish(event)
	if err != nil {
		tn.stats["errors"] = tn.stats["errors"].(int) + 1
		log.Printf("❌ TelegramNotifier: Ошибка публикации события: %v", err)
		return err
	}

	tn.stats["trading_signals_sent"] = tn.stats["trading_signals_sent"].(int) + 1
	log.Printf("✅ TelegramNotifier: Событие опубликовано в EventBus: %s %.2f%%",
		signal.Symbol, signal.ChangePercent)

	return nil
}

// PublishCounterSignal публикует сигнал от CounterAnalyzer
func (tn *TelegramNotifier) PublishCounterSignal(
	symbol string,
	direction string,
	change float64,
	signalCount int,
	maxSignals int,
	additionalData map[string]interface{},
) error {
	if !tn.enabled || tn.eventBus == nil {
		return nil
	}

	// Создаем структуру данных для CounterAnalyzer
	counterSignal := map[string]interface{}{
		"symbol":          symbol,
		"direction":       direction,
		"change":          change,
		"signal_count":    signalCount,
		"max_signals":     maxSignals,
		"source":          "counter_analyzer",
		"timestamp":       time.Now(),
		"additional_data": additionalData,
	}

	event := events.Event{
		Type:      events.EventSignalDetected,
		Source:    "counter_analyzer",
		Data:      counterSignal,
		Timestamp: time.Now(),
	}

	err := tn.eventBus.Publish(event)
	if err != nil {
		tn.stats["errors"] = tn.stats["errors"].(int) + 1
		log.Printf("❌ TelegramNotifier: Ошибка публикации Counter сигнала: %v", err)
		return err
	}

	tn.stats["counter_signals_sent"] = tn.stats["counter_signals_sent"].(int) + 1
	log.Printf("✅ TelegramNotifier: Counter сигнал опубликован: %s %s %.2f%%",
		symbol, direction, change)

	return nil
}

// SendDirectMessage отправляет сообщение напрямую (для системных сообщений)
// ИЗМЕНЕНО: метод может иметь другую сигнатуру в новом боте
func (tn *TelegramNotifier) SendDirectMessage(message string) error {
	if !tn.enabled || tn.mainBot == nil {
		return nil
	}

	// TODO: Реализовать отправку через новый интерфейс бота
	// Временная заглушка
	return nil
}

// SendControlMessage отправляет сообщение в основной чат
func (tn *TelegramNotifier) SendControlMessage(message string) error {
	if !tn.enabled || tn.mainBot == nil {
		return nil
	}

	// TODO: Реализовать отправку через новый интерфейс бота
	// Временная заглушка
	return nil
}

// SendTestMessage отправляет тестовое сообщение
func (tn *TelegramNotifier) SendTestMessage() error {
	if !tn.enabled || tn.mainBot == nil {
		return nil
	}

	// TODO: Реализовать отправку через новый интерфейс бота
	// Временная заглушка
	return nil
}

// Name возвращает имя
func (tn *TelegramNotifier) Name() string {
	return "telegram_notifier"
}

// IsEnabled возвращает статус
func (tn *TelegramNotifier) IsEnabled() bool {
	return tn.enabled
}

// SetEnabled включает/выключает
func (tn *TelegramNotifier) SetEnabled(enabled bool) {
	tn.enabled = enabled
}

// GetStats возвращает статистику
func (tn *TelegramNotifier) GetStats() map[string]interface{} {
	statsCopy := make(map[string]interface{})
	for k, v := range tn.stats {
		statsCopy[k] = v
	}
	return statsCopy
}
