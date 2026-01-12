// internal/delivery/telegram/user_notification_handler.go

package telegram

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

// UserNotificationHandler - обработчик уведомлений пользователей
type UserNotificationHandler struct {
	messageSender *MessageSender
	exchange      string
	rateLimiter   *RateLimiter
	mu            sync.RWMutex
	enabled       bool
}

// NewUserNotificationHandler создает новый обработчик
func NewUserNotificationHandler(messageSender *MessageSender, exchange string) *UserNotificationHandler {
	return &UserNotificationHandler{
		messageSender: messageSender,
		exchange:      exchange,
		rateLimiter:   NewRateLimiter(2 * time.Second),
		enabled:       true,
	}
}

// HandleEvent обрабатывает события пользовательских уведомлений
func (unh *UserNotificationHandler) HandleEvent(event types.Event) error {
	if !unh.enabled || unh.messageSender == nil {
		return nil
	}

	if event.Type != types.EventUserNotification { // ИСПРАВЛЕНО
		return nil
	}

	// Извлекаем данные
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data format")
	}

	// Получаем информацию пользователя
	chatID, _ := data["chat_id"].(string)
	if chatID == "" {
		return fmt.Errorf("chat_id not specified")
	}

	symbol, _ := data["symbol"].(string)
	direction, _ := data["direction"].(string)
	count, _ := data["signal_count"].(int)
	maxSignals, _ := data["max_signals"].(int)
	period, _ := data["period"].(string)

	// Форматируем и отправляем
	return unh.sendUserNotification(chatID, symbol, direction, count, maxSignals, period, data)
}

// sendUserNotification отправляет уведомление пользователю
func (unh *UserNotificationHandler) sendUserNotification(
	chatID, symbol, direction string,
	count, maxSignals int,
	period string,
	data map[string]interface{},
) error {
	// Проверяем rate limiting
	if !unh.rateLimiter.CanSend(chatID) {
		log.Printf("⚠️ Rate limit for chat %s, skipping", chatID)
		return nil
	}

	// Форматируем сообщение
	message, keyboard, err := unh.formatMessage(symbol, direction, count, maxSignals, period, data)
	if err != nil {
		return fmt.Errorf("failed to format message: %w", err)
	}

	// Создаем messageSender с chat_id пользователя
	userMessageSender := unh.messageSender.WithChatID(chatID)

	// Отправляем сообщение
	err = userMessageSender.SendTextMessage(message, keyboard, false)
	if err != nil {
		log.Printf("❌ Error sending to chat %s: %v", chatID, err)
		return err
	}

	log.Printf("✅ Notification sent to chat %s for %s", chatID, symbol)
	return nil
}

// formatMessage форматирует сообщение
func (unh *UserNotificationHandler) formatMessage(
	symbol, direction string,
	count, maxSignals int,
	period string,
	data map[string]interface{},
) (string, *InlineKeyboardMarkup, error) {
	// Используем существующий форматтер
	formatter := NewMarketMessageFormatter(unh.exchange)

	// Извлекаем дополнительные данные
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

	params := &MessageParams{
		Symbol:             symbol,
		Direction:          direction,
		Change:             float64(count) / float64(maxSignals) * 100,
		SignalCount:        count,
		MaxSignals:         maxSignals,
		CurrentPrice:       currentPrice,
		Volume24h:          volume24h,
		OpenInterest:       openInterest,
		OIChange24h:        oiChange24h,
		FundingRate:        fundingRate,
		AverageFunding:     0.0001,
		NextFundingTime:    time.Now().Add(1 * time.Hour),
		Period:             period,
		VolumeDelta:        volumeDelta,
		VolumeDeltaPercent: volumeDeltaPercent,
		RSI:                rsi,
		MACDSignal:         macdSignal,
		DeltaSource:        deltaSource,
	}

	message := formatter.FormatMessage(params)

	// Создаем клавиатуру
	builder := NewButtonURLBuilder(unh.exchange)
	periodMinutes := utils.ParsePeriodToMinutes(period) // ИСПРАВЛЕНО
	keyboard := builder.StandardNotificationKeyboard(symbol, periodMinutes)

	return message, keyboard, nil
}

// GetSubscribedEvents возвращает типы событий
func (unh *UserNotificationHandler) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventUserNotification, // ИСПРАВЛЕНО
	}
}

// GetName возвращает имя обработчика
func (unh *UserNotificationHandler) GetName() string {
	return "user_notification_handler"
}

// SetEnabled включает/выключает обработчик
func (unh *UserNotificationHandler) SetEnabled(enabled bool) {
	unh.mu.Lock()
	unh.enabled = enabled
	unh.mu.Unlock()
}

// IsEnabled возвращает статус
func (unh *UserNotificationHandler) IsEnabled() bool {
	unh.mu.RLock()
	defer unh.mu.RUnlock()
	return unh.enabled
}
