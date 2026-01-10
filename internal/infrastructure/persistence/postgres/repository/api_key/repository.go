// internal/infrastructure/persistence/postgres/repository/api_key/repository.go
package api_key

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/jmoiron/sqlx"
)

// APIKeyRepository интерфейс для работы с API ключами
type APIKeyRepository interface {
	// Основные методы
	Create(key *models.APIKeyWithSecrets) error
	FindByID(id int) (*models.APIKey, error)
	FindWithSecrets(id int) (*models.APIKeyWithSecrets, error)
	FindByUserIDExchange(userID int, exchange string) (*models.APIKey, error)
	FindByUserID(userID int) ([]*models.APIKey, error)
	Update(key *models.APIKey) error
	UpdateActivity(id int) error
	Deactivate(id int) error
	Delete(id int) error

	// Логирование и статистика
	LogUsage(log *models.APIKeyUsageLog) error
	GetUsageStats(ctx context.Context, apiKeyID int, days int) (map[string]interface{}, error)

	// Дополнительные методы
	CheckPermission(apiKeyID int, permission string) (bool, error)
	RotateKey(userID int, exchange, newAPIKey, newAPISecret, reason string) (int, error)
}

// JSONMap для работы с JSON полями
type JSONMap map[string]interface{}

// APIKeyRepositoryImpl реализация репозитория API ключей
type APIKeyRepositoryImpl struct {
	db            *sqlx.DB
	encryptionKey string
}

// NewAPIKeyRepository создает новый репозиторий API ключей
func NewAPIKeyRepository(db *sqlx.DB, encryptionKey string) *APIKeyRepositoryImpl {
	return &APIKeyRepositoryImpl{
		db:            db,
		encryptionKey: encryptionKey,
	}
}

// Create создает новый API ключ
func (r *APIKeyRepositoryImpl) Create(key *models.APIKeyWithSecrets) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Шифруем ключи
	encryptedKey, err := r.encrypt(key.APIKeyPlain)
	if err != nil {
		return fmt.Errorf("failed to encrypt api key: %w", err)
	}

	encryptedSecret, err := r.encrypt(key.APISecretPlain)
	if err != nil {
		return fmt.Errorf("failed to encrypt api secret: %w", err)
	}

	query := `
    INSERT INTO user_api_keys (
        user_id, exchange, api_key_encrypted, api_secret_encrypted,
        label, permissions, expires_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id, created_at, updated_at
    `

	permissionsJSON, _ := json.Marshal(key.Permissions)

	err = tx.QueryRow(
		query,
		key.UserID,
		key.Exchange,
		encryptedKey,
		encryptedSecret,
		key.Label,
		permissionsJSON,
		key.ExpiresAt,
	).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create api key: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID находит API ключ по ID
func (r *APIKeyRepositoryImpl) FindByID(id int) (*models.APIKey, error) {
	query := `
    SELECT
        id, user_id, exchange,
        label, permissions, is_active, last_used_at, expires_at,
        created_at, updated_at
    FROM user_api_keys
    WHERE id = $1
    `

	var key models.APIKey
	var permissionsJSON []byte
	var lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&key.ID,
		&key.UserID,
		&key.Exchange,
		&key.Label,
		&permissionsJSON,
		&key.IsActive,
		&lastUsedAt,
		&key.ExpiresAt,
		&key.CreatedAt,
		&key.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Устанавливаем время последнего использования
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	} else {
		key.LastUsedAt = nil
	}

	// Декодируем JSON с разрешениями
	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	return &key, nil
}

// FindWithSecrets находит API ключ с расшифрованными секретами
func (r *APIKeyRepositoryImpl) FindWithSecrets(id int) (*models.APIKeyWithSecrets, error) {
	query := `
    SELECT
        id, user_id, exchange, api_key_encrypted, api_secret_encrypted,
        label, permissions, is_active, last_used_at, expires_at,
        created_at, updated_at
    FROM user_api_keys
    WHERE id = $1
    `

	var key models.APIKey
	var encryptedKey, encryptedSecret string
	var permissionsJSON []byte
	var lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&key.ID,
		&key.UserID,
		&key.Exchange,
		&encryptedKey,
		&encryptedSecret,
		&key.Label,
		&permissionsJSON,
		&key.IsActive,
		&lastUsedAt,
		&key.ExpiresAt,
		&key.CreatedAt,
		&key.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Устанавливаем время последнего использования
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	} else {
		key.LastUsedAt = nil
	}

	// Декодируем JSON с разрешениями
	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	// Расшифровываем ключи
	apiKeyPlain, err := r.decrypt(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt api key: %w", err)
	}

	apiSecretPlain, err := r.decrypt(encryptedSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt api secret: %w", err)
	}

	return &models.APIKeyWithSecrets{
		APIKey:         key,
		APIKeyPlain:    apiKeyPlain,
		APISecretPlain: apiSecretPlain,
	}, nil
}

// FindByUserIDExchange находит активный API ключ пользователя для биржи
func (r *APIKeyRepositoryImpl) FindByUserIDExchange(userID int, exchange string) (*models.APIKey, error) {
	query := `
    SELECT
        id, user_id, exchange,
        label, permissions, is_active, last_used_at, expires_at,
        created_at, updated_at
    FROM user_api_keys
    WHERE user_id = $1
      AND exchange = $2
      AND is_active = TRUE
      AND (expires_at IS NULL OR expires_at > NOW())
    `

	var key models.APIKey
	var permissionsJSON []byte
	var lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, userID, exchange).Scan(
		&key.ID,
		&key.UserID,
		&key.Exchange,
		&key.Label,
		&permissionsJSON,
		&key.IsActive,
		&lastUsedAt,
		&key.ExpiresAt,
		&key.CreatedAt,
		&key.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Устанавливаем время последнего использования
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	} else {
		key.LastUsedAt = nil
	}
	// Декодируем JSON с разрешениями
	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	return &key, nil
}

// FindByUserID находит все API ключи пользователя
func (r *APIKeyRepositoryImpl) FindByUserID(userID int) ([]*models.APIKey, error) {
	query := `
    SELECT
        id, user_id, exchange,
        label, permissions, is_active, last_used_at, expires_at,
        created_at, updated_at
    FROM user_api_keys
    WHERE user_id = $1
    ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*models.APIKey

	for rows.Next() {
		var key models.APIKey
		var permissionsJSON []byte
		var lastUsedAt sql.NullTime

		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Exchange,
			&key.Label,
			&permissionsJSON,
			&key.IsActive,
			&lastUsedAt,
			&key.ExpiresAt,
			&key.CreatedAt,
			&key.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Устанавливаем время последнего использования
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		} else {
			key.LastUsedAt = nil
		}

		// Декодируем JSON с разрешениями
		if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
		}

		keys = append(keys, &key)
	}

	return keys, nil
}

// Update обновляет API ключ
func (r *APIKeyRepositoryImpl) Update(key *models.APIKey) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    UPDATE user_api_keys SET
        exchange = $1,
        label = $2,
        permissions = $3,
        is_active = $4,
        expires_at = $5,
        updated_at = NOW()
    WHERE id = $6 AND user_id = $7
    `

	permissionsJSON, _ := json.Marshal(key.Permissions)

	result, err := tx.Exec(
		query,
		key.Exchange,
		key.Label,
		permissionsJSON,
		key.IsActive,
		key.ExpiresAt,
		key.ID,
		key.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update api key: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateActivity обновляет время последнего использования
func (r *APIKeyRepositoryImpl) UpdateActivity(id int) error {
	query := `
    UPDATE user_api_keys
    SET last_used_at = NOW(),
        updated_at = NOW()
    WHERE id = $1
    `

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Deactivate деактивирует API ключ
func (r *APIKeyRepositoryImpl) Deactivate(id int) error {
	query := `
    UPDATE user_api_keys
    SET is_active = FALSE,
        updated_at = NOW()
    WHERE id = $1
    `

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete удаляет API ключ
func (r *APIKeyRepositoryImpl) Delete(id int) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `DELETE FROM user_api_keys WHERE id = $1`

	result, err := tx.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LogUsage логирует использование API ключа
func (r *APIKeyRepositoryImpl) LogUsage(log *models.APIKeyUsageLog) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    INSERT INTO api_key_usage_logs (
        api_key_id, action, endpoint, request_body,
        response_status, response_body, ip_address,
        user_agent, latency_ms, error_message
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    RETURNING id, created_at
    `

	requestBodyJSON, _ := json.Marshal(log.RequestBody)
	responseBodyJSON, _ := json.Marshal(log.ResponseBody)

	err = tx.QueryRow(
		query,
		log.APIKeyID,
		log.Action,
		log.Endpoint,
		requestBodyJSON,
		log.ResponseStatus,
		responseBodyJSON,
		log.IPAddress,
		log.UserAgent,
		log.LatencyMS,
		log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to log api key usage: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUsageStats возвращает статистику использования
func (r *APIKeyRepositoryImpl) GetUsageStats(ctx context.Context, apiKeyID int, days int) (map[string]interface{}, error) {
	query := `
    SELECT
        COUNT(*) as total_requests,
        COUNT(CASE WHEN response_status >= 400 OR error_message IS NOT NULL THEN 1 END) as error_count,
        AVG(latency_ms) as avg_latency,
        MIN(created_at) as first_request,
        MAX(created_at) as last_request
    FROM api_key_usage_logs
    WHERE api_key_id = $1
      AND created_at >= NOW() - INTERVAL '1 day' * $2
    `

	stats := make(map[string]interface{})

	// Используем временные переменные для сканирования
	var totalRequests, errorCount int64
	var avgLatency sql.NullFloat64
	var firstRequest, lastRequest sql.NullTime

	err := r.db.QueryRowContext(ctx, query, apiKeyID, days).Scan(
		&totalRequests,
		&errorCount,
		&avgLatency,
		&firstRequest,
		&lastRequest,
	)

	if err != nil {
		return nil, err
	}

	// Заполняем карту
	stats["total_requests"] = totalRequests
	stats["error_count"] = errorCount

	if avgLatency.Valid {
		stats["avg_latency"] = avgLatency.Float64
	} else {
		stats["avg_latency"] = 0.0
	}

	if firstRequest.Valid {
		stats["first_request"] = firstRequest.Time
	}

	if lastRequest.Valid {
		stats["last_request"] = lastRequest.Time
	}

	// Рассчитываем rate limit
	rateLimitQuery := `
    SELECT
        COUNT(*) as requests_last_hour,
        COUNT(DISTINCT ip_address) as unique_ips
    FROM api_key_usage_logs
    WHERE api_key_id = $1
      AND created_at >= NOW() - INTERVAL '1 hour'
    `

	var requestsLastHour, uniqueIPs int64

	err = r.db.QueryRowContext(ctx, rateLimitQuery, apiKeyID).Scan(
		&requestsLastHour,
		&uniqueIPs,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Если есть ошибка "no rows", устанавливаем нулевые значения
	if err == sql.ErrNoRows {
		requestsLastHour = 0
		uniqueIPs = 0
	}

	stats["requests_last_hour"] = requestsLastHour
	stats["unique_ips"] = uniqueIPs

	return stats, nil
}

// CheckPermission проверяет разрешение для API ключа
func (r *APIKeyRepositoryImpl) CheckPermission(apiKeyID int, permission string) (bool, error) {
	query := `
    SELECT check_api_key_permission($1, $2)
    `

	var hasPermission bool
	err := r.db.QueryRow(query, apiKeyID, permission).Scan(&hasPermission)
	if err != nil {
		return false, err
	}

	return hasPermission, nil
}

// RotateKey выполняет ротацию API ключа
func (r *APIKeyRepositoryImpl) RotateKey(userID int, exchange, newAPIKey, newAPISecret, reason string) (int, error) {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    SELECT rotate_api_key($1, $2, $3, $4, $5)
    `

	var newKeyID int
	err = tx.QueryRow(query, userID, exchange, newAPIKey, newAPISecret, reason).Scan(&newKeyID)
	if err != nil {
		return 0, err
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newKeyID, nil
}

// Вспомогательные методы для шифрования

func (r *APIKeyRepositoryImpl) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(r.encryptionKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (r *APIKeyRepositoryImpl) decrypt(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(r.encryptionKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Вспомогательная функция для преобразования времени в NullTime
func getNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}
