package session_stop

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// sessionStopHandler обрабатывает нажатие кнопки "🔴 Завершить сессию"
type sessionStopHandler struct {
	*base.BaseHandler
	service trading_session.Service
}

func newSessionStopHandler(service trading_session.Service) handlers.Handler {
	return &sessionStopHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "session_stop_handler",
			Command: constants.SessionButtonTexts.Stop + "*", // Добавляем * для паттерн-матчинга
			Type:    handlers.TypeMessage,
		},
		service: service,
	}
}

// Execute завершает торговую сессию
func (h *sessionStopHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	startKeyboard := telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.ReplyKeyboardButton{
			{{Text: constants.SessionButtonTexts.Start}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}

	if !h.service.IsActive(params.User.ID) {
		return handlers.HandlerResult{
			Message:  "ℹ️ Активной торговой сессии нет.",
			Keyboard: startKeyboard,
		}, nil
	}

	if err := h.service.Stop(params.User.ID); err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка завершения сессии: %w", err)
	}

	return handlers.HandlerResult{
		Message:  "🔴 *Сессия завершена*\n\nУведомления отключены.",
		Keyboard: startKeyboard,
	}, nil
}
