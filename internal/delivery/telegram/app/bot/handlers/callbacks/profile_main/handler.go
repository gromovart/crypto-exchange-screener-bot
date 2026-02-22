// internal/delivery/telegram/app/bot/handlers/callbacks/profile_main/handler.go
package profile_main

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/profile"
)

// profileMainHandler реализация обработчика профиля
type profileMainHandler struct {
	*base.BaseHandler
	profileService profile.Service
}

// NewHandler создает новый обработчик профиля
func NewHandler(profileService profile.Service) handlers.Handler {
	return &profileMainHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "profile_main_handler",
			Command: constants.CallbackProfileMain,
			Type:    handlers.TypeCallback,
		},
		profileService: profileService,
	}
}

// Execute выполняет обработку callback профиля
func (h *profileMainHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Создаем параметры для сервиса профиля
	profileParams := profile.ProfileParams{
		UserID: params.User.TelegramID,
		Action: "get",
	}

	// Вызываем сервис
	result, err := h.profileService.Exec(profileParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка получения профиля: %w", err)
	}

	// Приводим результат к нужному типу
	profileResult, ok := result.(profile.ProfileResult)
	if !ok {
		return handlers.HandlerResult{}, fmt.Errorf("неверный тип результата от сервиса профиля")
	}

	// ⭐ Если сервис вернул ошибку, показываем сообщение пользователю
	if !profileResult.Success {
		return handlers.HandlerResult{
			Message:  fmt.Sprintf("❌ %s", profileResult.Message),
			Keyboard: h.createProfileKeyboard(),
		}, nil
	}

	// Извлекаем данные для форматирования
	message := h.formatProfileMessage(profileResult.Data)
	keyboard := h.createProfileKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// formatProfileMessage форматирует сообщение профиля из данных сервиса
func (h *profileMainHandler) formatProfileMessage(data interface{}) string {
	// Приводим данные к map
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return "Ошибка форматирования данных профиля"
	}

	// Извлекаем данные пользователя
	userData, ok := dataMap["user"].(map[string]interface{})
	if !ok {
		return "Ошибка получения данных пользователя"
	}

	// Извлекаем данные подписки
	subData, ok := dataMap["subscription"].(map[string]interface{})
	if !ok {
		subData = make(map[string]interface{})
	}

	// Получаем значения с безопасным приведением типов
	id := getInt64(userData, "id")
	telegramID := getInt64(userData, "telegram_id")
	username := getString(userData, "username")
	firstName := getString(userData, "first_name")
	role := getString(userData, "role")

	planName := getString(subData, "plan_name")
	if planName == "" {
		planName = "Free"
	}

	subscriptionActive := getBool(subData, "is_active")
	signalsToday := getInt(userData, "signals_today")
	growthMin := getFloat64(userData, "min_growth_threshold")
	fallMin := getFloat64(userData, "min_fall_threshold")

	// Получаем даты
	createdAt := getTime(userData, "created_at")
	lastLoginAt := getTime(userData, "last_login_at")
	expiresAt := getTime(subData, "expires_at")

	// Форматируем имя
	displayName := firstName
	if displayName == "" {
		displayName = "Гость"
	}

	// Форматируем username
	displayUsername := username
	if displayUsername == "" {
		displayUsername = "не указан"
	} else {
		displayUsername = "@" + displayUsername
	}

	// Форматируем дату последнего входа
	lastLoginDisplay := "еще не входил"
	if !lastLoginAt.IsZero() {
		lastLoginDisplay = lastLoginAt.Format("02.01.2006 15:04")
	}

	// Определяем статус подписки
	subscriptionStatus := "❌ Неактивна"
	if subscriptionActive {
		subscriptionStatus = "✅ Активна"
	}

	// Форматируем дату окончания подписки - показываем всегда, если есть
	expiresAtDisplay := "—"
	if !expiresAt.IsZero() {
		expiresAtDisplay = expiresAt.Format("02.01.2006 15:04")

		// Добавляем индикатор, если подписка истекла
		if !subscriptionActive && expiresAt.Before(time.Now()) {
			expiresAtDisplay = expiresAtDisplay + " (истекла)"
		}
	}

	// Добавляем эмодзи для тарифа
	planEmoji := "🆓"
	if planName != "Free" && planName != "free" {
		planEmoji = "💎"
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"🆔 ID: %d\n"+
			"📱 Telegram ID: %d\n"+
			"👤 Имя: %s\n"+
			"📧 Username: %s\n"+
			"⭐ Роль: %s\n\n"+
			"💰 *Подписка*\n"+
			"   %s %s\n"+
			"   Статус: %s\n"+
			"   Действует до: %s\n\n"+
			"📊 *Статистика*\n"+
			"   📈 Сигналов сегодня: %d\n"+
			"   🎯 Мин. рост: %.2f%%\n"+
			"   📉 Мин. падение: %.2f%%\n\n"+
			"📅 Регистрация: %s\n"+
			"🔐 Последний вход: %s",
		constants.AuthButtonTexts.Profile,
		id,
		telegramID,
		displayName,
		displayUsername,
		h.getRoleDisplay(role),
		planEmoji,
		planName,
		subscriptionStatus,
		expiresAtDisplay,
		signalsToday,
		growthMin,
		fallMin,
		createdAt.Format("02.01.2006"),
		lastLoginDisplay,
	)
}

// getRoleDisplay возвращает отображаемое имя роли
func (h *profileMainHandler) getRoleDisplay(role string) string {
	switch role {
	case "admin":
		return "👑 Администратор"
	case "moderator":
		return "🛡️ Модератор"
	case "premium":
		return "💎 Премиум"
	default:
		return "👤 Пользователь"
	}
}

// createProfileKeyboard создает клавиатуру для профиля
func (h *profileMainHandler) createProfileKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			// {
			// 	{"text": constants.AuthButtonTexts.Stats, "callback_data": constants.CallbackProfileStats},
			// 	{"text": constants.AuthButtonTexts.Premium, "callback_data": constants.CallbackProfileSubscription},
			// },
			{
				{"text": constants.ButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}

// Вспомогательные функции для безопасного извлечения данных из map

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok && val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok && val != nil {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok && val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getTime(m map[string]interface{}, key string) time.Time {
	if val, ok := m[key]; ok && val != nil {
		switch v := val.(type) {
		case time.Time:
			return v
		case string:
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
