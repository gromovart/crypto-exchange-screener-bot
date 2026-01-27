// internal/delivery/telegram/controllers/counter/controller.go
package counter

import (
	counterService "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
)

// controllerImpl реализация CounterController
//
// АРХИТЕКТУРНАЯ РОЛЬ: EventSubscriber в Event-Driven Architecture
//
// Роль в системе EventBus:
// 1. ПОДПИСЧИК: реализует интерфейс types.EventSubscriber
// 2. ОБРАБОТЧИК: получает события types.EventCounterSignalDetected
// 3. АДАПТЕР: преобразует события EventBus в вызовы Use Cases
//
// СВЯЗЬ С EVENTBUS:
// ┌─────────────────────────────────────────────────────────────┐
// │                        EVENT BUS                            │
// │  (паттерн Publisher-Subscriber / Медиатор)                  │
// └─────────────────────┬───────────────────────────────────────┘
//
//	│ publishes EventCounterSignalDetected
//	▼
//
// ┌─────────────────────────────────────────────────────────────┐
// │                 CounterController (ЭТОТ ФАЙЛ)               │
// │  (реализует EventSubscriber: HandleEvent, GetName, etc.)    │
// └─────────────────────┬───────────────────────────────────────┘
//
//	│ вызывает service.Exec(CounterParams)
//	▼
//
// ┌─────────────────────────────────────────────────────────────┐
// │                 CounterService (Use Case)                   │
// │                (бизнес-логика обработки)                    │
// └─────────────────────────────────────────────────────────────┘
//
// ОБЯЗАННОСТИ КАК EVENT SUBSCRIBER:
// - Реализация контракта EventSubscriber (HandleEvent, GetName, GetSubscribedEvents)
// - Декларативная подписка на конкретные типы событий
// - Обработка событий в соответствии с контрактом EventBus
// - Возврат ошибок для обработки EventBus (retry, dead letter queue)
type controllerImpl struct {
	service counterService.Service
}

// NewController создает новый контроллер счетчика
//
// КОНТРАКТ СОЗДАНИЯ:
// 1. Контроллер регистрируется в EventBus через bus.Subscribe()
// 2. EventBus вызывает GetSubscribedEvents() для определения подписок
// 3. При публикации события EventBus вызывает HandleEvent()
//
// ПРИМЕР ИСПОЛЬЗОВАНИЯ:
//
//	bus := eventbus.NewEventBus()
//	service := counter.NewService(...)
//	controller := counter.NewController(service)
//	bus.Subscribe(types.EventCounterSignalDetected, controller)
func NewController(service counterService.Service) Controller {
	return &controllerImpl{service: service}
}

// HandleEvent обрабатывает событие от EventBus
//
// КОНТРАКТ EVENTSUBSCRIBER:
// - EventBus гарантирует вызов этого метода для подписанных событий
// - Метод должен быть идемпотентным (обработка повторных событий)
// - Ошибки возвращаются в EventBus для дальнейшей обработки
//
// ЖИЗНЕННЫЙ ЦИКЛ ОБРАБОТКИ СОБЫТИЯ:
// 1. EventBus публикует EventCounterSignalDetected
// 2. EventBus находит контроллер через GetSubscribedEvents()
// 3. EventBus вызывает HandleEvent(event) с Middleware цепочкой:
//   - Rate Limiting → Logging → Retry → Метод HandleEvent
//
// 4. Результат возвращается в EventBus для метрик и мониторинга
func (c *controllerImpl) HandleEvent(event types.Event) error {

	// [1] ВАЛИДАЦИЯ: проверяем структуру данных события
	// Контракт Event: event.Data может быть любого типа (interface{})
	// Наша ответственность: проверить ожидаемую структуру
	if err := ValidateEventData(event.Data); err != nil {
		logger.Error("❌ CounterController: Невалидные данные события: %v", err)
		return fmt.Errorf("валидация данных события: %w", err)
	}

	// [2] ПРЕОБРАЗОВАНИЕ: map[string]interface{} → CounterParams
	// Адаптация: преобразуем универсальный формат EventBus в специфичный для Use Case
	params, err := convertEventToParams(event)
	if err != nil {
		logger.Error("❌ CounterController: Ошибка преобразования данных: %v", err)
		return fmt.Errorf("преобразование данных события: %w", err)
	}

	// [3] ВЫЗОВ USE CASE (сервиса)
	// Делегирование: контроллер не содержит бизнес-логику, только координирует
	result, err := c.service.Exec(params)
	if err != nil {
		logger.Error("❌ CounterController: Ошибка обработки: %v", err)
		return fmt.Errorf("обработка сервисом: %w", err)
	}

	// [4] ЛОГИРОВАНИЕ РЕЗУЛЬТАТА
	// Мониторинг: EventBus также собирает метрики о успешных/неуспешных обработках
	logger.Debug("✅ CounterController: Результат: %+v", result)
	return nil
}

// GetName возвращает имя контроллера
//
// ИДЕНТИФИКАЦИЯ В EVENTBUS:
// - Уникальное имя для логирования и мониторинга
// - Используется в метриках EventBusMetrics.SubscribersCount
// - Помогает в диагностике распределенной системы
//
// СИСТЕМНЫЕ ТРЕБОВАНИЯ:
// - Должно быть уникальным в пределах одного типа события
// - Должно быть человеко-читаемым для логов
// - Должно быть постоянным (не меняться между запусками)
func (c *controllerImpl) GetName() string {
	return "counter_controller"
}

// GetSubscribedEvents возвращает типы событий для подписки
//
// ДЕКЛАРАТИВНАЯ ПОДПИСКА:
// - EventBus вызывает этот метод при регистрации
// - Определяет, на какие события реагирует контроллер
// - Позволяет EventBus оптимизировать маршрутизацию
//
// ПРИНЦИПЫ:
// - Минимализм: подписываться только на необходимые события
// - Специализация: один контроллер = один тип ответственности
// - Явность: явное указание лучше неявного
func (c *controllerImpl) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventCounterSignalDetected,
	}
}

// АРХИТЕКТУРНЫЕ ПРЕИМУЩЕСТВА ТАКОГО ПОДХОДА:
//
// 1. LOOSE COUPLING (СЛАБАЯ СВЯЗАННОСТЬ):
//    - Контроллер не знает, кто публикует события
//    - Публикатор не знает, кто обрабатывает события
//    - EventBus выступает медиатором
//
// 2. SCALABILITY (МАСШТАБИРУЕМОСТЬ):
//    - Можно добавить несколько контроллеров на одно событие
//    - EventBus распределяет нагрузку
//    - Легко добавлять новые типы событий
//
// 3. RESILIENCE (УСТОЙЧИВОСТЬ):
//    - Ошибки в контроллере не ломают EventBus
//    - EventBus может ретраить неудачные обработки
//    - Dead letter queue для проблемных событий
//
// 4. OBSERVABILITY (НАБЛЮДАЕМОСТЬ):
//    - EventBus собирает метрики по всем обработкам
//    - Централизованное логирование
//    - Мониторинг health check
//
// 5. TESTABILITY (ТЕСТИРУЕМОСТЬ):
//    - Контроллер тестируется изолированно
//    - Можно мокать EventBus
//    - Можно тестировать цепочки событий
//
// КЛЮЧЕВЫЕ ОТЛИЧИЯ ОТ ТРАДИЦИОННЫХ КОНТРОЛЛЕРОВ:
// - Не HTTP-based: работает с событиями, а не HTTP запросами
// - Не содержит роутинг: EventBus управляет маршрутизацией
// - Пассивный: реагирует на события, а не опрашивает
// - Статистический: один инстанс на всё приложение
