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

	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	displayName := user.FirstName
	if displayName == "" {
		displayName = "Гость"
	}

	displayUsername := "не указан"
	if user.Username != "" {
		displayUsername = "@" + user.Username
	}

	planName := user.SubscriptionTier
	if planName == "" {
		planName = "Free"
	}
	planEmoji := "🆓"
	if planName != "free" && planName != "Free" {
		planEmoji = "💎"
	}

	roleDisplay := getRoleDisplay(user.Role)

	lastLoginDisplay := "ещё не входил"
	if !user.LastLoginAt.IsZero() {
		lastLoginDisplay = user.LastLoginAt.Format("02.01.2006 15:04")
	}

	growthMin := user.MinGrowthThreshold
	if growthMin == 0 {
		growthMin = 2.0
	}
	fallMin := user.MinFallThreshold
	if fallMin == 0 {
		fallMin = 2.0
	}

	// MAX User ID — нужен для входа в Crypto Analyzer
	maxIDDisplay := "не привязан"
	if user.MaxUserID != nil {
		maxIDDisplay = fmt.Sprintf("%d", *user.MaxUserID)
	}

	msg := fmt.Sprintf(
		"🔑 Профиль\n\n"+
			"🆔 MAX ID: %s\n"+
			"👤 Имя: %s\n"+
			"📧 Username: %s\n"+
			"⭐ Роль: %s\n"+
			"💰 Подписка: %s %s\n\n"+
			"📊 Статистика:\n"+
			"· Сигналов сегодня: %d\n"+
			"· Мин. рост: %.2f%%\n"+
			"· Мин. падение: %.2f%%\n\n"+
			"📅 Регистрация: %s\n"+
			"🔐 Последний вход: %s\n\n"+
			"💡 MAX ID используется для входа в Crypto Analyzer",
		maxIDDisplay,
		displayName,
		displayUsername,
		roleDisplay,
		planEmoji, planName,
		user.SignalsToday,
		growthMin,
		fallMin,
		user.CreatedAt.Format("02.01.2006"),
		lastLoginDisplay,
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

func getRoleDisplay(role string) string {
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
