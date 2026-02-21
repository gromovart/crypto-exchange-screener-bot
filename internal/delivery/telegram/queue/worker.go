// internal/delivery/telegram/queue/worker.go
package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/go-redis/redis/v8"
)

const (
	globalRateLimitKey = "tg:ratelimit:global"
	userRateLimitKeyFmt = "tg:ratelimit:user:%d"
	maxAttempts        = 3
	workerPollInterval = 33 * time.Millisecond // ~30 msg/sec
)

// tokenBucketScript — атомарный Token Bucket через Lua
// Capacity: 30 токенов, refill: 30 токенов/сек
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = 30
local refill_rate = 30
local now = tonumber(ARGV[1])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

local elapsed = now - last_refill
local new_tokens = math.min(capacity, tokens + elapsed * refill_rate)

if new_tokens >= 1 then
    redis.call('HMSET', key, 'tokens', new_tokens - 1, 'last_refill', now)
    redis.call('EXPIRE', key, 60)
    return 1
else
    return 0
end
`)

// telegramRateLimitError ошибка 429 от Telegram API
type telegramRateLimitError struct {
	RetryAfter time.Duration
}

func (e *telegramRateLimitError) Error() string {
	return fmt.Sprintf("telegram: rate limit, retry after %v", e.RetryAfter)
}

func isRateLimitError(err error) bool {
	var rlErr *telegramRateLimitError
	return errors.As(err, &rlErr)
}

func parseRetryAfter(err error) time.Duration {
	var rlErr *telegramRateLimitError
	if errors.As(err, &rlErr) {
		return rlErr.RetryAfter
	}
	return 5 * time.Second
}

// Worker читает сообщения из Redis и отправляет в Telegram API
type Worker struct {
	redis      *redis.Client
	httpClient *http.Client
	baseURL    string
	testMode   bool
	enabled    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWorker создает Worker
func NewWorker(redisClient *redis.Client, botToken string, testMode, enabled bool) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		redis:      redisClient,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    fmt.Sprintf("https://api.telegram.org/bot%s/", botToken),
		testMode:   testMode,
		enabled:    enabled,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start запускает воркер в фоновой горутине
func (w *Worker) Start() {
	go w.run()
	logger.Info("✅ Telegram queue worker запущен")
}

// Stop останавливает воркер
func (w *Worker) Stop() {
	w.cancel()
	logger.Info("🛑 Telegram queue worker остановлен")
}

func (w *Worker) run() {
	queues := []string{
		string(PriorityHigh),
		string(PriorityNormal),
		string(PriorityLow),
	}

	for {
		// BRPOP — блокирующий pop с приоритетом (high → normal → low)
		result, err := w.redis.BRPop(w.ctx, 5*time.Second, queues...).Result()
		if err != nil {
			if w.ctx.Err() != nil {
				return // контекст отменён — завершаемся
			}
			if err != redis.Nil {
				logger.Error("❌ Queue worker: BRPOP error: %v", err)
			}
			continue
		}

		// result[0] — имя очереди, result[1] — payload
		var msg QueuedMessage
		if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
			logger.Error("❌ Queue worker: ошибка десериализации: %v", err)
			continue
		}

		// Проверяем TTL — устаревшие сигналы дропаем
		if time.Since(msg.CreatedAt) > MessageTTL {
			logger.Warn("⚠️ Queue worker: сообщение устарело (chatID=%d, возраст=%v), пропуск",
				msg.ChatID, time.Since(msg.CreatedAt).Round(time.Second))
			continue
		}

		// Ждём глобального rate limit (token bucket)
		for !w.canSendGlobal() {
			select {
			case <-w.ctx.Done():
				return
			case <-time.After(workerPollInterval):
			}
		}

		// Проверяем per-user rate limit (1 сообщение/сек на пользователя)
		if !w.canSendToUser(msg.ChatID) {
			// Откладываем на 1 секунду и кладём обратно
			go func(m QueuedMessage) {
				time.Sleep(time.Second)
				if err := w.enqueueBack(m); err != nil {
					logger.Error("❌ Queue worker: ошибка повторной постановки: %v", err)
				}
			}(msg)
			continue
		}

		// Отправляем
		if err := w.sendMessage(msg); err != nil {
			if isRateLimitError(err) {
				retryAfter := parseRetryAfter(err)
				msg.Attempts++
				if msg.Attempts < maxAttempts {
					go func(m QueuedMessage, delay time.Duration) {
						time.Sleep(delay)
						if err := w.enqueuePriority(m, PriorityHigh); err != nil {
							logger.Error("❌ Queue worker: ошибка retry постановки: %v", err)
						}
					}(msg, retryAfter)
				} else {
					logger.Warn("⚠️ Queue worker: дроп сообщения после %d попыток (chatID=%d)",
						msg.Attempts, msg.ChatID)
				}
			} else {
				logger.Error("❌ Queue worker: ошибка отправки (chatID=%d): %v", msg.ChatID, err)
			}
		}
	}
}

// canSendGlobal проверяет глобальный rate limit через Lua Token Bucket
func (w *Worker) canSendGlobal() bool {
	now := float64(time.Now().UnixNano()) / 1e9
	result, err := tokenBucketScript.Run(w.ctx, w.redis, []string{globalRateLimitKey}, now).Int()
	if err != nil {
		logger.Error("❌ Queue worker: ошибка token bucket: %v", err)
		return true // при ошибке Redis разрешаем отправку
	}
	return result == 1
}

// canSendToUser проверяет per-user rate limit (1 сообщение/сек)
func (w *Worker) canSendToUser(chatID int64) bool {
	key := fmt.Sprintf(userRateLimitKeyFmt, chatID)
	ok, err := w.redis.SetNX(w.ctx, key, 1, time.Second).Result()
	if err != nil {
		return true // при ошибке Redis разрешаем
	}
	return ok
}

// enqueueBack кладёт сообщение обратно в его приоритетную очередь
func (w *Worker) enqueueBack(msg QueuedMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return w.redis.LPush(context.Background(), string(msg.Priority), data).Err()
}

// enqueuePriority кладёт сообщение в указанную очередь
func (w *Worker) enqueuePriority(msg QueuedMessage, p Priority) error {
	msg.Priority = p
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return w.redis.LPush(context.Background(), string(p), data).Err()
}

// sendMessage отправляет сообщение в Telegram API
func (w *Worker) sendMessage(msg QueuedMessage) error {
	if !w.enabled {
		return nil
	}

	if w.testMode {
		preview := msg.Text
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		logger.Info("[TEST] Queue send to %d: %s", msg.ChatID, preview)
		return nil
	}

	request := map[string]interface{}{
		"chat_id":    msg.ChatID,
		"text":       msg.Text,
		"parse_mode": "Markdown",
	}
	if msg.Keyboard != nil {
		request["reply_markup"] = msg.Keyboard
	}

	return w.callTelegramAPI("sendMessage", request)
}

// callTelegramAPI выполняет HTTP запрос к Telegram Bot API
func (w *Worker) callTelegramAPI(method string, payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := w.httpClient.Post(
		w.baseURL+method,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var tgResp struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters,omitempty"`
	}

	if err := json.Unmarshal(body, &tgResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if !tgResp.OK {
		if tgResp.ErrorCode == 429 {
			retryAfter := time.Duration(tgResp.Parameters.RetryAfter) * time.Second
			if retryAfter == 0 {
				retryAfter = 5 * time.Second
			}
			return &telegramRateLimitError{RetryAfter: retryAfter}
		}
		return fmt.Errorf("telegram error %d: %s", tgResp.ErrorCode, tgResp.Description)
	}

	return nil
}
