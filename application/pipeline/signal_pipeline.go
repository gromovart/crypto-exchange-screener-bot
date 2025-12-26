// internal/pipeline/signal_pipeline.go (исправленная версия)
package pipeline

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	errors "crypto-exchange-screener-bot/internal/core/errors"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"log"
	"sync"
	"time"
)

// SignalPipeline обрабатывает и обогащает сигналы перед отправкой
type SignalPipeline struct {
	eventBus    *events.EventBus
	stages      []PipelineStage
	rateLimiter *RateLimiter
	stats       PipelineStats
	mu          sync.RWMutex
}

// NewSignalPipeline создает новый пайплайн сигналов
func NewSignalPipeline(eventBus *events.EventBus) *SignalPipeline {
	return &SignalPipeline{
		eventBus: eventBus,
		stages:   make([]PipelineStage, 0),
		rateLimiter: &RateLimiter{
			lastSent: make(map[string]time.Time),
			minDelay: 30 * time.Second, // Не чаще чем раз в 30 секунд на символ
		},
		stats: PipelineStats{},
	}
}

// AddStage добавляет этап обработки
func (p *SignalPipeline) AddStage(stage PipelineStage) {
	p.stages = append(p.stages, stage)
}

// Start запускает пайплайн
func (p *SignalPipeline) Start() {
	// Подписываемся на события сигналов
	subscriber := events.NewBaseSubscriber(
		"signal_pipeline",
		[]events.EventType{events.EventSignalDetected},
		p.handleSignal,
	)
	p.eventBus.Subscribe(events.EventSignalDetected, subscriber)

	log.Println("🚀 SignalPipeline запущен")
}

// handleSignal обрабатывает сигнал
func (p *SignalPipeline) handleSignal(event events.Event) error {
	startTime := time.Now()

	p.mu.Lock()
	p.stats.SignalsReceived++
	p.mu.Unlock()

	// Проверяем, что это сигнал
	signalData, ok := event.Data.(analysis.Signal)
	if !ok {
		log.Printf("⚠️ Неверный формат сигнала: %T", event.Data)
		return nil
	}

	// Проверяем rate limit
	key := signalData.Symbol + "_" + signalData.Type
	if !p.rateLimiter.CanSend(key) {
		p.mu.Lock()
		p.stats.SignalsFiltered++
		p.mu.Unlock()
		log.Printf("⏳ Пропуск сигнала %s (rate limit)", signalData.Symbol)
		return nil
	}

	// Обрабатываем через все этапы
	processedSignal := signalData
	var err error

	for _, stage := range p.stages {
		processedSignal, err = stage.Process(processedSignal)
		if err != nil {
			log.Printf("❌ Ошибка обработки сигнала %s на этапе %s: %v",
				signalData.Symbol, stage.Name(), err)
			return err
		}
	}

	// Обновляем статистику
	p.mu.Lock()
	p.stats.SignalsProcessed++
	p.stats.AverageTime = (p.stats.AverageTime*time.Duration(p.stats.SignalsProcessed-1) +
		time.Since(startTime)) / time.Duration(p.stats.SignalsProcessed)
	p.stats.LastProcessed = time.Now()
	p.mu.Unlock()

	// Публикуем обработанный сигнал
	p.eventBus.Publish(events.Event{
		Type:   "signal_processed",
		Source: "signal_pipeline",
		Data:   processedSignal,
		Metadata: events.Metadata{
			CorrelationID: processedSignal.ID,
			Priority:      int(processedSignal.Confidence / 10),
			Tags:          processedSignal.Metadata.Tags,
		},
		Timestamp: time.Now(),
	})

	log.Printf("✅ Сигнал обработан: %s %s (уверенность: %.0f%%)",
		processedSignal.Symbol, processedSignal.Direction, processedSignal.Confidence)

	return nil
}

// CanSend проверяет, можно ли отправить сигнал
func (rl *RateLimiter) CanSend(key string) bool {
	rl.mu.RLock()
	last, exists := rl.lastSent[key]
	rl.mu.RUnlock()

	if exists && time.Since(last) < rl.minDelay {
		return false
	}

	rl.mu.Lock()
	rl.lastSent[key] = time.Now()
	rl.mu.Unlock()

	return true
}

// GetStats возвращает статистику
func (p *SignalPipeline) GetStats() PipelineStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// ValidationStage этап валидации сигнала
type ValidationStage struct{}

func (s *ValidationStage) Name() string { return "validation" }

func (s *ValidationStage) Process(signal analysis.Signal) (analysis.Signal, error) {
	if signal.Confidence < 50 {
		return signal, errors.ErrLowConfidence
	}

	if signal.ChangePercent == 0 {
		return signal, errors.ErrNoChange
	}

	if signal.Symbol == "" {
		return signal, errors.ErrInvalidSymbol
	}

	return signal, nil
}

// EnrichmentStage этап обогащения сигнала
type EnrichmentStage struct{}

func (s *EnrichmentStage) Name() string { return "enrichment" }

func (s *EnrichmentStage) Process(signal analysis.Signal) (analysis.Signal, error) {
	// Добавляем дополнительные теги
	if signal.Confidence > 80 {
		signal.Metadata.Tags = append(signal.Metadata.Tags, "high_confidence")
	}

	if signal.ChangePercent > 10 {
		signal.Metadata.Tags = append(signal.Metadata.Tags, "strong_move")
	}

	// Добавляем временную метку обработки
	if signal.Metadata.Indicators == nil {
		signal.Metadata.Indicators = make(map[string]float64)
	}
	signal.Metadata.Indicators["processed_at"] = float64(time.Now().Unix())

	return signal, nil
}
