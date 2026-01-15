package signal_settings

// updateSensitivity обновляет чувствительность
func (s *serviceImpl) updateSensitivity(params SignalSettingsParams) (SignalSettingsResult, error) {
	// TODO: Реализовать после добавления поля sensitivity в модель User
	return SignalSettingsResult{
		Success: true,
		Message: "Настройка чувствительности в разработке",
		UserID:  params.UserID,
	}, nil
}

// updateQuietHours обновляет тихие часы
func (s *serviceImpl) updateQuietHours(params SignalSettingsParams) (SignalSettingsResult, error) {
	// TODO: Реализовать настройку тихих часов
	return SignalSettingsResult{
		Success: true,
		Message: "Настройка тихих часов в разработке",
		UserID:  params.UserID,
	}, nil
}
