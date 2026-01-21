// internal/delivery/telegram/transport/factory.go
package transport

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
)

// TransportType тип транспорта Telegram
type TransportType string

const (
	TransportTypePolling TransportType = "polling"
	TransportTypeWebhook TransportType = "webhook"
)

// TelegramTransport интерфейс транспорта Telegram
type TelegramTransport interface {
	// Запуск транспорта
	Start() error

	// Остановка транспорта
	Stop() error

	// Проверка работы
	IsRunning() bool

	// Получение имени транспорта
	Name() string

	// Получение типа транспорта
	Type() TransportType

	// Получение бота (если нужно)
	GetBot() *bot.TelegramBot
}

// TransportFactory фабрика для создания транспорта Telegram
type TransportFactory struct {
	config *config.Config
	bot    *bot.TelegramBot
}

// NewTransportFactory создает новую фабрику транспорта
func NewTransportFactory(cfg *config.Config, botInstance *bot.TelegramBot) *TransportFactory {
	return &TransportFactory{
		config: cfg,
		bot:    botInstance,
	}
}

// CreateTransport создает транспорт на основе конфигурации
func (f *TransportFactory) CreateTransport() (TelegramTransport, error) {
	if f.config.IsWebhookMode() {
		return f.createWebhookTransport()
	}
	return f.createPollingTransport()
}

// createWebhookTransport создает вебхук транспорт
func (f *TransportFactory) createWebhookTransport() (TelegramTransport, error) {
	if f.bot == nil {
		return nil, fmt.Errorf("бот не инициализирован для создания вебхука")
	}

	// Создаем вебхук сервер
	webhookServer := bot.NewWebhookServer(f.config, f.bot)
	if webhookServer == nil {
		return nil, fmt.Errorf("не удалось создать вебхук сервер")
	}

	return &webhookTransport{
		server: webhookServer,
		bot:    f.bot,
		name:   "WebhookTransport",
	}, nil
}

// createPollingTransport создает polling транспорт
func (f *TransportFactory) createPollingTransport() (TelegramTransport, error) {
	if f.bot == nil {
		return nil, fmt.Errorf("бот не инициализирован для создания polling")
	}

	return &pollingTransport{
		bot:  f.bot,
		name: "PollingTransport",
	}, nil
}

// ============================================
// РЕАЛИЗАЦИЯ WEBHOOK ТРАНСПОРТА
// ============================================

type webhookTransport struct {
	server  *bot.WebhookServer
	bot     *bot.TelegramBot
	name    string
	running bool
}

func (wt *webhookTransport) Start() error {
	if wt.server == nil {
		return fmt.Errorf("вебхук сервер не инициализирован")
	}

	if err := wt.server.Start(); err != nil {
		return fmt.Errorf("ошибка запуска вебхук сервера: %w", err)
	}

	wt.running = true
	return nil
}

func (wt *webhookTransport) Stop() error {
	if wt.server == nil || !wt.running {
		return nil
	}

	if err := wt.server.Stop(); err != nil {
		return fmt.Errorf("ошибка остановки вебхук сервера: %w", err)
	}

	wt.running = false
	return nil
}

func (wt *webhookTransport) IsRunning() bool {
	return wt.running && wt.server != nil
}

func (wt *webhookTransport) Name() string {
	return wt.name
}

func (wt *webhookTransport) Type() TransportType {
	return TransportTypeWebhook
}

func (wt *webhookTransport) GetBot() *bot.TelegramBot {
	return wt.bot
}

// ============================================
// РЕАЛИЗАЦИЯ POLLING ТРАНСПОРТА
// ============================================

type pollingTransport struct {
	bot     *bot.TelegramBot
	name    string
	running bool
}

func (pt *pollingTransport) Start() error {
	if pt.bot == nil {
		return fmt.Errorf("бот не инициализирован")
	}

	// Проверяем есть ли метод StartPolling
	if botWithPolling, ok := interface{}(pt.bot).(interface{ StartPolling() error }); ok {
		if err := botWithPolling.StartPolling(); err != nil {
			return fmt.Errorf("ошибка запуска polling: %w", err)
		}
	} else {
		return fmt.Errorf("бот не поддерживает polling")
	}

	pt.running = true
	return nil
}

func (pt *pollingTransport) Stop() error {
	if pt.bot == nil || !pt.running {
		return nil
	}

	// Проверяем есть ли метод StopPolling
	if botWithPolling, ok := interface{}(pt.bot).(interface{ StopPolling() error }); ok {
		if err := botWithPolling.StopPolling(); err != nil {
			return fmt.Errorf("ошибка остановки polling: %w", err)
		}
	}

	pt.running = false
	return nil
}

func (pt *pollingTransport) IsRunning() bool {
	if pt.bot == nil {
		return false
	}

	// Проверяем статус polling через бота
	if pollingBot, ok := interface{}(pt.bot).(interface{ IsPolling() bool }); ok {
		return pollingBot.IsPolling()
	}

	return pt.running
}

func (pt *pollingTransport) Name() string {
	return pt.name
}

func (pt *pollingTransport) Type() TransportType {
	return TransportTypePolling
}

func (pt *pollingTransport) GetBot() *bot.TelegramBot {
	return pt.bot
}

// CreateDefaultTransport создает транспорт по умолчанию
func CreateDefaultTransport(cfg *config.Config, botInstance *bot.TelegramBot) (TelegramTransport, error) {
	factory := NewTransportFactory(cfg, botInstance)
	return factory.CreateTransport()
}
