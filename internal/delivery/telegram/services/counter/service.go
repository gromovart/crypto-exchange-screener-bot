// internal/delivery/telegram/services/counter/service.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"time"
)

// serviceImpl реализация CounterService
type serviceImpl struct {
	userService   *users.Service
	formatter     *formatters.FormatterProvider
	messageSender message_sender.MessageSender
}

// NewService создает новый сервис счетчика
func NewService(userService *users.Service, formatter *formatters.FormatterProvider, messageSender message_sender.MessageSender) Service {
	return &serviceImpl{
		userService:   userService,
		formatter:     formatter,
		messageSender: messageSender,
	}
}

// Exec выполняет обработку события счетчика
func (s *serviceImpl) Exec(params interface{}) (interface{}, error) {
	// Приводим параметры к нужному типу
	parsedParams, ok := params.(CounterParams)
	if !ok {
		return CounterResult{Processed: false},
			fmt.Errorf("неверный тип параметров: ожидается CounterParams")
	}

	if parsedParams.Event.Type != types.EventCounterSignalDetected {
		return CounterResult{Processed: false},
			fmt.Errorf("неподдерживаемый тип события: %s", parsedParams.Event.Type)
	}

	// Извлекаем данные счетчика
	rawData, err := s.extractRawData(parsedParams.Event.Data)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка извлечения данных счетчика: %w", err)
	}

	// Конвертируем в форматтер данные
	counterData := s.convertToFormatterData(rawData)

	// Получаем пользователей для отправки
	usersToNotify, err := s.getUsersToNotify(rawData)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка получения пользователей: %w", err)
	}

	// Отправляем уведомления
	sentCount := 0
	for _, user := range usersToNotify {
		if err := s.sendNotification(user, counterData); err != nil {
			log.Printf("❌ Ошибка отправки уведомления пользователю %s: %v", user.Username, err)
		} else {
			sentCount++
		}
	}

	return CounterResult{
		Processed: true,
		Message:   fmt.Sprintf("Отправлено %d уведомлений для %s", sentCount, rawData.Symbol),
		SentTo:    sentCount,
	}, nil
}

// sendNotification отправляет уведомление пользователю
func (s *serviceImpl) sendNotification(user *models.User, data formatters.CounterData) error {
	// Форматируем сообщение
	formattedMessage := s.formatter.FormatCounterSignal(data)

	// ЛОГИРУЕМ ПОЛНОЕ СООБЩЕНИЕ
	log.Printf("📨 DEBUG: Полное сообщение для %s:\n%s",
		data.Symbol, formattedMessage)

	log.Printf("📨 Отправка counter уведомления для %s пользователю %s (chat_id: %s)",
		data.Symbol, user.Username, user.ChatID)

	// Проверяем message sender
	if s.messageSender == nil {
		log.Printf("❌ MessageSender is NIL!")
		return fmt.Errorf("message sender not initialized")
	}

	// Проверяем тип message sender
	log.Printf("📱 MessageSender type: %T", s.messageSender)

	// Проверяем тестовый режим если есть метод
	if sender, ok := s.messageSender.(interface{ IsTestMode() bool }); ok {
		log.Printf("🧪 MessageSender test mode: %v", sender.IsTestMode())
	}

	log.Printf("📨 Отправка counter уведомления для %s пользователю %s (chat_id: %s)",
		data.Symbol, user.Username, user.ChatID)

	// Проверяем chat_id
	if user.ChatID == "" {
		return fmt.Errorf("пустой chat_id у пользователя %s", user.Username)
	}

	// Конвертируем chat_id из строки в int64
	var chatID int64
	_, err := fmt.Sscanf(user.ChatID, "%d", &chatID)
	if err != nil {
		return fmt.Errorf("неверный формат chat_id у пользователя %s: %s", user.Username, user.ChatID)
	}

	// Отправляем через message sender
	if s.messageSender != nil {
		err := s.messageSender.SendTextMessage(chatID, formattedMessage, nil)
		if err != nil {
			return fmt.Errorf("ошибка отправки в Telegram: %w", err)
		}
		log.Printf("✅ Сообщение отправлено пользователю %s", user.Username)
	} else {
		log.Printf("⚠️ MessageSender не инициализирован, сообщение не отправлено")
		return fmt.Errorf("message sender not initialized")
	}

	return nil
}

// RawCounterData сырые данные счетчика
type RawCounterData struct {
	Symbol             string    `json:"symbol"`
	Direction          string    `json:"direction"`
	ChangePercent      float64   `json:"change"`
	SignalCount        int       `json:"signal_count"`
	MaxSignals         int       `json:"max_signals"`
	Period             string    `json:"period"`
	CurrentPrice       float64   `json:"current_price"`
	Volume24h          float64   `json:"volume_24h"`
	OpenInterest       float64   `json:"open_interest"`
	OIChange24h        float64   `json:"oi_change_24h"`
	FundingRate        float64   `json:"funding_rate"`
	NextFundingTime    time.Time `json:"next_funding_time"`
	LiquidationVolume  float64   `json:"liquidation_volume"`
	LongLiqVolume      float64   `json:"long_liq_volume"`
	ShortLiqVolume     float64   `json:"short_liq_volume"`
	VolumeDelta        float64   `json:"volume_delta"`
	VolumeDeltaPercent float64   `json:"volume_delta_percent"`
	RSI                float64   `json:"rsi"`
	MACDSignal         float64   `json:"macd_signal"`
	DeltaSource        string    `json:"delta_source"`
	Confidence         float64   `json:"confidence"`
	Timestamp          time.Time `json:"timestamp"`
}

// extractRawData извлекает сырые данные счетчика из события (map[string]interface{})
func (s *serviceImpl) extractRawData(eventData interface{}) (RawCounterData, error) {
	// Приводим к map[string]interface{}
	dataMap, ok := eventData.(map[string]interface{})
	if !ok {
		return RawCounterData{}, fmt.Errorf("неверный формат данных события")
	}

	// Извлекаем данные с проверкой типов
	data := RawCounterData{
		Timestamp: time.Now(),
	}

	// Строковые поля
	if s, ok := dataMap["symbol"].(string); ok {
		data.Symbol = s
	}
	if d, ok := dataMap["direction"].(string); ok {
		data.Direction = d
	}
	if p, ok := dataMap["period"].(string); ok {
		data.Period = p
	}
	if ds, ok := dataMap["delta_source"].(string); ok {
		data.DeltaSource = ds
	}

	// Числовые поля (float64)
	floatFields := map[string]*float64{
		"change":               &data.ChangePercent,
		"current_price":        &data.CurrentPrice,
		"volume_24h":           &data.Volume24h,
		"open_interest":        &data.OpenInterest,
		"oi_change_24h":        &data.OIChange24h,
		"volume_delta":         &data.VolumeDelta,
		"volume_delta_percent": &data.VolumeDeltaPercent,
		"rsi":                  &data.RSI,
		"funding_rate":         &data.FundingRate,
		"confidence":           &data.Confidence,
		"liquidation_volume":   &data.LiquidationVolume,
		"long_liq_volume":      &data.LongLiqVolume,
		"short_liq_volume":     &data.ShortLiqVolume,
		"macd_signal":          &data.MACDSignal,
	}

	for key, ptr := range floatFields {
		if val, ok := dataMap[key].(float64); ok {
			*ptr = val
		}
	}

	// Целочисленные поля
	if sc, ok := dataMap["signal_count"].(int); ok {
		data.SignalCount = sc
	} else if scf, ok := dataMap["signal_count"].(float64); ok {
		data.SignalCount = int(scf)
	}

	if ms, ok := dataMap["max_signals"].(int); ok {
		data.MaxSignals = ms
	} else if msf, ok := dataMap["max_signals"].(float64); ok {
		data.MaxSignals = int(msf)
	}

	// Время next_funding_time
	if nft, ok := dataMap["next_funding_time"].(time.Time); ok {
		data.NextFundingTime = nft
	} else if nftStr, ok := dataMap["next_funding_time"].(string); ok {
		// Пробуем распарсить строку времени
		if t, err := time.Parse(time.RFC3339, nftStr); err == nil {
			data.NextFundingTime = t
		}
	}

	log.Printf("🔢 CounterService: Извлечены данные для %s (%s: %.2f%%, сигналов: %d/%d)",
		data.Symbol, data.Direction, data.ChangePercent, data.SignalCount, data.MaxSignals)

	return data, nil
}

// convertToFormatterData конвертирует сырые данные в форматтер данные
func (s *serviceImpl) convertToFormatterData(raw RawCounterData) formatters.CounterData {
	return formatters.CounterData{
		Symbol:             raw.Symbol,
		Direction:          raw.Direction,
		ChangePercent:      raw.ChangePercent,
		SignalCount:        raw.SignalCount,
		MaxSignals:         raw.MaxSignals,
		Period:             raw.Period,
		CurrentPrice:       raw.CurrentPrice,
		Volume24h:          raw.Volume24h,
		OpenInterest:       raw.OpenInterest,
		OIChange24h:        raw.OIChange24h,
		FundingRate:        raw.FundingRate,
		NextFundingTime:    raw.NextFundingTime,
		LiquidationVolume:  raw.LiquidationVolume,
		LongLiqVolume:      raw.LongLiqVolume,
		ShortLiqVolume:     raw.ShortLiqVolume,
		VolumeDelta:        raw.VolumeDelta,
		VolumeDeltaPercent: raw.VolumeDeltaPercent,
		RSI:                raw.RSI,
		MACDSignal:         raw.MACDSignal,
		DeltaSource:        raw.DeltaSource,
		Confidence:         raw.Confidence,
		Timestamp:          raw.Timestamp,
	}
}

// getUsersToNotify возвращает пользователей, которым нужно отправить уведомление
func (s *serviceImpl) getUsersToNotify(data RawCounterData) ([]*models.User, error) {
	if s.userService == nil {
		return nil, fmt.Errorf("userService not initialized")
	}

	// Получаем всех пользователей
	allUsers, err := s.userService.GetAllUsers(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Фильтруем пользователей
	var filteredUsers []*models.User
	for _, user := range allUsers {
		if s.shouldSendToUser(user, data) {
			filteredUsers = append(filteredUsers, user)
		}
	}

	return filteredUsers, nil
}

// shouldSendToUser проверяет, нужно ли отправлять пользователю
func (s *serviceImpl) shouldSendToUser(user *models.User, data RawCounterData) bool {
	if user == nil {
		return false
	}

	// Проверяем ChatID
	if user.ChatID == "" {
		log.Printf("⚠️ User %d (%s) skipped: empty chat_id", user.ID, user.Username)
		return false
	}

	// Проверяем активность
	if !user.IsActive {
		log.Printf("⚠️ User %d (%s) skipped: not active", user.ID, user.Username)
		return false
	}

	// Базовые проверки из модели User
	if !user.CanReceiveNotifications() {
		log.Printf("⚠️ User %d (%s) skipped: notifications disabled", user.ID, user.Username)
		return false
	}

	// Проверяем тип сигнала
	if data.Direction == "growth" && !user.CanReceiveGrowthSignals() {
		log.Printf("⚠️ User %d (%s) skipped: growth signals disabled", user.ID, user.Username)
		return false
	}
	if data.Direction == "fall" && !user.CanReceiveFallSignals() {
		log.Printf("⚠️ User %d (%s) skipped: fall signals disabled", user.ID, user.Username)
		return false
	}

	// Проверяем тихие часы
	if user.IsInQuietHours() {
		log.Printf("⚠️ User %d (%s) skipped: in quiet hours (%d-%d)",
			user.ID, user.Username, user.QuietHoursStart, user.QuietHoursEnd)
		return false
	}

	// Проверяем лимиты
	if user.HasReachedDailyLimit() {
		log.Printf("⚠️ User %d (%s) skipped: daily limit reached (%d/%d)",
			user.ID, user.Username, user.SignalsToday, user.MaxSignalsPerDay)
		return false
	}

	// Проверяем пороги
	fillPercentage := float64(data.SignalCount) / float64(data.MaxSignals) * 100
	if data.Direction == "growth" && fillPercentage < user.MinGrowthThreshold {
		log.Printf("⚠️ User %d (%s) skipped: growth threshold not met (%.1f%% < %.1f%%)",
			user.ID, user.Username, fillPercentage, user.MinGrowthThreshold)
		return false
	}
	if data.Direction == "fall" && fillPercentage < user.MinFallThreshold {
		log.Printf("⚠️ User %d (%s) skipped: fall threshold not met (%.1f%% < %.1f%%)",
			user.ID, user.Username, fillPercentage, user.MinFallThreshold)
		return false
	}

	log.Printf("✅ User %d (%s) passed all checks", user.ID, user.Username)
	return true
}
