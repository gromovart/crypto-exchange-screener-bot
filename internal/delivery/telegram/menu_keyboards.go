// internal/delivery/telegram/menu_keyboards.go
package telegram

// MenuKeyboards - клавиатуры меню (2 ряда для устранения скролла)
type MenuKeyboards struct{}

// NewMenuKeyboards создает новые клавиатуры меню
func NewMenuKeyboards() *MenuKeyboards {
	return &MenuKeyboards{}
}

// GetMainMenu возвращает главное меню (использует централизованный метод)
func (mk *MenuKeyboards) GetMainMenu() ReplyKeyboardMarkup {
	return GetMainMenuKeyboard()
}

// GetSettingsMenu возвращает меню настроек (использует централизованный метод)
func (mk *MenuKeyboards) GetSettingsMenu() ReplyKeyboardMarkup {
	return GetSettingsMenuKeyboard()
}

// GetNotificationsMenu возвращает меню уведомлений (использует централизованный метод)
func (mk *MenuKeyboards) GetNotificationsMenu() ReplyKeyboardMarkup {
	return GetNotificationsMenuKeyboard()
}

// GetSignalTypesMenu возвращает меню типов сигналов (использует централизованный метод)
func (mk *MenuKeyboards) GetSignalTypesMenu() ReplyKeyboardMarkup {
	return GetSignalTypesMenuKeyboard()
}

// GetPeriodsMenu возвращает меню периодов (использует централизованный метод)
func (mk *MenuKeyboards) GetPeriodsMenu() ReplyKeyboardMarkup {
	return GetPeriodsMenuKeyboard()
}

// GetResetMenu возвращает меню сброса (использует централизованный метод)
func (mk *MenuKeyboards) GetResetMenu() ReplyKeyboardMarkup {
	return GetResetMenuKeyboard()
}

// GetInlineMenuSettings возвращает inline меню настроек (использует централизованный метод)
func (mk *MenuKeyboards) GetInlineMenuSettings() *InlineKeyboardMarkup {
	return CreateSettingsKeyboard()
}

// GetInlineMenuPeriods возвращает inline меню периодов (использует централизованный метод)
func (mk *MenuKeyboards) GetInlineMenuPeriods() *InlineKeyboardMarkup {
	return CreatePeriodSelectionKeyboard()
}

// GetInlineMenuReset возвращает inline меню сброса (использует централизованный метод)
func (mk *MenuKeyboards) GetInlineMenuReset() *InlineKeyboardMarkup {
	return CreateResetKeyboard()
}
