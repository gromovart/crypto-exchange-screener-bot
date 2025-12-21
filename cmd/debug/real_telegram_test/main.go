// cmd/debug/real_telegram_test/main.go
package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	// Парсинг флагов
	var (
		configPath = flag.String("config", ".env", "Путь к файлу конфигурации")
		testCount  = flag.Int("count", 3, "Количество тестовых уведомлений")
		chatID     = flag.String("chat-id", "", "ID чата (переопределяет .env)")
		debugMode  = flag.Bool("debug", false, "Режим отладки")
	)
	flag.Parse()

	fmt.Println("🤖 ТЕСТ РЕАЛЬНОГО TELEGRAM БОТА")
	fmt.Println(strings.Repeat("=", 60))

	// Загрузка конфигурации
	fmt.Println("1. 📋 Загрузка конфигурации...")
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Ошибка загрузки конфигурации: %v\n", err)
		fmt.Println("💡 Используйте: --config=.env или создайте .env файл")
		os.Exit(1)
	}

	// Проверка настроек Telegram
	if !cfg.TelegramEnabled {
		fmt.Println("⚠️  Telegram отключен в конфигурации")
		fmt.Println("   Установите TELEGRAM_ENABLED=true в .env")
		cfg.TelegramEnabled = true
	}

	if cfg.TelegramBotToken == "" || cfg.TelegramBotToken == "your_telegram_bot_token_here" {
		fmt.Println("❌ Telegram Bot Token не настроен")
		fmt.Println("💡 Получите токен у @BotFather и добавьте в .env:")
		fmt.Println("   TG_API_KEY=ваш_токен_бота")
		os.Exit(1)
	}

	if cfg.TelegramChatID == "" || cfg.TelegramChatID == "your_telegram_chat_id_here" {
		fmt.Println("❌ Telegram Chat ID не настроен")
		fmt.Println("💡 Получите Chat ID у @userinfobot и добавьте в .env:")
		fmt.Println("   TG_CHAT_ID=ваш_chat_id")
		os.Exit(1)
	}

	// Переопределение Chat ID если указан флаг
	if *chatID != "" {
		cfg.TelegramChatID = *chatID
		fmt.Printf("✅ Используется Chat ID из флага: %s\n", *chatID)
	}

	// Вывод информации о конфигурации
	fmt.Println("✅ Конфигурация загружена:")
	fmt.Printf("   • Telegram Bot Token: %s...%s\n",
		cfg.TelegramBotToken[:10],
		cfg.TelegramBotToken[len(cfg.TelegramBotToken)-10:])
	fmt.Printf("   • Chat ID: %s\n", cfg.TelegramChatID)
	fmt.Printf("   • Уведомления роста: %v\n", cfg.TelegramNotifyGrowth)
	fmt.Printf("   • Уведомления падения: %v\n", cfg.TelegramNotifyFall)

	// Создание Telegram бота
	fmt.Println("\n2. 🤖 Создание Telegram бота...")
	bot := telegram.NewTelegramBot(cfg)
	if bot == nil {
		fmt.Println("❌ Не удалось создать Telegram бота")
		os.Exit(1)
	}
	fmt.Println("✅ Telegram бот создан")

	// Тест 1: Отправка тестового сообщения
	fmt.Println("\n3. 📨 Отправка тестового сообщения...")
	err = bot.SendTestMessage()
	if err != nil {
		fmt.Printf("❌ Ошибка отправки тестового сообщения: %v\n", err)
		fmt.Println("💡 Проверьте:")
		fmt.Println("   - Правильность токена бота")
		fmt.Println("   - Правильность Chat ID")
		fmt.Println("   - Бот добавлен в чат")
		os.Exit(1)
	}
	fmt.Println("✅ Тестовое сообщение отправлено!")
	fmt.Println("   Проверьте Telegram чат")

	// Пауза для проверки
	time.Sleep(2 * time.Second)

	// Тест 2: Отправка сигналов роста
	fmt.Println("\n4. 📈 Отправка тестовых сигналов роста...")
	testGrowthSignals(bot, cfg, *testCount, *debugMode)

	// Тест 3: Отправка сигналов падения
	fmt.Println("\n5. 📉 Отправка тестовых сигналов падения...")
	testFallSignals(bot, cfg, *testCount, *debugMode)

	// Тест 4: Отправка уведомлений счетчика
	fmt.Println("\n6. 🔢 Отправка уведомлений счетчика...")
	testCounterNotifications(bot, cfg, *testCount, *debugMode)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎉 ТЕСТИРОВАНИЕ ЗАВЕРШЕНО!")
	fmt.Println("📱 Проверьте все уведомления в Telegram чате")
	fmt.Println(strings.Repeat("=", 60))
}

// testGrowthSignals тестирует отправку сигналов роста
func testGrowthSignals(bot *telegram.TelegramBot, cfg *config.Config, count int, debug bool) {
	if !cfg.TelegramNotifyGrowth {
		fmt.Println("⚠️  Уведомления о росте отключены в конфигурации")
		return
	}

	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT"}

	for i := 0; i < count && i < len(symbols); i++ {
		symbol := symbols[i]

		signal := types.GrowthSignal{
			Symbol:        symbol,
			Direction:     "growth",
			GrowthPercent: 1.5 + float64(i)*0.5,
			PeriodMinutes: 5 * (i + 1),
			Timestamp:     time.Now(),
			Confidence:    60.0 + float64(i)*10,
			Volume24h:     1000000 * float64(i+1),
		}

		fmt.Printf("   📤 Отправка сигнала роста: %s %.2f%%\n",
			signal.Symbol, signal.GrowthPercent)

		err := bot.SendNotification(signal)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("   ✅ Отправлено\n")

			if debug {
				message := bot.FormatSignalMessage(signal)
				fmt.Printf("   📋 Сообщение:\n")
				lines := strings.Split(message, "\n")
				for _, line := range lines {
					fmt.Printf("      %s\n", line)
				}
			}
		}

		// Пауза между сообщениями для rate limiting
		if i < count-1 {
			time.Sleep(2 * time.Second)
		}
	}
}

// testFallSignals тестирует отправку сигналов падения
func testFallSignals(bot *telegram.TelegramBot, cfg *config.Config, count int, debug bool) {
	if !cfg.TelegramNotifyFall {
		fmt.Println("⚠️  Уведомления о падении отключены в конфигурации")
		return
	}

	symbols := []string{"DOGEUSDT", "MATICUSDT", "DOTUSDT", "AVAXUSDT", "XRPUSDT"}

	for i := 0; i < count && i < len(symbols); i++ {
		symbol := symbols[i]

		signal := types.GrowthSignal{
			Symbol:        symbol,
			Direction:     "fall",
			FallPercent:   1.0 + float64(i)*0.5,
			PeriodMinutes: 5 * (i + 1),
			Timestamp:     time.Now(),
			Confidence:    65.0 + float64(i)*10,
			Volume24h:     500000 * float64(i+1),
		}

		fmt.Printf("   📤 Отправка сигнала падения: %s %.2f%%\n",
			signal.Symbol, signal.FallPercent)

		err := bot.SendNotification(signal)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("   ✅ Отправлено\n")

			if debug {
				message := bot.FormatSignalMessage(signal)
				fmt.Printf("   📋 Сообщение:\n")
				lines := strings.Split(message, "\n")
				for _, line := range lines {
					fmt.Printf("      %s\n", line)
				}
			}
		}

		// Пауза между сообщениями для rate limiting
		if i < count-1 {
			time.Sleep(2 * time.Second)
		}
	}
}

// testCounterNotifications тестирует отправку уведомлений счетчика
func testCounterNotifications(bot *telegram.TelegramBot, cfg *config.Config, count int, debug bool) {
	if !cfg.CounterAnalyzer.Enabled {
		fmt.Println("⚠️  Counter Analyzer отключен в конфигурации")
		return
	}

	if !cfg.CounterAnalyzer.NotificationEnabled {
		fmt.Println("⚠️  Уведомления счетчика отключены в конфигурации")
		return
	}

	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	periods := []string{"15 минут", "30 минут", "1 час"}

	for i := 0; i < count && i < len(symbols); i++ {
		symbol := symbols[i]
		period := periods[i%len(periods)]

		// Используем SendCounterNotification если метод экспортирован
		// Для теста создадим сообщение вручную
		message := createCounterMessage(symbol, period, i+1, 8)

		fmt.Printf("   📤 Отправка уведомления счетчика: %s\n", symbol)

		err := bot.SendMessage(message)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("   ✅ Отправлено (счетчик: %d/8)\n", i+1)

			if debug {
				fmt.Printf("   📋 Сообщение:\n")
				lines := strings.Split(message, "\n")
				for _, line := range lines {
					fmt.Printf("      %s\n", line)
				}
			}
		}

		// Пауза между сообщениями для rate limiting
		if i < count-1 {
			time.Sleep(3 * time.Second)
		}
	}
}

// createCounterMessage создает сообщение счетчика
func createCounterMessage(symbol string, period string, count int, maxSignals int) string {
	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("2006/01/02 15:04:05")

	return fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"⚫ Символ: %s\n"+
			"🕐 Время: %s\n"+
			"⏱️  Период: %s\n"+
			"🟢 Направление: РОСТ\n"+
			"📈 Счетчик: %d/%d (%.0f%%)\n"+
			"📊 Базовый период: 1 мин",
		symbol,
		timeStr,
		period,
		count, maxSignals, percentage,
	)
}
