// internal/delivery/telegram/app/bot/handlers/commands/terms/handler.go
package terms

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// termsCommandHandler реализация обработчика команды /terms
type termsCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /terms
func NewHandler() handlers.Handler {
	return &termsCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "terms_command_handler",
			Command: "terms",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /terms
func (h *termsCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	message := fmt.Sprintf(
		"📜 *Условия использования*\n\n" +
			"*1. Общие положения*\n" +
			"1.1. Бот предоставляет информацию о рынке криптовалют в реальном времени.\n" +
			"1.2. Используя бота, вы подтверждаете, что ознакомились и согласны с данными условиями.\n\n" +

			"*2. Информационный характер*\n" +
			"2.1. Все сигналы и данные носят исключительно информационный характер.\n" +
			"2.2. Бот не дает финансовых рекомендаций и не является руководством к действию.\n" +
			"2.3. Вы самостоятельно принимаете все торговые решения и несете за них ответственность.\n\n" +

			"*3. Риски*\n" +
			"3.1. Рынок криптовалют характеризуется высокой волатильностью.\n" +
			"3.2. Торговля связана с высокими рисками потери капитала.\n" +
			"3.3. Прошлые результаты не гарантируют будущей доходности.\n\n" +

			"*4. Подписки и платежи*\n" +
			"4.1. Оплата производится в Telegram Stars через официальные методы.\n" +
			"4.2. Подписка активируется автоматически после успешной оплаты.\n" +
			"4.3. Длительность подписки указывается при выборе тарифа.\n" +
			"4.4. Средства за неиспользованный период не возвращаются.\n\n" +

			"*5. Возвраты и споры*\n" +
			"5.1. Возвраты возможны только в случаях технических проблем с ботом.\n" +
			"5.2. Для возврата используйте команду /paysupport.\n" +
			"5.3. Решение о возврате принимается в течение 7 рабочих дней.\n\n" +

			"*6. Ответственность*\n" +
			"6.1. Мы не несем ответственности за прямые или косвенные убытки.\n" +
			"6.2. Бот предоставляется \"как есть\" без гарантий.\n" +
			"6.3. Мы не отвечаем за задержки в работе Telegram или биржи.\n\n" +

			"*7. Изменение условий*\n" +
			"7.1. Мы оставляем право изменять условия использования.\n" +
			"7.2. Продолжение использования бота означает согласие с изменениями.\n\n" +

			"*8. Контакты*\n" +
			"📧 Email: support@gromovart.ru\n" +
			"💬 Telegram: @crypto_exchange_screener\n\n" +

			"*9. Подтверждение*\n" +
			"Нажимая кнопку \"💳 Оплатить\", вы подтверждаете, что прочитали и согласны с условиями.",
	)

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "✅ Принять и продолжить", "callback_data": constants.CallbackMenuMain},
			},
			{
				{"text": "📋 Помощь", "callback_data": constants.CallbackHelp},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
			"command": "terms",
		},
	}, nil
}
