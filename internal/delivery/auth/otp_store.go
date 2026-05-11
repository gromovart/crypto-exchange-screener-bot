// internal/delivery/auth/otp_store.go
package auth

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

const (
	maxAttempts = 3               // максимум попыток верификации
	ratePeriod  = 5 * time.Minute // окно для rate limiting запросов OTP
	maxOTPReqs  = 3               // максимум запросов OTP за ratePeriod
)

type otpEntry struct {
	code         string
	expiresAt    time.Time
	attempts     int    // неудачные попытки верификации
	messageID    int64  // ID сообщения в Telegram (int64)
	messageIDStr string // ID сообщения в MAX (string)
	otpCode      string // сохранённый OTP-код для поиска копий-сообщений при удалении
}

type rateEntry struct {
	count     int
	windowEnd time.Time
}

// OTPStore — потокобезопасное in-memory хранилище OTP
type OTPStore struct {
	mu      sync.Mutex
	otps    map[int64]*otpEntry  // max_user_id → OTP
	rate    map[int64]*rateEntry // max_user_id → rate limit
	ttl     time.Duration
}

// NewOTPStore создаёт хранилище с заданным TTL для кодов
func NewOTPStore(ttl time.Duration) *OTPStore {
	s := &OTPStore{
		otps: make(map[int64]*otpEntry),
		rate: make(map[int64]*rateEntry),
		ttl:  ttl,
	}
	go s.cleanupLoop()
	return s
}

// Generate генерирует и сохраняет новый OTP для пользователя.
// Возвращает ошибку если превышен rate limit.
func (s *OTPStore) Generate(maxUserID int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Rate limit: не более maxOTPReqs запросов за ratePeriod
	now := time.Now()
	r := s.rate[maxUserID]
	if r == nil || now.After(r.windowEnd) {
		r = &rateEntry{count: 0, windowEnd: now.Add(ratePeriod)}
		s.rate[maxUserID] = r
	}
	if r.count >= maxOTPReqs {
		remaining := r.windowEnd.Sub(now).Truncate(time.Second)
		return "", fmt.Errorf("слишком много запросов, повторите через %s", remaining)
	}
	r.count++

	code := generateCode()
	s.otps[maxUserID] = &otpEntry{
		code:      code,
		expiresAt: now.Add(s.ttl),
		otpCode:   code,
	}
	return code, nil
}

// Verify проверяет OTP. Возвращает true при совпадении.
// После maxAttempts неудачных попыток код инвалидируется.
func (s *OTPStore) Verify(maxUserID int64, code string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.otps[maxUserID]
	if !ok {
		return false, fmt.Errorf("код не найден или уже использован")
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.otps, maxUserID)
		return false, fmt.Errorf("код истёк")
	}
	if entry.code != code {
		entry.attempts++
		if entry.attempts >= maxAttempts {
			delete(s.otps, maxUserID)
			return false, fmt.Errorf("превышено количество попыток")
		}
		return false, fmt.Errorf("неверный код, осталось попыток: %d", maxAttempts-entry.attempts)
	}

	// Успех — удаляем использованный код
	delete(s.otps, maxUserID)
	return true, nil
}

// SetMessageID сохраняет ID сообщения с кодом для последующего удаления.
func (s *OTPStore) SetMessageID(userID int64, msgID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.otps[userID]; ok {
		e.messageID = msgID
	}
}

// GetMessageID возвращает ID сообщения с кодом (0 если не задан).
func (s *OTPStore) GetMessageID(userID int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.otps[userID]; ok {
		return e.messageID
	}
	return 0
}

// SetMessageIDStr сохраняет строковый ID сообщения (MAX) для последующего удаления.
func (s *OTPStore) SetMessageIDStr(userID int64, msgID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.otps[userID]; ok {
		e.messageIDStr = msgID
	}
}

// GetMessageIDStr возвращает строковый ID сообщения (MAX) ("" если не задан).
func (s *OTPStore) GetMessageIDStr(userID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.otps[userID]; ok {
		return e.messageIDStr
	}
	return ""
}

// GetOTPCode возвращает сохранённый OTP-код ("" если запись не найдена).
func (s *OTPStore) GetOTPCode(userID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.otps[userID]; ok {
		return e.otpCode
	}
	return ""
}

// Invalidate принудительно удаляет OTP для пользователя (например при смене кода)
func (s *OTPStore) Invalidate(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.otps, userID)
}

// generateCode генерирует 6-значный цифровой код
func generateCode() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n)
}

// cleanupLoop периодически удаляет просроченные записи
func (s *OTPStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, e := range s.otps {
			if now.After(e.expiresAt) {
				delete(s.otps, id)
			}
		}
		for id, r := range s.rate {
			if now.After(r.windowEnd) {
				delete(s.rate, id)
			}
		}
		s.mu.Unlock()
	}
}
