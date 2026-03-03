// internal/delivery/max/bot/handlers/commands/terms/handler.go
package terms

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

type termsHandler struct{ *base.BaseHandler }

func NewHandler() handlers.Handler {
	return &termsHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "max_terms_handler",
			Command: "terms",
			Type:    handlers.TypeCommand,
		},
	}
}

func (h *termsHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	msg := "📜 *Условия использования*\n\n" +

		"*1. Общие положения*\n" +
		"Бот предоставляет информацию о рынке криптовалют в реальном времени. " +
		"Используя бота, вы подтверждаете согласие с данными условиями.\n\n" +

		"*2. Информационный характер*\n" +
		"Все сигналы и данные носят исключительно информационный характер. " +
		"Бот не даёт финансовых рекомендаций и не является руководством к действию. " +
		"Вы самостоятельно принимаете все торговые решения.\n\n" +

		"*3. Риски*\n" +
		"Рынок криптовалют характеризуется высокой волатильностью. " +
		"Торговля связана с высокими рисками потери капитала. " +
		"Прошлые результаты не гарантируют будущей доходности.\n\n" +

		"*4. Ответственность*\n" +
		"Мы не несём ответственности за прямые или косвенные убытки. " +
		"Бот предоставляется «как есть» без гарантий. " +
		"Мы не отвечаем за задержки в работе MAX или биржи.\n\n" +

		"*5. Изменение условий*\n" +
		"Мы оставляем право изменять условия использования. " +
		"Продолжение использования бота означает согласие с изменениями.\n\n" +

		"*6. Контакты*\n" +
		"📧 Email: support@gromovart.ru\n" +
		"💬 Telegram: @artemgrrr"

	keyboard := kb.Keyboard([][]map[string]string{
		{kb.B(kb.Btn.Help, kb.CbHelp), kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	})

	return handlers.HandlerResult{Message: msg, Keyboard: keyboard}, nil
}
