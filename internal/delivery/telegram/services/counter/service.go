// internal/delivery/telegram/services/counter/service.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
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
// Теперь принимает конкретный тип CounterParams вместо interface{}
func (s *serviceImpl) Exec(params CounterParams) (CounterResult, error) {

	log.Printf("🔍 Service.Exec: получены параметры: %s %s %.4f%%, RSI=%.2f, MACD=%.2f",
		params.Symbol, params.Direction, params.ChangePercent, params.RSI, params.MACDSignal)

	// Извлекаем данные счетчика из CounterParams
	rawData, err := s.extractRawDataFromParams(params)
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
			logger.Error("❌ Ошибка отправки уведомления пользователю %s: %v", user.Username, err)
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

	logger.Debug("📨 Отправка counter уведомления для %s пользователю %s",
		data.Symbol, user.Username)

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
		logger.Debug("🛠️ Создана клавиатура для %s", data.Symbol)
	}

	// Отправляем через message sender с клавиатурой
	if s.messageSender != nil {
		err := s.messageSender.SendTextMessage(chatID, formattedMessage, keyboard)
		if err != nil {
			return fmt.Errorf("ошибка отправки в Telegram: %w", err)
		}
		logger.Debug("✅ Сообщение с клавиатурой отправлено пользователю %s", user.Username)
	} else {
		logger.Error("⚠️ MessageSender не инициализирован, сообщение не отправлено")
		return fmt.Errorf("message sender not initialized")
	}

	return nil
}
