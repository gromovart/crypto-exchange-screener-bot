// internal/delivery/telegram/auth_handlers.go
package telegram

import (
	"fmt"
	"log"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// AuthHandlers обработчики авторизации для Telegram бота
type AuthHandlers struct {
	bot            *TelegramBot
	userService    *users.Service
	authMiddleware *AuthMiddleware
}

// NewAuthHandlers создает новые обработчики авторизации
func NewAuthHandlers(bot *TelegramBot, userService *users.Service) *AuthHandlers {
	authMiddleware := NewAuthMiddleware(userService, bot.config.TelegramBotToken)

	return &AuthHandlers{
		bot:            bot,
		userService:    userService,
		authMiddleware: authMiddleware,
	}
}

// RegisterHandlers регистрирует обработчики команд авторизации
func (h *AuthHandlers) RegisterHandlers() {
	log.Println("🔐 Регистрация обработчиков авторизации...")

	// Интеграция будет происходить через существующие обработчики Telegram бота
	// Команды будут обрабатываться через menuManager
	log.Println("✅ Обработчики авторизации готовы к интеграции с Telegram ботом")
}

// GetAuthMiddleware возвращает middleware аутентификации
func (h *AuthHandlers) GetAuthMiddleware() *AuthMiddleware {
	return h.authMiddleware
}

// handleStart обработчик команды /start
func (h *AuthHandlers) handleStart(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	// Приветственное сообщение
	message := fmt.Sprintf(
		"👋 *Добро пожаловать, %s!*\n\n"+
			"🚀 *Crypto Exchange Screener Bot*\n\n"+
			"✅ Ваш аккаунт: @%s\n"+
			"👤 Имя: %s\n"+
			"⭐ Роль: %s\n"+
			"📅 Дата регистрации: %s\n\n"+
			"Бот анализирует рынок криптовалют и отправляет уведомления о сильных движениях.\n\n"+
			"*Основные команды:*\n"+
			"/profile - Ваш профиль\n"+
			"/settings - Настройки\n"+
			"/notifications - Управление уведомлениями\n"+
			"/help - Справка\n\n"+
			"Используйте меню ниже для управления ботом:",
		user.FirstName,
		user.Username,
		user.FirstName,
		getRoleDisplayName(user.Role),
		user.CreatedAt.Format("02.01.2006"),
	)

	// Создаем приветственную клавиатуру
	keyboard := h.authMiddleware.CreateAuthInlineKeyboard()

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleProfile обработчик команды /profile
func (h *AuthHandlers) handleProfile(user *models.User, update *TelegramUpdate) error {

	log.Printf("🔍 DEBUG: FirstName: %q (contains *: %v, contains _: %v)",
		user.FirstName,
		strings.Contains(user.FirstName, "*"),
		strings.Contains(user.FirstName, "_"))
	log.Printf("🔍 DEBUG: Username: %q", user.Username)
	log.Printf("🔍 DEBUG: CreatedAt: %s", user.CreatedAt.Format("02.01.2006"))
	log.Printf("🔍 DEBUG: LastLoginAt: %s", user.LastLoginAt.Format("02.01.2006 15:04"))
	chatID := h.authMiddleware.getChatID(update)

	// Получаем статистику пользователя
	stats := h.getUserStats(user)

	// Форматируем сообщение профиля
	message := fmt.Sprintf(
		"👤 *Ваш профиль*\n\n"+
			"🆔 ID: %d\n"+
			"📱 Telegram ID: %d\n"+
			"👤 Имя: %s\n"+ // user.FirstName может содержать *
			"📧 Username: @%s\n"+
			"⭐ Роль: %s\n"+
			"💰 Тариф: %s\n"+
			"✅ Статус: %s\n"+
			"📅 Регистрация: %s\n"+
			"🔐 Последний вход: %s\n\n",
		user.ID,
		user.TelegramID,
		"Test User", // ВРЕМЕННО: безопасное имя
		user.Username,
		getRoleDisplayName(user.Role),
		getSubscriptionTierDisplayName(user.SubscriptionTier),
		getStatusDisplay(user.IsActive),
		user.CreatedAt.Format("02.01.2006"),
		user.LastLoginAt.Format("02.01.2006 15:04"),
	)

	// Добавляем статистику если есть
	message += fmt.Sprintf(
		"📊 *Статистика*\n"+ // Теперь есть закрывающий *
			"📈 Сигналов сегодня: %d/%d\n"+
			"🎯 Минимальный рост: %.2f%%\n"+
			"📉 Минимальное падение: %.2f%%\n\n",
		stats.SignalsToday,
		stats.MaxSignalsPerDay,
		stats.MinGrowthThreshold,
		stats.MinFallThreshold,
	)

	// Клавиатура для управления профилем
	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "⚙️ Настройки", CallbackData: "auth_settings"},
				{Text: "🔔 Уведомления", CallbackData: "auth_notifications"},
			},
			{
				{Text: "📊 Статистика", CallbackData: "auth_stats"},
				{Text: "📈 Пороги", CallbackData: "auth_thresholds"},
			},
		},
	}
	log.Printf("🔍 DEBUG: Profile message length: %d bytes", len(message))
	log.Printf("🔍 DEBUG: First 400 chars: %s", message[:min(400, len(message))])
	log.Printf("🔍 DEBUG: Chars 300-350: %s", message[300:min(350, len(message))])
	log.Printf("🔍 DEBUG: Chars 320-340: %q", message[320:min(340, len(message))])

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleLogin обработчик команды /login
func (h *AuthHandlers) handleLogin(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	// Пользователь уже авторизован через Telegram
	message := fmt.Sprintf(
		"✅ *Вы уже авторизованы!*\n\n"+
			"👤 Имя: %s\n"+
			"📧 Username: @%s\n"+
			"⭐ Роль: %s\n"+
			"📅 В системе с: %s\n\n"+
			"Последний вход: %s",
		user.FirstName,
		user.Username,
		getRoleDisplayName(user.Role),
		user.CreatedAt.Format("02.01.2006"),
		user.LastLoginAt.Format("02.01.2006 15:04"),
	)

	// Обновляем время последнего входа
	user.LastLoginAt = time.Now()
	if err := h.userService.UpdateUser(user); err != nil {
		log.Printf("⚠️ Не удалось обновить время входа: %v", err)
	}

	return h.authMiddleware.sendMessage(chatID, message, nil)
}

// handleLogout обработчик команды /logout
func (h *AuthHandlers) handleLogout(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	// В Telegram боте выход не требует удаления сессии
	// Просто отправляем сообщение
	message := fmt.Sprintf("👋 *До свидания, %s!*\n\nВы можете вернуться в любое время, отправив /start", user.FirstName)

	return h.authMiddleware.sendMessage(chatID, message, nil)
}

// handleSettings обработчик команды /settings
func (h *AuthHandlers) handleSettings(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	message := fmt.Sprintf(
		"⚙️ *Настройки профиля*\n\n"+
			"Текущие настройки:\n\n"+
			"🔊 *Уведомления:* %s\n"+
			"📈 *Отслеживать рост:* %s\n"+
			"📉 *Отслеживать падение:* %s\n"+
			"🎯 *Порог роста:* %.2f%%\n"+
			"📉 *Порог падения:* %.2f%%\n"+
			"⏰ *Тихие часы:* %02d:00 - %02d:00\n"+
			"🌐 *Язык:* %s\n"+
			"🕐 *Часовой пояс:* %s\n"+
			"👁️ *Режим отображения:* %s\n\n"+
			"Выберите настройку для изменения:",
		getBoolDisplay(user.NotificationsEnabled),
		getBoolDisplay(user.NotifyGrowth),
		getBoolDisplay(user.NotifyFall),
		user.MinGrowthThreshold,
		user.MinFallThreshold,
		user.QuietHoursStart,
		user.QuietHoursEnd,
		user.Language,
		user.Timezone,
		user.DisplayMode,
	)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔔 Уведомления", CallbackData: "settings_notifications"},
				{Text: "📈 Пороги", CallbackData: "settings_thresholds"},
			},
			{
				{Text: "⏰ Тихие часы", CallbackData: "settings_quiet_hours"},
				{Text: "🌐 Язык", CallbackData: "settings_language"},
			},
			{
				{Text: "🕐 Часовой пояс", CallbackData: "settings_timezone"},
				{Text: "👁️ Отображение", CallbackData: "settings_display"},
			},
			{
				{Text: "🔄 Сбросить настройки", CallbackData: "settings_reset"},
				{Text: "🔙 Назад", CallbackData: "settings_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleNotifications обработчик команды /notifications
func (h *AuthHandlers) handleNotifications(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	message := fmt.Sprintf(
		"🔔 *Управление уведомлениями*\n\n"+
			"Текущие настройки:\n\n"+
			"🔊 Общие уведомления: %s\n"+
			"📈 Уведомления о росте: %s\n"+
			"📉 Уведомления о падении: %s\n"+
			"🔄 Непрерывные сигналы: %s\n"+
			"⏰ Тихие часы: %02d:00 - %02d:00\n\n"+
			"Выберите настройку для изменения:",
		getBoolDisplay(user.NotificationsEnabled),
		getBoolDisplay(user.NotifyGrowth),
		getBoolDisplay(user.NotifyFall),
		getBoolDisplay(user.NotifyContinuous),
		user.QuietHoursStart,
		user.QuietHoursEnd,
	)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: getToggleText("🔔 Общие", user.NotificationsEnabled),
					CallbackData: "settings_toggle_notifications"},
			},
			{
				{Text: getToggleText("📈 Рост", user.NotifyGrowth),
					CallbackData: "settings_toggle_growth"},
				{Text: getToggleText("📉 Падение", user.NotifyFall),
					CallbackData: "settings_toggle_fall"},
			},
			{
				{Text: getToggleText("🔄 Непрерывные", user.NotifyContinuous),
					CallbackData: "settings_toggle_continuous"},
				{Text: "⏰ Тихие часы", CallbackData: "settings_set_quiet_hours"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "settings_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleThresholds обработчик команды /thresholds
func (h *AuthHandlers) handleThresholds(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	message := fmt.Sprintf(
		"🎯 *Настройка порогов*\n\n"+
			"Текущие пороги:\n\n"+
			"📈 Минимальный рост: %.2f%%\n"+
			"📉 Минимальное падение: %.2f%%\n\n"+
			"Порог определяет, насколько сильным должно быть движение,\n"+
			"чтобы бот отправил вам уведомление.\n\n"+
			"Рекомендуемые значения: 2.0%% - 5.0%%\n\n"+
			"Выберите порог для изменения:",
		user.MinGrowthThreshold,
		user.MinFallThreshold,
	)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("📈 Рост: %.2f%%", user.MinGrowthThreshold),
					CallbackData: "settings_set_growth_threshold"},
			},
			{
				{Text: fmt.Sprintf("📉 Падение: %.2f%%", user.MinFallThreshold),
					CallbackData: "settings_set_fall_threshold"},
			},
			{
				{Text: "2.0% (по умолчанию)", CallbackData: "settings_threshold_2"},
				{Text: "3.0% (средний)", CallbackData: "settings_threshold_3"},
			},
			{
				{Text: "5.0% (строгий)", CallbackData: "settings_threshold_5"},
				{Text: "🔙 Назад", CallbackData: "settings_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handlePeriods обработчик команды /periods
func (h *AuthHandlers) handlePeriods(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	// Преобразуем периоды в строку
	periodsStr := "Не настроены"
	if len(user.PreferredPeriods) > 0 {
		var periods []string
		for _, p := range user.PreferredPeriods {
			periods = append(periods, fmt.Sprintf("%dм", p))
		}
		periodsStr = strings.Join(periods, ", ")
	}

	message := fmt.Sprintf(
		"⏱️ *Настройка периодов анализа*\n\n"+
			"Текущие периоды: %s\n\n"+
			"Периоды определяют, за какие временные интервалы\n"+
			"бот анализирует движение цены.\n\n"+
			"Выберите периоды для отслеживания:",
		periodsStr,
	)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "1м", CallbackData: "settings_period_1"},
				{Text: "5м", CallbackData: "settings_period_5"},
				{Text: "15м", CallbackData: "settings_period_15"},
			},
			{
				{Text: "30м", CallbackData: "settings_period_30"},
				{Text: "1ч", CallbackData: "settings_period_60"},
				{Text: "4ч", CallbackData: "settings_period_240"},
			},
			{
				{Text: "1д", CallbackData: "settings_period_1440"},
				{Text: "🔙 Назад", CallbackData: "settings_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleLanguage обработчик команды /language
func (h *AuthHandlers) handleLanguage(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	message := fmt.Sprintf(
		"🌐 *Выбор языка*\n\n"+
			"Текущий язык: %s\n\n"+
			"Выберите язык интерфейса:",
		getLanguageDisplayName(user.Language),
	)

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🇷🇺 Русский", CallbackData: "settings_language_ru"},
				{Text: "🇺🇸 English", CallbackData: "settings_language_en"},
			},
			{
				{Text: "🇪🇸 Español", CallbackData: "settings_language_es"},
				{Text: "🇨🇳 中文", CallbackData: "settings_language_zh"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "settings_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleHelp обработчик команды /help
func (h *AuthHandlers) handleHelp(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	message := "📋 *Справка по командам*\n\n" +
		"*Основные команды:*\n" +
		"/start - Начало работы\n" +
		"/profile - Ваш профиль\n" +
		"/settings - Настройки профиля\n" +
		"/help - Эта справка\n\n" +
		"*Управление уведомлениями:*\n" +
		"/notifications - Настройки уведомлений\n" +
		"/thresholds - Настройка порогов\n" +
		"/periods - Настройка периодов\n" +
		"/language - Выбор языка\n\n" +
		"*Для премиум пользователей:*\n" +
		"/premium - Премиум функции\n" +
		"/advanced - Расширенная аналитика\n\n" +
		"*Для администраторов:*\n" +
		"/admin - Панель администратора\n" +
		"/stats - Статистика системы\n" +
		"/users - Управление пользователями\n\n" +
		"*Как работает бот:*\n" +
		"1️⃣ Анализирует рынок в реальном времени\n" +
		"2️⃣ Обнаруживает сильные движения цен\n" +
		"3️⃣ Отправляет уведомления при превышении порогов\n" +
		"4️⃣ Считает сигналы по периодам\n\n" +
		"*Настройки по умолчанию:*\n" +
		"📈 Рост: 2.0%\n" +
		"📉 Падение: 2.0%\n" +
		"⏱️ Периоды: 5м, 15м, 30м\n" +
		"🔔 Уведомления: включены\n\n" +
		"Используйте команды выше или меню для настройки."

	return h.authMiddleware.sendMessage(chatID, message, nil)
}

// handlePremium обработчик команды /premium
func (h *AuthHandlers) handlePremium(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	if !h.authMiddleware.isPremiumUser(user) {
		message := "🌟 *Премиум функции*\n\n" +
			"Эта функция доступна только премиум пользователям.\n\n" +
			"*Преимущества премиум аккаунта:*\n" +
			"✅ Расширенные сигналы\n" +
			"✅ Детальная аналитика\n" +
			"✅ Приоритетная очередь\n" +
			"✅ Кастомные уведомления\n" +
			"✅ Увеличенные лимиты\n\n" +
			"Для получения премиум статуса обратитесь к администратору."

		return h.authMiddleware.sendMessage(chatID, message, nil)
	}

	message := "🚀 *Премиум функции*\n\n" +
		"*Доступные функции:*\n" +
		"✅ Расширенные сигналы - больше типов сигналов\n" +
		"✅ Детальная аналитика - графики и статистика\n" +
		"✅ Приоритетная очередь - быстрые уведомления\n" +
		"✅ Кастомные уведомления - гибкая настройка\n" +
		"✅ Увеличенные лимиты - больше сигналов в день\n\n" +
		"Используйте команду /advanced для доступа к расширенной аналитике."

	keyboard := h.authMiddleware.CreatePremiumInlineKeyboard()

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleAdvanced обработчик команды /advanced
func (h *AuthHandlers) handleAdvanced(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	if !h.authMiddleware.isPremiumUser(user) {
		return h.handlePremium(user, update)
	}

	message := "📊 *Расширенная аналитика*\n\n" +
		"*Доступные инструменты:*\n" +
		"📈 Детальные графики - цена, объем, индикаторы\n" +
		"📊 Статистика по символам - исторические данные\n" +
		"🔍 Углубленный анализ - паттерны и тренды\n" +
		"📉 Риск-менеджмент - оценка рисков\n" +
		"📋 Отчеты - ежедневные/еженедельные отчеты\n\n" +
		"*Будущие обновления:*\n" +
		"🤖 AI анализ - прогнозы на основе ИИ\n" +
		"📱 Мобильное приложение - уведомления на телефон\n" +
		"🌐 Web интерфейс - управление через браузер\n\n" +
		"Выберите инструмент для работы:"

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📈 Графики", CallbackData: "advanced_charts"},
				{Text: "📊 Статистика", CallbackData: "advanced_stats"},
			},
			{
				{Text: "🔍 Анализ", CallbackData: "advanced_analysis"},
				{Text: "📉 Риски", CallbackData: "advanced_risks"},
			},
			{
				{Text: "📋 Отчеты", CallbackData: "advanced_reports"},
				{Text: "🔙 Назад", CallbackData: "advanced_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleAdmin обработчик команды /admin
func (h *AuthHandlers) handleAdmin(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	if !user.IsAdmin() {
		return h.authMiddleware.sendAuthError(chatID, "Эта команда доступна только администраторам")
	}

	message := "👑 *Панель администратора*\n\n" +
		"*Доступные функции:*\n" +
		"👥 Управление пользователями - просмотр, редактирование\n" +
		"📊 Статистика системы - метрики и аналитика\n" +
		"⚙️ Настройки системы - конфигурация бота\n" +
		"🔄 Логи - просмотр журналов событий\n" +
		"🔧 Технические инструменты - обслуживание\n\n" +
		"Выберите раздел для управления:"

	keyboard := h.authMiddleware.CreateAdminInlineKeyboard()

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// handleStats обработчик команды /stats (исправленная версия)
func (h *AuthHandlers) handleStats(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	if !user.IsAdmin() {
		return h.authMiddleware.sendAuthError(chatID, "Эта команда доступна только администраторам")
	}

	// Упрощенная статистика без вызова методов с контекстом
	totalUsers := 0
	activeUsers := 0

	// Логируем попытку получения статистики
	log.Printf("📊 Получение статистики системы для администратора %d", user.ID)

	message := fmt.Sprintf(
		"📊 *Статистика системы*\n\n"+
			"*Пользователи:*\n"+
			"👥 Всего пользователей: %d\n"+
			"✅ Активных пользователей: %d\n"+
			"🌟 Премиум пользователей: %s\n"+
			"👑 Администраторов: %s\n\n"+
			"*Производительность:*\n"+
			"⚡ Активных сессий: %s\n"+
			"📈 Сигналов сегодня: %s\n"+
			"⏱️ Среднее время ответа: %s\n\n"+
			"*Система:*\n"+
			"🖥️ Версия бота: 1.0.0\n"+
			"📅 Время работы: %s\n"+
			"💾 Использование памяти: %s\n",
		totalUsers,
		activeUsers,
		"N/A", // Премиум пользователей
		"N/A", // Администраторов
		"N/A", // Активных сессий
		"N/A", // Сигналов сегодня
		"N/A", // Среднее время ответа
		time.Since(h.bot.startupTime).Round(time.Second).String(),
		"N/A", // Использование памяти
	)

	return h.authMiddleware.sendMessage(chatID, message, nil)
}

// handleUsers обработчик команды /users
func (h *AuthHandlers) handleUsers(user *models.User, update *TelegramUpdate) error {
	chatID := h.authMiddleware.getChatID(update)

	if !user.IsAdmin() {
		return h.authMiddleware.sendAuthError(chatID, "Эта команда доступна только администраторам")
	}

	message := "👥 *Управление пользователями*\n\n" +
		"*Доступные действия:*\n" +
		"🔍 Поиск пользователей - по ID, username\n" +
		"📋 Список пользователей - с пагинацией\n" +
		"👑 Изменение ролей - назначение прав\n" +
		"✅ Активация/деактивация - управление доступом\n" +
		"📊 Статистика пользователя - детальная информация\n\n" +
		"Выберите действие:"

	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔍 Поиск", CallbackData: "admin_users_search"},
				{Text: "📋 Список", CallbackData: "admin_users_list"},
			},
			{
				{Text: "👑 Роли", CallbackData: "admin_users_roles"},
				{Text: "✅ Статус", CallbackData: "admin_users_status"},
			},
			{
				{Text: "📊 Статистика", CallbackData: "admin_users_stats"},
				{Text: "🔙 Назад", CallbackData: "admin_back"},
			},
		},
	}

	return h.authMiddleware.sendMessage(chatID, message, keyboard)
}

// ... (остальные методы остаются без изменений, но убраны вызовы updatesRouter и callbackRouter)

// Вспомогательные функции

// getUserStats возвращает статистику пользователя
func (h *AuthHandlers) getUserStats(user *models.User) *UserStats {
	return &UserStats{
		SignalsToday:       user.SignalsToday,
		MaxSignalsPerDay:   user.MaxSignalsPerDay,
		MinGrowthThreshold: user.MinGrowthThreshold,
		MinFallThreshold:   user.MinFallThreshold,
	}
}

// UserStats структура статистики
type UserStats struct {
	SignalsToday       int
	MaxSignalsPerDay   int
	MinGrowthThreshold float64
	MinFallThreshold   float64
}

// getRoleDisplayName возвращает отображаемое имя роли
func getRoleDisplayName(role string) string {
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

// getSubscriptionTierDisplayName возвращает отображаемое имя тарифа
func getSubscriptionTierDisplayName(tier string) string {
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

// getStatusDisplay возвращает отображение статуса
func getStatusDisplay(isActive bool) string {
	if isActive {
		return "✅ Активен"
	}
	return "❌ Деактивирован"
}

// getBoolDisplay возвращает отображение булевого значения
func getBoolDisplay(value bool) string {
	if value {
		return "✅ Включено"
	}
	return "❌ Выключено"
}

// getToggleText возвращает текст для переключателя
func getToggleText(baseText string, isEnabled bool) string {
	if isEnabled {
		return "✅ " + baseText
	}
	return "❌ " + baseText
}

// getLanguageDisplayName возвращает отображаемое имя языка
func getLanguageDisplayName(language string) string {
	switch language {
	case "ru":
		return "🇷🇺 Русский"
	case "en":
		return "🇺🇸 English"
	case "es":
		return "🇪🇸 Español"
	case "zh":
		return "🇨🇳 中文"
	default:
		return language
	}
}

// getDisplayModeName возвращает имя режима отображения
func getDisplayModeName(mode string) string {
	switch mode {
	case "compact":
		return "Компактный"
	case "detailed":
		return "Детальный"
	case "pro":
		return "Профессиональный"
	default:
		return mode
	}
}
