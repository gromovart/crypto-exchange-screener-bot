// internal/users/settings_manager.go (исправленный)
package users

// import (
// 	"context"
// 	"crypto-exchange-screener-bot/persistence/postgres/repository/users"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"math"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// 	"github.com/jmoiron/sqlx"
// 	"github.com/lib/pq"
// )

// // SettingsManager управляет настройками пользователей
// type SettingsManager struct {
// 	db          *sqlx.DB
// 	redisClient *redis.Client
// 	cachePrefix string
// 	cacheTTL    time.Duration
// }

// // NewSettingsManager создает новый менеджер настроек
// func NewSettingsManager(db *sqlx.DB, redisClient *redis.Client) *SettingsManager {
// 	return &SettingsManager{
// 		db:          db,
// 		redisClient: redisClient,
// 		cachePrefix: "user_settings:",
// 		cacheTTL:    30 * time.Minute,
// 	}
// }

// // NotificationUpdateRequest запрос на обновление уведомлений
// type NotificationUpdateRequest struct {
// 	Enabled    *bool `json:"enabled,omitempty"`
// 	Growth     *bool `json:"growth,omitempty"`
// 	Fall       *bool `json:"fall,omitempty"`
// 	Continuous *bool `json:"continuous,omitempty"`
// 	QuietHours struct {
// 		Start *int `json:"start,omitempty"`
// 		End   *int `json:"end,omitempty"`
// 	} `json:"quiet_hours,omitempty"`
// }

// // ThresholdsUpdateRequest запрос на обновление порогов
// type ThresholdsUpdateRequest struct {
// 	MinGrowth *float64 `json:"min_growth,omitempty"`
// 	MinFall   *float64 `json:"min_fall,omitempty"`
// }

// // PeriodsUpdateRequest запрос на обновление периодов
// type PeriodsUpdateRequest struct {
// 	PreferredPeriods []int `json:"preferred_periods,omitempty"`
// }

// // DisplayUpdateRequest запрос на обновление отображения
// type DisplayUpdateRequest struct {
// 	DisplayMode *string `json:"display_mode,omitempty"`
// 	Language    *string `json:"language,omitempty"`
// 	Timezone    *string `json:"timezone,omitempty"`
// }

// // GetUserSettings получает настройки пользователя
// func (sm *SettingsManager) GetUserSettings(userID int) (*users.UserSettings, error) {
// 	// Пробуем получить из кэша
// 	cacheKey := sm.cachePrefix + strconv.Itoa(userID)
// 	if cached, err := sm.redisClient.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var settings users.UserSettings
// 		if err := json.Unmarshal([]byte(cached), &settings); err == nil {
// 			return &settings, nil
// 		}
// 	}

// 	// Получаем из базы данных
// 	query := `
//     SELECT
//         min_growth_threshold,
//         min_fall_threshold,
//         preferred_periods,
//         language,
//         timezone,
//         display_mode,
//         min_volume_filter,
//         exclude_patterns
//     FROM users
//     WHERE id = $1
//     `

// 	var settings users.UserSettings
// 	var periods []int64
// 	var excludePatterns []string

// 	err := sm.db.QueryRow(query, userID).Scan(
// 		&settings.MinGrowthThreshold,
// 		&settings.MinFallThreshold,
// 		&periods,
// 		&settings.Language,
// 		&settings.Timezone,
// 		&settings.DisplayMode,
// 		&settings.MinVolumeFilter,
// 		&excludePatterns,
// 	)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user settings: %w", err)
// 	}

// 	// Конвертируем массивы в срезы
// 	settings.PreferredPeriods = make([]int, len(periods))
// 	for i, p := range periods {
// 		settings.PreferredPeriods[i] = int(p)
// 	}
// 	settings.ExcludePatterns = excludePatterns

// 	// Кэшируем
// 	if data, err := json.Marshal(settings); err == nil {
// 		sm.redisClient.Set(context.Background(), cacheKey, data, sm.cacheTTL)
// 	}

// 	return &settings, nil
// }

// // UpdateNotificationSettings обновляет настройки уведомлений
// func (sm *SettingsManager) UpdateNotificationSettings(userID int, req NotificationUpdateRequest) error {
// 	// Получаем текущие настройки
// 	current, err := sm.getCurrentNotifications(userID)
// 	if err != nil {
// 		return err
// 	}

// 	// Применяем изменения
// 	if req.Enabled != nil {
// 		current.Enabled = *req.Enabled
// 	}
// 	if req.Growth != nil {
// 		current.Growth = *req.Growth
// 	}
// 	if req.Fall != nil {
// 		current.Fall = *req.Fall
// 	}
// 	if req.Continuous != nil {
// 		current.Continuous = *req.Continuous
// 	}
// 	if req.QuietHours.Start != nil {
// 		current.QuietHoursStart = *req.QuietHours.Start
// 	}
// 	if req.QuietHours.End != nil {
// 		current.QuietHoursEnd = *req.QuietHours.End
// 	}

// 	// Валидация
// 	if err := sm.validateNotificationSettings(current); err != nil {
// 		return err
// 	}

// 	// Обновляем в базе
// 	query := `
//     UPDATE users
//     SET notifications_enabled = $1,
//         notify_growth = $2,
//         notify_fall = $3,
//         notify_continuous = $4,
//         quiet_hours_start = $5,
//         quiet_hours_end = $6,
//         updated_at = NOW()
//     WHERE id = $7
//     `

// 	result, err := sm.db.Exec(query,
// 		current.Enabled,
// 		current.Growth,
// 		current.Fall,
// 		current.Continuous,
// 		current.QuietHoursStart,
// 		current.QuietHoursEnd,
// 		userID,
// 	)

// 	if err != nil {
// 		return fmt.Errorf("failed to update notification settings: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	sm.invalidateUserCache(userID)

// 	// Логируем изменение
// 	sm.logSettingChange(userID, "notifications", req)

// 	return nil
// }

// // UpdateThresholdSettings обновляет пороги анализа
// func (sm *SettingsManager) UpdateThresholdSettings(userID int, req ThresholdsUpdateRequest) error {
// 	// Получаем текущие настройки
// 	current, err := sm.GetUserSettings(userID)
// 	if err != nil {
// 		return err
// 	}

// 	// Применяем изменения
// 	if req.MinGrowth != nil {
// 		current.MinGrowthThreshold = *req.MinGrowth
// 	}
// 	if req.MinFall != nil {
// 		current.MinFallThreshold = *req.MinFall
// 	}

// 	// Валидация
// 	if err := sm.validateThresholdSettings(current); err != nil {
// 		return err
// 	}

// 	// Обновляем в базе
// 	query := `
//     UPDATE users
//     SET min_growth_threshold = $1,
//         min_fall_threshold = $2,
//         updated_at = NOW()
//     WHERE id = $3
//     `

// 	result, err := sm.db.Exec(query,
// 		current.MinGrowthThreshold,
// 		current.MinFallThreshold,
// 		userID,
// 	)

// 	if err != nil {
// 		return fmt.Errorf("failed to update threshold settings: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	sm.invalidateUserCache(userID)

// 	// Логируем изменение
// 	sm.logSettingChange(userID, "thresholds", req)

// 	return nil
// }

// // UpdatePeriodSettings обновляет периоды анализа
// func (sm *SettingsManager) UpdatePeriodSettings(userID int, req PeriodsUpdateRequest) error {
// 	// Валидация периодов
// 	if err := sm.validatePeriods(req.PreferredPeriods); err != nil {
// 		return err
// 	}

// 	// Обновляем в базе
// 	query := `
//     UPDATE users
//     SET preferred_periods = $1,
//         updated_at = NOW()
//     WHERE id = $2
//     `

// 	result, err := sm.db.Exec(query, pq.Array(req.PreferredPeriods), userID)
// 	if err != nil {
// 		return fmt.Errorf("failed to update period settings: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	sm.invalidateUserCache(userID)

// 	// Логируем изменение
// 	sm.logSettingChange(userID, "periods", req)

// 	return nil
// }

// // UpdateDisplaySettings обновляет настройки отображения
// func (sm *SettingsManager) UpdateDisplaySettings(userID int, req DisplayUpdateRequest) error {
// 	// Получаем текущие настройки
// 	current, err := sm.GetUserSettings(userID)
// 	if err != nil {
// 		return err
// 	}

// 	// Применяем изменения
// 	updateFields := []string{}
// 	updateValues := []interface{}{}
// 	paramCount := 1

// 	if req.DisplayMode != nil {
// 		if !sm.isValidDisplayMode(*req.DisplayMode) {
// 			return fmt.Errorf("invalid display mode: %s", *req.DisplayMode)
// 		}
// 		current.DisplayMode = *req.DisplayMode
// 		updateFields = append(updateFields, fmt.Sprintf("display_mode = $%d", paramCount))
// 		updateValues = append(updateValues, current.DisplayMode)
// 		paramCount++
// 	}

// 	if req.Language != nil {
// 		if !sm.isValidLanguage(*req.Language) {
// 			return fmt.Errorf("invalid language: %s", *req.Language)
// 		}
// 		current.Language = *req.Language
// 		updateFields = append(updateFields, fmt.Sprintf("language = $%d", paramCount))
// 		updateValues = append(updateValues, current.Language)
// 		paramCount++
// 	}

// 	if req.Timezone != nil {
// 		if !sm.isValidTimezone(*req.Timezone) {
// 			return fmt.Errorf("invalid timezone: %s", *req.Timezone)
// 		}
// 		current.Timezone = *req.Timezone
// 		updateFields = append(updateFields, fmt.Sprintf("timezone = $%d", paramCount))
// 		updateValues = append(updateValues, current.Timezone)
// 		paramCount++
// 	}

// 	// Если нет изменений
// 	if len(updateFields) == 0 {
// 		return nil
// 	}

// 	// Строим запрос
// 	updateValues = append(updateValues, userID)
// 	query := fmt.Sprintf(`
//     UPDATE users
//     SET %s,
//         updated_at = NOW()
//     WHERE id = $%d
//     `, strings.Join(updateFields, ", "), paramCount)

// 	result, err := sm.db.Exec(query, updateValues...)
// 	if err != nil {
// 		return fmt.Errorf("failed to update display settings: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	sm.invalidateUserCache(userID)

// 	// Логируем изменение
// 	sm.logSettingChange(userID, "display", req)

// 	return nil
// }

// // UpdateAllSettings обновляет все настройки пользователя
// func (sm *SettingsManager) UpdateAllSettings(userID int, settings *users.UserSettings) error {
// 	// Валидация
// 	if err := sm.validateThresholdSettings(settings); err != nil {
// 		return err
// 	}

// 	if err := sm.validatePeriods(settings.PreferredPeriods); err != nil {
// 		return err
// 	}

// 	if !sm.isValidDisplayMode(settings.DisplayMode) {
// 		return fmt.Errorf("invalid display mode: %s", settings.DisplayMode)
// 	}

// 	if !sm.isValidLanguage(settings.Language) {
// 		return fmt.Errorf("invalid language: %s", settings.Language)
// 	}

// 	if !sm.isValidTimezone(settings.Timezone) {
// 		return fmt.Errorf("invalid timezone: %s", settings.Timezone)
// 	}

// 	// Обновляем в базе
// 	query := `
//     UPDATE users
//     SET min_growth_threshold = $1,
//         min_fall_threshold = $2,
//         preferred_periods = $3,
//         language = $4,
//         timezone = $5,
//         display_mode = $6,
//         min_volume_filter = $7,
//         exclude_patterns = $8,
//         updated_at = NOW()
//     WHERE id = $9
//     `

// 	result, err := sm.db.Exec(query,
// 		settings.MinGrowthThreshold,
// 		settings.MinFallThreshold,
// 		pq.Array(settings.PreferredPeriods),
// 		settings.Language,
// 		settings.Timezone,
// 		settings.DisplayMode,
// 		settings.MinVolumeFilter,
// 		pq.Array(settings.ExcludePatterns),
// 		userID,
// 	)

// 	if err != nil {
// 		return fmt.Errorf("failed to update all settings: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	sm.invalidateUserCache(userID)

// 	// Логируем изменение
// 	sm.logSettingChange(userID, "all_settings", settings)

// 	return nil
// }

// // ResetToDefault сбрасывает настройки пользователя к значениям по умолчанию
// func (sm *SettingsManager) ResetToDefault(userID int) error {
// 	defaultSettings := sm.GetDefaultSettings()

// 	query := `
//     UPDATE users
//     SET min_growth_threshold = $1,
//         min_fall_threshold = $2,
//         preferred_periods = $3,
//         language = $4,
//         timezone = $5,
//         display_mode = $6,
//         min_volume_filter = $7,
//         exclude_patterns = $8,
//         updated_at = NOW()
//     WHERE id = $9
//     `

// 	result, err := sm.db.Exec(query,
// 		defaultSettings.MinGrowthThreshold,
// 		defaultSettings.MinFallThreshold,
// 		pq.Array(defaultSettings.PreferredPeriods),
// 		defaultSettings.Language,
// 		defaultSettings.Timezone,
// 		defaultSettings.DisplayMode,
// 		defaultSettings.MinVolumeFilter,
// 		pq.Array(defaultSettings.ExcludePatterns),
// 		userID,
// 	)

// 	if err != nil {
// 		return fmt.Errorf("failed to reset settings: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	sm.invalidateUserCache(userID)

// 	// Логируем сброс
// 	sm.logSettingChange(userID, "reset_to_default", nil)

// 	return nil
// }

// // GetDefaultSettings возвращает настройки по умолчанию
// func (sm *SettingsManager) GetDefaultSettings() *users.UserSettings {
// 	return &users.UserSettings{
// 		MinGrowthThreshold: 2.0,
// 		MinFallThreshold:   2.0,
// 		PreferredPeriods:   []int{5, 15, 30},
// 		MinVolumeFilter:    0.0,
// 		ExcludePatterns:    []string{},
// 		Language:           "ru",
// 		Timezone:           "Europe/Moscow",
// 		DisplayMode:        "compact",
// 	}
// }

// // GetNotificationPreferences получает настройки уведомлений пользователя
// func (sm *SettingsManager) GetNotificationPreferences(userID int) (*users.NotificationSettings, error) {
// 	cacheKey := sm.cachePrefix + "notifications:" + strconv.Itoa(userID)

// 	// Пробуем получить из кэша
// 	if cached, err := sm.redisClient.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var settings users.NotificationSettings
// 		if err := json.Unmarshal([]byte(cached), &settings); err == nil {
// 			return &settings, nil
// 		}
// 	}

// 	// Получаем из базы
// 	query := `
//     SELECT
//         notifications_enabled,
//         notify_growth,
//         notify_fall,
//         notify_continuous,
//         quiet_hours_start,
//         quiet_hours_end
//     FROM users
//     WHERE id = $1
//     `

// 	var settings users.NotificationSettings

// 	err := sm.db.QueryRow(query, userID).Scan(
// 		&settings.Enabled,
// 		&settings.Growth,
// 		&settings.Fall,
// 		&settings.Continuous,
// 		&settings.QuietHoursStart,
// 		&settings.QuietHoursEnd,
// 	)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get notification preferences: %w", err)
// 	}

// 	// Кэшируем
// 	if data, err := json.Marshal(settings); err == nil {
// 		sm.redisClient.Set(context.Background(), cacheKey, data, sm.cacheTTL)
// 	}

// 	return &settings, nil
// }

// // ToggleNotification переключает конкретный тип уведомления
// func (sm *SettingsManager) ToggleNotification(userID int, notificationType string) (bool, error) {
// 	preferences, err := sm.GetNotificationPreferences(userID)
// 	if err != nil {
// 		return false, err
// 	}

// 	var newValue bool
// 	var fieldName string

// 	switch notificationType {
// 	case "all":
// 		preferences.Enabled = !preferences.Enabled
// 		newValue = preferences.Enabled
// 		fieldName = "notifications_enabled"
// 	case "growth":
// 		preferences.Growth = !preferences.Growth
// 		newValue = preferences.Growth
// 		fieldName = "notify_growth"
// 	case "fall":
// 		preferences.Fall = !preferences.Fall
// 		newValue = preferences.Fall
// 		fieldName = "notify_fall"
// 	case "continuous":
// 		preferences.Continuous = !preferences.Continuous
// 		newValue = preferences.Continuous
// 		fieldName = "notify_continuous"
// 	default:
// 		return false, fmt.Errorf("unknown notification type: %s", notificationType)
// 	}

// 	// Обновляем в базе
// 	query := fmt.Sprintf(`
//     UPDATE users
//     SET %s = $1,
//         updated_at = NOW()
//     WHERE id = $2
//     `, fieldName)

// 	result, err := sm.db.Exec(query, newValue, userID)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to toggle notification: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return false, sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш уведомлений
// 	sm.redisClient.Del(context.Background(), sm.cachePrefix+"notifications:"+strconv.Itoa(userID))

// 	// Инвалидируем основной кэш
// 	sm.invalidateUserCache(userID)

// 	return newValue, nil
// }

// // SetQuietHours устанавливает тихие часы
// func (sm *SettingsManager) SetQuietHours(userID, startHour, endHour int) error {
// 	// Валидация
// 	if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
// 		return fmt.Errorf("hours must be between 0 and 23")
// 	}

// 	query := `
//     UPDATE users
//     SET quiet_hours_start = $1,
//         quiet_hours_end = $2,
//         updated_at = NOW()
//     WHERE id = $3
//     `

// 	result, err := sm.db.Exec(query, startHour, endHour, userID)
// 	if err != nil {
// 		return fmt.Errorf("failed to set quiet hours: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш уведомлений
// 	sm.redisClient.Del(context.Background(), sm.cachePrefix+"notifications:"+strconv.Itoa(userID))

// 	// Логируем изменение
// 	sm.logSettingChange(userID, "quiet_hours", map[string]interface{}{
// 		"start": startHour,
// 		"end":   endHour,
// 	})

// 	return nil
// }

// // GetSettingsForAnalysis получает настройки пользователя для анализа
// // Оптимизированная версия для использования в пайплайне анализа
// func (sm *SettingsManager) GetSettingsForAnalysis(userID int) (*AnalysisSettings, error) {
// 	cacheKey := sm.cachePrefix + "analysis:" + strconv.Itoa(userID)

// 	// Пробуем получить из кэша
// 	if cached, err := sm.redisClient.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var settings AnalysisSettings
// 		if err := json.Unmarshal([]byte(cached), &settings); err == nil {
// 			return &settings, nil
// 		}
// 	}

// 	// Получаем из базы (JOIN для оптимизации)
// 	query := `
//     SELECT
//         u.min_growth_threshold,
//         u.min_fall_threshold,
//         u.preferred_periods,
//         u.notifications_enabled,
//         u.notify_growth,
//         u.notify_fall,
//         u.quiet_hours_start,
//         u.quiet_hours_end,
//         u.signals_today,
//         u.max_signals_per_day,
//         COALESCE(s.max_symbols, 100) as max_symbols,
//         COALESCE(s.max_signals_per_day, u.max_signals_per_day) as effective_max_signals
//     FROM users u
//     LEFT JOIN subscription_plans s ON u.subscription_tier = s.code
//     WHERE u.id = $1 AND u.is_active = TRUE
//     `

// 	var settings AnalysisSettings
// 	var periods []int64

// 	err := sm.db.QueryRow(query, userID).Scan(
// 		&settings.MinGrowthThreshold,
// 		&settings.MinFallThreshold,
// 		&periods,
// 		&settings.NotificationsEnabled,
// 		&settings.NotifyGrowth,
// 		&settings.NotifyFall,
// 		&settings.QuietHoursStart,
// 		&settings.QuietHoursEnd,
// 		&settings.SignalsToday,
// 		&settings.MaxSignalsPerDay,
// 		&settings.MaxSymbols,
// 		&settings.EffectiveMaxSignals,
// 	)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get analysis settings: %w", err)
// 	}

// 	// Конвертируем массив в срез
// 	settings.PreferredPeriods = make([]int, len(periods))
// 	for i, p := range periods {
// 		settings.PreferredPeriods[i] = int(p)
// 	}

// 	// Рассчитываем текущий час для проверки тихих часов
// 	settings.CurrentHour = time.Now().UTC().Hour()

// 	// Кэшируем (короткий TTL, т.к. часто меняется)
// 	if data, err := json.Marshal(settings); err == nil {
// 		sm.redisClient.Set(context.Background(), cacheKey, data, 5*time.Minute)
// 	}

// 	return &settings, nil
// }

// // AnalysisSettings оптимизированная структура для анализа
// type AnalysisSettings struct {
// 	MinGrowthThreshold   float64 `json:"min_growth_threshold"`
// 	MinFallThreshold     float64 `json:"min_fall_threshold"`
// 	PreferredPeriods     []int   `json:"preferred_periods"`
// 	NotificationsEnabled bool    `json:"notifications_enabled"`
// 	NotifyGrowth         bool    `json:"notify_growth"`
// 	NotifyFall           bool    `json:"notify_fall"`
// 	QuietHoursStart      int     `json:"quiet_hours_start"`
// 	QuietHoursEnd        int     `json:"quiet_hours_end"`
// 	SignalsToday         int     `json:"signals_today"`
// 	MaxSignalsPerDay     int     `json:"max_signals_per_day"`
// 	MaxSymbols           int     `json:"max_symbols"`
// 	EffectiveMaxSignals  int     `json:"effective_max_signals"`
// 	CurrentHour          int     `json:"current_hour"`
// }

// // ShouldSendSignal проверяет, нужно ли отправлять сигнал пользователю
// func (as *AnalysisSettings) ShouldSendSignal(signalType string, changePercent float64) bool {
// 	// Быстрая проверка уведомлений
// 	if !as.NotificationsEnabled {
// 		return false
// 	}

// 	// Проверка типа сигнала
// 	if signalType == "growth" && !as.NotifyGrowth {
// 		return false
// 	}
// 	if signalType == "fall" && !as.NotifyFall {
// 		return false
// 	}

// 	// Проверка порогов
// 	if signalType == "growth" && changePercent < as.MinGrowthThreshold {
// 		return false
// 	}
// 	if signalType == "fall" && math.Abs(changePercent) < as.MinFallThreshold {
// 		return false
// 	}

// 	// Проверка тихих часов
// 	if as.QuietHoursStart != 0 || as.QuietHoursEnd != 0 {
// 		if as.QuietHoursStart > as.QuietHoursEnd {
// 			// Например, 23-8 (ночные часы)
// 			if as.CurrentHour >= as.QuietHoursStart || as.CurrentHour < as.QuietHoursEnd {
// 				return false
// 			}
// 		} else {
// 			// Например, 14-18 (дневные часы)
// 			if as.CurrentHour >= as.QuietHoursStart && as.CurrentHour < as.QuietHoursEnd {
// 				return false
// 			}
// 		}
// 	}

// 	// Проверка лимитов
// 	if as.SignalsToday >= as.EffectiveMaxSignals {
// 		return false
// 	}

// 	return true
// }

// // Вспомогательные методы

// func (sm *SettingsManager) getCurrentNotifications(userID int) (*users.NotificationSettings, error) {
// 	query := `
//     SELECT
//         notifications_enabled,
//         notify_growth,
//         notify_fall,
//         notify_continuous,
//         quiet_hours_start,
//         quiet_hours_end
//     FROM users
//     WHERE id = $1
//     `

// 	var settings users.NotificationSettings

// 	err := sm.db.QueryRow(query, userID).Scan(
// 		&settings.Enabled,
// 		&settings.Growth,
// 		&settings.Fall,
// 		&settings.Continuous,
// 		&settings.QuietHoursStart,
// 		&settings.QuietHoursEnd,
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return &settings, nil
// }

// func (sm *SettingsManager) validateNotificationSettings(settings *users.NotificationSettings) error {
// 	// Проверка тихих часов
// 	if settings.QuietHoursStart < 0 || settings.QuietHoursStart > 23 {
// 		return fmt.Errorf("quiet_hours_start must be between 0 and 23")
// 	}
// 	if settings.QuietHoursEnd < 0 || settings.QuietHoursEnd > 23 {
// 		return fmt.Errorf("quiet_hours_end must be between 0 and 23")
// 	}

// 	return nil
// }

// func (sm *SettingsManager) validateThresholdSettings(settings *users.UserSettings) error {
// 	if settings.MinGrowthThreshold < 0.1 || settings.MinGrowthThreshold > 50.0 {
// 		return fmt.Errorf("min_growth_threshold must be between 0.1%% and 50%%")
// 	}
// 	if settings.MinFallThreshold < 0.1 || settings.MinFallThreshold > 50.0 {
// 		return fmt.Errorf("min_fall_threshold must be between 0.1%% and 50%%")
// 	}

// 	return nil
// }

// func (sm *SettingsManager) validatePeriods(periods []int) error {
// 	if len(periods) == 0 {
// 		return fmt.Errorf("at least one period must be specified")
// 	}

// 	validPeriods := map[int]bool{
// 		1:    true,
// 		5:    true,
// 		15:   true,
// 		30:   true,
// 		60:   true,
// 		240:  true,
// 		1440: true,
// 	}

// 	for _, period := range periods {
// 		if !validPeriods[period] {
// 			return fmt.Errorf("invalid period: %d minutes. Valid periods: 1, 5, 15, 30, 60, 240, 1440", period)
// 		}
// 	}

// 	return nil
// }

// func (sm *SettingsManager) isValidDisplayMode(mode string) bool {
// 	validModes := map[string]bool{
// 		"compact":  true,
// 		"detailed": true,
// 		"minimal":  true,
// 	}
// 	return validModes[mode]
// }

// func (sm *SettingsManager) isValidLanguage(lang string) bool {
// 	validLanguages := map[string]bool{
// 		"ru": true,
// 		"en": true,
// 		"es": true,
// 		"zh": true,
// 	}
// 	return validLanguages[lang]
// }

// func (sm *SettingsManager) isValidTimezone(tz string) bool {
// 	// Простая проверка - в реальном приложении используйте библиотеку time/tzdata
// 	validTimezones := map[string]bool{
// 		"Europe/Moscow":    true,
// 		"UTC":              true,
// 		"America/New_York": true,
// 		"Europe/London":    true,
// 		"Asia/Tokyo":       true,
// 	}
// 	return validTimezones[tz]
// }

// func (sm *SettingsManager) invalidateUserCache(userID int) {
// 	keys := []string{
// 		sm.cachePrefix + strconv.Itoa(userID),
// 		sm.cachePrefix + "notifications:" + strconv.Itoa(userID),
// 		sm.cachePrefix + "analysis:" + strconv.Itoa(userID),
// 	}

// 	sm.redisClient.Del(context.Background(), keys...)
// }

// func (sm *SettingsManager) logSettingChange(userID int, settingType string, data interface{}) {
// 	log.Printf("User %d updated %s settings: %v", userID, settingType, data)

// 	// В реальном приложении можно сохранять в таблицу аудита
// 	// sm.db.Exec(`
// 	// INSERT INTO user_settings_audit (user_id, setting_type, old_value, new_value)
// 	// VALUES ($1, $2, $3, $4)
// 	// `, userID, settingType, oldValue, newValue)
// }

// // Telegram команды для управления настройками

// func (sm *SettingsManager) RegisterTelegramCommands(bot interface{}) {
// 	// В реальной реализации регистрируем команды
// 	// Например:
// 	// bot.HandleCommand("/settings", sm.handleSettingsCommand)
// 	// bot.HandleCommand("/thresholds", sm.handleThresholdsCommand)
// 	// bot.HandleCommand("/notifications", sm.handleNotificationsCommand)
// }

// // Экспорт настроек для API
// func (sm *SettingsManager) ExportSettings(userID int) (map[string]interface{}, error) {
// 	settings, err := sm.GetUserSettings(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	notifications, err := sm.GetNotificationPreferences(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return map[string]interface{}{
// 		"analysis": map[string]interface{}{
// 			"min_growth_threshold": settings.MinGrowthThreshold,
// 			"min_fall_threshold":   settings.MinFallThreshold,
// 			"preferred_periods":    settings.PreferredPeriods,
// 			"min_volume_filter":    settings.MinVolumeFilter,
// 			"exclude_patterns":     settings.ExcludePatterns,
// 		},
// 		"notifications": map[string]interface{}{
// 			"enabled":    notifications.Enabled,
// 			"growth":     notifications.Growth,
// 			"fall":       notifications.Fall,
// 			"continuous": notifications.Continuous,
// 			"quiet_hours": map[string]int{
// 				"start": notifications.QuietHoursStart,
// 				"end":   notifications.QuietHoursEnd,
// 			},
// 		},
// 		"display": map[string]interface{}{
// 			"language":     settings.Language,
// 			"timezone":     settings.Timezone,
// 			"display_mode": settings.DisplayMode,
// 		},
// 		"limits": map[string]interface{}{
// 			"signals_today":       0, // Будет получено отдельно
// 			"max_signals_per_day": 0, // Будет получено отдельно
// 		},
// 	}, nil
// }
