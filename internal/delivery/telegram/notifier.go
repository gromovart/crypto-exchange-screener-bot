// internal/delivery/telegram/notifier.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

// Notifier - нотификатор Telegram (слушает EventBus)
type Notifier struct {
	config           *config.Config
	messageSender    *MessageSender
	messageFormatter *MarketMessageFormatter
	telegramBot      *TelegramBot // Добавляем TelegramBot
	rateLimiter      *RateLimiter
	enabled          bool
	mu               sync.RWMutex
}

// NewNotifier создает новый нотификатор
func NewNotifier(cfg *config.Config) *Notifier {
	// Используем биржу из конфигурации
	exchange := cfg.Exchange
	if exchange == "" {
		exchange = "bybit" // значение по умолчанию
	}

	return &Notifier{
		config:           cfg,
		messageFormatter: NewMarketMessageFormatter(exchange),
		rateLimiter:      NewRateLimiter(2 * time.Second),
		enabled:          cfg.TelegramEnabled,
	}
}

// SetTelegramBot устанавливает TelegramBot
func (n *Notifier) SetTelegramBot(bot *TelegramBot) {
	n.telegramBot = bot
	if bot != nil {
		n.messageSender = bot.messageSender
	}
}

// SetMessageSender устанавливает отправителя сообщений
func (n *Notifier) SetMessageSender(sender *MessageSender) {
	n.messageSender = sender
}

// HandleEvent обрабатывает события из EventBus
func (n *Notifier) HandleEvent(event types.Event) error {
	if !n.IsEnabled() || n.telegramBot == nil {
		return nil
	}

	log.Printf("🤖 telegram.Notifier: Событие %s от %s", event.Type, event.Source)

	switch event.Type {
	case types.EventSignalDetected:
		return n.handleSignalEvent(event)
	case types.EventCounterSignalDetected:
		return n.handleCounterSignalEvent(event)
	case types.EventCounterNotificationRequest:
		return n.handleCounterNotification(event)
	}

	return nil
}

// handleSignalEvent обрабатывает обычные сигналы
func (n *Notifier) handleSignalEvent(event types.Event) error {
	// Пропускаем сигналы от counter_analyzer - они обрабатываются отдельно
	if event.Source == "counter_analyzer" {
		log.Printf("⚠️ Пропуск сигнала counter_analyzer в telegram.Notifier")
		return nil
	}

	// Обработка обычных сигналов
	signal, ok := event.Data.(types.TrendSignal)
	if !ok {
		return nil
	}

	log.Printf("🤖 telegram.Notifier: Отправка сигнала %s %.2f%%",
		signal.Symbol, signal.ChangePercent)

	// Используем TelegramBot для отправки
	return n.telegramBot.SendNotification(types.GrowthSignal{
		Symbol:        signal.Symbol,
		Direction:     signal.Direction,
		GrowthPercent: signal.ChangePercent,
		FallPercent:   0,
		Confidence:    signal.Confidence,
		DataPoints:    signal.DataPoints,
		StartPrice:    0,
		EndPrice:      0,
		Timestamp:     signal.Timestamp,
	})
}

// handleCounterSignalEvent обрабатывает Counter сигналы
func (n *Notifier) handleCounterSignalEvent(event types.Event) error {
	log.Printf("🤖 telegram.Notifier: Обработка Counter сигнала от %s", event.Source)

	// Извлекаем данные Counter сигнала
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		log.Printf("❌ Неверный формат данных Counter сигнала")
		return nil
	}

	// Извлекаем данные для форматирования
	symbol, _ := data["symbol"].(string)
	direction, _ := data["direction"].(string)
	change, _ := data["change"].(float64)
	signalCount, _ := data["signal_count"].(int)
	maxSignals, _ := data["max_signals"].(int)
	periodStr, _ := data["period"].(string)
	currentPrice, _ := data["current_price"].(float64)
	volume24h, _ := data["volume_24h"].(float64)
	openInterest, _ := data["open_interest"].(float64)
	oiChange24h, _ := data["oi_change_24h"].(float64)
	fundingRate, _ := data["funding_rate"].(float64)
	volumeDelta, _ := data["volume_delta"].(float64)
	volumeDeltaPercent, _ := data["volume_delta_percent"].(float64)
	rsi, _ := data["rsi"].(float64)
	macdSignal, _ := data["macd_signal"].(float64)
	deltaSource, _ := data["delta_source"].(string)

	if symbol == "" {
		log.Printf("❌ Не указан символ")
		return nil
	}

	if periodStr == "" {
		periodStr = "1h"
	}

	log.Printf("✅ Counter сигнал: %s %s %.2f%% (сигналов: %d/%d)",
		symbol, direction, change, signalCount, maxSignals)

	// Создаем MessageParams для полного форматирования
	params := &MessageParams{
		Symbol:             symbol,
		Direction:          direction,
		Change:             change,
		SignalCount:        signalCount,
		MaxSignals:         maxSignals,
		CurrentPrice:       currentPrice,
		Volume24h:          volume24h,
		OpenInterest:       openInterest,
		OIChange24h:        oiChange24h,
		FundingRate:        fundingRate,
		AverageFunding:     0.0001, // default
		NextFundingTime:    time.Now().Add(1 * time.Hour),
		Period:             periodStr, // Используем оригинальный период
		LiquidationVolume:  0,
		LongLiqVolume:      0,
		ShortLiqVolume:     0,
		VolumeDelta:        volumeDelta,
		VolumeDeltaPercent: volumeDeltaPercent,
		RSI:                rsi,
		MACDSignal:         macdSignal,
		DeltaSource:        deltaSource,
	}

	// Форматируем полное сообщение
	message := n.messageFormatter.FormatMessage(params)

	// ПОЛУЧАЕМ ПЕРИОД В МИНУТАХ ЧЕРЕЗ pkg/utils
	periodMinutes := utils.ParsePeriodToMinutes(periodStr)
	periodName := utils.PeriodToName(periodStr)

	log.Printf("📊 Период: %s → %s (%d минут)", periodStr, periodName, periodMinutes)

	// СОЗДАЕМ КЛАВИАТУРУ С КНОПКАМИ "ТОРГОВАТЬ" И "ГРАФИКИ"
	var keyboard *InlineKeyboardMarkup

	// Вариант 1: Через keyboardSystem из menuManager
	if n.telegramBot != nil && n.telegramBot.menuManager != nil {
		keyboardSystem := n.telegramBot.menuManager.GetKeyboardSystem()
		if keyboardSystem != nil {
			keyboard = keyboardSystem.CreateNotificationKeyboard(symbol, periodMinutes)
			log.Printf("✅ Создана клавиатура для %s (период: %d мин)", symbol, periodMinutes)
		}
	}

	// Вариант 2: Fallback - создаем напрямую через ButtonURLBuilder
	if keyboard == nil && n.config != nil {
		exchange := n.config.Exchange
		if exchange == "" {
			exchange = "bybit"
		}
		builder := NewButtonURLBuilder(exchange)
		keyboard = builder.StandardNotificationKeyboard(symbol, periodMinutes)
		log.Printf("✅ Создана клавиатура для %s через ButtonURLBuilder", symbol)
	}

	// Отправляем через TelegramBot С КЛАВИАТУРОЙ
	if n.telegramBot != nil && n.messageSender != nil {
		log.Printf("📨 Отправка сообщения с клавиатурой для %s", symbol)
		return n.messageSender.SendTextMessage(message, keyboard, false)
	}

	return fmt.Errorf("telegram bot or message sender not initialized")
}

// handleCounterNotification обрабатывает запросы уведомлений
func (n *Notifier) handleCounterNotification(event types.Event) error {
	log.Printf("📨 telegram.Notifier: Обработка запроса уведомлений")

	// Можно добавить специальную логику для запросов уведомлений
	// Например, форматирование с дополнительными данными

	return nil
}

// GetName возвращает имя нотификатора (для интерфейса EventSubscriber)
func (n *Notifier) GetName() string {
	return "telegram_notifier"
}

// GetSubscribedEvents возвращает типы событий
func (n *Notifier) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventSignalDetected,
		types.EventCounterSignalDetected,
		types.EventCounterNotificationRequest,
	}
}

// SetEnabled включает/выключает уведомления
func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	n.enabled = enabled
	n.mu.Unlock()
}

// IsEnabled возвращает статус уведомлений
func (n *Notifier) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}
