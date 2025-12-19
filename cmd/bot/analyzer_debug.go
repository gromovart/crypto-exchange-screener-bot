package main

import (
	"crypto-exchange-screener-bot/internal/analysis/analyzers"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"strings"
	"time"
)

func main() {
	fmt.Println("🔧 ТЕСТИРОВАНИЕ АНАЛИЗАТОРОВ")
	fmt.Println(strings.Repeat("=", 60))

	// Тестовые данные
	testData := createTestData()

	// Тестируем GrowthAnalyzer
	fmt.Println("\n🧪 ТЕСТ GROWTH ANALYZER:")
	testGrowthAnalyzer(testData)

	// Тестируем FallAnalyzer
	fmt.Println("\n🧪 ТЕСТ FALL ANALYZER:")
	testFallAnalyzer(testData)

	// Тестируем ContinuousAnalyzer
	fmt.Println("\n🧪 ТЕСТ CONTINUOUS ANALYZER:")
	testContinuousAnalyzer(testData)

	fmt.Println("\n✅ Тестирование завершено")
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

func testGrowthAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        1.0,
		MinConfidence: 10.0, // Очень низкая уверенность
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_growth":           0.01, // Всего 0.01%!
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	analyzer := analyzers.NewGrowthAnalyzer(config)

	fmt.Println("   Конфигурация:")
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

		// Проверяем данные
		if len(data) > 0 {
			startPrice := data[0].Price
			endPrice := data[len(data)-1].Price
			actualChange := ((endPrice - startPrice) / startPrice) * 100
			fmt.Printf("      • Фактическое изменение: %.4f%%\n", actualChange)
		}
	}

	if len(signals) == 0 {
		fmt.Println("   ⚠️  Нет сигналов, даже с порогом 0.01%!")
		fmt.Println("   🔍 Проблемы с анализатором роста!")
	}
}

func testFallAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        1.0,
		MinConfidence: 1.0, // СНИЖАЕМ до 1%!
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_fall":             0.001, // ЕЩЕ НИЖЕ - 0.001%!
			"continuity_threshold": 0.5,
			"volume_weight":        0.2,
		},
	}

	analyzer := analyzers.NewFallAnalyzer(config)

	fmt.Println("   Конфигурация:")
	fmt.Printf("      • MinFall: %.3f%%\n", config.CustomSettings["min_fall"])
	fmt.Printf("      • MinConfidence: %.1f%%\n", config.MinConfidence)
	fmt.Printf("      • Вес: %.1f\n", config.Weight)

	signals, err := analyzer.Analyze(data, config)
	if err != nil {
		fmt.Printf("   ❌ Ошибка: %v\n", err)
		return
	}

	fmt.Printf("   📊 Результаты: %d сигналов\n", len(signals))

	// Детальная информация о данных
	fmt.Println("   📈 Анализ данных:")
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

	// Общее изменение
	totalChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100
	fmt.Printf("   📊 Общее изменение: %.4f%%\n", totalChange)

	for i, signal := range signals {
		fmt.Printf("      Сигнал %d:\n", i+1)
		fmt.Printf("      • Символ: %s\n", signal.Symbol)
		fmt.Printf("      • Тип: %s\n", signal.Type)
		fmt.Printf("      • Направление: %s\n", signal.Direction)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
		fmt.Printf("      • Уверенность: %.1f%%\n", signal.Confidence)

		// Обратите внимание: для падения ChangePercent должен быть отрицательным
		if signal.ChangePercent > 0 && signal.Direction == "down" {
			fmt.Printf("      ⚠️  Внимание: ChangePercent положительный для падения!\n")
		}
	}

	if len(signals) == 0 {
		fmt.Println("   ⚠️  Нет сигналов падения!")
		fmt.Println("   🔍 Возможные причины:")
		fmt.Println("      • ChangePercent должен быть отрицательным для падения")
		fmt.Println("      • Порог min_fall слишком высокий")
		fmt.Println("      • Анализатор неправильно рассчитывает изменения")
		fmt.Println("      • Не учитываются промежуточные падения")
	}
}

func testContinuousAnalyzer(data []types.PriceData) {
	config := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"min_continuous_points": 2,
			"max_gap_ratio":         0.3,
		},
	}

	analyzer := analyzers.NewContinuousAnalyzer(config)

	fmt.Println("   Конфигурация:")
	fmt.Printf("      • MinContinuousPoints: %d\n", config.CustomSettings["min_continuous_points"])

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
		fmt.Printf("      • Непрерывный: %v\n", signal.Metadata.IsContinuous)
		fmt.Printf("      • Изменение: %.4f%%\n", signal.ChangePercent)
	}
}
