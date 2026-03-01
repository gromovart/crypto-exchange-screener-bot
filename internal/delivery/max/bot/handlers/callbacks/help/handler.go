// internal/delivery/max/bot/handlers/callbacks/help/handler.go
package help

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик помощи (callback)
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик помощи
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("help_cb", kb.CbHelp, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	msg := "📋 *Помощь*\n\n" +
		"*Команды:*\n" +
		"/start — Главное меню\n" +
		"/help — Эта справка\n\n" +
		"*Разделы:*\n" +
		"• 🔔 Уведомления — вкл/выкл и типы\n" +
		"• 📈 Сигналы — настройка порогов\n" +
		"• ⏱️ Периоды — выбор таймфреймов\n" +
		"• 👤 Профиль — информация о подписке\n" +
		"• 🔄 Сбросить — сброс настроек\n\n" +
		"*Как работает:*\n" +
		"Бот отслеживает рынок криптовалют и присылает сигналы когда цена меняется на заданный процент.\n\n" +
		"Вы можете настроить:\n" +
		"— Пороги изменения (от 0.5% до 50%)\n" +
		"— Направление (рост / падение / оба)\n" +
		"— Таймфреймы для анализа"

	rows := [][]map[string]string{
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
