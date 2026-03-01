// internal/delivery/max/bot/bot.go
package bot

import (
	"context"
	"strings"

	maxpkg "crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/middleware"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/router"
	"crypto-exchange-screener-bot/internal/delivery/max/transport"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Bot — оркестратор MAX бота: polling + маршрутизация
type Bot struct {
	client         *maxpkg.Client
	sender         message_sender.MessageSender
	router         router.Router
	authMiddleware *middleware.AuthMiddleware
	poller         *transport.Poller
}

// NewBot создаёт MAX бота с зарегистрированными хэндлерами
func NewBot(client *maxpkg.Client, deps Dependencies) *Bot {
	b := &Bot{
		client:         client,
		sender:         message_sender.NewSender(client, true),
		router:         router.NewRouter(),
		authMiddleware: middleware.NewAuthMiddleware(deps.UserService),
	}

	// Регистрируем все хэндлеры
	RegisterAll(b.router, deps)

	// Создаём поллер
	b.poller = transport.NewPoller(client, b.HandleUpdate)

	return b
}

// HandleUpdate обрабатывает одно входящее обновление
func (b *Bot) HandleUpdate(upd maxpkg.Update) {
	// Аутентифицируем и получаем HandlerParams
	params, err := b.authMiddleware.ProcessUpdate(upd)
	if err != nil {
		logger.Warn("⚠️ MAX HandleUpdate: auth: %v", err)
		return
	}

	// Определяем команду/callback для маршрутизации
	command := resolveCommand(upd, params)
	if command == "" {
		logger.Debug("⏭️ MAX HandleUpdate: пустая команда, пропускаем")
		return
	}

	logger.Info("📨 MAX HandleUpdate: route='%s' user=%d", command, params.User.ID)

	// Маршрутизируем
	result, err := b.router.Handle(command, params)
	if err != nil {
		logger.Warn("⚠️ MAX router: %v", err)
		// Отправляем fallback-ответ
		_ = b.sender.SendTextMessage(params.ChatID, "❓ Неизвестная команда. Попробуйте /start", nil)
		return
	}

	// Отвечаем на callback (убираем loading spinner)
	if params.CallbackID != "" {
		if cbErr := b.sender.AnswerCallback(params.CallbackID, "", false); cbErr != nil {
			logger.Debug("⚠️ MAX AnswerCallback: %v", cbErr)
		}
	}

	if result.Message == "" {
		return
	}

	// Редактируем или отправляем новое сообщение
	if result.EditMessage && params.MessageID > 0 {
		if editErr := b.sender.EditMessageText(params.ChatID, params.MessageID, result.Message, result.Keyboard); editErr != nil {
			logger.Warn("⚠️ MAX EditMessage failed (%v), sending new", editErr)
			_ = b.sender.SendMenuMessage(params.ChatID, result.Message, result.Keyboard)
		}
	} else {
		_ = b.sender.SendMenuMessage(params.ChatID, result.Message, result.Keyboard)
	}
}

// Start запускает long-polling. Блокирует до отмены ctx.
func (b *Bot) Start(ctx context.Context) {
	logger.Info("🤖 MAX Bot запущен")
	b.SetMyCommands()
	b.poller.Run(ctx)
	logger.Info("🤖 MAX Bot остановлен")
}

// SetMyCommands регистрирует команды в меню бота
func (b *Bot) SetMyCommands() {
	cmds := []maxpkg.BotCommand{
		{Command: "start", Description: "Главное меню"},
		{Command: "help", Description: "Помощь"},
		{Command: "menu", Description: "Открыть меню"},
	}
	if err := b.client.SetMyCommands(cmds); err != nil {
		logger.Warn("⚠️ MAX SetMyCommands: %v", err)
	} else {
		logger.Info("✅ MAX: команды бота зарегистрированы")
	}
}

// resolveCommand определяет строку-команду для роутера из входящего Update
func resolveCommand(upd maxpkg.Update, params handlers.HandlerParams) string {
	switch {
	case upd.CallbackQuery != nil:
		// Callback — используем data напрямую
		return upd.CallbackQuery.Data

	case upd.Message != nil:
		text := strings.TrimSpace(upd.Message.Text)
		if text == "" {
			return ""
		}
		// Команда начинается с /
		if strings.HasPrefix(text, "/") {
			// Убираем @botname если есть
			cmd := strings.SplitN(text, "@", 2)[0]
			return cmd
		}
		// Обычный текст — обрабатываем как текст
		return text

	default:
		return ""
	}
}
