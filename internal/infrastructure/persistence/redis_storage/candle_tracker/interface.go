// internal/infrastructure/persistence/redis_storage/candle_tracker/interface.go
package candletracker

// CandleTrackerInterface интерфейс для отслеживания обработанных свечей
type CandleTrackerInterface interface {
	// Initialize инициализирует трекер
	Initialize() error

	// MarkCandleProcessedAtomically атомарно помечает свечу как обработанную
	// Возвращает true если свеча была успешно помечена как обработанная (не была обработана ранее)
	// Возвращает false если свеча уже была обработана
	MarkCandleProcessedAtomically(symbol, period string, startTime int64) (bool, error)

	// IsCandleProcessed проверяет была ли свеча обработана
	IsCandleProcessed(symbol, period string, startTime int64) (bool, error)

	// MarkCandleProcessedUnsafe помечает свечу как обработанную (без атомарной проверки)
	MarkCandleProcessedUnsafe(symbol, period string, startTime int64) error

	// CleanupOldEntries очищает старые записи
	CleanupOldEntries() (int64, error)

	// GetStats возвращает статистику трекера
	GetStats() (map[string]interface{}, error)

	// TestConnection тестирует подключение к Redis
	TestConnection() error
}
