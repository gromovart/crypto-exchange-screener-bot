package main

import (
	"crypto-exchange-screener-bot/internal/analysis/analyzers"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"strings"
	"time"
)

func main() {
	fmt.Println("🧪 ТЕСТИРОВАНИЕ COUNTER ANALYZER")
	fmt.Println(strings.Repeat("=", 60))

	// Базовый тест
	fmt.Println("\n📊 БАЗОВЫЙ ТЕСТ:")
	runBasicCounterTest()

	// Тест периодов
	fmt.Println("\n⏱️  ТЕСТ ПЕРИОДОВ:")
	runPeriodTest()

	// Тест статистики
	fmt.Println("\n📈 ТЕСТ СТАТИСТИКИ:")
	runStatisticsTest()

	fmt.Println("\n✅ Тестирование CounterAnalyzer завершено")
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

	fmt.Println("   🔄 Тест 1: Рост 0.2%")
	testData1 := []types.PriceData{
		{Symbol: "BTCUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "BTCUSDT", Price: 100.2, Timestamp: now.Add(-1 * time.Minute)},
	}

	signals1, err1 := analyzer.Analyze(testData1, config)
	if err1 != nil {
		fmt.Printf("      ❌ Ошибка: %v\n", err1)
	} else if len(signals1) > 0 {
		fmt.Printf("      ✅ Обнаружен рост: %.4f%%\n", signals1[0].ChangePercent)
	} else {
		fmt.Println("      ⚠️  Сигнал не обнаружен")
	}

	fmt.Println("\n   🔄 Тест 2: Падение 0.2%")
	testData2 := []types.PriceData{
		{Symbol: "ETHUSDT", Price: 200.0, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "ETHUSDT", Price: 199.6, Timestamp: now.Add(-1 * time.Minute)},
	}

	signals2, err2 := analyzer.Analyze(testData2, config)
	if err2 != nil {
		fmt.Printf("      ❌ Ошибка: %v\n", err2)
	} else if len(signals2) > 0 {
		fmt.Printf("      ✅ Обнаружено падение: %.4f%%\n", signals2[0].ChangePercent)
	} else {
		fmt.Println("      ⚠️  Сигнал не обнаружен")
	}

	fmt.Println("\n   🔄 Тест 3: Малое изменение 0.05%")
	testData3 := []types.PriceData{
		{Symbol: "XRPUSDT", Price: 0.5, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "XRPUSDT", Price: 0.50025, Timestamp: now.Add(-1 * time.Minute)},
	}

	signals3, err3 := analyzer.Analyze(testData3, config)
	if err3 != nil {
		fmt.Printf("      ❌ Ошибка: %v\n", err3)
	} else if len(signals3) == 0 {
		fmt.Println("      ✅ Правильно: нет сигнала (ниже порога 0.1%)")
	} else {
		fmt.Printf("      ⚠️  Неожиданный сигнал: %.4f%%\n", signals3[0].ChangePercent)
	}
}

func runPeriodTest() {
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

	fmt.Println("   ⏱️  Тестирование периодов:")

	periods := []struct {
		period types.CounterPeriod
		name   string
	}{
		{types.Period5Min, "5 минут"},
		{types.Period15Min, "15 минут"},
		{types.Period30Min, "30 минут"},
		{types.Period1Hour, "1 час"},
		{types.Period4Hours, "4 часа"},
		{types.Period1Day, "1 день"},
	}

	for _, p := range periods {
		analyzer.SetAnalysisPeriod(p.period)
		fmt.Printf("      ✅ Установлен период: %s\n", p.name)

		// Простая проверка - анализируем тестовые данные
		testData := []types.PriceData{
			{Symbol: "TESTUSDT", Price: 100.0, Timestamp: time.Now().Add(-2 * time.Minute)},
			{Symbol: "TESTUSDT", Price: 100.15, Timestamp: time.Now().Add(-1 * time.Minute)},
		}

		signals, _ := analyzer.Analyze(testData, config)
		if len(signals) > 0 {
			fmt.Printf("         • Сигнал: %s %.4f%%\n", signals[0].Direction, signals[0].ChangePercent)
		}
	}
}

func runStatisticsTest() {
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

	fmt.Println("   📈 Тест статистики:")

	// Создаем тестовые данные для нескольких символов
	now := time.Now()
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}

	for _, symbol := range symbols {
		testData := []types.PriceData{
			{Symbol: symbol, Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
			{Symbol: symbol, Price: 100.15, Timestamp: now.Add(-1 * time.Minute)},
		}

		// Анализируем 3 раза для накопления счетчика
		for i := 0; i < 3; i++ {
			analyzer.Analyze(testData, config)
		}
		fmt.Printf("      • %s: проанализирован 3 раза\n", symbol)
	}

	// Получаем всю статистику
	allCounters := analyzer.GetAllCounters()
	fmt.Printf("   📊 Результаты:\n")
	fmt.Printf("      • Всего счетчиков: %d\n", len(allCounters))

	totalGrowth := 0
	totalFall := 0

	for symbol, counter := range allCounters {
		fmt.Printf("      • %s: рост=%d, падение=%d\n",
			symbol, counter.GrowthCount, counter.FallCount)
		totalGrowth += counter.GrowthCount
		totalFall += counter.FallCount
	}

	fmt.Printf("   🧮 ИТОГО:\n")
	fmt.Printf("      • Всего сигналов роста: %d\n", totalGrowth)
	fmt.Printf("      • Всего сигналов падения: %d\n", totalFall)
	fmt.Printf("      • Общее количество сигналов: %d\n", totalGrowth+totalFall)
}
