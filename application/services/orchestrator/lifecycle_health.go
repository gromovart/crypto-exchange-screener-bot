// application/services/orchestrator/lifecycle_health.go
package orchestrator

import (
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// healthCheckLoop цикл проверки здоровья
func (lm *LifecycleManager) healthCheckLoop() {
	for {
		select {
		case <-lm.healthTicker.C:
			lm.performHealthCheck()
		case <-lm.stopChan:
			if lm.healthTicker != nil {
				lm.healthTicker.Stop()
			}
			return
		}
	}
}

// performHealthCheck выполняет проверку здоровья
func (lm *LifecycleManager) performHealthCheck() {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	health := lm.registry.CheckHealth()

	// Публикуем событие проверки здоровья
	lm.publishEvent(types.Event{
		Type:   types.EventHealthCheck,
		Source: "LifecycleManager",
		Data: map[string]interface{}{
			"сообщение":   "Проверка здоровья выполнена",
			"данные":      health,
			"время":       time.Now(),
			"серьезность": "информация",
		},
		Timestamp: time.Now(),
	})

	// Перезапускаем неудачные сервисы если включено
	if lm.config.RestartOnFailure {
		for service, healthy := range health {
			if !healthy {
				lm.scheduleRestart(service)
			}
		}
	}
}
