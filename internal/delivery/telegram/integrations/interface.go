package integrations

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
)

// TelegramBotClient интерфейс для работы с Telegram ботом
type TelegramBotClient interface {
	// Методы для отправки сообщений
	SendTextMessage(chatID int64, text string, keyboard interface{}) error
	GetMessageSender() message_sender.MessageSender

	// Методы для работы с обновлениями
	HandleUpdate(update interface{}) error

	// Информация о боте
	IsRunning() bool
	GetConfig() *config.Config
}

// TelegramPackageService главный сервис Telegram пакета
type TelegramPackageService interface {
	// Управление профилем пользователя
	GetUserProfile(userID int64) (*ProfileData, error)

	// Обработка событий EventBus
	HandleCounterSignal(event types.Event) error
	HandleRegularSignal(event types.Event) error

	// Управление уведомлениями
	SendUserNotification(userID int64, message string) error

	// Статистика и мониторинг
	GetPackageStats() map[string]interface{}
	GetHealthStatus() HealthStatus

	// Управление жизненным циклом
	Start() error
	Stop() error
	IsRunning() bool
}

// ProfileData данные профиля пользователя
type ProfileData struct {
	User         interface{} `json:"user"`
	Subscription interface{} `json:"subscription"`
	Stats        interface{} `json:"stats"`
	Message      string      `json:"message"`
}

// HealthStatus статус здоровья сервиса
type HealthStatus struct {
	Status      string            `json:"status"`
	Services    map[string]string `json:"services"`
	EventBus    EventBusStatus    `json:"event_bus"`
	LastUpdated string            `json:"last_updated"`
}

// EventBusStatus статус EventBus
type EventBusStatus struct {
	Connected    bool  `json:"connected"`
	Subscribers  int   `json:"subscribers"`
	EventsSent   int64 `json:"events_sent"`
	EventsFailed int64 `json:"events_failed"`
}
