// internal/delivery/telegram/app/bot/handlers/commands/profile/handler.go
package profile

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// profileCommandHandler реализация обработчика команды /profile
type profileCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /profile
func NewHandler() handlers.Handler {
	return &profileCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "profile_command_handler",
			Command: "profile",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /profile
func (h *profileCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := h.formatProfileMessage(params.User)
	keyboard := h.createProfileKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// formatProfileMessage форматирует сообщение профиля
func (h *profileCommandHandler) formatProfileMessage(user *models.User) string {
	firstName := user.FirstName
	if firstName == "" {
		firstName = "Гость"
	}

	username := user.Username
	if username == "" {
		username = "не указан"
	} else {
		username = "@" + username
	}

	// Форматируем дату последнего входа
	lastLoginDisplay := "еще не входил"
	if !user.LastLoginAt.IsZero() {
		lastLoginDisplay = user.LastLoginAt.Format("02.01.2006 15:04")
	}

	// Определяем статус пользователя (не подписки)
	userStatus := "✅ Активен"
	if !user.IsActive {
		userStatus = "❌ Заблокирован"
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"🆔 ID: %d\n"+
			"📱 Telegram ID: %d\n"+
			"👤 Имя: %s\n"+
			"📧 Username: %s\n"+
			"⭐ Роль: %s\n"+
			"💰 Тариф: %s\n"+
			"👤 Статус пользователя: %s\n"+ // ⭐ Изменено: явно указано "Статус пользователя"
			"📅 Регистрация: %s\n"+
			"🔐 Последний вход: %s\n\n"+
			"%s\n"+
			"📈 Сигналов сегодня: %d/%d\n"+
			"🎯 Мин. рост: %.2f%%\n"+
			"📉 Мин. падение: %.2f%%\n",
		constants.AuthButtonTexts.Profile,
		user.ID,
		user.TelegramID,
		firstName,
		username,
		h.GetRoleDisplay(user.Role),
		h.GetSubscriptionTierDisplayName(user.SubscriptionTier),
		userStatus, // ⭐ Статус пользователя (активен/заблокирован)
		user.CreatedAt.Format("02.01.2006"),
		lastLoginDisplay,
		constants.AuthButtonTexts.Stats,
		user.SignalsToday,
		user.MaxSignalsPerDay,
		user.MinGrowthThreshold,
		user.MinFallThreshold,
	)
}

// createProfileKeyboard создает клавиатуру для профиля
func (h *profileCommandHandler) createProfileKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.AuthButtonTexts.Stats, "callback_data": constants.CallbackProfileStats},
				{"text": constants.AuthButtonTexts.Premium, "callback_data": constants.CallbackProfileSubscription},
			},
			{
				{"text": constants.ButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
