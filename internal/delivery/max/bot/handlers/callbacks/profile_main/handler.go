// internal/delivery/max/bot/handlers/callbacks/profile_main/handler.go
package profile_main

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик главной страницы профиля
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик профиля
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("profile_main", kb.CbProfileMain, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	name := "Пользователь"
	username := ""
	tier := "Free"

	if user != nil {
		if user.FirstName != "" {
			name = user.FirstName
		}
		if user.Username != "" {
			username = "@" + user.Username
		}
		if user.SubscriptionTier != "" {
			tier = user.SubscriptionTier
		}
	}

	msg := fmt.Sprintf(
		"👤 *Профиль*\n\n"+
			"Имя: %s\n"+
			"%s\n"+
			"Подписка: %s\n\n"+
			"Выберите раздел:",
		name, username, tier,
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.ProfileStats, kb.CbProfileStats)},
		{kb.B(kb.Btn.ProfileSubscription, kb.CbProfileSubscription)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
