// internal/delivery/telegram/app/bot/message_sender/rate_limiter.go
package message_sender

import (
	"sync"
	"time"
)

// RateLimiter ограничитель частоты отправки
type RateLimiter struct {
	interval time.Duration
	lastSend time.Time
	mu       sync.Mutex
}

// NewRateLimiter создает новый ограничитель
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		interval: interval,
		lastSend: time.Now().Add(-interval), // Можно отправлять сразу
	}
}

// CanSend проверяет, можно ли отправлять сообщение
func (rl *RateLimiter) CanSend() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastSend) < rl.interval {
		return false
	}

	rl.lastSend = now
	return true
}
