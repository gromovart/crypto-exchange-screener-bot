// internal/delivery/telegram/auth_handlers_wrapper.go
package telegram

import (
	"fmt"
	"log"
	"strings"
)

// isAuthCallback проверяет, относится ли callback к авторизации
func (mh *MenuHandlers) isAuthCallback(callbackData string) bool {
	if mh.authHandlers == nil {
		return false
	}

	// Проверяем префиксы callback'ов авторизации
	authPrefixes := []string{
		"auth_",
		"settings_",
		"admin_",
		"premium_",
		"advanced_",
	}

	for _, prefix := range authPrefixes {
		if strings.HasPrefix(callbackData, prefix) {
			return true
		}
	}

	return false
}

// handleAuthCallback обрабатывает callback'ы авторизации
func (mh *MenuHandlers) handleAuthCallback(callbackData string, chatID string) error {
	if mh.authHandlers == nil {
		return fmt.Errorf("auth handlers not initialized")
	}

	// TODO: Реализовать полную обработку через AuthMiddleware
	// Пока просто логируем и возвращаем заглушку
	log.Printf("🔐 Auth callback detected: %s for chat %s", callbackData, chatID)

	// Временное сообщение
	message := "🔐 *Функционал авторизации*\n\n" +
		"Система авторизации в процессе интеграции.\n" +
		"Доступные команды:\n" +
		"/profile - Ваш профиль\n" +
		"/settings - Настройки\n" +
		"/notifications - Управление уведомлениями\n\n" +
		"Callback получен: " + callbackData

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// isAuthCommand проверяет, является ли команда командой авторизации
func (mh *MenuHandlers) isAuthCommand(text string) bool {
	authCommands := []string{
		"/profile",
		"/settings",
		"/notifications",
		"/thresholds",
		"/periods",
		"/language",
		"/premium",
		"/advanced",
		"/admin",
		"/stats",
		"/users",
		"/login",
		"/logout",
	}

	for _, cmd := range authCommands {
		if strings.HasPrefix(text, cmd) {
			return true
		}
	}

	return false
}

// handleAuthMessage обрабатывает сообщения авторизации
func (mh *MenuHandlers) handleAuthMessage(text, chatID string) error {
	if mh.authHandlers == nil {
		return fmt.Errorf("auth handlers not initialized")
	}

	log.Printf("🔐 Auth command detected: %s for chat %s", text, chatID)

	// Временное сообщение
	message := "🔐 *Команда авторизации*\n\n" +
		"Система авторизации в процессе интеграции.\n" +
		"Скоро будут доступны:\n" +
		"• Персональный профиль\n" +
		"• Настройки уведомлений\n" +
		"• История сигналов\n" +
		"• Премиум функции\n\n" +
		"Команда: " + text

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}
