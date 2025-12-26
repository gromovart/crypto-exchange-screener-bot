// persistence/redis/session_store.go
package redis

// import (
// 	"context"
// 	"crypto/rand"
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"crypto-exchange-screener-bot/internal/users"

// 	"github.com/go-redis/redis/v8"
// )

// // SessionStore управляет сессиями пользователей в Redis
// type SessionStore struct {
// 	client     *redis.Client
// 	prefix     string
// 	sessionTTL time.Duration
// }

// // SessionData структура данных сессии
// type SessionData struct {
// 	UserID       int                    `json:"user_id"`
// 	TelegramID   int64                  `json:"telegram_id"`
// 	Username     string                 `json:"username"`
// 	FirstName    string                 `json:"first_name"`
// 	Role         string                 `json:"role"`
// 	Permissions  []string               `json:"permissions"`
// 	Settings     map[string]interface{} `json:"settings"`
// 	DeviceInfo   map[string]interface{} `json:"device_info"`
// 	IPAddress    string                 `json:"ip_address"`
// 	UserAgent    string                 `json:"user_agent"`
// 	CreatedAt    time.Time              `json:"created_at"`
// 	LastActivity time.Time              `json:"last_activity"`
// 	ExpiresAt    time.Time              `json:"expires_at"`
// }

// // NewSessionStore создает новое хранилище сессий
// func NewSessionStore(client *redis.Client) *SessionStore {
// 	return &SessionStore{
// 		client:     client,
// 		prefix:     "session:",
// 		sessionTTL: 24 * time.Hour, // 24 часа по умолчанию
// 	}
// }

// // SetSessionTTL устанавливает TTL для сессий
// func (s *SessionStore) SetSessionTTL(ttl time.Duration) {
// 	s.sessionTTL = ttl
// }

// // CreateSession создает новую сессию
// func (s *SessionStore) CreateSession(user *users.User, deviceInfo map[string]interface{}, ip, userAgent string) (string, *SessionData, error) {
// 	// Генерируем токен сессии
// 	token, err := s.generateToken()
// 	if err != nil {
// 		return "", nil, fmt.Errorf("failed to generate session token: %w", err)
// 	}

// 	// Создаем данные сессии
// 	sessionData := &SessionData{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		Role:         user.Role,
// 		Permissions:  s.getUserPermissions(user),
// 		Settings:     s.extractUserSettings(user),
// 		DeviceInfo:   deviceInfo,
// 		IPAddress:    ip,
// 		UserAgent:    userAgent,
// 		CreatedAt:    time.Now(),
// 		LastActivity: time.Now(),
// 		ExpiresAt:    time.Now().Add(s.sessionTTL),
// 	}

// 	// Сохраняем сессию в Redis
// 	if err := s.saveSession(token, sessionData); err != nil {
// 		return "", nil, fmt.Errorf("failed to save session: %w", err)
// 	}

// 	// Сохраняем связь пользователь -> токены
// 	if err := s.linkUserToSession(user.ID, token); err != nil {
// 		// Если не удалось сохранить связь, удаляем сессию
// 		s.DeleteSession(token)
// 		return "", nil, fmt.Errorf("failed to link user to session: %w", err)
// 	}

// 	// Логируем создание сессии
// 	s.logSessionActivity(token, "session_created", map[string]interface{}{
// 		"user_id":    user.ID,
// 		"ip_address": ip,
// 		"user_agent": userAgent,
// 	})

// 	return token, sessionData, nil
// }

// // GetSession получает сессию по токену
// func (s *SessionStore) GetSession(token string) (*SessionData, error) {
// 	ctx := context.Background()
// 	key := s.getSessionKey(token)

// 	// Получаем данные сессии
// 	data, err := s.client.Get(ctx, key).Result()
// 	if err != nil {
// 		if err == redis.Nil {
// 			return nil, nil // Сессия не найдена
// 		}
// 		return nil, fmt.Errorf("failed to get session: %w", err)
// 	}

// 	// Декодируем JSON
// 	var sessionData SessionData
// 	if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
// 	}

// 	// Проверяем срок действия
// 	if time.Now().After(sessionData.ExpiresAt) {
// 		// Сессия истекла, удаляем её
// 		s.DeleteSession(token)
// 		return nil, nil
// 	}

// 	return &sessionData, nil
// }

// // UpdateSessionActivity обновляет время последней активности
// func (s *SessionStore) UpdateSessionActivity(token string) error {
// 	sessionData, err := s.GetSession(token)
// 	if err != nil {
// 		return err
// 	}
// 	if sessionData == nil {
// 		return fmt.Errorf("session not found")
// 	}

// 	// Обновляем время активности
// 	sessionData.LastActivity = time.Now()

// 	// Обновляем время истечения (рефрешим TTL)
// 	sessionData.ExpiresAt = time.Now().Add(s.sessionTTL)

// 	// Сохраняем обновленные данные
// 	return s.saveSession(token, sessionData)
// }

// // DeleteSession удаляет сессию
// func (s *SessionStore) DeleteSession(token string) error {
// 	ctx := context.Background()
// 	key := s.getSessionKey(token)

// 	// Получаем данные сессии перед удалением
// 	sessionData, _ := s.GetSession(token)
// 	if sessionData != nil {
// 		// Удаляем связь пользователь -> токен
// 		s.unlinkUserFromSession(sessionData.UserID, token)
// 	}

// 	// Удаляем сессию
// 	if err := s.client.Del(ctx, key).Err(); err != nil {
// 		return fmt.Errorf("failed to delete session: %w", err)
// 	}

// 	return nil
// }

// // DeleteAllUserSessions удаляет все сессии пользователя
// func (s *SessionStore) DeleteAllUserSessions(userID int) error {
// 	ctx := context.Background()
// 	userSessionsKey := s.getUserSessionsKey(userID)

// 	// Получаем все токены пользователя
// 	tokens, err := s.client.SMembers(ctx, userSessionsKey).Result()
// 	if err != nil {
// 		return fmt.Errorf("failed to get user sessions: %w", err)
// 	}

// 	// Удаляем каждую сессию
// 	for _, token := range tokens {
// 		s.DeleteSession(token)
// 	}

// 	// Удаляем множество токенов пользователя
// 	if err := s.client.Del(ctx, userSessionsKey).Err(); err != nil {
// 		return fmt.Errorf("failed to delete user sessions set: %w", err)
// 	}

// 	return nil
// }

// // RevokeSession отзывает сессию (помечает как недействительную)
// func (s *SessionStore) RevokeSession(token, reason string) error {
// 	sessionData, err := s.GetSession(token)
// 	if err != nil {
// 		return err
// 	}
// 	if sessionData == nil {
// 		return fmt.Errorf("session not found")
// 	}

// 	// Добавляем метку отзыва в данные сессии
// 	if sessionData.DeviceInfo == nil {
// 		sessionData.DeviceInfo = make(map[string]interface{})
// 	}
// 	sessionData.DeviceInfo["revoked"] = true
// 	sessionData.DeviceInfo["revoked_at"] = time.Now().Format(time.RFC3339)
// 	sessionData.DeviceInfo["revoked_reason"] = reason

// 	// Сохраняем с коротким TTL (1 час для аудита)
// 	key := s.getSessionKey(token)
// 	data, _ := json.Marshal(sessionData)

// 	ctx := context.Background()
// 	if err := s.client.SetEX(ctx, key, data, time.Hour).Err(); err != nil {
// 		return fmt.Errorf("failed to revoke session: %w", err)
// 	}

// 	// Логируем отзыв
// 	s.logSessionActivity(token, "session_revoked", map[string]interface{}{
// 		"reason": reason,
// 	})

// 	return nil
// }

// // GetUserActiveSessions возвращает все активные сессии пользователя
// func (s *SessionStore) GetUserActiveSessions(userID int) ([]*SessionData, error) {
// 	ctx := context.Background()
// 	userSessionsKey := s.getUserSessionsKey(userID)

// 	// Получаем все токены пользователя
// 	tokens, err := s.client.SMembers(ctx, userSessionsKey).Result()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user sessions: %w", err)
// 	}

// 	var sessions []*SessionData
// 	for _, token := range tokens {
// 		sessionData, err := s.GetSession(token)
// 		if err != nil {
// 			continue // Пропускаем невалидные сессии
// 		}
// 		if sessionData != nil {
// 			sessions = append(sessions, sessionData)
// 		}
// 	}

// 	return sessions, nil
// }

// // GetUserSessionsCount возвращает количество активных сессий пользователя
// func (s *SessionStore) GetUserSessionsCount(userID int) (int, error) {
// 	ctx := context.Background()
// 	userSessionsKey := s.getUserSessionsKey(userID)

// 	count, err := s.client.SCard(ctx, userSessionsKey).Result()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get user sessions count: %w", err)
// 	}

// 	return int(count), nil
// }

// // CleanupExpiredSessions очищает истекшие сессии
// func (s *SessionStore) CleanupExpiredSessions() (int, error) {
// 	ctx := context.Background()
// 	pattern := s.prefix + "*"

// 	// Получаем все ключи сессий
// 	keys, err := s.client.Keys(ctx, pattern).Result()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get session keys: %w", err)
// 	}

// 	deletedCount := 0
// 	for _, key := range keys {
// 		// Проверяем TTL
// 		ttl, err := s.client.TTL(ctx, key).Result()
// 		if err != nil {
// 			continue
// 		}

// 		// Если TTL отрицательный (ключ истек или не имеет TTL), удаляем
// 		if ttl < 0 {
// 			// Получаем токен из ключа
// 			token := key[len(s.prefix):]

// 			// Получаем данные сессии для удаления связи пользователь -> токен
// 			sessionData, _ := s.GetSession(token)
// 			if sessionData != nil {
// 				s.unlinkUserFromSession(sessionData.UserID, token)
// 			}

// 			// Удаляем сессию
// 			if err := s.client.Del(ctx, key).Err(); err == nil {
// 				deletedCount++
// 			}
// 		}
// 	}

// 	return deletedCount, nil
// }

// // CleanupInactiveSessions очищает неактивные сессии (не было активности в течение N времени)
// func (s *SessionStore) CleanupInactiveSessions(maxInactiveTime time.Duration) (int, error) {
// 	ctx := context.Background()
// 	pattern := s.prefix + "*"

// 	keys, err := s.client.Keys(ctx, pattern).Result()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get session keys: %w", err)
// 	}

// 	deletedCount := 0
// 	cutoffTime := time.Now().Add(-maxInactiveTime)

// 	for _, key := range keys {
// 		// Получаем данные сессии
// 		data, err := s.client.Get(ctx, key).Result()
// 		if err != nil {
// 			continue
// 		}

// 		var sessionData SessionData
// 		if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
// 			continue
// 		}

// 		// Проверяем время последней активности
// 		if sessionData.LastActivity.Before(cutoffTime) {
// 			// Получаем токен из ключа
// 			token := key[len(s.prefix):]

// 			// Удаляем связь пользователь -> токен
// 			s.unlinkUserFromSession(sessionData.UserID, token)

// 			// Удаляем сессию
// 			if err := s.client.Del(ctx, key).Err(); err == nil {
// 				deletedCount++
// 			}
// 		}
// 	}

// 	return deletedCount, nil
// }

// // GetSessionStats возвращает статистику сессий
// func (s *SessionStore) GetSessionStats() (map[string]interface{}, error) {
// 	ctx := context.Background()
// 	pattern := s.prefix + "*"

// 	keys, err := s.client.Keys(ctx, pattern).Result()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get session keys: %w", err)
// 	}

// 	stats := map[string]interface{}{
// 		"total_sessions":     len(keys),
// 		"active_users":       0,
// 		"sessions_by_device": map[string]int{},
// 		"sessions_by_hour":   map[int]int{},
// 	}

// 	// Счетчики для уникальных пользователей и устройств
// 	uniqueUsers := make(map[int]bool)
// 	deviceCounts := make(map[string]int)
// 	hourCounts := make(map[int]int)

// 	for _, key := range keys {
// 		data, err := s.client.Get(ctx, key).Result()
// 		if err != nil {
// 			continue
// 		}

// 		var sessionData SessionData
// 		if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
// 			continue
// 		}

// 		// Считаем уникальных пользователей
// 		uniqueUsers[sessionData.UserID] = true

// 		// Считаем устройства
// 		if deviceType, ok := sessionData.DeviceInfo["type"].(string); ok {
// 			deviceCounts[deviceType]++
// 		}

// 		// Считаем по часам создания
// 		hour := sessionData.CreatedAt.Hour()
// 		hourCounts[hour]++
// 	}

// 	stats["active_users"] = len(uniqueUsers)
// 	stats["sessions_by_device"] = deviceCounts
// 	stats["sessions_by_hour"] = hourCounts

// 	// Добавляем информацию о распределении по ролям
// 	roleStats := make(map[string]int)
// 	for _, key := range keys {
// 		data, err := s.client.Get(ctx, key).Result()
// 		if err != nil {
// 			continue
// 		}

// 		var sessionData SessionData
// 		if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
// 			continue
// 		}

// 		roleStats[sessionData.Role]++
// 	}
// 	stats["sessions_by_role"] = roleStats

// 	return stats, nil
// }

// // ExtendSession продлевает срок действия сессии
// func (s *SessionStore) ExtendSession(token string, extension time.Duration) error {
// 	sessionData, err := s.GetSession(token)
// 	if err != nil {
// 		return err
// 	}
// 	if sessionData == nil {
// 		return fmt.Errorf("session not found")
// 	}

// 	// Обновляем время истечения
// 	sessionData.ExpiresAt = sessionData.ExpiresAt.Add(extension)

// 	// Сохраняем обновленные данные
// 	return s.saveSession(token, sessionData)
// }

// // IsSessionValid проверяет валидность сессии
// func (s *SessionStore) IsSessionValid(token string) (bool, error) {
// 	sessionData, err := s.GetSession(token)
// 	if err != nil {
// 		return false, err
// 	}

// 	if sessionData == nil {
// 		return false, nil
// 	}

// 	// Проверяем, не отозвана ли сессия
// 	if revoked, ok := sessionData.DeviceInfo["revoked"].(bool); ok && revoked {
// 		return false, nil
// 	}

// 	return true, nil
// }

// // MigrateSession переносит сессию на нового пользователя
// func (s *SessionStore) MigrateSession(token string, newUserID int) error {
// 	sessionData, err := s.GetSession(token)
// 	if err != nil {
// 		return err
// 	}
// 	if sessionData == nil {
// 		return fmt.Errorf("session not found")
// 	}

// 	// Удаляем старую связь
// 	s.unlinkUserFromSession(sessionData.UserID, token)

// 	// Обновляем ID пользователя
// 	sessionData.UserID = newUserID

// 	// Сохраняем обновленную сессию
// 	if err := s.saveSession(token, sessionData); err != nil {
// 		return err
// 	}

// 	// Создаем новую связь
// 	return s.linkUserToSession(newUserID, token)
// }

// // LogSessionActivity логирует активность сессии
// func (s *SessionStore) LogSessionActivity(token, activityType string, details map[string]interface{}) error {
// 	ctx := context.Background()
// 	logKey := s.getSessionLogKey(token)

// 	logEntry := map[string]interface{}{
// 		"type":      activityType,
// 		"timestamp": time.Now().Format(time.RFC3339),
// 		"details":   details,
// 	}

// 	data, err := json.Marshal(logEntry)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal log entry: %w", err)
// 	}

// 	// Сохраняем лог (ограничиваем 100 последних записей)
// 	if err := s.client.LPush(ctx, logKey, data).Err(); err != nil {
// 		return fmt.Errorf("failed to log activity: %w", err)
// 	}

// 	// Ограничиваем размер списка
// 	if err := s.client.LTrim(ctx, logKey, 0, 99).Err(); err != nil {
// 		return fmt.Errorf("failed to trim log list: %w", err)
// 	}

// 	// Устанавливаем TTL для логов (7 дней)
// 	if err := s.client.Expire(ctx, logKey, 7*24*time.Hour).Err(); err != nil {
// 		return fmt.Errorf("failed to set log TTL: %w", err)
// 	}

// 	return nil
// }

// // GetSessionActivityLog возвращает лог активности сессии
// func (s *SessionStore) GetSessionActivityLog(token string, limit int) ([]map[string]interface{}, error) {
// 	ctx := context.Background()
// 	logKey := s.getSessionLogKey(token)

// 	// Получаем логи
// 	logs, err := s.client.LRange(ctx, logKey, 0, int64(limit-1)).Result()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get session logs: %w", err)
// 	}

// 	var activities []map[string]interface{}
// 	for _, log := range logs {
// 		var activity map[string]interface{}
// 		if err := json.Unmarshal([]byte(log), &activity); err != nil {
// 			continue
// 		}
// 		activities = append(activities, activity)
// 	}

// 	return activities, nil
// }

// // Вспомогательные методы

// // generateToken генерирует случайный токен сессии
// func (s *SessionStore) generateToken() (string, error) {
// 	b := make([]byte, 32)
// 	if _, err := rand.Read(b); err != nil {
// 		return "", err
// 	}
// 	return base64.URLEncoding.EncodeToString(b), nil
// }

// // getSessionKey возвращает ключ Redis для сессии
// func (s *SessionStore) getSessionKey(token string) string {
// 	return s.prefix + token
// }

// // getUserSessionsKey возвращает ключ Redis для множества сессий пользователя
// func (s *SessionStore) getUserSessionsKey(userID int) string {
// 	return fmt.Sprintf("user:%d:sessions", userID)
// }

// // getSessionLogKey возвращает ключ Redis для логов сессии
// func (s *SessionStore) getSessionLogKey(token string) string {
// 	return s.prefix + token + ":logs"
// }

// // saveSession сохраняет сессию в Redis
// func (s *SessionStore) saveSession(token string, data *SessionData) error {
// 	ctx := context.Background()
// 	key := s.getSessionKey(token)

// 	// Кодируем данные в JSON
// 	jsonData, err := json.Marshal(data)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal session data: %w", err)
// 	}

// 	// Сохраняем с TTL
// 	ttl := time.Until(data.ExpiresAt)
// 	if err := s.client.SetEX(ctx, key, jsonData, ttl).Err(); err != nil {
// 		return fmt.Errorf("failed to save session to redis: %w", err)
// 	}

// 	return nil
// }

// // linkUserToSession связывает пользователя с токеном сессии
// func (s *SessionStore) linkUserToSession(userID int, token string) error {
// 	ctx := context.Background()
// 	userSessionsKey := s.getUserSessionsKey(userID)

// 	// Добавляем токен в множество сессий пользователя
// 	if err := s.client.SAdd(ctx, userSessionsKey, token).Err(); err != nil {
// 		return fmt.Errorf("failed to link user to session: %w", err)
// 	}

// 	// Устанавливаем TTL для множества сессий (немного больше TTL сессии)
// 	if err := s.client.Expire(ctx, userSessionsKey, s.sessionTTL+time.Hour).Err(); err != nil {
// 		return fmt.Errorf("failed to set user sessions TTL: %w", err)
// 	}

// 	return nil
// }

// // unlinkUserFromSession удаляет связь пользователя с токеном сессии
// func (s *SessionStore) unlinkUserFromSession(userID int, token string) error {
// 	ctx := context.Background()
// 	userSessionsKey := s.getUserSessionsKey(userID)

// 	// Удаляем токен из множества сессий пользователя
// 	if err := s.client.SRem(ctx, userSessionsKey, token).Err(); err != nil {
// 		return fmt.Errorf("failed to unlink user from session: %w", err)
// 	}

// 	return nil
// }

// // getUserPermissions возвращает разрешения пользователя
// func (s *SessionStore) getUserPermissions(user *users.User) []string {
// 	permissions := []string{}

// 	switch user.Role {
// 	case "admin":
// 		permissions = []string{"read", "write", "delete", "admin", "manage_users"}
// 	case "premium":
// 		permissions = []string{"read", "write", "premium_features"}
// 	case "user":
// 		permissions = []string{"read", "basic_write"}
// 	}

// 	return permissions
// }

// // extractUserSettings извлекает настройки пользователя
// func (s *SessionStore) extractUserSettings(user *users.User) map[string]interface{} {
// 	return map[string]interface{}{
// 		"notifications": map[string]bool{
// 			"enabled":    user.Notifications.Enabled,
// 			"growth":     user.Notifications.Growth,
// 			"fall":       user.Notifications.Fall,
// 			"continuous": user.Notifications.Continuous,
// 		},
// 		"thresholds": map[string]float64{
// 			"min_growth": user.Settings.MinGrowthThreshold,
// 			"min_fall":   user.Settings.MinFallThreshold,
// 		},
// 		"display": map[string]interface{}{
// 			"mode":     user.DisplayMode,
// 			"language": user.Language,
// 			"timezone": user.Timezone,
// 		},
// 	}
// }

// // SessionManager высокоуровневый менеджер сессий
// type SessionManager struct {
// 	store         *SessionStore
// 	postgresStore interface{} // Опционально: PostgreSQL репозиторий
// }

// // NewSessionManager создает новый менеджер сессий
// func NewSessionManager(redisClient *redis.Client) *SessionManager {
// 	return &SessionManager{
// 		store: NewSessionStore(redisClient),
// 	}
// }

// // SetPostgresStore устанавливает PostgreSQL хранилище для синхронизации
// func (sm *SessionManager) SetPostgresStore(postgresStore interface{}) {
// 	sm.postgresStore = postgresStore
// }

// // Create создает сессию и синхронизирует с PostgreSQL если нужно
// func (sm *SessionManager) Create(user *users.User, deviceInfo map[string]interface{}, ip, userAgent string) (string, *SessionData, error) {
// 	token, sessionData, err := sm.store.CreateSession(user, deviceInfo, ip, userAgent)
// 	if err != nil {
// 		return "", nil, err
// 	}

// 	// Синхронизация с PostgreSQL если настроено
// 	if sm.postgresStore != nil {
// 		// Здесь можно сохранить сессию в PostgreSQL для долговременного хранения
// 		// Например: sm.postgresStore.CreateSession(sessionData)
// 	}

// 	return token, sessionData, nil
// }

// // Validate проверяет сессию и обновляет активность
// func (sm *SessionManager) Validate(token string) (*SessionData, error) {
// 	// Получаем сессию
// 	sessionData, err := sm.store.GetSession(token)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if sessionData == nil {
// 		return nil, nil
// 	}

// 	// Обновляем активность
// 	if err := sm.store.UpdateSessionActivity(token); err != nil {
// 		// Не прерываем валидацию при ошибке обновления активности
// 		// Просто логируем
// 		fmt.Printf("Failed to update session activity: %v\n", err)
// 	}

// 	return sessionData, nil
// }

// // Destroy удаляет сессию со всех хранилищ
// func (sm *SessionManager) Destroy(token string) error {
// 	// Удаляем из Redis
// 	if err := sm.store.DeleteSession(token); err != nil {
// 		return err
// 	}

// 	// Удаляем из PostgreSQL если настроено
// 	if sm.postgresStore != nil {
// 		// Например: sm.postgresStore.DeleteSession(token)
// 	}

// 	return nil
// }

// // StartCleanupWorker запускает воркер для очистки устаревших сессий
// func (sm *SessionManager) StartCleanupWorker() {
// 	go func() {
// 		ticker := time.NewTicker(time.Hour)
// 		defer ticker.Stop()

// 		for range ticker.C {
// 			// Очищаем истекшие сессии
// 			deleted, err := sm.store.CleanupExpiredSessions()
// 			if err != nil {
// 				fmt.Printf("Failed to cleanup expired sessions: %v\n", err)
// 			} else if deleted > 0 {
// 				fmt.Printf("Cleaned up %d expired sessions\n", deleted)
// 			}

// 			// Очищаем неактивные сессии (не было активности 7 дней)
// 			inactiveDeleted, err := sm.store.CleanupInactiveSessions(7 * 24 * time.Hour)
// 			if err != nil {
// 				fmt.Printf("Failed to cleanup inactive sessions: %v\n", err)
// 			} else if inactiveDeleted > 0 {
// 				fmt.Printf("Cleaned up %d inactive sessions\n", inactiveDeleted)
// 			}
// 		}
// 	}()
// }
