// application/services/orchestrator/lifecycle_deps.go
package orchestrator

import (
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// StartAll запускает все сервисы в правильном порядке
func (lm *LifecycleManager) StartAll() map[string]error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Получаем порядок запуска на основе зависимостей
	order := lm.getStartOrder()

	errors := make(map[string]error)

	for _, serviceName := range order {
		service, exists := lm.registry.Get(serviceName)
		if !exists {
			continue
		}

		if err := service.Start(); err != nil {
			errors[serviceName] = err
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State: StateError,
				Error: err.Error(),
			})

			lm.publishEvent(types.Event{
				Type:   types.EventServiceError,
				Source: serviceName,
				Data: map[string]interface{}{
					"сервис":   serviceName,
					"ошибка":   err.Error(),
					"действие": "ошибка_запуска",
				},
				Timestamp: time.Now(),
			})

			// Пытаемся перезапустить если включено
			if lm.config.RestartOnFailure {
				lm.scheduleRestart(serviceName)
			}
		} else {
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State:     StateRunning,
				StartedAt: time.Now(),
			})

			lm.publishEvent(types.Event{
				Type:   types.EventServiceStarted,
				Source: serviceName,
				Data: map[string]interface{}{
					"сервис":   serviceName,
					"действие": "запущен",
					"статус":   "работает",
				},
				Timestamp: time.Now(),
			})
		}
	}

	return errors
}

// StopAll останавливает все сервисы в обратном порядке
func (lm *LifecycleManager) StopAll() map[string]error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Получаем порядок остановки (обратный порядку запуска)
	order := lm.getStopOrder()

	errors := make(map[string]error)

	for _, serviceName := range order {
		service, exists := lm.registry.Get(serviceName)
		if !exists {
			continue
		}

		if err := service.Stop(); err != nil {
			errors[serviceName] = err
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State: StateError,
				Error: err.Error(),
			})

			lm.publishEvent(types.Event{
				Type:   types.EventServiceError,
				Source: serviceName,
				Data: map[string]interface{}{
					"сервис":   serviceName,
					"ошибка":   err.Error(),
					"действие": "ошибка_остановки",
				},
				Timestamp: time.Now(),
			})
		} else {
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State:     StateStopped,
				StoppedAt: time.Now(),
			})

			lm.publishEvent(types.Event{
				Type:   types.EventServiceStopped,
				Source: serviceName,
				Data: map[string]interface{}{
					"сервис":   serviceName,
					"действие": "остановлен",
					"статус":   "остановлен",
				},
				Timestamp: time.Now(),
			})
		}
	}

	return errors
}

// getStartOrder возвращает порядок запуска на основе зависимостей
func (lm *LifecycleManager) getStartOrder() []string {
	// Простая топологическая сортировка
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	order := make([]string, 0)

	var visit func(string)
	visit = func(service string) {
		if temp[service] {
			// Циклическая зависимость
			return
		}

		if !visited[service] {
			temp[service] = true

			// Сначала посещаем зависимости
			if deps, exists := lm.dependencyGraph.Services[service]; exists {
				for _, dep := range deps {
					visit(dep)
				}
			}

			temp[service] = false
			visited[service] = true
			order = append(order, service)
		}
	}

	// Запускаем для всех сервисов
	for service := range lm.registry.GetAll() {
		if !visited[service] {
			visit(service)
		}
	}

	return order
}

// getStopOrder возвращает порядок остановки
func (lm *LifecycleManager) getStopOrder() []string {
	// Обратный порядок запуска
	startOrder := lm.getStartOrder()
	stopOrder := make([]string, len(startOrder))

	for i, service := range startOrder {
		stopOrder[len(startOrder)-1-i] = service
	}

	return stopOrder
}

// scheduleRestart планирует перезапуск сервиса
func (lm *LifecycleManager) scheduleRestart(name string) {
	attempts := lm.restartAttempts[name]
	attempts++
	lm.restartAttempts[name] = attempts

	// Проверяем максимальное количество попыток
	if attempts > lm.config.MaxRestartAttempts && lm.config.MaxRestartAttempts > 0 {
		lm.publishEvent(types.Event{
			Type:   types.EventServiceError,
			Source: "LifecycleManager",
			Data: map[string]interface{}{
				"сервис":      name,
				"сообщение":   "Превышено максимальное количество попыток перезапуска, отказ",
				"время":       time.Now(),
				"серьезность": "ошибка",
			},
			Timestamp: time.Now(),
		})
		return
	}

	// Запланировать перезапуск
	go func() {
		time.Sleep(lm.config.RestartDelay)

		lm.publishEvent(types.Event{
			Type:   types.EventServiceStarted,
			Source: "LifecycleManager",
			Data: map[string]interface{}{
				"сервис":      name,
				"сообщение":   "Попытка перезапуска",
				"время":       time.Now(),
				"серьезность": "информация",
			},
			Timestamp: time.Now(),
		})

		if err := lm.StartService(name); err != nil {
			lm.publishEvent(types.Event{
				Type:   types.EventServiceError,
				Source: "LifecycleManager",
				Data: map[string]interface{}{
					"сервис":      name,
					"сообщение":   "Перезапуск не удался: " + err.Error(),
					"время":       time.Now(),
					"серьезность": "ошибка",
				},
				Timestamp: time.Now(),
			})
		}
	}()
}
