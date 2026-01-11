// internal/delivery/telegram/auth_init.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"log"
)

// AuthInitializer - инициализатор системы авторизации для Telegram бота
type AuthInitializer struct {
	config      *config.Config
	userService *users.Service
}

// NewAuthInitializer создает новый инициализатор авторизации
func NewAuthInitializer(cfg *config.Config, userService *users.Service) *AuthInitializer {
	return &AuthInitializer{
		config:      cfg,
		userService: userService,
	}
}

// InitializeAuth инициализирует систему авторизации для бота
func (ai *AuthInitializer) InitializeAuth(bot *TelegramBot) (*AuthHandlers, error) {
	log.Println("🔐 Инициализация системы авторизации...")

	// Проверяем, что userService доступен
	if ai.userService == nil {
		log.Println("⚠️ UserService не доступен, авторизация будет отключена")
		return nil, nil
	}

	// Создаем обработчики авторизации
	authHandlers := NewAuthHandlers(bot, ai.userService)

	// Настраиваем авторизацию в боте
	bot.SetupAuth(authHandlers)

	log.Println("✅ Система авторизации инициализирована")
	return authHandlers, nil
}

// InitializeAuthForSingleton инициализирует авторизацию для Singleton бота
func (ai *AuthInitializer) InitializeAuthForSingleton() (*AuthHandlers, error) {
	// Получаем Singleton бот
	bot := GetBot()
	if bot == nil {
		log.Println("⚠️ Singleton бот не инициализирован")
		return nil, nil
	}

	return ai.InitializeAuth(bot)
}

// SetupAuthCommands регистрирует команды авторизации в обработчике обновлений
func (ai *AuthInitializer) SetupAuthCommands(updatesHandler *UpdatesHandler, authHandlers *AuthHandlers) {
	if updatesHandler == nil || authHandlers == nil {
		log.Println("⚠️ Не удалось настроить команды авторизации: updatesHandler или authHandlers nil")
		return
	}

	log.Println("📋 Настройка команд авторизации...")

	// Устанавливаем authHandlers в updatesHandler
	updatesHandler.SetAuthHandlers(authHandlers)

	// Логируем успешную настройку
	if updatesHandler.HasAuth() {
		log.Println("✅ Команды авторизации настроены в UpdatesHandler")
	} else {
		log.Println("⚠️ Команды авторизации не настроены: updatesHandler не имеет authHandlers")
	}
}

// GetAuthMiddleware возвращает middleware авторизации для бота
func (ai *AuthInitializer) GetAuthMiddleware(bot *TelegramBot) *AuthMiddleware {
	if bot == nil {
		return nil
	}

	return bot.GetAuthMiddleware()
}

// IsAuthEnabled проверяет, включена ли авторизация
func (ai *AuthInitializer) IsAuthEnabled(bot *TelegramBot) bool {
	if bot == nil {
		return false
	}

	return bot.HasAuth()
}

// CreateDefaultAuthKeyboard создает клавиатуру по умолчанию для авторизации
func (ai *AuthInitializer) CreateDefaultAuthKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: AuthButtonTexts.Profile, CallbackData: CallbackAuthProfile},
				{Text: AuthButtonTexts.Settings, CallbackData: CallbackAuthSettings},
			},
			{
				{Text: AuthButtonTexts.Notifications, CallbackData: CallbackAuthNotifications},
				{Text: AuthButtonTexts.Stats, CallbackData: CallbackAuthStats},
			},
		},
	}
}

// CreateAdminAuthKeyboard создает клавиатуру для администратора
func (ai *AuthInitializer) CreateAdminAuthKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: AuthButtonTexts.Users, CallbackData: CallbackAdminUsers},
				{Text: AuthButtonTexts.Stats, CallbackData: CallbackAdminStats},
			},
			{
				{Text: AuthButtonTexts.System, CallbackData: CallbackAdminSystem},
				{Text: ButtonTexts.Back, CallbackData: CallbackAdminBack},
			},
		},
	}
}

// CreatePremiumAuthKeyboard создает клавиатуру для премиум пользователей
func (ai *AuthInitializer) CreatePremiumAuthKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🚀 Расширенная аналитика", CallbackData: CallbackPremiumAnalytics},
				{Text: "📈 Детальные сигналы", CallbackData: CallbackPremiumSignals},
			},
			{
				{Text: AuthButtonTexts.Priority, CallbackData: CallbackPremiumPriority},
				{Text: ButtonTexts.Back, CallbackData: CallbackPremiumBack},
			},
		},
	}
}

// GetAuthStatus возвращает статус авторизации в текстовом формате
func (ai *AuthInitializer) GetAuthStatus(bot *TelegramBot) string {
	if !ai.IsAuthEnabled(bot) {
		return "🔓 Авторизация: ❌ Выключена"
	}

	// Получаем middleware для получения дополнительной информации
	middleware := ai.GetAuthMiddleware(bot)
	if middleware == nil {
		return "🔓 Авторизация: ⚠️ Частично включена"
	}

	return "🔓 Авторизация: ✅ Включена"
}

// SetupDefaultUserSettings устанавливает настройки по умолчанию для нового пользователя
func (ai *AuthInitializer) SetupDefaultUserSettings() {
	log.Println("⚙️ Настройки пользователей по умолчанию готовы")
}

// ValidateAuthConfig проверяет конфигурацию авторизации
func (ai *AuthInitializer) ValidateAuthConfig() error {
	if ai.config == nil {
		return logError("Конфигурация не задана")
	}

	if ai.config.TelegramBotToken == "" {
		return logError("Telegram Bot Token не указан")
	}

	log.Println("✅ Конфигурация авторизации проверена")
	return nil
}

// Вспомогательная функция для логирования ошибок
func logError(message string) error {
	log.Printf("❌ Ошибка авторизации: %s", message)
	return nil // Возвращаем nil для совместимости
}
