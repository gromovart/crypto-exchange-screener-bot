// internal/infrastructure/http/tbank/token.go
package tbank

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// GenerateToken генерирует подпись запроса по алгоритму Т-Банк:
// 1. Собираем параметры + пароль
// 2. Сортируем по ключу алфавитно
// 3. Конкатенируем только значения
// 4. SHA-256
func GenerateToken(params map[string]string, password string) string {
	p := make(map[string]string, len(params)+1)
	for k, v := range params {
		p[k] = v
	}
	p["Password"] = password

	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(p[k])
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

// VerifyToken проверяет токен входящего уведомления от Т-Банк
func VerifyToken(params map[string]string, password string) bool {
	received := params["Token"]
	if received == "" {
		return false
	}

	// Копируем параметры без Token и вложенных объектов (Data, Receipt)
	clean := make(map[string]string, len(params))
	for k, v := range params {
		if k != "Token" {
			clean[k] = v
		}
	}

	expected := GenerateToken(clean, password)
	if !strings.EqualFold(received, expected) {
		// Временный debug-лог для диагностики
		keys := make([]string, 0, len(clean))
		for k := range clean {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(k + "=" + clean[k] + " ")
		}
		println("[DEBUG TOKEN] params:", sb.String())
		println("[DEBUG TOKEN] received:", received)
		println("[DEBUG TOKEN] expected:", expected)
		return false
	}
	return true
}
