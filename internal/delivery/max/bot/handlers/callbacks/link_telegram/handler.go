// internal/delivery/max/bot/handlers/callbacks/link_telegram/handler.go
package link_telegram

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — экран привязки Telegram-аккаунта.
// Показывает инструкцию: получи код через /link в Telegram-боте,
// затем введи его здесь командой /link КОД.
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("link_telegram", kb.CbLinkTelegram, handlers.TypeCallback),
	}
}

// Execute показывает инструкцию привязки аккаунта
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	// Если уже привязан
	if user != nil && user.HasLinkedTelegram() {
		msg := "✅ *Telegram-аккаунт уже привязан*\n\n" +
			"Ваш аккаунт MAX связан с Telegram.\n" +
			"Подписка и настройки синхронизированы."

		rows := [][]map[string]string{
			kb.BackRow(kb.CbProfileMain),
		}
		return handlers.HandlerResult{
			Message:     msg,
			Keyboard:    kb.Keyboard(rows),
			EditMessage: params.MessageID != "",
		}, nil
	}

	msg := "🔗 *Привязка Telegram-аккаунта*\n\n" +
		"Привяжите ваш Telegram-аккаунт, чтобы использовать подписку в MAX.\n\n" +
		"*Как получить код:*\n" +
		"1️⃣ Откройте бота в Telegram\n" +
		"2️⃣ Отправьте команду `/link`\n" +
		"3️⃣ Получите 6-символьный код\n\n" +
		"*Как ввести код:*\n" +
		"Отправьте сюда: `/link КОД`\n" +
		"Например: `/link ABC123`\n\n" +
		"Код действителен 15 минут."

	rows := [][]map[string]string{
		kb.BackRow(kb.CbProfileMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
