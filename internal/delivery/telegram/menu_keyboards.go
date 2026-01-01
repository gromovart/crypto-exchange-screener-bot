// internal/delivery/telegram/menu_keyboards.go
package telegram

// MenuKeyboards - клавиатуры меню (2 ряда для устранения скролла)
type MenuKeyboards struct{}

// NewMenuKeyboards создает новые клавиатуры меню
func NewMenuKeyboards() *MenuKeyboards {
	return &MenuKeyboards{}
}

// GetMainMenu возвращает главное меню (2 ряда)
func (mk *MenuKeyboards) GetMainMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "⚙️ Настройки"},
				{Text: "📊 Статус"},
				{Text: "🔔 Уведомления"},
			},
			{
				{Text: "📈 Сигналы"},
				{Text: "⏱️ Периоды"},
				{Text: "📋 Помощь"},
			},
		},
		ResizeKeyboard:  true,  // Адаптивные кнопки
		OneTimeKeyboard: false, // Постоянное меню
		Selective:       false,
		IsPersistent:    true, // Сохраняется после использования
	}
}

// GetSettingsMenu возвращает меню настроек (2 ряда)
func (mk *MenuKeyboards) GetSettingsMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "🔔 Вкл/Выкл"},
				{Text: "📈 Тип сигналов"},
				{Text: "🔄 Сбросить"},
			},
			{
				{Text: "⏱️ 5мин"},
				{Text: "⏱️ 15мин"},
				{Text: "🔙 Назад"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetNotificationsMenu возвращает меню уведомлений (2 ряда)
func (mk *MenuKeyboards) GetNotificationsMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "✅ Включить"},
				{Text: "❌ Выключить"},
				{Text: "📊 Все сигналы"},
			},
			{
				{Text: "📈 Только рост"},
				{Text: "📉 Только падение"},
				{Text: "🔙 Назад"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetSignalTypesMenu возвращает меню типов сигналов (2 ряда)
func (mk *MenuKeyboards) GetSignalTypesMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "📈 Только рост"},
				{Text: "📉 Только падение"},
				{Text: "📊 Все сигналы"},
			},
			{
				{Text: "🔔 Настройки уведомлений"},
				{Text: "📊 Статус"},
				{Text: "🔙 Главное меню"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetPeriodsMenu возвращает меню периодов (2 ряда)
func (mk *MenuKeyboards) GetPeriodsMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "⏱️ 5 мин"},
				{Text: "⏱️ 15 мин"},
				{Text: "⏱️ 30 мин"},
			},
			{
				{Text: "⏱️ 1 час"},
				{Text: "⏱️ 4 часа"},
				{Text: "🔙 Назад"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetResetMenu возвращает меню сброса (2 ряда)
func (mk *MenuKeyboards) GetResetMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "🔄 Все счетчики"},
				{Text: "📊 По символу"},
				{Text: "📈 Счетчик роста"},
			},
			{
				{Text: "📉 Счетчик падения"},
				{Text: "⚙️ Настройки"},
				{Text: "🔙 Главное меню"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetInlineMenuSettings возвращает inline меню настроек (для быстрых действий)
func (mk *MenuKeyboards) GetInlineMenuSettings() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔔 Уведомления", CallbackData: "menu_notify"},
				{Text: "📊 Тип сигналов", CallbackData: "menu_signals"},
			},
			{
				{Text: "⏱️ Периоды", CallbackData: "menu_periods"},
				{Text: "🔄 Сбросить", CallbackData: "menu_reset"},
			},
		},
	}
}

// GetInlineMenuPeriods возвращает inline меню периодов
func (mk *MenuKeyboards) GetInlineMenuPeriods() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "5 мин", CallbackData: "period_5m"},
				{Text: "15 мин", CallbackData: "period_15m"},
				{Text: "30 мин", CallbackData: "period_30m"},
			},
			{
				{Text: "1 час", CallbackData: "period_1h"},
				{Text: "4 часа", CallbackData: "period_4h"},
				{Text: "🔙 Назад", CallbackData: "menu_back"},
			},
		},
	}
}

// GetInlineMenuReset возвращает inline меню сброса
func (mk *MenuKeyboards) GetInlineMenuReset() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Все счетчики", CallbackData: "reset_all"},
				{Text: "📊 По символу", CallbackData: "reset_symbol"},
			},
			{
				{Text: "📈 Счетчик роста", CallbackData: "reset_growth"},
				{Text: "📉 Счетчик падения", CallbackData: "reset_fall"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "menu_back"},
			},
		},
	}
}
