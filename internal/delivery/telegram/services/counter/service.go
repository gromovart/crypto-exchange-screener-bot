// internal/delivery/telegram/services/counter/service.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"math"
	"time"
)

// serviceImpl реализация CounterService
type serviceImpl struct {
	userService   *users.Service
	formatter     *formatters.FormatterProvider
	messageSender message_sender.MessageSender
	buttonBuilder *buttons.ButtonBuilder
}

// NewService создает новый сервис счетчика
func NewService(
	userService *users.Service,
	formatter *formatters.FormatterProvider,
	messageSender message_sender.MessageSender,
	buttonBuilder *buttons.ButtonBuilder,
) Service {
	return &serviceImpl{
		userService:   userService,
		formatter:     formatter,
		messageSender: messageSender,
		buttonBuilder: buttonBuilder,
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

	// СОЗДАЕМ КЛАВИАТУРУ С КНОПКАМИ "График" и "Торговать"
	var keyboard interface{} = nil
	if s.buttonBuilder != nil {
		keyboard = s.buttonBuilder.CreateSignalKeyboard(data.Symbol)
		log.Printf("🛠️ Создана клавиатура для %s с кнопками: График, Торговать", data.Symbol)
	} else {
		log.Printf("⚠️ ButtonBuilder не инициализирован, клавиатура не будет добавлена")
	}

	// Отправляем через message sender с клавиатурой
	if s.messageSender != nil {
		err := s.messageSender.SendTextMessage(chatID, formattedMessage, keyboard)
		if err != nil {
			return fmt.Errorf("ошибка отправки в Telegram: %w", err)
		}
		log.Printf("✅ Сообщение с клавиатурой отправлено пользователю %s", user.Username)
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
	SignalCount        int       `json:"signal_count"` // старый формат (для обратной совместимости)
	MaxSignals         int       `json:"max_signals"`  // старый формат
	Period             string    `json:"period"`       // "5m", "15m", "30m", "1h", "4h", "1d"
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

	// НОВЫЕ ПОЛЯ для системы подтверждений
	Confirmations         int `json:"confirmations"`          // текущие подтверждения
	RequiredConfirmations int `json:"required_confirmations"` // нужно подтверждений
	TotalSlots            int `json:"total_slots"`            // всего слотов (групп)
	FilledSlots           int `json:"filled_slots"`           // заполненные слоты

	// НОВЫЕ ПОЛЯ для отслеживания прогресса
	NextAnalysis       time.Time `json:"next_analysis"`       // следующий анализ
	NextSignal         time.Time `json:"next_signal"`         // следующий сигнал
	ProgressPercentage float64   `json:"progress_percentage"` // процент прогресса (вычисляемое)
}

// Вспомогательная функция для получения ключей мапы
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// extractRawData извлекает сырые данные счетчика из события (map[string]interface{})
func (s *serviceImpl) extractRawData(eventData interface{}) (RawCounterData, error) {
	fmt.Printf("\n🔍 DEBUG extractRawData ДЕТАЛЬНО:\n")

	// Приводим к map
	dataMap, ok := eventData.(map[string]interface{})
	if !ok {
		fmt.Printf("   ❌ eventData не map: %T\n", eventData)
		return RawCounterData{}, fmt.Errorf("неверный формат данных события")
	}

	// Выводим все ключи
	fmt.Printf("   Все ключи в eventData (%d):\n", len(dataMap))
	for key, val := range dataMap {
		fmt.Printf("      %s: %v (тип: %T)\n", key, val, val)
	}

	data := RawCounterData{
		Timestamp: time.Now(),
	}

	// 1. Пробуем извлечь change_percent напрямую
	if change, ok := dataMap["change_percent"].(float64); ok {
		data.ChangePercent = change
		fmt.Printf("   ✅ Извлечен change_percent напрямую: %.4f%%\n", change)
	} else {
		fmt.Printf("   ❌ change_percent не найден как float64\n")

		// 2. Пробуем через Custom
		if custom, ok := dataMap["custom"].(map[string]interface{}); ok {
			if change, ok := custom["change_percent"].(float64); ok {
				data.ChangePercent = change
				fmt.Printf("   ✅ Извлечен change_percent из Custom: %.4f%%\n", change)
			}
		}
	}

	// 3. Пробуем извлечь period_string
	if period, ok := dataMap["period_string"].(string); ok {
		data.Period = period
		fmt.Printf("   ✅ Извлечен period_string напрямую: %s\n", period)
	} else if custom, ok := dataMap["custom"].(map[string]interface{}); ok {
		if period, ok := custom["period_string"].(string); ok {
			data.Period = period
			fmt.Printf("   ✅ Извлечен period_string из Custom: %s\n", period)
		}
	}

	// 4. Пробуем извлечь confirmations
	if confirmations, ok := dataMap["confirmations"].(int); ok {
		data.Confirmations = confirmations
		data.RequiredConfirmations = GetRequiredConfirmations(data.Period)
		fmt.Printf("   ✅ Извлечены confirmations: %d/%d\n",
			confirmations, data.RequiredConfirmations)
	} else if confirmations, ok := dataMap["confirmations"].(float64); ok {
		data.Confirmations = int(confirmations)
		data.RequiredConfirmations = GetRequiredConfirmations(data.Period)
		fmt.Printf("   ✅ Извлечены confirmations из float64: %d/%d\n",
			data.Confirmations, data.RequiredConfirmations)
	}

	// 5. Символ и направление
	if symbol, ok := dataMap["symbol"].(string); ok {
		data.Symbol = symbol
		fmt.Printf("   ✅ Извлечен symbol: %s\n", symbol)
	}

	if direction, ok := dataMap["direction"].(string); ok {
		data.Direction = direction
		fmt.Printf("   ✅ Извлечено direction: %s\n", direction)
	}

	// 6. Другие поля с добавлением RSI и MACD
	floatFields := map[string]*float64{
		"current_price": &data.CurrentPrice,
		"volume_24h":    &data.Volume24h,
		"open_interest": &data.OpenInterest,
		"funding_rate":  &data.FundingRate,
		"rsi":           &data.RSI,        // ДОБАВЛЕНО
		"macd_signal":   &data.MACDSignal, // ДОБАВЛЕНО
	}

	// Сначала пробуем из indicators как map[string]float64 (актуальный тип из логов)
	if indicators, ok := dataMap["indicators"].(map[string]float64); ok {
		fmt.Printf("   ✅ indicators как map[string]float64\n")

		for key, ptr := range floatFields {
			if val, ok := indicators[key]; ok {
				*ptr = val
				fmt.Printf("   ✅ Извлечен %s из indicators: %v\n", key, val)
			} else {
				fmt.Printf("   ❌ Ключ '%s' не найден в indicators\n", key)
			}
		}

		// Дельта объемов
		if volumeDelta, ok := indicators["volume_delta"]; ok {
			data.VolumeDelta = volumeDelta
			fmt.Printf("   ✅ Извлечен volume_delta: %.2f\n", volumeDelta)
		}
		if volumeDeltaPercent, ok := indicators["volume_delta_percent"]; ok {
			data.VolumeDeltaPercent = volumeDeltaPercent
			fmt.Printf("   ✅ Извлечен volume_delta_percent: %.2f%%\n", volumeDeltaPercent)
		}

		// Логируем извлеченные значения
		fmt.Printf("   📊 ИЗВЛЕЧЕННЫЕ ДАННЫЕ из indicators:\n")
		fmt.Printf("      OI: %.2f, Объем 24ч: %.2f, RSI: %.2f, MACD: %.2f\n",
			data.OpenInterest, data.Volume24h, data.RSI, data.MACDSignal)

	} else {
		fmt.Printf("   ❌ indicators не map[string]float64: %T\n", dataMap["indicators"])

		// Попробуем как map[string]interface{} для обратной совместимости
		if indicators, ok := dataMap["indicators"].(map[string]interface{}); ok {
			fmt.Printf("   ✅ indicators как map[string]interface{} (обратная совместимость)\n")

			for key, ptr := range floatFields {
				if val, ok := indicators[key].(float64); ok {
					*ptr = val
					fmt.Printf("   ✅ Извлечен %s из indicators: %v\n", key, val)
				} else {
					fmt.Printf("   ❌ Ключ '%s' не найден или не float64 в indicators\n", key)
				}
			}

			// Дельта объемов
			if volumeDelta, ok := indicators["volume_delta"].(float64); ok {
				data.VolumeDelta = volumeDelta
				fmt.Printf("   ✅ Извлечен volume_delta: %.2f\n", volumeDelta)
			}
			if volumeDeltaPercent, ok := indicators["volume_delta_percent"].(float64); ok {
				data.VolumeDeltaPercent = volumeDeltaPercent
				fmt.Printf("   ✅ Извлечен volume_delta_percent: %.2f%%\n", volumeDeltaPercent)
			}

			// Логируем извлеченные значения
			fmt.Printf("   📊 ИЗВЛЕЧЕННЫЕ ДАННЫЕ из indicators (interface{}):\n")
			fmt.Printf("      OI: %.2f, Объем 24ч: %.2f, RSI: %.2f, MACD: %.2f\n",
				data.OpenInterest, data.Volume24h, data.RSI, data.MACDSignal)
		} else {
			fmt.Printf("   ❌ indicators вообще не мапа\n")
		}
	}

	fmt.Printf("   📊 ИТОГО извлечено: %s %s %.4f%% (период: %s, подтверждений: %d/%d)\n",
		data.Symbol, data.Direction, data.ChangePercent, data.Period,
		data.Confirmations, data.RequiredConfirmations)

	// Дополнительная проверка значений
	fmt.Printf("   🔍 ПРОВЕРКА ЗНАЧЕНИЙ после извлечения:\n")
	fmt.Printf("      OpenInterest: %.2f\n", data.OpenInterest)
	fmt.Printf("      Volume24h: %.2f\n", data.Volume24h)
	fmt.Printf("      RSI: %.2f\n", data.RSI)
	fmt.Printf("      MACDSignal: %.2f\n", data.MACDSignal)
	fmt.Printf("      VolumeDelta: %.2f (%.2f%%)\n", data.VolumeDelta, data.VolumeDeltaPercent)

	return data, nil
}

// calculateNextAnalysis рассчитывает время следующего анализа (через 1 минуту)
func (s *serviceImpl) calculateNextAnalysis(timestamp time.Time, period string) time.Time {
	// Анализ всегда через 1 минуту
	next := timestamp.Add(1 * time.Minute)

	// Округляем до следующей целой минуты
	next = next.Truncate(time.Minute)
	if next.Before(timestamp) || next.Equal(timestamp) {
		next = next.Add(1 * time.Minute)
	}

	return next
}

// calculateNextSignal рассчитывает время следующего сигнала
func (s *serviceImpl) calculateNextSignal(timestamp time.Time, period string, confirmations, requiredConfirmations int) time.Time {
	if requiredConfirmations == 0 {
		requiredConfirmations = GetRequiredConfirmations(period)
	}

	if confirmations >= requiredConfirmations {
		// Если уже есть все подтверждения, следующий сигнал = начало следующего периода
		return s.calculateNextPeriodStart(timestamp, period)
	}

	// Если не все подтверждения, следующий сигнал = когда будет следующее подтверждение
	// Каждое подтверждение = 1 минута анализа
	remainingConfirmations := requiredConfirmations - confirmations

	// Время до следующего сигнала = оставшиеся подтверждения × 1 минута
	next := timestamp.Add(time.Duration(remainingConfirmations) * time.Minute)

	// Округляем до целой минуты
	next = next.Truncate(time.Minute)
	if next.Before(timestamp) || next.Equal(timestamp) {
		next = next.Add(1 * time.Minute)
	}

	return next
}

// getMinutesPerGroup возвращает минут в одной группе для прогресс-бара
func (s *serviceImpl) getMinutesPerGroup(period string) int {
	switch period {
	case "5m":
		return 1 // 5 групп по 1 минуте
	case "15m":
		return 3 // 5 групп по 3 минуты
	case "30m":
		return 5 // 6 групп по 5 минут
	case "1h":
		return 10 // 6 групп по 10 минут
	case "4h":
		return 30 // 8 групп по 30 минут
	case "1d":
		return 120 // 12 групп по 2 часа
	default:
		return 1
	}
}

// getGroupedSlotsInfo возвращает информацию о группировке слотов
func (s *serviceImpl) getGroupedSlotsInfo(period string) (totalGroups int, minutesPerGroup int) {
	switch period {
	case "5m":
		return 5, 1 // 5 групп по 1 минуте
	case "15m":
		return 5, 3 // 5 групп по 3 минуты
	case "30m":
		return 6, 5 // 6 групп по 5 минут
	case "1h":
		return 6, 10 // 6 групп по 10 минут
	case "4h":
		return 8, 30 // 8 групп по 30 минут
	case "1d":
		return 12, 120 // 12 групп по 2 часа
	default:
		return 5, 1
	}
}

// calculateNextPeriodStart рассчитывает начало следующего периода
func (s *serviceImpl) calculateNextPeriodStart(timestamp time.Time, period string) time.Time {
	periodMinutes := s.periodToMinutes(period)

	// Текущая минута от начала часа
	currentMinute := timestamp.Minute()

	// Находим следующий период
	periodsPassed := currentMinute / periodMinutes
	nextPeriodStartMinute := (periodsPassed + 1) * periodMinutes

	// Если следующее начало периода в этом часу
	if nextPeriodStartMinute < 60 {
		next := time.Date(
			timestamp.Year(), timestamp.Month(), timestamp.Day(),
			timestamp.Hour(), nextPeriodStartMinute, 0, 0,
			timestamp.Location(),
		)

		// Если следующее начало уже прошло, берем следующее
		if !next.After(timestamp) {
			next = next.Add(time.Duration(periodMinutes) * time.Minute)
		}

		return next
	}

	// Иначе в следующем часу
	next := time.Date(
		timestamp.Year(), timestamp.Month(), timestamp.Day(),
		timestamp.Hour()+1, 0, 0, 0,
		timestamp.Location(),
	)
	return next
}

// getMinutesPerConfirmation возвращает минут между подтверждениями для периода
func (s *serviceImpl) getMinutesPerConfirmation(period string, requiredConfirmations int) int {
	periodMinutes := s.periodToMinutes(period)

	// Для 5m с 3 подтверждениями: 5 / 3 = 1.66 ≈ 2 минуты между подтверждениями
	// Для 1h с 6 подтверждениями: 60 / 6 = 10 минут между подтверждениями
	if requiredConfirmations <= 0 {
		return 1
	}

	minutes := math.Ceil(float64(periodMinutes) / float64(requiredConfirmations))
	if minutes < 1 {
		return 1
	}

	return int(minutes)
}

// convertToFormatterData конвертирует сырые данные в форматтер данные
func (s *serviceImpl) convertToFormatterData(raw RawCounterData) formatters.CounterData {
	// Рассчитываем процент прогресса
	progressPercentage := 0.0
	if raw.RequiredConfirmations > 0 {
		progressPercentage = float64(raw.Confirmations) / float64(raw.RequiredConfirmations) * 100
	} else if raw.MaxSignals > 0 {
		// Обратная совместимость
		progressPercentage = float64(raw.SignalCount) / float64(raw.MaxSignals) * 100
	}

	// РАССЧИТЫВАЕМ СЛЕДУЮЩИЙ АНАЛИЗ (всегда через 1 минуту)
	nextAnalysis := s.calculateNextAnalysis(raw.Timestamp, raw.Period)

	// РАССЧИТЫВАЕМ СЛЕДУЮЩИЙ СИГНАЛ
	nextSignal := s.calculateNextSignal(raw.Timestamp, raw.Period, raw.Confirmations, raw.RequiredConfirmations)

	// РАССЧИТЫВАЕМ ГРУППИРОВКУ для прогресс-бара
	totalGroups, _ := s.getGroupedSlotsInfo(raw.Period)
	filledGroups := s.calculateFilledGroups(raw.Confirmations, raw.RequiredConfirmations, totalGroups)

	return formatters.CounterData{
		Symbol:        raw.Symbol,
		Direction:     raw.Direction,
		ChangePercent: raw.ChangePercent,

		// Используем новые поля подтверждений, если есть
		SignalCount: raw.Confirmations,         // теперь это подтверждения
		MaxSignals:  raw.RequiredConfirmations, // теперь это требуемые подтверждения

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

		// НОВЫЕ ПОЛЯ для прогресса с группировкой
		Confirmations:         raw.Confirmations,
		RequiredConfirmations: raw.RequiredConfirmations,
		TotalSlots:            totalGroups,  // Теперь это группы (не отдельные минуты)
		FilledSlots:           filledGroups, // Заполненные группы
		ProgressPercentage:    progressPercentage,
		NextAnalysis:          nextAnalysis,
		NextSignal:            nextSignal,
	}
}

// calculateFilledGroups рассчитывает заполненные группы для прогресс-бара
func (s *serviceImpl) calculateFilledGroups(confirmations, requiredConfirmations, totalGroups int) int {
	if requiredConfirmations == 0 {
		return 0
	}

	// Каждая группа подтверждается если большинство минут в ней подтверждены
	// Упрощенная логика: группы = (подтверждения / требуемые) × всего групп
	filled := float64(confirmations) / float64(requiredConfirmations) * float64(totalGroups)

	// Округляем вверх, но не больше totalGroups
	filledInt := int(math.Ceil(filled))
	if filledInt > totalGroups {
		filledInt = totalGroups
	}

	return filledInt
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
	// ВРЕМЕННО: ПРОСТОЙ ТЕСТ
	fmt.Printf("\n🔍 DEBUG shouldSendToUser:\n")
	fmt.Printf("   Пользователь: %s (ID: %d)\n", user.Username, user.ID)
	fmt.Printf("   Сигнал: %s %s %.4f%% (период: %s)\n",
		data.Symbol, data.Direction, data.ChangePercent, data.Period)
	fmt.Printf("   ChatID: %s, Активен: %v\n", user.ChatID, user.IsActive)

	// БАЗОВЫЕ ПРОВЕРКИ
	if user == nil {
		fmt.Printf("   ❌ Пользователь nil\n")
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
func GetRequiredConfirmations(period string) int {
	if period == "" {
		return 3 // дефолт
	}

	switch period {
	case "5m":
		return 3
	case "15m":
		return 3
	case "30m":
		return 4
	case "1h":
		return 6
	case "4h":
		return 8
	case "1d":
		return 12
	default:
		return 3
	}
}

// periodToMinutes конвертирует период строки в минуты
func (s *serviceImpl) periodToMinutes(period string) int {
	switch period {
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 15 // дефолт
	}
}
