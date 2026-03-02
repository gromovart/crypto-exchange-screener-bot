// internal/delivery/max/bot/handlers/callbacks/auth_login/handler.go
package auth_login

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик входа (авторизация автоматическая через middleware)
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("auth_login", kb.CbAuthLogin, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
// Авторизация происходит автоматически через AuthMiddleware,
// поэтому здесь просто показываем главное меню
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	name := "!"
	if user != nil && user.FirstName != "" {
		name = ", " + user.FirstName + "!"
	}

	msg := "✅ *Вы авторизованы" + name + "*\n\nДобро пожаловать в Crypto Screener Bot."

	rows := [][]map[string]string{
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
