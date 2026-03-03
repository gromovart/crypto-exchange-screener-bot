// internal/delivery/max/bot/handlers/commands/link/handler.go
package link

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик команды /link [КОД] в MAX боте.
// Без аргументов: перенаправляет на экран инструкций (CbLinkTelegram).
// С аргументом-кодом: выполняет привязку Telegram-аккаунта.
type Handler struct {
	*base.BaseHandler
	userService *users.Service
}

// New создаёт обработчик
func New(svc *users.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("link_command", "/link", handlers.TypeCommand),
		userService: svc,
	}
}

// Execute обрабатывает /link или /link КОД
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	// Если уже привязан
	if user.HasLinkedTelegram() {
		return handlers.HandlerResult{
			Message: "✅ Telegram-аккаунт уже привязан. Подписка синхронизирована.",
			Keyboard: kb.Keyboard([][]map[string]string{
				kb.BackRow(kb.CbMenuMain),
			}),
			EditMessage: false,
		}, nil
	}

	// /link без кода — показываем инструкции
	code := strings.TrimSpace(params.Data)
	if code == "" {
		// Перенаправляем на экран CbLinkTelegram
		return handlers.HandlerResult{
			Message: "🔗 *Привязка Telegram-аккаунта*\n\n" +
				"Получите код в Telegram-боте командой `/link`,\n" +
				"затем введите: `/link КОД`\n\n" +
				"Например: `/link ABC123`",
			Keyboard: kb.Keyboard([][]map[string]string{
				kb.BackRow(kb.CbMenuMain),
			}),
			EditMessage: false,
		}, nil
	}

	// Нормализуем код: убираем пробелы, делаем uppercase
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 6 {
		return handlers.HandlerResult{
			Message: "❌ Неверный код. Код должен состоять из 6 символов.\n\nПопробуйте ещё раз: `/link КОД`",
			Keyboard: kb.Keyboard([][]map[string]string{
				kb.BackRow(kb.CbMenuMain),
			}),
			EditMessage: false,
		}, nil
	}

	// Выполняем привязку
	maxChatID := strconv.FormatInt(params.ChatID, 10)
	linkedUser, err := h.userService.LinkMaxAccount(user.TelegramID, maxChatID, code)
	if err != nil {
		return handlers.HandlerResult{
			Message: fmt.Sprintf("❌ Не удалось привязать аккаунт.\n\n%s\n\nПроверьте код и попробуйте снова.", friendlyError(err)),
			Keyboard: kb.Keyboard([][]map[string]string{
				kb.BackRow(kb.CbMenuMain),
			}),
			EditMessage: false,
		}, nil
	}

	tier := linkedUser.SubscriptionTier
	if tier == "" {
		tier = "Free"
	}

	return handlers.HandlerResult{
		Message: fmt.Sprintf(
			"✅ *Telegram-аккаунт успешно привязан!*\n\n"+
				"Ваша подписка: *%s*\n\n"+
				"Теперь вы можете пользоваться всеми возможностями бота.",
			tier,
		),
		Keyboard: kb.Keyboard([][]map[string]string{
			{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
		}),
		EditMessage: false,
	}, nil
}

func friendlyError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "неверный или истёкший") {
		return "Код недействителен или истёк срок его действия."
	}
	return "Внутренняя ошибка сервера."
}
