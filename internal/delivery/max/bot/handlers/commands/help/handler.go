// internal/delivery/max/bot/handlers/commands/help/handler.go
package help

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

type helpHandler struct{ *base.BaseHandler }

func NewHandler() handlers.Handler {
	return &helpHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "max_help_handler",
			Command: "help",
			Type:    handlers.TypeCommand,
		},
	}
}

func (h *helpHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	keyboard := kb.Keyboard([][]map[string]string{
		{kb.BUrl("📚 Документация", "https://teletype.in/@gromovart/pj2UIVlmr55")},
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	})
	return handlers.HandlerResult{Message: HelpText(), Keyboard: keyboard}, nil
}

// HelpText — общий текст справки, используется в command и callback хэндлерах.
func HelpText() string {
	return "📋 *Справка по командам*\n\n" +
		"━━━ 🤖 КОМАНДЫ ━━━\n\n" +
		"/start — 🏠 Главное меню\n" +
		"/help — 📋 Эта справка\n" +
		"/menu — 📋 Открыть меню\n" +
		"/profile — 👤 Ваш профиль\n" +
		"/settings — ⚙️ Настройки\n" +
		"/notifications — 🔔 Уведомления\n" +
		"/signals — 📈 Сигналы\n" +
		"/periods — ⏱️ Периоды\n" +
		"/thresholds — 🎯 Пороги срабатывания\n" +
		"/stats — 📊 Статистика\n" +
		"/link — 🔗 Привязать Telegram-аккаунт\n" +
		"/buy — 💎 Купить подписку\n" +
		"/paysupport — 🛟 Поддержка\n" +
		"/terms — 📜 Условия использования\n\n" +
		"━━━ 📊 КАК РАБОТАЕТ БОТ ━━━\n\n" +
		"1️⃣ Анализирует рынок фьючерсов Bybit в реальном времени\n" +
		"2️⃣ Обнаруживает сильные движения цен (рост/падение)\n" +
		"3️⃣ Отправляет уведомления при превышении заданных порогов\n" +
		"4️⃣ Считает сигналы по выбранным периодам\n\n" +
		"━━━ 📐 НАСТРОЙКИ ПО УМОЛЧАНИЮ ━━━\n\n" +
		"📈 Рост: 2.0%\n" +
		"📉 Падение: 2.0%\n" +
		"⏱️ Периоды: 5м, 15м, 30м\n" +
		"🔔 Уведомления: включены\n\n" +
		"━━━ 📞 ПОДДЕРЖКА ━━━\n\n" +
		"📧 Email: support@gromovart.ru\n" +
		"💬 Telegram: @crypto_exchange_screener"
}
