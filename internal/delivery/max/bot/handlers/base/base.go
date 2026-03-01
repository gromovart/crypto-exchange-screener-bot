// internal/delivery/max/bot/handlers/base/base.go
package base

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// BaseHandler — базовая структура для всех MAX хэндлеров
type BaseHandler struct {
	Name    string
	Command string
	Type    handlers.HandlerType
}

// New создаёт новый базовый хэндлер
func New(name, command string, handlerType handlers.HandlerType) *BaseHandler {
	return &BaseHandler{Name: name, Command: command, Type: handlerType}
}

func (h *BaseHandler) GetName() string               { return h.Name }
func (h *BaseHandler) GetCommand() string            { return h.Command }
func (h *BaseHandler) GetType() handlers.HandlerType { return h.Type }

// GetBoolDisplay возвращает текстовое представление bool
func (h *BaseHandler) GetBoolDisplay(v bool) string {
	if v {
		return "✅ Включено"
	}
	return "❌ Выключено"
}

// GetToggleText возвращает кнопку с эмодзи-индикатором
func (h *BaseHandler) GetToggleText(base string, enabled bool) string {
	if enabled {
		return "✅ " + base
	}
	return "❌ " + base
}

// GetRoleDisplay возвращает отображаемое имя роли
func (h *BaseHandler) GetRoleDisplay(role string) string {
	switch role {
	case models.RoleAdmin:
		return "👑 Администратор"
	case models.RolePremium:
		return "🌟 Премиум"
	default:
		return "👤 Пользователь"
	}
}

// GetStatusDisplay возвращает текст статуса
func (h *BaseHandler) GetStatusDisplay(active bool) string {
	if active {
		return "✅ Активен"
	}
	return "❌ Неактивен"
}
