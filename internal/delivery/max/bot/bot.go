// internal/delivery/max/bot/bot.go
package bot

import (
	"context"
	"strings"
	"time"

	maxpkg "crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/internal/delivery/auth"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/router"
	cbWatchlistToggle "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/watchlist_toggle"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/middleware"
	"crypto-exchange-screener-bot/internal/delivery/max/transport"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Bot — оркестратор MAX бота: polling/webhook + маршрутизация
type Bot struct {
	client           *maxpkg.Client
	sender           message_sender.MessageSender
	router           router.Router
	authMiddleware   *middleware.AuthMiddleware
	poller           *transport.Poller
	webhookServer    *transport.WebhookServer
	otpServer        *auth.Server
	mode             string // "polling" или "webhook"
	userService      *users.Service
	watchlistService watchlistSvc.Service
}

// NewBot создаёт MAX бота с зарегистрированными хэндлерами
func NewBot(client *maxpkg.Client, deps Dependencies) *Bot {
	sender := message_sender.NewSender(client, true)
	b := &Bot{
		client:           client,
		sender:           sender,
		router:           router.NewRouter(),
		authMiddleware:   middleware.NewAuthMiddleware(deps.UserService),
		otpServer:        newOTPServer(deps.AuthConfig, deps.UserService, sender),
		mode:             "polling", // По умолчанию polling
		userService:      deps.UserService,
		watchlistService: deps.WatchlistService,
	}

	// Регистрируем все хэндлеры
	RegisterAll(b.router, deps)

	// Создаём поллер для polling режима
	b.poller = transport.NewPoller(client, b.HandleUpdate)

	return b
}

// SetWebhookMode настраивает бота для работы в режиме webhook
func (b *Bot) SetWebhookMode(config transport.WebhookConfig) error {
	b.mode = "webhook"
	b.webhookServer = transport.NewWebhookServer(config, b.client, b.HandleUpdate)
	return nil
}

// HandleUpdate обрабатывает одно входящее обновление
func (b *Bot) HandleUpdate(upd maxpkg.Update) {
	// Аутентифицируем и получаем HandlerParams
	params, err := b.authMiddleware.ProcessUpdate(upd)
	if err != nil {
		logger.Warn("⚠️ MAX HandleUpdate: auth: %v", err)
		return
	}

	// ⭐ FSM: перехватываем не-командный текст если пользователь в режиме поиска вотчлиста
	if upd.UpdateType == "message_created" && upd.Message != nil {
		text := strings.TrimSpace(upd.Message.Body.Text)
		if text != "" && !strings.HasPrefix(text, "/") &&
			b.userService != nil && b.watchlistService != nil {
			state, _ := b.userService.GetUserState(params.User.ID)
			if state == "watchlist_search" {
				_ = b.userService.ClearUserState(params.User.ID)
				result, err := cbWatchlistToggle.ExecuteSearch(b.watchlistService, params.User.ID, text)
				if err != nil {
					_ = b.sender.SendTextMessage(params.ChatID, "Ошибка поиска: "+err.Error(), nil)
					return
				}
				_ = b.sender.SendMenuMessage(params.ChatID, result.Message, result.Keyboard)
				return
			}
		}
	}

	// Определяем команду/callback для маршрутизации
	command := resolveCommand(upd, params)
	if command == "" {
		logger.Debug("⏭️ MAX HandleUpdate: пустая команда, пропускаем")
		return
	}

	logger.Info("📨 MAX HandleUpdate: route='%s' user=%d mode=%s", command, params.User.ID, b.mode)

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
	} else if result.AutoDeleteAfter > 0 {
		// Отправляем с получением mid, затем удаляем через заданный интервал
		mid, err := b.sender.SendMenuMessageWithID(params.ChatID, result.Message, result.Keyboard)
		if err != nil {
			logger.Warn("⚠️ MAX SendMenuMessageWithID: %v", err)
		} else if mid != "" {
			delay := result.AutoDeleteAfter
			go func() {
				time.Sleep(delay)
				if delErr := b.sender.DeleteMessage(mid); delErr != nil {
					logger.Debug("⚠️ MAX AutoDelete: %v", delErr)
				} else {
					logger.Info("🗑️ MAX AutoDelete: удалено сообщение mid=%s (через %s)", mid, delay)
				}
			}()
		}
	} else {
		_ = b.sender.SendMenuMessage(params.ChatID, result.Message, result.Keyboard)
	}
}

// GetSender возвращает message sender MAX бота (для уведомлений из других сервисов)
func (b *Bot) GetSender() message_sender.MessageSender {
	return b.sender
}

// GetOTPServer возвращает auth OTP-сервер (nil если не настроен)
func (b *Bot) GetOTPServer() *auth.Server {
	return b.otpServer
}

// Start запускает бот в соответствующем режиме. Блокирует до отмены ctx.
func (b *Bot) Start(ctx context.Context) {
	// Запускаем OTP auth-сервер (если настроен)
	if b.otpServer != nil {
		if err := b.otpServer.Start(); err != nil {
			logger.Error("❌ Не удалось запустить Auth OTP-сервер: %v", err)
		} else {
			logger.Info("✅ Auth OTP-сервер запущен")
		}
	}

	if b.mode == "webhook" {
		if b.webhookServer == nil {
			logger.Error("❌ MAX Bot: webhook сервер не инициализирован")
			return
		}
		logger.Info("🤖 MAX Bot запущен (режим: webhook)")
		if err := b.webhookServer.Start(); err != nil {
			logger.Error("❌ Не удалось запустить MAX webhook: %v", err)
			return
		}
		// Блокируем до отмены контекста
		<-ctx.Done()
		logger.Info("🛑 Остановка MAX webhook сервера...")
		b.webhookServer.Stop()
	} else {
		logger.Info("🤖 MAX Bot запущен (режим: polling)")
		b.poller.Run(ctx)
	}
	logger.Info("🤖 MAX Bot остановлен")
}

// Stop останавливает бота
func (b *Bot) Stop() error {
	if b.otpServer != nil {
		if err := b.otpServer.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки Auth OTP-сервера: %v", err)
		}
	}
	if b.mode == "webhook" && b.webhookServer != nil {
		return b.webhookServer.Stop()
	}
	return nil
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
			// Убираем "@botname", но сохраняем аргументы для роутера
			// "/help@bot" → "/help", "/link ABC123" → "/link ABC123"
			cmd := strings.SplitN(text, "@", 2)[0]
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
