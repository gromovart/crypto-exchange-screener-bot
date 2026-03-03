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
		if cbErr := b.sender.AnswerCallback(params.CallbackID, ""); cbErr != nil {
			logger.Debug("⚠️ MAX AnswerCallback: %v", cbErr)
		}
	}

	if result.Message == "" {
		return
	}

	// Редактируем или отправляем новое сообщение.
	// EditMessage выполняем только для callback-ов (params.CallbackID != ""),
	// чтобы не пытаться редактировать сообщение пользователя при вводе команды.
	if result.EditMessage && params.MessageID != "" && params.CallbackID != "" {
		if editErr := b.sender.EditMessageText(params.MessageID, result.Message, result.Keyboard); editErr != nil {
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
	b.poller.Run(ctx)
	logger.Info("🤖 MAX Bot остановлен")
}


// resolveCommand определяет строку-команду для роутера из входящего Update
func resolveCommand(upd maxpkg.Update, params handlers.HandlerParams) string {
	switch upd.UpdateType {
	case "message_callback":
		if upd.Callback != nil {
			return upd.Callback.Payload
		}
		return ""

	case "message_created":
		if upd.Message == nil {
			return ""
		}
		text := strings.TrimSpace(upd.Message.Body.Text)
		if text == "" {
			return ""
		}
		// Команда с \/: "/help", "/start", "/notifications" и т.п.
		if strings.HasPrefix(text, "/") {
			// Убираем "@botname" и аргументы для чистого имени команды
			cmd := strings.SplitN(text, "@", 2)[0]  // "/help@bot" → "/help"
			cmd = strings.SplitN(cmd, " ", 2)[0]    // "/start arg" → "/start"
			return cmd
		}
		return ""

	case "bot_started":
		// Пользователь нажал START — маршрутизируем как /start
		return "/start"

	default:
		return ""
	}
}
