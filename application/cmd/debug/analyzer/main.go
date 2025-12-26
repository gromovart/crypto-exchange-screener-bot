package main

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	analyzers "crypto-exchange-screener-bot/internal/core/domain/signals/detectors"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"math"
	"strings"
	"time"
)

func main() {
	logger.Debug("🔧 ТЕСТИРОВАНИЕ АНАЛИЗАТОРОВ")
	logger.Debug(strings.Repeat("=", 60))

	// Тестовые данные для других тестов
	testData := createTestData()

	// Тестируем CounterAnalyzer (расширенный тест)
	logger.Debug("\n🧪 ТЕСТ COUNTER ANALYZER:")
	testCounterAnalyzerExtended()

	// Тестируем новый FallAnalyzer
	testNewFallAnalyzer()

	// Тестируем GrowthAnalyzer
	logger.Debug("\n🧪 ТЕСТ GROWTH ANALYZER:")
	testGrowthAnalyzer(testData)

	// Тестируем старый FallAnalyzer (для сравнения)
	logger.Debug("\n🧪 ТЕСТ СТАРОГО FALL ANALYZER:")
	testFallAnalyzer(testData)

	// Тестируем ContinuousAnalyzer
	logger.Debug("\n🧪 ТЕСТ CONTINUOUS ANALYZER:")
	testContinuousAnalyzer(testData)

	// Тестируем VolumeAnalyzer
	logger.Debug("\n🧪 ТЕСТ VOLUME ANALYZER:")
	testVolumeAnalyzer(testData)

	// Тестируем все анализаторы вместе
	logger.Debug("\n🧪 ИНТЕГРАЦИОННЫЙ ТЕСТ ВСЕХ АНАЛИЗАТОРОВ:")
	testAllAnalyzersIntegration()

	logger.Debug("\n✅ Тестирование завершено")
}

func testCounterAnalyzerExtended() {
	fmt.Println("   🔄 Расширенный тест CounterAnalyzer...")

	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"base_period_minutes":    1,
			"analysis_period":        "15m",
			"growth_threshold":       0.1,
			"fall_threshold":         0.1,
			"track_growth":           true,
			"track_fall":             true,
			"notification_threshold": 1,
			"max_signals_5m":         5,
			"max_signals_15m":        8,
			"max_signals_30m":        10,
			"max_signals_1h":         12,
			"max_signals_4h":         15,
			"max_signals_1d":         20,
			"chart_provider":         "coinglass",
		},
	}

	analyzer := analyzers.NewCounterAnalyzer(config, nil, nil)

	// Тест 1: Многократный анализ одного символа
	fmt.Println("   📈 Тест 1: Многократный анализ BTCUSDT")
	now := time.Now()
	btcData := []types.PriceData{
		{Symbol: "BTCUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 100.2, Timestamp: now.Add(-1 * time.Minute)}, // +0.2%
	}

	var signals []analysis.Signal
	for i := 1; i <= 5; i++ {
		sigs, err := analyzer.Analyze(btcData, config)
		if err != nil {
			fmt.Printf("      ❌ Итерация %d: ошибка - %v\n", i, err)
			continue
		}
		signals = append(signals, sigs...)
		if len(sigs) > 0 {
			fmt.Printf("      %d. Сигнал роста: счетчик=%d, уверенность=%.1f%%\n",
				i, i, sigs[0].Confidence)
		}
	}

	// Проверяем статистику
	counters := analyzer.GetAllCounters()
	if btcCounter, ok := counters["BTCUSDT"]; ok {
		fmt.Printf("   📊 Статистика BTCUSDT: рост=%d, падение=%d\n",
			btcCounter.GrowthCount, btcCounter.FallCount)

		// Рассчитываем ожидаемую уверенность
		maxSignals := 8 // для 15-минутного периода
		expectedConfidence := float64(btcCounter.GrowthCount) / float64(maxSignals) * 100
		fmt.Printf("      • Ожидаемая уверенность: %.1f%%\n", expectedConfidence)

		if len(signals) > 0 {
			lastSignal := signals[len(signals)-1]
			fmt.Printf("      • Фактическая уверенность: %.1f%%\n", lastSignal.Confidence)

			if math.Abs(lastSignal.Confidence-expectedConfidence) < 1.0 {
				fmt.Println("      ✅ Уверенность рассчитана правильно")
			} else {
				fmt.Printf("      ❌ Расхождение в уверенности: %.1f%% vs %.1f%%\n",
					lastSignal.Confidence, expectedConfidence)
			}
		}
	}

	// Тест 2: Анализ нескольких символов
	fmt.Println("\n   📈 Тест 2: Анализ нескольких символов")
	symbols := []string{"ETHUSDT", "SOLUSDT", "ADAUSDT"}

	for _, symbol := range symbols {
		symbolData := []types.PriceData{
			{Symbol: symbol, Price: 50.0, Timestamp: now.Add(-2 * time.Minute)},
			{Symbol: symbol, Price: 50.1, Timestamp: now.Add(-1 * time.Minute)}, // +0.2%
		}

		sigs, _ := analyzer.Analyze(symbolData, config)
		if len(sigs) > 0 {
			fmt.Printf("      • %s: %s %.2f%%\n", symbol, sigs[0].Direction, sigs[0].ChangePercent)
		}
	}

	// Общая статистика
	allCounters := analyzer.GetAllCounters()
	fmt.Printf("   📊 Общая статистика: %d символов\n", len(allCounters))

	totalGrowth := 0
	totalFall := 0
	for symbol, counter := range allCounters {
		fmt.Printf("      • %s: рост=%d, падение=%d\n",
			symbol, counter.GrowthCount, counter.FallCount)
		totalGrowth += counter.GrowthCount
		totalFall += counter.FallCount
	}

	fmt.Printf("   🧮 Итого: рост=%d, падение=%d, всего=%d\n",
		totalGrowth, totalFall, totalGrowth+totalFall)

	// Тест 3: Проверка метаданных
	fmt.Println("\n   🔍 Тест 3: Проверка метаданных сигналов")
	if len(signals) > 0 {
		signal := signals[0]
		fmt.Printf("      • Тип сигнала: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
		fmt.Printf("      • Точки данных: %d\n", signal.DataPoints)
		fmt.Printf("      • Стратегия: %s\n", signal.Metadata.Strategy)
		fmt.Printf("      • Тэги: %v\n", signal.Metadata.Tags)
		fmt.Printf("      • Индикаторы: %v\n", signal.Metadata.Indicators)

		// Проверяем ключевые индикаторы
		if count, ok := signal.Metadata.Indicators["count"]; ok {
			fmt.Printf("      • Счетчик в индикаторах: %.0f\n", count)
		}
		if period, ok := signal.Metadata.Indicators["period"]; ok {
			fmt.Printf("      • Период в индикаторах: %.0f мин\n", period)
		}
	}

	// Тест 4: Сброс периода
	fmt.Println("\n   🔄 Тест 4: Сброс периода")
	originalCount := len(allCounters)
	analyzer.SetAnalysisPeriod(analyzers.Period5Min)

	countersAfterReset := analyzer.GetAllCounters()
	fmt.Printf("      • Счетчиков до сброса: %d\n", originalCount)
	fmt.Printf("      • Счетчиков после сброса: %d\n", len(countersAfterReset))

	// Проверяем сброс счетчиков
	allReset := true
	for _, counter := range countersAfterReset {
		if counter.GrowthCount != 0 || counter.FallCount != 0 {
			allReset = false
			fmt.Printf("      ❌ Счетчик %s не сброшен: рост=%d, падение=%d\n",
				counter.Symbol, counter.GrowthCount, counter.FallCount)
		}
	}

	if allReset {
		fmt.Println("      ✅ Все счетчики сброшены при смене периода")
	}

	// Статистика анализатора
	stats := analyzer.GetStats()
	fmt.Println("\n   📈 Статистика анализатора:")
	fmt.Printf("      • Всего вызовов: %d\n", stats.TotalCalls)
	fmt.Printf("      • Успешных: %d\n", stats.SuccessCount)
	fmt.Printf("      • Ошибок: %d\n", stats.ErrorCount)
	fmt.Printf("      • Среднее время: %v\n", stats.AverageTime)
}
func testAllAnalyzersIntegration() {
	fmt.Println("   🔄 Интеграционный тест всех анализаторов...")

	// Создаем тестовые данные
	now := time.Now()
	testData := []types.PriceData{
		{Symbol: "BTCUSDT", Price: 100.0, Volume24h: 1000000, Timestamp: now.Add(-5 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 101.0, Volume24h: 1100000, Timestamp: now.Add(-4 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 102.0, Volume24h: 1200000, Timestamp: now.Add(-3 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 101.5, Volume24h: 1150000, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 100.5, Volume24h: 1050000, Timestamp: now.Add(-1 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 101.0, Volume24h: 1100000, Timestamp: now},
	}

	// Конфигурации для разных анализаторов
	growthConfig := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 50.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_growth":           1.0,
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	fallConfig := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 50.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_fall":             1.0,
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	counterConfig := analyzers.AnalyzerConfig{
		Enabled:        true,
		Weight:         0.7,
		MinConfidence:  10.0,
		MinDataPoints:  2,
		CustomSettings: analyzers.DefaultCounterConfig.CustomSettings,
	}

	// Исправленная конфигурация для ContinuousAnalyzer
	continuousConfig := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.6,
		MinConfidence: 30.0,
		MinDataPoints: 3,
		CustomSettings: map[string]interface{}{
			"min_continuous_points": 3,
		},
	}

	// Создаем анализаторы
	growthAnalyzer := analyzers.NewGrowthAnalyzer(growthConfig)
	fallAnalyzer := analyzers.NewFallAnalyzer(fallConfig)
	counterAnalyzer := analyzers.NewCounterAnalyzer(counterConfig, nil, nil)
	continuousAnalyzer := analyzers.NewContinuousAnalyzer(continuousConfig)

	// Запускаем все анализаторы
	fmt.Println("   📊 Запуск всех анализаторов на одних данных:")

	analyzersList := []struct {
		name     string
		analyzer analyzers.Analyzer
		config   analyzers.AnalyzerConfig
	}{
		{"GrowthAnalyzer", growthAnalyzer, growthConfig},
		{"FallAnalyzer", fallAnalyzer, fallConfig},
		{"CounterAnalyzer", counterAnalyzer, counterConfig},
		{"ContinuousAnalyzer", continuousAnalyzer, continuousConfig},
	}

	totalSignals := 0
	for _, item := range analyzersList {
		signals, err := item.analyzer.Analyze(testData, item.config)
		if err != nil {
			fmt.Printf("      ❌ %s: ошибка - %v\n", item.name, err)
			continue
		}

		fmt.Printf("      • %s: %d сигналов\n", item.name, len(signals))
		totalSignals += len(signals)

		// Показываем детали для CounterAnalyzer
		if item.name == "CounterAnalyzer" && len(signals) > 0 {
			for _, signal := range signals {
				fmt.Printf("        - %s: %s %.4f%% (уверенность: %.1f%%)\n",
					signal.Symbol, signal.Direction, signal.ChangePercent, signal.Confidence)
				fmt.Printf("          Тэги: %v\n", signal.Metadata.Tags)
			}
		}

		// Показываем детали для ContinuousAnalyzer
		if item.name == "ContinuousAnalyzer" && len(signals) > 0 {
			for _, signal := range signals {
				fmt.Printf("        - %s: непрерывный %s %.4f%% (%d точек)\n",
					signal.Symbol, signal.Direction, signal.ChangePercent, signal.DataPoints)
			}
		}
	}

	fmt.Printf("   📈 Всего сигналов от всех анализаторов: %d\n", totalSignals)

	// Проверяем согласованность результатов
	if totalSignals > 0 {
		fmt.Println("   ✅ Анализаторы работают совместно")
	} else {
		fmt.Println("   ⚠️  Ни один анализатор не обнаружил сигналов")
		fmt.Println("   💡 Возможные причины:")
		fmt.Println("      • Пороги слишком высокие")
		fmt.Println("      • Тестовые данные не содержат значительных изменений")
		fmt.Println("      • Анализаторы настроены слишком строго")
	}
}

func testCounterAnalyzer(testData []types.PriceData) {
	// Создаем тестовые данные для счетчика
	counterTestData := createCounterTestData()

	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"base_period_minutes":    1,
			"analysis_period":        "15m",
			"growth_threshold":       0.1,
			"fall_threshold":         0.1,
			"track_growth":           true,
			"track_fall":             true,
			"notification_threshold": 1,
			"max_signals_5m":         5,
			"max_signals_15m":        8,
			"max_signals_30m":        10,
			"max_signals_1h":         12,
			"max_signals_4h":         15,
			"max_signals_1d":         20,
			"chart_provider":         "coinglass",
		},
	}

	analyzer := analyzers.NewCounterAnalyzer(config, nil, nil)

	logger.Debug("   Конфигурация CounterAnalyzer:")
	fmt.Printf("      • Базовый период: %d мин\n", config.CustomSettings["base_period_minutes"])
	fmt.Printf("      • Период анализа: %s\n", config.CustomSettings["analysis_period"])
	fmt.Printf("      • Порог роста: %.2f%%\n", config.CustomSettings["growth_threshold"])
	fmt.Printf("      • Порог падения: %.2f%%\n", config.CustomSettings["fall_threshold"])
	fmt.Printf("      • Макс сигналов (15м): %d\n", config.CustomSettings["max_signals_15m"])

	// Тест 1: Рост
	logger.Debug("\n   📈 Тест 1: Сигналы роста")
	for i, data := range counterTestData.growthTest {
		signals, err := analyzer.Analyze(data, config)
		if err != nil {
			fmt.Printf("      ❌ Ошибка анализа роста %d: %v\n", i+1, err)
			continue
		}

		if len(signals) > 0 {
			fmt.Printf("      ✅ Тест роста %d: %d сигналов\n", i+1, len(signals))
			for _, signal := range signals {
				fmt.Printf("         • %s: %.4f%% (уверенность: %.1f%%)\n",
					signal.Direction, signal.ChangePercent, signal.Confidence)
			}
		} else {
			fmt.Printf("      ⚠️  Тест роста %d: нет сигналов\n", i+1)
		}
	}

	// Тест 2: Падение
	logger.Debug("\n   📉 Тест 2: Сигналы падения")
	for i, data := range counterTestData.fallTest {
		signals, err := analyzer.Analyze(data, config)
		if err != nil {
			fmt.Printf("      ❌ Ошибка анализа падения %d: %v\n", i+1, err)
			continue
		}

		if len(signals) > 0 {
			fmt.Printf("      ✅ Тест падения %d: %d сигналов\n", i+1, len(signals))
			for _, signal := range signals {
				fmt.Printf("         • %s: %.4f%% (уверенность: %.1f%%)\n",
					signal.Direction, signal.ChangePercent, signal.Confidence)
			}
		} else {
			fmt.Printf("      ⚠️  Тест падения %d: нет сигналов\n", i+1)
		}
	}

	// Тест 3: Смешанные данные
	logger.Debug("\n   🔄 Тест 3: Смешанные сигналы")
	signals, err := analyzer.Analyze(counterTestData.mixedTest, config)
	if err != nil {
		fmt.Printf("      ❌ Ошибка анализа смешанных данных: %v\n", err)
	} else {
		fmt.Printf("      📊 Смешанный тест: %d сигналов\n", len(signals))
		for _, signal := range signals {
			fmt.Printf("         • %s: %.4f%%\n", signal.Direction, signal.ChangePercent)
		}
	}

	// Тест 4: Получение статистики
	logger.Debug("\n   📊 Тест 4: Статистика счетчика")

	// Анализируем несколько раз для накопления счетчика
	for i := 0; i < 3; i++ {
		analyzer.Analyze(counterTestData.growthTest[0], config)
	}

	// Получаем статистику
	counters := analyzer.GetAllCounters()
	fmt.Printf("      • Всего счетчиков: %d\n", len(counters))

	for _, counter := range counters {
		fmt.Printf("      • %s: рост=%d, падение=%d\n",
			counter.Symbol, counter.GrowthCount, counter.FallCount)
	}

	// Тест 5: Сброс периода
	logger.Debug("\n   🔄 Тест 5: Сброс периода")
	analyzer.SetAnalysisPeriod(analyzers.Period5Min)
	fmt.Printf("      ✅ Период изменен на 5 минут\n")

	// Проверяем сброс счетчиков
	countersAfterReset := analyzer.GetAllCounters()
	fmt.Printf("      • Счетчиков после сброса: %d\n", len(countersAfterReset))
}

// Структура для тестовых данных счетчика
type counterTestDataStruct struct {
	growthTest [][]types.PriceData
	fallTest   [][]types.PriceData
	mixedTest  []types.PriceData
}

func createCounterTestData() counterTestDataStruct {
	now := time.Now()

	return counterTestDataStruct{
		// Тест роста (0.5% рост за 1 минуту)
		growthTest: [][]types.PriceData{
			{
				{Symbol: "BTCUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "BTCUSDT", Price: 100.5, Timestamp: now.Add(-1 * time.Minute)},
			},
			{
				{Symbol: "ETHUSDT", Price: 200.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "ETHUSDT", Price: 201.0, Timestamp: now.Add(-1 * time.Minute)},
			},
		},

		// Тест падения (0.5% падение за 1 минуту)
		fallTest: [][]types.PriceData{
			{
				{Symbol: "BTCUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "BTCUSDT", Price: 99.5, Timestamp: now.Add(-1 * time.Minute)},
			},
			{
				{Symbol: "ETHUSDT", Price: 200.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "ETHUSDT", Price: 199.0, Timestamp: now.Add(-1 * time.Minute)},
			},
		},

		// Смешанный тест
		mixedTest: []types.PriceData{
			{Symbol: "BTCUSDT", Price: 100.0, Timestamp: now.Add(-3 * time.Minute)},
			{Symbol: "BTCUSDT", Price: 100.3, Timestamp: now.Add(-2 * time.Minute)},
			{Symbol: "BTCUSDT", Price: 99.8, Timestamp: now.Add(-1 * time.Minute)},
		},
	}
}

func createTestData() []types.PriceData {
	now := time.Now()
	return []types.PriceData{
		{
			Symbol:    "BTCUSDT",
			Price:     100.0,
			Volume24h: 1000000,
			Timestamp: now.Add(-5 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     101.0, // +1% рост
			Volume24h: 1100000,
			Timestamp: now.Add(-4 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     102.0, // еще +1% рост
			Volume24h: 1200000,
			Timestamp: now.Add(-3 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     101.5, // -0.5% падение
			Volume24h: 1150000,
			Timestamp: now.Add(-2 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     100.5, // еще -1% падение
			Volume24h: 1050000,
			Timestamp: now.Add(-1 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     101.0, // +0.5% рост
			Volume24h: 1100000,
			Timestamp: now,
		},
	}
}

func createTestDataForFall() []types.PriceData {
	now := time.Now()
	return []types.PriceData{
		{
			Symbol:    "BTCUSDT",
			Price:     100.0,
			Volume24h: 1000000,
			Timestamp: now.Add(-5 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     101.0, // +1%
			Volume24h: 1100000,
			Timestamp: now.Add(-4 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     100.0, // -1% - ЯВНОЕ ПАДЕНИЕ
			Volume24h: 900000,
			Timestamp: now.Add(-3 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     99.0, // -1% - ЕЩЕ ПАДЕНИЕ
			Volume24h: 800000,
			Timestamp: now.Add(-2 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     99.5, // +0.5%
			Volume24h: 850000,
			Timestamp: now.Add(-1 * time.Minute),
		},
		{
			Symbol:    "BTCUSDT",
			Price:     99.0, // -0.5%
			Volume24h: 800000,
			Timestamp: now,
		},
	}
}

func testNewFallAnalyzer() {
	logger.Debug("\n🧪 ТЕСТ НОВОГО FALL ANALYZER (версия 2.0):")

	data := createTestDataForFall()

	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        1.0,
		MinConfidence: 1.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_fall":             0.01,
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	analyzer := analyzers.NewFallAnalyzer(config)

	logger.Debug("   📊 Тестовые данные:")
	for i, point := range data {
		fmt.Printf("      %d. %.2f (объем: %.0f) время: %v\n",
			i+1, point.Price, point.Volume24h,
			point.Timestamp.Format("15:04:05"))
	}

	signals, err := analyzer.Analyze(data, config)
	if err != nil {
		fmt.Printf("   ❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("   📊 Результаты: %d сигналов\n", len(signals))

	if len(signals) == 0 {
		logger.Debug("   ⚠️  НЕТ СИГНАЛОВ!")

		logger.Debug("   📈 Все изменения:")
		for i := 1; i < len(data); i++ {
			change := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
			trend := "↑"
			if change < 0 {
				trend = "↓"
			}
			fmt.Printf("      %d→%d: %.2f → %.2f (%s%.4f%%)\n",
				i-1, i, data[i-1].Price, data[i].Price, trend, change)
		}
	}

	for i, signal := range signals {
		fmt.Printf("      Сигнал %d:\n", i+1)
		fmt.Printf("      • Символ: %s\n", signal.Symbol)
		fmt.Printf("      • Тип: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
		fmt.Printf("      • Уверенность: %.1f%%\n", signal.Confidence)
		fmt.Printf("      • Период: %d мин\n", signal.Period)
		fmt.Printf("      • Начало: %.2f → Конец: %.2f\n",
			signal.StartPrice, signal.EndPrice)

		if signal.ChangePercent > 0 && signal.Direction == "down" {
			logger.Debug("      ⚠️  ВНИМАНИЕ: ChangePercent положительный при падении!")
		}
	}
}

func testGrowthAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        1.0,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_growth":           0.01,
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	analyzer := analyzers.NewGrowthAnalyzer(config)

	logger.Debug("   Конфигурация:")
	fmt.Printf("      • MinGrowth: %.2f%%\n", config.CustomSettings["min_growth"])
	fmt.Printf("      • MinConfidence: %.1f%%\n", config.MinConfidence)
	fmt.Printf("      • MinDataPoints: %d\n", config.MinDataPoints)

	signals, err := analyzer.Analyze(data, config)
	if err != nil {
		fmt.Printf("   ❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("   📊 Результаты: %d сигналов\n", len(signals))

	for i, signal := range signals {
		fmt.Printf("      Сигнал %d:\n", i+1)
		fmt.Printf("      • Символ: %s\n", signal.Symbol)
		fmt.Printf("      • Тип: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
		fmt.Printf("      • Уверенность: %.1f%%\n", signal.Confidence)
		fmt.Printf("      • Точки данных: %d\n", signal.DataPoints)

		if len(data) > 0 {
			startPrice := data[0].Price
			endPrice := data[len(data)-1].Price
			actualChange := ((endPrice - startPrice) / startPrice) * 100
			fmt.Printf("      • Фактическое изменение: %.4f%%\n", actualChange)
		}
	}

	if len(signals) == 0 {
		logger.Debug("   ⚠️  Нет сигналов, даже с порогом 0.01%!")
		logger.Debug("   🔍 Проблемы с анализатором роста!")
	}
}

func testFallAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        1.0,
		MinConfidence: 1.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_fall":             0.001,
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	analyzer := analyzers.NewFallAnalyzer(config)

	logger.Debug("   Конфигурация:")
	fmt.Printf("      • MinFall: %.3f%%\n", config.CustomSettings["min_fall"])
	fmt.Printf("      • MinConfidence: %.1f%%\n", config.MinConfidence)
	fmt.Printf("      • Вес: %.1f\n", config.Weight)

	signals, err := analyzer.Analyze(data, config)
	if err != nil {
		fmt.Printf("   ❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("   📊 Результаты: %d сигналов\n", len(signals))

	logger.Debug("   📈 Анализ данных:")
	for i, point := range data {
		if i > 0 {
			change := ((point.Price - data[i-1].Price) / data[i-1].Price) * 100
			trend := "↑"
			if change < 0 {
				trend = "↓"
			}
			fmt.Printf("      %d → %d: %.2f → %.2f (%s%.4f%%)\n",
				i, i+1, data[i-1].Price, point.Price, trend, change)
		}
	}

	totalChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100
	fmt.Printf("   📊 Общее изменение: %.4f%%\n", totalChange)

	for i, signal := range signals {
		fmt.Printf("      Сигнал %d:\n", i+1)
		fmt.Printf("      • Символ: %s\n", signal.Symbol)
		fmt.Printf("      • Тип: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
		fmt.Printf("      • Уверенность: %.1f%%\n", signal.Confidence)

		if signal.ChangePercent > 0 && signal.Direction == "down" {
			fmt.Printf("      ⚠️  Внимание: ChangePercent положительный для падения!\n")
		}
	}

	if len(signals) == 0 {
		logger.Debug("   ⚠️  Нет сигналов падения!")
		logger.Debug("   🔍 Возможные причины:")
		logger.Debug("      • ChangePercent должен быть отрицательным для падения")
		logger.Debug("      • Порог min_fall слишком высокий")
		logger.Debug("      • Анализатор неправильно рассчитывает изменения")
		logger.Debug("      • Не учитываются промежуточные падения")
	}
}

func testContinuousAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 1.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_continuous_points": 2,
			"max_gap_ratio":         0.3,
		},
	}

	analyzer := analyzers.NewContinuousAnalyzer(config)

	logger.Debug("   Конфигурация:")
	fmt.Printf("      • MinContinuousPoints: %d\n", config.CustomSettings["min_continuous_points"])
	fmt.Printf("      • MinConfidence: %.1f%%\n", config.MinConfidence)

	signals, err := analyzer.Analyze(data, config)
	if err != nil {
		fmt.Printf("   ❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("   📊 Результаты: %d сигналов\n", len(signals))

	logger.Debug("   📈 Анализ непрерывности:")
	for i := 1; i < len(data); i++ {
		change1 := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100

		if i+1 < len(data) {
			change2 := ((data[i+1].Price - data[i].Price) / data[i].Price) * 100

			if change1 > 0 && change2 > 0 {
				fmt.Printf("      %d-%d-%d: РОСТ %.4f%% → %.4f%%\n",
					i-1, i, i+1, change1, change2)
			} else if change1 < 0 && change2 < 0 {
				fmt.Printf("      %d-%d-%d: ПАДЕНИЕ %.4f%% → %.4f%%\n",
					i-1, i, i+1, change1, change2)
			}
		}
	}

	for i, signal := range signals {
		fmt.Printf("      Сигнал %d:\n", i+1)
		fmt.Printf("      • Символ: %s\n", signal.Symbol)
		fmt.Printf("      • Тип: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Непрерывный: %v\n", signal.Metadata.IsContinuous)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
		fmt.Printf("      • Уверенность: %.1f%%\n", signal.Confidence)
	}

	if len(signals) == 0 {
		logger.Debug("   ⚠️  Нет сигналов непрерывности!")
		logger.Debug("   🔍 В данных есть последовательные изменения:")
		logger.Debug("      - Рост: точки 0→1→2 (+1% → +1%)")
		logger.Debug("      - Падение: точки 2→3→4 (-0.5% → -1%)")
	}
}
func testVolumeAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.5,
		MinConfidence: 30.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_volume":              100000.0, // Минимальный объем
			"volume_change_threshold": 50.0,     // Порог изменения объема
		},
	}

	analyzer := analyzers.NewVolumeAnalyzer(config)

	logger.Debug("   Конфигурация:")
	fmt.Printf("      • MinVolume: %.0f\n", config.CustomSettings["min_volume"])
	fmt.Printf("      • VolumeChangeThreshold: %.0f%%\n", config.CustomSettings["volume_change_threshold"])
	fmt.Printf("      • MinConfidence: %.1f%%\n", config.MinConfidence)

	// Покажем объемы
	logger.Debug("   📊 Объемы данных:")
	for i, point := range data {
		fmt.Printf("      %d. Цена: %.2f, Объем: %.0f\n",
			i+1, point.Price, point.Volume24h)
	}

	signals, err := analyzer.Analyze(data, config)
	if err != nil {
		fmt.Printf("   ❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("   📊 Результаты: %d сигналов\n", len(signals))

	for i, signal := range signals {
		fmt.Printf("      Сигнал %d:\n", i+1)
		fmt.Printf("      • Символ: %s\n", signal.Symbol)
		fmt.Printf("      • Тип: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Уверенность: %.1f%%\n", signal.Confidence)

		if avgVolume, ok := signal.Metadata.Indicators["avg_volume"]; ok {
			fmt.Printf("      • Средний объем: %.0f\n", avgVolume)
		}

		if volumeChange, ok := signal.Metadata.Indicators["volume_change"]; ok {
			fmt.Printf("      • Изменение объема: %.1f%%\n", volumeChange)

			// Проверяем значительное изменение объема
			threshold := config.CustomSettings["volume_change_threshold"].(float64)
			if math.Abs(volumeChange) > threshold {
				fmt.Printf("      ⚡ ЗНАЧИТЕЛЬНОЕ ИЗМЕНЕНИЕ ОБЪЕМА!\n")
			}
		}
	}

	if len(signals) == 0 {
		logger.Debug("   ⚠️  Нет сигналов объема!")
		// Рассчитаем средний объем вручную
		var totalVolume float64
		hasVolume := false
		for _, point := range data {
			if point.Volume24h > 0 {
				totalVolume += point.Volume24h
				hasVolume = true
			}
		}

		if hasVolume {
			avgVolume := totalVolume / float64(len(data))
			fmt.Printf("   📈 Средний объем: %.0f\n", avgVolume)
			fmt.Printf("   🔍 Минимальный порог: %.0f\n", config.CustomSettings["min_volume"])

			if avgVolume < config.CustomSettings["min_volume"].(float64) {
				logger.Debug("   💡 Объем ниже минимального порога!")
			}
		} else {
			logger.Debug("   💡 В данных нет информации об объеме!")
		}
	}
}
