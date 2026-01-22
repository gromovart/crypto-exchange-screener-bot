// internal/delivery/telegram/components/factory/factory.go
package components_factory

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ComponentFactory фабрика компонентов инфраструктуры Telegram
type ComponentFactory struct {
	config   *config.Config
	exchange string
}

// ComponentDependencies зависимости для фабрики компонентов
type ComponentDependencies struct {
	Config   *config.Config
	Exchange string
}

// NewComponentFactory создает фабрику компонентов
func NewComponentFactory(deps ComponentDependencies) *ComponentFactory {
	logger.Info("🛠️  Создание фабрики компонентов...")

	if deps.Exchange == "" {
		deps.Exchange = "BYBIT"
	}

	return &ComponentFactory{
		config:   deps.Config,
		exchange: deps.Exchange,
	}
}

// CreateMessageSender создает MessageSender
func (f *ComponentFactory) CreateMessageSender() message_sender.MessageSender {
	if f.config == nil {
		logger.Error("❌ ComponentFactory: конфиг не доступен")
		return nil
	}

	if !f.config.Telegram.Enabled || f.config.TelegramBotToken == "" {
		logger.Warn("⚠️ Telegram отключен, создаю заглушку MessageSender")
		return &stubMessageSender{}
	}

	return message_sender.NewMessageSender(f.config)
}

// CreateButtonBuilder создает ButtonBuilder
func (f *ComponentFactory) CreateButtonBuilder() *buttons.ButtonBuilder {
	return buttons.NewButtonBuilder()
}

// CreateFormatterProvider создает FormatterProvider
func (f *ComponentFactory) CreateFormatterProvider() *formatters.FormatterProvider {
	return formatters.NewFormatterProvider(f.exchange)
}

// CreateAllComponents создает все компоненты
func (f *ComponentFactory) CreateAllComponents() ComponentSet {
	return ComponentSet{
		MessageSender:     f.CreateMessageSender(),
		ButtonBuilder:     f.CreateButtonBuilder(),
		FormatterProvider: f.CreateFormatterProvider(),
	}
}

// ComponentSet набор компонентов инфраструктуры
type ComponentSet struct {
	MessageSender     message_sender.MessageSender
	ButtonBuilder     *buttons.ButtonBuilder
	FormatterProvider *formatters.FormatterProvider
}

// Validate проверяет фабрику компонентов
func (f *ComponentFactory) Validate() bool {
	if f.config == nil {
		logger.Warn("⚠️ ComponentFactory: конфиг не доступен")
		return false
	}

	logger.Info("✅ ComponentFactory валидирована")
	return true
}

// stubMessageSender заглушка для MessageSender
type stubMessageSender struct{}

func (s *stubMessageSender) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	logger.Debug("[STUB] Отправка сообщения в %d: %s", chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubMessageSender) SendCounterMessage(chatID int64, text string, keyboard interface{}) error {
	return nil
}

func (s *stubMessageSender) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	logger.Debug("[STUB] Отправка сообщения с клавиатурой в %d: %s", chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubMessageSender) EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error {
	logger.Debug("[STUB] Редактирование сообщения %d в %d: %s", messageID, chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubMessageSender) DeleteMessage(chatID, messageID int64) error {
	logger.Debug("[STUB] Удаление сообщения %d в %d", messageID, chatID)
	return nil
}

func (s *stubMessageSender) AnswerCallback(callbackID, text string, showAlert bool) error {
	logger.Debug("[STUB] Ответ на callback %s: %s (showAlert: %v)", callbackID, text, showAlert)
	return nil
}

func (s *stubMessageSender) SetChatID(chatID int64) {
	logger.Debug("[STUB] Установка chat ID: %d", chatID)
}

func (s *stubMessageSender) GetChatID() int64 {
	return 0
}

func (s *stubMessageSender) SetTestMode(enabled bool) {
	logger.Debug("[STUB] Установка тестового режима: %v", enabled)
}

func (s *stubMessageSender) IsTestMode() bool {
	return false
}
