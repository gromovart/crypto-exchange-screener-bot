// internal/delivery/telegram/app/bot/handlers/base/base.go
package base

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// BaseHandler базовая структура для всех хэндлеров
type BaseHandler struct {
	Name    string
	Command string
	Type    handlers.HandlerType
}

// GetName возвращает имя хэндлера
func (h *BaseHandler) GetName() string {
	return h.Name
}

// GetCommand возвращает команду/callback
func (h *BaseHandler) GetCommand() string {
	return h.Command
}

// GetType возвращает тип хэндлера
func (h *BaseHandler) GetType() handlers.HandlerType {
	return h.Type
}

// GetRoleDisplay возвращает отображаемое имя роли
func (h *BaseHandler) GetRoleDisplay(role string) string {
	switch role {
	case models.RoleAdmin:
		return "👑 Администратор"
	case models.RolePremium:
		return "🌟 Премиум"
	case models.RoleUser:
		return "👤 Пользователь"
	default:
		return role
	}
}

// GetBoolDisplay возвращает отображение булевого значения
func (h *BaseHandler) GetBoolDisplay(value bool) string {
	if value {
		return "✅ Включено"
	}
	return "❌ Выключено"
}

// GetToggleText возвращает текст для переключателя
func (h *BaseHandler) GetToggleText(baseText string, isEnabled bool) string {
	if isEnabled {
		return "✅ " + baseText
	}
	return "❌ " + baseText
}

// GetStatusDisplay возвращает отображение статуса
func (h *BaseHandler) GetStatusDisplay(isActive bool) string {
	if isActive {
		return "✅ Активен"
	}
	return "❌ Деактивирован"
}

// GetSubscriptionTierDisplayName возвращает отображаемое имя тарифа
func (h *BaseHandler) GetSubscriptionTierDisplayName(tier string) string {
	switch tier {
	case "enterprise":
		return "🏢 Доступ на 12 месяцев"
	case "pro":
		return "🚀 Доступ на 3 месяца"
	case "basic":
		return "📱 Доступ на 1 месяц"
	case "free":
		return "🆓 Free"
	default:
		return tier
	}
}
