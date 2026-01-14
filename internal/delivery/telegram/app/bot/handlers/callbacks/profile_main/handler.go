// internal/delivery/telegram/app/bot/handlers/callbacks/profile_main/handler.go
package profile_main

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// profileMainHandler реализация обработчика профиля
type profileMainHandler struct {
	*base.BaseHandler // Изменено на указатель
}

// NewHandler создает новый обработчик профиля
func NewHandler() handlers.Handler {
	return &profileMainHandler{
		BaseHandler: &base.BaseHandler{ // Изменено на указатель
			Name:    "profile_main_handler",
			Command: constants.CallbackProfileMain,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback профиля
func (h *profileMainHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
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
func (h *profileMainHandler) formatProfileMessage(user *models.User) string {
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

	return fmt.Sprintf(
		"%s\n\n"+
			"🆔 ID: %d\n"+
			"📱 Telegram ID: %d\n"+
			"👤 Имя: %s\n"+
			"📧 Username: %s\n"+
			"⭐ Роль: %s\n"+
			"💰 Тариф: %s\n"+
			"✅ Статус: %s\n"+
			"📅 Регистрация: %s\n"+
			"🔐 Последний вход: %s\n\n"+
			"%s\n"+
			"📈 Сигналов сегодня: %d/%d\n"+
			"🎯 Мин. рост: %.2f%%\n"+
			"📉 Мин. падение: %.2f%%\n",
		constants.MenuButtonTexts.Profile,
		user.ID,
		user.TelegramID,
		firstName,
		username,
		h.GetRoleDisplay(user.Role),
		h.GetSubscriptionTierDisplayName(user.SubscriptionTier),
		h.GetStatusDisplay(user.IsActive),
		user.CreatedAt.Format("02.01.2006"),
		lastLoginDisplay,
		constants.AuthButtonTexts.Stats, // Используем AuthButtonTexts.Stats для "Статистика"
		user.SignalsToday,
		user.MaxSignalsPerDay,
		user.MinGrowthThreshold,
		user.MinFallThreshold,
	)
}

// createProfileKeyboard создает клавиатуру для профиля
func (h *profileMainHandler) createProfileKeyboard() interface{} {
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

// GetSubscriptionTierDisplayName возвращает отображаемое имя тарифа
func (h *profileMainHandler) GetSubscriptionTierDisplayName(tier string) string {
	switch tier {
	case "enterprise":
		return "🏢 Enterprise"
	case "pro":
		return "🚀 Pro"
	case "basic":
		return "📱 Basic"
	case "free":
		return "🆓 Free"
	default:
		return tier
	}
}

// GetStatusDisplay возвращает отображение статуса
func (h *profileMainHandler) GetStatusDisplay(isActive bool) string {
	if isActive {
		return "✅ Активен"
	}
	return "❌ Деактивирован"
}
