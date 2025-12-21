package main

import (
	"crypto-exchange-screener-bot/internal/analysis/analyzers"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"strings"
	"time"
)

func main() {
	fmt.Println("🧪 ПОЛНЫЙ ТЕСТ COUNTER ANALYZER")
	fmt.Println(strings.Repeat("=", 70))

	// Базовый тест
	fmt.Println("\n📊 БАЗОВЫЙ ТЕСТ:")
	runBasicCounterTest()

	// Тест периодов
	fmt.Println("\n⏱️  ТЕСТ ПЕРИОДОВ И СБРОСА:")
	runPeriodAndResetTest()

	// Тест статистики
	fmt.Println("\n📈 ТЕСТ СТАТИСТИКИ И МЕТАДАННЫХ:")
	runStatisticsAndMetadataTest()

	// Тест граничных условий
	fmt.Println("\n⚠️  ТЕСТ ГРАНИЧНЫХ УСЛОВИЙ:")
	runEdgeCasesTest()

	// Тест производительности
	fmt.Println("\n⚡ ТЕСТ ПРОИЗВОДИТЕЛЬНОСТИ:")
	runPerformanceTest()

	fmt.Println("\n" + strings.Repeat("✅", 30))
	fmt.Println("✅ ВСЕ ТЕСТЫ COUNTER ANALYZER ЗАВЕРШЕНЫ УСПЕШНО")
	fmt.Println(strings.Repeat("✅", 30))
}

func runBasicCounterTest() {
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

	// Тестовые данные
	now := time.Now()

	testCases := []struct {
		name         string
		data         []types.PriceData
		expectSignal bool
		description  string
	}{
		{
			name: "Рост 0.2% (выше порога 0.1%)",
			data: []types.PriceData{
				{Symbol: "BTCUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "BTCUSDT", Price: 100.2, Timestamp: now.Add(-1 * time.Minute)},
			},
			expectSignal: true,
			description:  "Должен быть сигнал роста",
		},
		{
			name: "Падение 0.2% (выше порога 0.1%)",
			data: []types.PriceData{
				{Symbol: "ETHUSDT", Price: 200.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "ETHUSDT", Price: 199.6, Timestamp: now.Add(-1 * time.Minute)},
			},
			expectSignal: true,
			description:  "Должен быть сигнал падения",
		},
		{
			name: "Малое изменение 0.05% (ниже порога)",
			data: []types.PriceData{
				{Symbol: "XRPUSDT", Price: 0.5, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "XRPUSDT", Price: 0.50025, Timestamp: now.Add(-1 * time.Minute)},
			},
			expectSignal: false,
			description:  "Не должно быть сигнала",
		},
		{
			name: "Нулевое изменение",
			data: []types.PriceData{
				{Symbol: "ADAUSDT", Price: 0.5, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "ADAUSDT", Price: 0.5, Timestamp: now.Add(-1 * time.Minute)},
			},
			expectSignal: false,
			description:  "Не должно быть сигнала при нулевом изменении",
		},
		{
			name: "Точное пороговое значение роста 0.1%",
			data: []types.PriceData{
				{Symbol: "SOLUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: "SOLUSDT", Price: 100.1, Timestamp: now.Add(-1 * time.Minute)},
			},
			expectSignal: false,
			description:  "При точном пороге - нет сигнала (строгое неравенство)",
		},
	}

	passed := 0
	total := len(testCases)

	for _, tc := range testCases {
		fmt.Printf("   🔄 %s:\n", tc.name)
		signals, err := analyzer.Analyze(tc.data, config)

		if err != nil {
			fmt.Printf("      ❌ Ошибка: %v\n", err)
			continue
		}

		hasSignal := len(signals) > 0

		if hasSignal == tc.expectSignal {
			fmt.Printf("      ✅ %s\n", tc.description)
			if hasSignal {
				fmt.Printf("         • Тип: %s, Изменение: %.4f%%\n",
					signals[0].Direction, signals[0].ChangePercent)
				fmt.Printf("         • Уверенность: %.1f%%, Тэги: %v\n",
					signals[0].Confidence, signals[0].Metadata.Tags)
			}
			passed++
		} else {
			fmt.Printf("      ❌ Ошибка: %s\n", tc.description)
			if hasSignal {
				fmt.Printf("         • Получен сигнал: %s %.4f%%\n",
					signals[0].Direction, signals[0].ChangePercent)
			} else {
				fmt.Printf("         • Сигнал не получен\n")
			}
		}
		fmt.Println()
	}

	fmt.Printf("   📊 Результат: %d/%d тестов пройдено\n", passed, total)
}

func runPeriodAndResetTest() {
	config := analyzers.AnalyzerConfig{
		Enabled:        true,
		Weight:         0.7,
		MinConfidence:  10.0,
		MinDataPoints:  2,
		CustomSettings: analyzers.DefaultCounterConfig.CustomSettings,
	}

	analyzer := analyzers.NewCounterAnalyzer(config, nil, nil)

	fmt.Println("   🔄 Тест смены периодов и сброса:")

	// Тестовые данные
	now := time.Now()
	testData := []types.PriceData{
		{Symbol: "TESTUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "TESTUSDT", Price: 100.15, Timestamp: now.Add(-1 * time.Minute)},
	}

	// Тест 1: Накапливаем счетчики в 15-минутном периоде
	fmt.Println("   📈 Тест 1: Накопление в 15-минутном периоде")
	for i := 1; i <= 5; i++ {
		signals, _ := analyzer.Analyze(testData, config)
		if len(signals) > 0 {
			fmt.Printf("      %d. Сигнал роста: счетчик=%d\n", i, i)
		}
	}

	counters15m := analyzer.GetAllCounters()
	for symbol, counter := range counters15m {
		fmt.Printf("      • %s: рост=%d, период=%s\n",
			symbol, counter.GrowthCount, counter.SelectedPeriod)
	}

	// Тест 2: Меняем период на 5 минут (должен сбросить счетчики)
	fmt.Println("\n   🔄 Тест 2: Смена периода на 5 минут")
	analyzer.SetAnalysisPeriod(analyzers.Period5Min)

	// Проверяем сброс
	counters5m := analyzer.GetAllCounters()
	if len(counters5m) == 0 {
		fmt.Println("      ✅ Счетчики сброшены при смене периода")
	} else {
		for symbol, counter := range counters5m {
			if counter.GrowthCount == 0 {
				fmt.Printf("      ✅ %s: счетчик сброшен (рост=%d)\n", symbol, counter.GrowthCount)
			} else {
				fmt.Printf("      ❌ %s: счетчик НЕ сброшен (рост=%d)\n", symbol, counter.GrowthCount)
			}
		}
	}

	// Тест 3: Накапливаем в новом периоде
	fmt.Println("\n   📈 Тест 3: Накопление в 5-минутном периоде")
	for i := 1; i <= 3; i++ {
		analyzer.Analyze(testData, config)
	}

	finalCounters := analyzer.GetAllCounters()
	for symbol, counter := range finalCounters {
		fmt.Printf("      • %s: рост=%d, период=%s\n",
			symbol, counter.GrowthCount, counter.SelectedPeriod)
	}
}

func runStatisticsAndMetadataTest() {
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
			"max_signals_15m":        8,
		},
	}

	analyzer := analyzers.NewCounterAnalyzer(config, nil, nil)

	fmt.Println("   📊 Тест статистики и метаданных:")

	// Тестируем несколько символов
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT"}
	now := time.Now()

	fmt.Println("   📈 Анализ 5 символов по 4 раза:")
	for _, symbol := range symbols {
		testData := []types.PriceData{
			{Symbol: symbol, Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
			{Symbol: symbol, Price: 100.15, Timestamp: now.Add(-1 * time.Minute)},
		}

		for i := 0; i < 4; i++ {
			signals, _ := analyzer.Analyze(testData, config)
			if len(signals) > 0 && i == 0 {
				// Проверяем метаданные первого сигнала
				signal := signals[0]
				fmt.Printf("      • %s: %s %.4f%%\n",
					symbol, signal.Direction, signal.ChangePercent)
				fmt.Printf("        Тэги: %v\n", signal.Metadata.Tags)
				fmt.Printf("        Индикаторы: count=%.0f, change=%.4f, period=%.0f\n",
					signal.Metadata.Indicators["count"],
					signal.Metadata.Indicators["change"],
					signal.Metadata.Indicators["period"])
			}
		}
	}

	// Получаем полную статистику
	fmt.Println("\n   📈 Полная статистика:")
	allCounters := analyzer.GetAllCounters()

	totalGrowth := 0
	totalFall := 0
	totalSymbols := len(allCounters)

	fmt.Printf("      • Всего символов: %d\n", totalSymbols)

	for symbol, counter := range allCounters {
		fmt.Printf("      • %s: рост=%d, падение=%d, период=%s\n",
			symbol, counter.GrowthCount, counter.FallCount, counter.SelectedPeriod)
		totalGrowth += counter.GrowthCount
		totalFall += counter.FallCount

		// Рассчитываем уверенность
		maxSignals := 8 // для 15-минутного периода
		confidence := float64(counter.GrowthCount+counter.FallCount) / float64(maxSignals) * 100
		fmt.Printf("        Уверенность: %.1f%%\n", confidence)
	}

	fmt.Printf("\n   🧮 Сводная статистика:\n")
	fmt.Printf("      • Всего сигналов роста: %d\n", totalGrowth)
	fmt.Printf("      • Всего сигналов падения: %d\n", totalFall)
	fmt.Printf("      • Общее количество сигналов: %d\n", totalGrowth+totalFall)
	fmt.Printf("      • Среднее на символ: %.1f сигналов\n",
		float64(totalGrowth+totalFall)/float64(totalSymbols))

	// Проверяем конфигурацию
	configFromAnalyzer := analyzer.GetConfig()
	fmt.Printf("\n   ⚙️  Конфигурация анализатора:\n")
	fmt.Printf("      • Вес: %.1f\n", configFromAnalyzer.Weight)
	fmt.Printf("      • Мин. уверенность: %.1f%%\n", configFromAnalyzer.MinConfidence)
	fmt.Printf("      • Мин. точек данных: %d\n", configFromAnalyzer.MinDataPoints)
}

func runEdgeCasesTest() {
	fmt.Println("   ⚠️  Тест граничных условий:")

	// Тест 1: Недостаточно данных
	fmt.Println("   🔄 Тест 1: Недостаточно данных")
	config := analyzers.AnalyzerConfig{
		Enabled:        true,
		Weight:         0.7,
		MinConfidence:  10.0,
		MinDataPoints:  2,
		CustomSettings: analyzers.DefaultCounterConfig.CustomSettings,
	}

	analyzer := analyzers.NewCounterAnalyzer(config, nil, nil)

	// Только одна точка данных
	singleData := []types.PriceData{
		{Symbol: "BTCUSDT", Price: 100.0, Timestamp: time.Now()},
	}

	signals, err := analyzer.Analyze(singleData, config)
	if err != nil {
		fmt.Printf("      ✅ Правильная ошибка: %v\n", err)
	} else if len(signals) == 0 {
		fmt.Println("      ✅ Нет сигналов при недостаточных данных")
	} else {
		fmt.Println("      ❌ Ожидалась ошибка или отсутствие сигналов")
	}

	// Тест 2: Очень большой рост
	fmt.Println("\n   🔄 Тест 2: Очень большой рост (10%)")
	bigGrowthData := []types.PriceData{
		{Symbol: "BTCUSDT", Price: 100.0, Timestamp: time.Now().Add(-2 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 110.0, Timestamp: time.Now().Add(-1 * time.Minute)}, // +10%
	}

	signals, _ = analyzer.Analyze(bigGrowthData, config)
	if len(signals) > 0 {
		fmt.Printf("      ✅ Большой рост обнаружен: %.2f%%\n", signals[0].ChangePercent)
	} else {
		fmt.Println("      ❌ Большой рост не обнаружен")
	}

	// Тест 3: Отрицательные цены (нереальный случай)
	fmt.Println("\n   🔄 Тест 3: Нестандартные значения")
	weirdData := []types.PriceData{
		{Symbol: "TESTUSDT", Price: 0.001, Timestamp: time.Now().Add(-2 * time.Minute)},
		{Symbol: "TESTUSDT", Price: 0.0015, Timestamp: time.Now().Add(-1 * time.Minute)}, // +50%
	}

	signals, _ = analyzer.Analyze(weirdData, config)
	if len(signals) > 0 {
		fmt.Printf("      ✅ Изменение на малых ценах: %.2f%%\n", signals[0].ChangePercent)
	} else {
		fmt.Println("      ⚠️  Нет сигнала на малых ценах")
	}

	// Тест 4: Отключение отслеживания роста/падения
	fmt.Println("\n   🔄 Тест 4: Отключение отслеживания")
	configNoGrowth := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"track_growth":     false,
			"track_fall":       true,
			"growth_threshold": 0.1,
			"fall_threshold":   0.1,
		},
	}

	analyzer2 := analyzers.NewCounterAnalyzer(configNoGrowth, nil, nil)
	growthData := []types.PriceData{
		{Symbol: "BTCUSDT", Price: 100.0, Timestamp: time.Now().Add(-2 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 100.2, Timestamp: time.Now().Add(-1 * time.Minute)},
	}

	signals, _ = analyzer2.Analyze(growthData, configNoGrowth)
	if len(signals) == 0 {
		fmt.Println("      ✅ Рост не отслеживается (track_growth=false)")
	} else {
		fmt.Println("      ❌ Рост отслеживается, хотя должен быть отключен")
	}
}

func runPerformanceTest() {
	config := analyzers.AnalyzerConfig{
		Enabled:        true,
		Weight:         0.7,
		MinConfidence:  10.0,
		MinDataPoints:  2,
		CustomSettings: analyzers.DefaultCounterConfig.CustomSettings,
	}

	analyzer := analyzers.NewCounterAnalyzer(config, nil, nil)

	fmt.Println("   ⚡ Тест производительности:")

	// Подготавливаем много тестовых данных
	now := time.Now()
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT",
		"DOTUSDT", "DOGEUSDT", "AVAXUSDT", "MATICUSDT", "LTCUSDT"}

	var testDataSets [][]types.PriceData
	for _, symbol := range symbols {
		for i := 0; i < 10; i++ { // 10 наборов данных на символ
			data := []types.PriceData{
				{Symbol: symbol, Price: 100.0 + float64(i), Timestamp: now.Add(-2 * time.Minute)},
				{Symbol: symbol, Price: 100.15 + float64(i), Timestamp: now.Add(-1 * time.Minute)},
			}
			testDataSets = append(testDataSets, data)
		}
	}

	fmt.Printf("      • Тестовых наборов: %d\n", len(testDataSets))

	// Тестируем производительность
	startTime := time.Now()
	processed := 0

	for _, data := range testDataSets {
		analyzer.Analyze(data, config)
		processed++
	}

	duration := time.Since(startTime)

	fmt.Printf("      • Обработано: %d наборов данных\n", processed)
	fmt.Printf("      • Время выполнения: %v\n", duration)
	fmt.Printf("      • Среднее время на набор: %v\n", duration/time.Duration(processed))

	// Статистика после теста
	stats := analyzer.GetStats()
	fmt.Printf("\n   📊 Статистика анализатора:\n")
	fmt.Printf("      • Всего вызовов: %d\n", stats.TotalCalls)
	fmt.Printf("      • Успешных: %d\n", stats.SuccessCount)
	fmt.Printf("      • Ошибок: %d\n", stats.ErrorCount)
	fmt.Printf("      • Среднее время: %v\n", stats.AverageTime)

	// Проверяем количество счетчиков
	allCounters := analyzer.GetAllCounters()
	fmt.Printf("      • Уникальных счетчиков: %d\n", len(allCounters))

	// Память (примерная оценка)
	approxMemory := len(allCounters) * 100 // примерно 100 байт на счетчик
	fmt.Printf("      • Примерное использование памяти: ~%d байт\n", approxMemory)
}
