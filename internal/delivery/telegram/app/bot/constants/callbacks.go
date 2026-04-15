// internal/delivery/telegram/app/bot/constants/callbacks.go
package constants

// Callback constants
const (
	// ============== MAIN MENU ==============
	CallbackStats             = "stats"              // 📊 Статус
	CallbackSettingsMain      = "settings_main"      // ⚙️ Настройки
	CallbackNotificationsMenu = "notifications_menu" // 🔔 Уведомления
	CallbackSignalsMenu       = "signals_menu"       // 📈 Сигналы
	CallbackPeriodsMenu       = "periods_menu"       // ⏱️ Периоды
	CallbackResetMenu         = "reset_menu"         // 🔄 Сбросить
	CallbackHelp              = "help"               // 📋 Помощь

	// ============== SETTINGS MENU ==============
	CallbackAuthLogin = "auth_login" // 🔑 Войти / Авторизация
	// Для АВТОРИЗОВАННЫХ
	CallbackProfileMain    = "profile_main"    // 👤 Мой профиль
	CallbackThresholdsMenu = "thresholds_menu" // 📊 Пороги сигналов
	CallbackPeriodManage   = "period_manage"   // ⏱️ Периоды (расширенный)
	CallbackResetSettings  = "reset_settings"  // ⚙️ Сбросить настройки

	// Навигация
	CallbackMenuBack = "menu_back" // 🔙 Назад
	CallbackMenuMain = "menu_main" // 🔙 Главное меню

	// ============== NOTIFICATIONS MENU ==============
	// (одинаковое для всех)
	CallbackNotifyToggleAll  = "notify_toggle_all"  // ✅/❌ Включить/Выключить
	CallbackNotifyGrowthOnly = "notify_growth_only" // 📈 Только рост
	CallbackNotifyFallOnly   = "notify_fall_only"   // 📉 Только падение
	CallbackNotifyBoth       = "notify_both"        // 📊 Все сигналы

	// ============== THRESHOLDS MENU ==============
	CallbackThresholdGrowth = "threshold_growth" // 📈 Мин. рост: X%
	CallbackThresholdFall   = "threshold_fall"   // 📉 Мин. падение: X%

	// ============== PERIODS MENU ==============
	// Базовый (для всех)
	CallbackPeriod1m  = "period_1m"  // ⏱️ 1 минута
	CallbackPeriod5m  = "period_5m"  // ⏱️ 5 минут
	CallbackPeriod15m = "period_15m" // ⏱️ 15 минут
	CallbackPeriod30m = "period_30m" // ⏱️ 30 минут
	CallbackPeriod1h  = "period_1h"  // ⏱️ 1 час
	CallbackPeriod4h  = "period_4h"  // ⏱️ 4 часа

	// Расширенный - управление предпочтительными
	CallbackPeriod1d           = "period_1d"            // ⏱️ 1 день
	CallbackPeriodManageAdd    = "period_manage_add"    // ➕ Добавить период
	CallbackPeriodManageRemove = "period_manage_remove" // ➖ Удалить период
	CallbackPeriodManageReset  = "period_manage_reset"  // 🔄 Сбросить выбор

	// ============== PROFILE MENU ==============
	CallbackProfileStats        = "profile_stats"        // 📊 Статистика
	CallbackProfileSubscription = "profile_subscription" // 💎 Подписка: X

	// ============== AUTH CALLBACKS ==============
	CallbackAuthLogout   = "auth_logout"   // 👋 Выйти
	CallbackAuthForgot   = "auth_forgot"   // 🔓 Забыли пароль?
	CallbackAuthRegister = "auth_register" // 📝 Регистрация

	// ============== RESET MENU ==============
	CallbackResetAll      = "reset_all"       // 🗑️ Сбросить все
	CallbackResetCounters = "reset_counters"  // 📊 Сбросить счетчики
	CallbackResetBySymbol = "reset_by_symbol" // 🔤 Сбросить по символу

	// ============== SYMBOL SELECTION ==============
	CallbackSymbolBTCUSDT = "symbol_btcusdt" // BTC/USDT
	CallbackSymbolETHUSDT = "symbol_ethusdt" // ETH/USDT
	CallbackSymbolBNBUSDT = "symbol_bnbusdt" // BNB/USDT
	CallbackSymbolSOLUSDT = "symbol_solusdt" // SOL/USDT
	CallbackSymbolXRPUSDT = "symbol_xrpusdt" // XRP/USDT
	CallbackSymbolBack    = "symbol_back"    // 🔙 Назад к сбросу

	// ============== SIGNALS MENU ==============
	CallbackSignalToggleGrowth       = "signal_toggle_growth"        // 📈 Вкл/Выкл рост
	CallbackSignalToggleFall         = "signal_toggle_fall"          // 📉 Вкл/Выкл падение
	CallbackSignalSetGrowthThreshold = "signal_set_growth_threshold" // 📈 Установить порог роста
	CallbackSignalSetFallThreshold   = "signal_set_fall_threshold"   // 📉 Установить порог падения
	CallbackSignalSetSensitivity     = "signal_set_sensitivity"      // 🎯 Настроить чувствительность
	CallbackSignalHistory            = "signal_history"              // 📊 История сигналов
	CallbackSignalTest               = "signal_test"                 // ⚡ Тестовый сигнал

	// ============== SESSION ==============
	CallbackSessionStart    = "session_start"     // 🟢 Начать торговую сессию
	CallbackSessionStop     = "session_stop"      // 🔴 Завершить сессию
	CallbackSessionDuration = "session_duration:" // session_duration:{2h|4h|8h|day}

	// ============== WATCHLIST ==============
	CallbackWatchlistMenu   = "watchlist_menu"    // 📋 Вотчлист
	CallbackWatchlistSearch = "watchlist_search"  // 🔍 Поиск монеты
	CallbackWatchlistReset  = "watchlist_reset"    // 🗑️ Очистить вотчлист
	CallbackWatchlistAddAll = "watchlist_add_all"  // ➕ Добавить все монеты
	// Wildcard: watchlist_toggle:{SYMBOL}
	CallbackWatchlistTogglePrefix = "watchlist_toggle:"
	// Wildcard: watchlist_letter:{LETTER}:{PAGE}
	CallbackWatchlistLetterPrefix = "watchlist_letter:"
	// Wildcard: watchlist_page:{PAGE}
	CallbackWatchlistPagePrefix = "watchlist_page:"

	// ============== TEST & DEBUG ==============
	CallbackTest           = "test"             // 🧪 Тестовое сообщение
	CallbackTestOK         = "test_ok"          // ✅ Тест OK
	CallbackTestCancel     = "test_cancel"      // ❌ Тест отмена
	CallbackToggleTestMode = "toggle_test_mode" // 🧪 Переключить тестовый режим
	CallbackChart          = "chart"            // 📈 Графики
)
