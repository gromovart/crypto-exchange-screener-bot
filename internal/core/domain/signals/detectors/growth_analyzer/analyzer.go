// internal/core/domain/signals/detectors/growth_analyzer/analyzer.go
package growth_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/growth_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/growth_analyzer/manager"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"fmt"
	"time"
)

// GrowthAnalyzer - анализатор роста (рефакторинг по модульной структуре)
type GrowthAnalyzer struct {
	config       common.AnalyzerConfig
	stateManager *manager.StateManager
}

func convertToManagerConfig(cfg common.AnalyzerConfig) manager.AnalyzerConfigWrapper {
	return manager.AnalyzerConfigWrapper{
		Enabled:        cfg.Enabled,
		Weight:         cfg.Weight,
		MinConfidence:  cfg.MinConfidence,
		MinDataPoints:  cfg.MinDataPoints,
		CustomSettings: cfg.CustomSettings,
	}
}

// NewGrowthAnalyzer - создает новый анализатор роста
func NewGrowthAnalyzer(cfg common.AnalyzerConfig) *GrowthAnalyzer {
	if err := ValidateGrowthConfig(cfg); err != nil {
		// Используем конфигурацию по умолчанию при ошибке валидации
		cfg = NewGrowthConfig()
	}

	return &GrowthAnalyzer{
		config:       cfg,
		stateManager: manager.NewStateManager(convertToManagerConfig(cfg)),
	}
}

// Name возвращает имя анализатора
func (a *GrowthAnalyzer) Name() string {
	return "growth_analyzer"
}

// Version возвращает версию
func (a *GrowthAnalyzer) Version() string {
	return "2.0.0" // Обновленная версия после рефакторинга
}

// Supports проверяет поддержку символа
func (a *GrowthAnalyzer) Supports(symbol string) bool {
	// Проверяем rate limiting
	return a.stateManager.ShouldProcessSymbol(symbol)
}

// Analyze анализирует данные на рост
func (a *GrowthAnalyzer) Analyze(data []redis_storage.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()
	success := false
	var growthPercent float64
	var isContinuous bool

	defer func() {
		a.stateManager.UpdateStats(time.Since(startTime), success, growthPercent, isContinuous)
	}()

	// Валидация входных данных
	if err := ValidatePriceData(data); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if len(data) < cfg.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: got %d, need %d", len(data), cfg.MinDataPoints)
	}

	// Получаем конфигурацию роста
	growthConfig := GetGrowthConfig(cfg)

	// Подготавливаем данные
	sortedData := SortPriceDataByTime(data)

	// Преобразуем GrowthConfig в CalculatorConfig для калькулятора
	calcConfig := convertGrowthConfigToCalculatorConfig(growthConfig)

	// Используем калькулятор роста для анализа
	calculatorInput := calculator.GrowthCalculatorInput{
		PriceData:   sortedData,
		Config:      calcConfig,
		CurrentTime: time.Now(),
	}

	calculatorOutput, err := calculator.CalculateGrowth(calculatorInput)
	if err != nil {
		return nil, fmt.Errorf("growth calculation error: %w", err)
	}

	// Проверяем минимальный рост
	if calculatorOutput.GrowthPercent < growthConfig.MinGrowthPercent {
		return nil, nil // Недостаточный рост
	}

	// Создаем результат анализа
	analysisResult := GrowthAnalysisResult{
		Symbol:        sortedData[0].Symbol,
		GrowthPercent: calculatorOutput.GrowthPercent,
		SignalType:    GrowthSignalType(calculatorOutput.SignalType),
		IsContinuous:  calculatorOutput.IsContinuous,
		TrendStrength: calculatorOutput.TrendStrength,
		Volatility:    calculatorOutput.Volatility,
		DataPoints:    len(sortedData),
		StartPrice:    sortedData[0].Price,
		EndPrice:      sortedData[len(sortedData)-1].Price,
		Timestamp:     time.Now(),
		RawData:       sortedData,
	}

	// Преобразуем результат в формат для калькулятора уверенности
	calcResult := calculator.CalculatorResult{
		GrowthPercent:   calculatorOutput.GrowthPercent,
		IsContinuous:    calculatorOutput.IsContinuous,
		ContinuityScore: calculatorOutput.ContinuityScore,
		SignalType:      calculatorOutput.SignalType,
		TrendStrength:   calculatorOutput.TrendStrength,
		Volatility:      calculatorOutput.Volatility,
		RawData:         sortedData,
	}

	// Рассчитываем итоговую уверенность
	analysisResult.Confidence = calculator.CalculateConfidence(calcResult, calcConfig)

	// Проверяем минимальную уверенность
	if analysisResult.Confidence < cfg.MinConfidence {
		return nil, nil // Недостаточная уверенность
	}

	// Обновляем данные по символу
	// Создаем manager-совместимый результат
	managerResult := manager.GrowthAnalysisResult{
		Symbol:        analysisResult.Symbol,
		GrowthPercent: analysisResult.GrowthPercent,
		SignalType:    string(analysisResult.SignalType),
		Confidence:    analysisResult.Confidence,
		IsContinuous:  analysisResult.IsContinuous,
		TrendStrength: analysisResult.TrendStrength,
		Volatility:    analysisResult.Volatility,
		DataPoints:    analysisResult.DataPoints,
		StartPrice:    analysisResult.StartPrice,
		EndPrice:      analysisResult.EndPrice,
		Timestamp:     analysisResult.Timestamp,
		RawData:       data,
	}

	a.stateManager.UpdateSymbolData(managerResult.Symbol, managerResult)

	// Преобразуем в общий сигнал
	signal := analysisResult.ToAnalyzerSignal()

	// Обновляем переменные для deferred функции
	success = true
	growthPercent = calculatorOutput.GrowthPercent
	isContinuous = calculatorOutput.IsContinuous

	return []analysis.Signal{signal}, nil
}

// GetConfig возвращает конфигурацию
func (a *GrowthAnalyzer) GetConfig() common.AnalyzerConfig {
	return a.config
}

// GetStats возвращает статистику
func (a *GrowthAnalyzer) GetStats() common.AnalyzerStats {
	managerStats := a.stateManager.GetAnalyzerStats()

	// Преобразуем manager.AnalyzerStatsWrapper в common.AnalyzerStats
	return common.AnalyzerStats{
		TotalCalls:   managerStats.TotalCalls,
		SuccessCount: managerStats.SuccessCount,
		ErrorCount:   managerStats.ErrorCount,
		TotalTime:    managerStats.TotalTime,
		AverageTime:  managerStats.AverageTime,
		LastCallTime: managerStats.LastCallTime,
	}
}

// GetGrowthStats возвращает расширенную статистику роста
func (a *GrowthAnalyzer) GetGrowthStats() GrowthStats {
	managerStats := a.stateManager.GetStats()

	// Получаем базовую статистику через GetStats
	baseStats := a.GetStats()

	return GrowthStats{
		AnalyzerStats:         baseStats, // Используем AnalyzerStatsCopy
		TotalGrowthSignals:    managerStats.TotalGrowthSignals,
		AverageGrowthPercent:  managerStats.AverageGrowthPercent,
		MaxGrowthPercent:      managerStats.MaxGrowthPercent,
		ContinuousGrowthCount: managerStats.ContinuousGrowthCount,
	}
}

// GetSymbolStats возвращает статистику по символу
func (a *GrowthAnalyzer) GetSymbolStats(symbol string) map[string]interface{} {
	return a.stateManager.GetSymbolStats(symbol)
}

// GetRecentSignals возвращает последние сигналы для символа
func (a *GrowthAnalyzer) GetRecentSignals(symbol string, limit int) []GrowthAnalysisResult {
	managerResults := a.stateManager.GetRecentSignals(symbol, limit)
	results := make([]GrowthAnalysisResult, len(managerResults))

	for i, mr := range managerResults {
		results[i] = GrowthAnalysisResult{
			Symbol:        mr.Symbol,
			GrowthPercent: mr.GrowthPercent,
			SignalType:    GrowthSignalType(mr.SignalType),
			Confidence:    mr.Confidence,
			IsContinuous:  mr.IsContinuous,
			TrendStrength: mr.TrendStrength,
			Volatility:    mr.Volatility,
			DataPoints:    mr.DataPoints,
			StartPrice:    mr.StartPrice,
			EndPrice:      mr.EndPrice,
			Timestamp:     mr.Timestamp,
			RawData:       mr.RawData,
		}
	}

	return results
}

// Reset сбрасывает состояние анализатора
func (a *GrowthAnalyzer) Reset() {
	a.stateManager.Reset()
}

// GetUptime возвращает время работы анализатора
func (a *GrowthAnalyzer) GetUptime() time.Duration {
	return a.stateManager.GetUptime()
}

// convertGrowthConfigToCalculatorConfig - преобразует GrowthConfig в CalculatorConfig
func convertGrowthConfigToCalculatorConfig(growthConfig GrowthConfig) calculator.CalculatorConfig {
	return calculator.CalculatorConfig{
		MinGrowthPercent:      growthConfig.MinGrowthPercent,
		ContinuityThreshold:   growthConfig.ContinuityThreshold,
		AccelerationThreshold: growthConfig.AccelerationThreshold,
		VolumeWeight:          growthConfig.VolumeWeight,
		TrendStrengthWeight:   growthConfig.TrendStrengthWeight,
		VolatilityWeight:      growthConfig.VolatilityWeight,
	}
}
func (r *GrowthAnalysisResult) ToAnalyzerSignal() analysis.Signal {
	tags := []string{"growth", "bullish"}
	if r.IsContinuous {
		tags = append(tags, "continuous")
	}
	if string(r.SignalType) != "" {
		tags = append(tags, string(r.SignalType))
	}

	// mapGrowthSignalTypeToValue уже определена в types.go
	growthTypeValue := mapGrowthSignalTypeToValue(r.SignalType)

	return analysis.Signal{
		Symbol:        r.Symbol,
		Type:          "growth",
		Direction:     "up",
		ChangePercent: r.GrowthPercent,
		Confidence:    r.Confidence,
		DataPoints:    r.DataPoints,
		StartPrice:    r.StartPrice,
		EndPrice:      r.EndPrice,
		Timestamp:     r.Timestamp,
		Metadata: analysis.Metadata{
			Strategy:     "growth_detection",
			Tags:         tags,
			IsContinuous: r.IsContinuous,
			Indicators: map[string]float64{
				"trend_strength": r.TrendStrength,
				"volatility":     r.Volatility,
				"growth_type":    growthTypeValue,
			},
		},
	}
}
