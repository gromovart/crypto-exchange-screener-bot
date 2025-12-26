// persistence/postgres/repository/api_key_repository.go
package api_key

// import (
// 	"context"
// 	"crypto/aes"
// 	"crypto/cipher"
// 	"crypto/rand"
// 	"database/sql"
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"time"

// 	"github.com/jmoiron/sqlx"
// )

// // APIKeyRepository управляет API ключами пользователей
// type APIKeyRepository struct {
// 	db            *sqlx.DB
// 	encryptionKey string
// }

// // NewAPIKeyRepository создает новый репозиторий API ключей
// func NewAPIKeyRepository(db *sqlx.DB, encryptionKey string) *APIKeyRepository {
// 	return &APIKeyRepository{
// 		db:            db,
// 		encryptionKey: encryptionKey,
// 	}
// }

// // APIKey структура для хранения API ключа
// type APIKey struct {
// 	ID          int       `db:"id"`
// 	UserID      int       `db:"user_id"`
// 	Exchange    string    `db:"exchange"`
// 	Label       string    `db:"label"`
// 	Permissions JSONMap   `db:"permissions"`
// 	IsActive    bool      `db:"is_active"`
// 	LastUsedAt  time.Time `db:"last_used_at"`
// 	ExpiresAt   time.Time `db:"expires_at"`
// 	CreatedAt   time.Time `db:"created_at"`
// 	UpdatedAt   time.Time `db:"updated_at"`

// 	// Зашифрованные поля (не экспортируются)
// 	apiKeyEncrypted    string `db:"api_key_encrypted"`
// 	apiSecretEncrypted string `db:"api_secret_encrypted"`
// }

// // APIKeyWithSecrets структура с расшифрованными ключами
// type APIKeyWithSecrets struct {
// 	APIKey
// 	APIKeyPlain    string `json:"api_key"`
// 	APISecretPlain string `json:"api_secret"`
// }

// // JSONMap для работы с JSON полями
// type JSONMap map[string]interface{}

// // Create создает новый API ключ
// func (r *APIKeyRepository) Create(key *APIKeyWithSecrets) error {
// 	// Шифруем ключи
// 	encryptedKey, err := r.encrypt(key.APIKeyPlain)
// 	if err != nil {
// 		return fmt.Errorf("failed to encrypt api key: %w", err)
// 	}

// 	encryptedSecret, err := r.encrypt(key.APISecretPlain)
// 	if err != nil {
// 		return fmt.Errorf("failed to encrypt api secret: %w", err)
// 	}

// 	query := `
//     INSERT INTO user_api_keys (
//         user_id, exchange, api_key_encrypted, api_secret_encrypted,
//         label, permissions, expires_at
//     ) VALUES ($1, $2, $3, $4, $5, $6, $7)
//     RETURNING id, created_at, updated_at
//     `

// 	permissionsJSON, _ := json.Marshal(key.Permissions)

// 	return r.db.QueryRow(
// 		query,
// 		key.UserID,
// 		key.Exchange,
// 		encryptedKey,
// 		encryptedSecret,
// 		key.Label,
// 		permissionsJSON,
// 		key.ExpiresAt,
// 	).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)
// }

// // FindByID находит API ключ по ID
// func (r *APIKeyRepository) FindByID(id int) (*APIKey, error) {
// 	query := `
//     SELECT
//         id, user_id, exchange, api_key_encrypted, api_secret_encrypted,
//         label, permissions, is_active, last_used_at, expires_at,
//         created_at, updated_at
//     FROM user_api_keys
//     WHERE id = $1
//     `

// 	var key APIKey
// 	var permissionsJSON []byte

// 	err := r.db.QueryRow(query, id).Scan(
// 		&key.ID,
// 		&key.UserID,
// 		&key.Exchange,
// 		&key.apiKeyEncrypted,
// 		&key.apiSecretEncrypted,
// 		&key.Label,
// 		&permissionsJSON,
// 		&key.IsActive,
// 		&key.LastUsedAt,
// 		&key.ExpiresAt,
// 		&key.CreatedAt,
// 		&key.UpdatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON с разрешениями
// 	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
// 	}

// 	return &key, nil
// }

// // FindWithSecrets находит API ключ с расшифрованными секретами
// func (r *APIKeyRepository) FindWithSecrets(id int) (*APIKeyWithSecrets, error) {
// 	key, err := r.FindByID(id)
// 	if err != nil || key == nil {
// 		return nil, err
// 	}

// 	// Расшифровываем ключи
// 	apiKeyPlain, err := r.decrypt(key.apiKeyEncrypted)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decrypt api key: %w", err)
// 	}

// 	apiSecretPlain, err := r.decrypt(key.apiSecretEncrypted)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decrypt api secret: %w", err)
// 	}

// 	return &APIKeyWithSecrets{
// 		APIKey:         *key,
// 		APIKeyPlain:    apiKeyPlain,
// 		APISecretPlain: apiSecretPlain,
// 	}, nil
// }

// // FindByUserIDExchange находит активный API ключ пользователя для биржи
// func (r *APIKeyRepository) FindByUserIDExchange(userID int, exchange string) (*APIKey, error) {
// 	query := `
//     SELECT
//         id, user_id, exchange, api_key_encrypted, api_secret_encrypted,
//         label, permissions, is_active, last_used_at, expires_at,
//         created_at, updated_at
//     FROM user_api_keys
//     WHERE user_id = $1
//       AND exchange = $2
//       AND is_active = TRUE
//       AND (expires_at IS NULL OR expires_at > NOW())
//     `

// 	var key APIKey
// 	var permissionsJSON []byte

// 	err := r.db.QueryRow(query, userID, exchange).Scan(
// 		&key.ID,
// 		&key.UserID,
// 		&key.Exchange,
// 		&key.apiKeyEncrypted,
// 		&key.apiSecretEncrypted,
// 		&key.Label,
// 		&permissionsJSON,
// 		&key.IsActive,
// 		&key.LastUsedAt,
// 		&key.ExpiresAt,
// 		&key.CreatedAt,
// 		&key.UpdatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON с разрешениями
// 	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
// 	}

// 	return &key, nil
// }

// // FindByUserID находит все API ключи пользователя
// func (r *APIKeyRepository) FindByUserID(userID int) ([]*APIKey, error) {
// 	query := `
//     SELECT
//         id, user_id, exchange, api_key_encrypted, api_secret_encrypted,
//         label, permissions, is_active, last_used_at, expires_at,
//         created_at, updated_at
//     FROM user_api_keys
//     WHERE user_id = $1
//     ORDER BY created_at DESC
//     `

// 	rows, err := r.db.Query(query, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var keys []*APIKey

// 	for rows.Next() {
// 		var key APIKey
// 		var permissionsJSON []byte

// 		err := rows.Scan(
// 			&key.ID,
// 			&key.UserID,
// 			&key.Exchange,
// 			&key.apiKeyEncrypted,
// 			&key.apiSecretEncrypted,
// 			&key.Label,
// 			&permissionsJSON,
// 			&key.IsActive,
// 			&key.LastUsedAt,
// 			&key.ExpiresAt,
// 			&key.CreatedAt,
// 			&key.UpdatedAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON с разрешениями
// 		if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
// 		}

// 		keys = append(keys, &key)
// 	}

// 	return keys, nil
// }

// // UpdateActivity обновляет время последнего использования
// func (r *APIKeyRepository) UpdateActivity(id int) error {
// 	query := `
//     UPDATE user_api_keys
//     SET last_used_at = NOW(),
//         updated_at = NOW()
//     WHERE id = $1
//     `

// 	result, err := r.db.Exec(query, id)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // Deactivate деактивирует API ключ
// func (r *APIKeyRepository) Deactivate(id int) error {
// 	query := `
//     UPDATE user_api_keys
//     SET is_active = FALSE,
//         updated_at = NOW()
//     WHERE id = $1
//     `

// 	result, err := r.db.Exec(query, id)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // Delete удаляет API ключ
// func (r *APIKeyRepository) Delete(id int) error {
// 	query := `DELETE FROM user_api_keys WHERE id = $1`

// 	result, err := r.db.Exec(query, id)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // LogUsage логирует использование API ключа
// func (r *APIKeyRepository) LogUsage(log *APIKeyUsageLog) error {
// 	query := `
//     INSERT INTO api_key_usage_logs (
//         api_key_id, action, endpoint, request_body,
//         response_status, response_body, ip_address,
//         user_agent, latency_ms, error_message
//     ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
//     `

// 	requestBodyJSON, _ := json.Marshal(log.RequestBody)
// 	responseBodyJSON, _ := json.Marshal(log.ResponseBody)

// 	_, err := r.db.Exec(
// 		query,
// 		log.APIKeyID,
// 		log.Action,
// 		log.Endpoint,
// 		requestBodyJSON,
// 		log.ResponseStatus,
// 		responseBodyJSON,
// 		log.IPAddress,
// 		log.UserAgent,
// 		log.LatencyMs,
// 		log.ErrorMessage,
// 	)

// 	return err
// }

// // GetUsageStats возвращает статистику использования
// func (r *APIKeyRepository) GetUsageStats(ctx context.Context, apiKeyID int, days int) (map[string]interface{}, error) {
// 	query := `
//     SELECT
//         COUNT(*) as total_requests,
//         COUNT(CASE WHEN response_status >= 400 OR error_message IS NOT NULL THEN 1 END) as error_count,
//         AVG(latency_ms) as avg_latency,
//         MIN(created_at) as first_request,
//         MAX(created_at) as last_request
//     FROM api_key_usage_logs
//     WHERE api_key_id = $1
//       AND created_at >= NOW() - INTERVAL '1 day' * $2
//     `

// 	stats := make(map[string]interface{})

// 	// Используем временные переменные для сканирования
// 	var totalRequests, errorCount int64
// 	var avgLatency sql.NullFloat64
// 	var firstRequest, lastRequest sql.NullTime

// 	err := r.db.QueryRowContext(ctx, query, apiKeyID, days).Scan(
// 		&totalRequests,
// 		&errorCount,
// 		&avgLatency,
// 		&firstRequest,
// 		&lastRequest,
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	// Заполняем карту
// 	stats["total_requests"] = totalRequests
// 	stats["error_count"] = errorCount
// 	stats["avg_latency"] = avgLatency.Float64
// 	stats["first_request"] = firstRequest.Time
// 	stats["last_request"] = lastRequest.Time

// 	// Рассчитываем rate limit
// 	rateLimitQuery := `
//     SELECT
//         COUNT(*) as requests_last_hour,
//         COUNT(DISTINCT ip_address) as unique_ips
//     FROM api_key_usage_logs
//     WHERE api_key_id = $1
//       AND created_at >= NOW() - INTERVAL '1 hour'
//     `

// 	var requestsLastHour, uniqueIPs int64

// 	err = r.db.QueryRowContext(ctx, rateLimitQuery, apiKeyID).Scan(
// 		&requestsLastHour,
// 		&uniqueIPs,
// 	)

// 	if err != nil && err != sql.ErrNoRows {
// 		return nil, err
// 	}

// 	// Если есть ошибка "no rows", устанавливаем нулевые значения
// 	if err == sql.ErrNoRows {
// 		requestsLastHour = 0
// 		uniqueIPs = 0
// 	}

// 	stats["requests_last_hour"] = requestsLastHour
// 	stats["unique_ips"] = uniqueIPs

// 	return stats, nil
// }

// // CheckPermission проверяет разрешение для API ключа
// func (r *APIKeyRepository) CheckPermission(apiKeyID int, permission string) (bool, error) {
// 	query := `
//     SELECT check_api_key_permission($1, $2)
//     `

// 	var hasPermission bool
// 	err := r.db.QueryRow(query, apiKeyID, permission).Scan(&hasPermission)
// 	if err != nil {
// 		return false, err
// 	}

// 	return hasPermission, nil
// }

// // RotateKey выполняет ротацию API ключа
// func (r *APIKeyRepository) RotateKey(userID int, exchange, newAPIKey, newAPISecret, reason string) (int, error) {
// 	query := `
//     SELECT rotate_api_key($1, $2, $3, $4, $5)
//     `

// 	var newKeyID int
// 	err := r.db.QueryRow(query, userID, exchange, newAPIKey, newAPISecret, reason).Scan(&newKeyID)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return newKeyID, nil
// }

// // Вспомогательные методы для шифрования

// func (r *APIKeyRepository) encrypt(plaintext string) (string, error) {
// 	block, err := aes.NewCipher([]byte(r.encryptionKey))
// 	if err != nil {
// 		return "", err
// 	}

// 	gcm, err := cipher.NewGCM(block)
// 	if err != nil {
// 		return "", err
// 	}

// 	nonce := make([]byte, gcm.NonceSize())
// 	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
// 		return "", err
// 	}

// 	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
// 	return base64.StdEncoding.EncodeToString(ciphertext), nil
// }

// func (r *APIKeyRepository) decrypt(encrypted string) (string, error) {
// 	data, err := base64.StdEncoding.DecodeString(encrypted)
// 	if err != nil {
// 		return "", err
// 	}

// 	block, err := aes.NewCipher([]byte(r.encryptionKey))
// 	if err != nil {
// 		return "", err
// 	}

// 	gcm, err := cipher.NewGCM(block)
// 	if err != nil {
// 		return "", err
// 	}

// 	nonceSize := gcm.NonceSize()
// 	if len(data) < nonceSize {
// 		return "", fmt.Errorf("ciphertext too short")
// 	}

// 	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
// 	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	return string(plaintext), nil
// }

// // Структуры для логов
// type APIKeyUsageLog struct {
// 	APIKeyID       int                    `db:"api_key_id"`
// 	Action         string                 `db:"action"`
// 	Endpoint       string                 `db:"endpoint"`
// 	RequestBody    map[string]interface{} `db:"request_body"`
// 	ResponseStatus int                    `db:"response_status"`
// 	ResponseBody   map[string]interface{} `db:"response_body"`
// 	IPAddress      string                 `db:"ip_address"`
// 	UserAgent      string                 `db:"user_agent"`
// 	LatencyMs      int                    `db:"latency_ms"`
// 	ErrorMessage   string                 `db:"error_message"`
// 	CreatedAt      time.Time              `db:"created_at"`
// }
