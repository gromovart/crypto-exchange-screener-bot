// cmd/debug/real_telegram_test/main.go
package main

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
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
		configPath  = flag.String("config", ".env", "Путь к файлу конфигурации")
		testCount   = flag.Int("count", 1, "Количество тестовых уведомлений (ПО УМОЛЧАНИЮ 1)")
		chatID      = flag.String("chat-id", "", "ID чата (переопределяет .env)")
		debugMode   = flag.Bool("debug", false, "Режим отладки")
		skipWelcome = flag.Bool("skip-welcome", false, "Пропустить приветственное сообщение")
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
	fmt.Printf("   • Тестовых сообщений: %d\n", *testCount)

	// Создание Telegram бота
	fmt.Println("\n2. 🤖 Создание Telegram бота...")
	bot := telegram.NewTelegramBot(cfg)
	if bot == nil {
		fmt.Println("❌ Не удалось создать Telegram бота")
		os.Exit(1)
	}
	fmt.Println("✅ Telegram бот создан")

	// ПАУЗА для того чтобы основной бот отправил свое приветственное сообщение
	if !*skipWelcome {
		fmt.Println("\n⏳ Ожидание 3 секунды перед отправкой тестовых сообщений...")
		time.Sleep(3 * time.Second)
	}

	// Тест 1: Отправка тестового сообщения (ТОЛЬКО ЕСЛИ НЕ ПРОПУСКАЕМ)
	if !*skipWelcome && *testCount > 0 {
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
		time.Sleep(2 * time.Second)
	} else if *skipWelcome {
		fmt.Println("\n3. ⏭️  Пропуск тестового сообщения (--skip-welcome)")
	}

	// Тест 2: Отправка сигналов роста (ОГРАНИЧЕННОЕ КОЛИЧЕСТВО)
	if *testCount > 0 && cfg.TelegramNotifyGrowth {
		fmt.Println("\n4. 📈 Отправка тестовых сигналов роста...")
		testGrowthSignals(bot, cfg, *testCount, *debugMode)
	} else {
		fmt.Println("\n4. ⏭️  Пропуск сигналов роста (отключено или count=0)")
	}

	// Тест 3: Отправка сигналов падения (ОГРАНИЧЕННОЕ КОЛИЧЕСТВО)
	if *testCount > 0 && cfg.TelegramNotifyFall {
		fmt.Println("\n5. 📉 Отправка тестовых сигналов падения...")
		testFallSignals(bot, cfg, *testCount, *debugMode)
	} else {
		fmt.Println("\n5. ⏭️  Пропуск сигналов падения (отключено или count=0)")
	}

	// Тест 4: Отправка уведомлений счетчика (ОГРАНИЧЕННОЕ КОЛИЧЕСТВО)
	if *testCount > 0 && cfg.AnalyzerConfigs.CounterAnalyzer.Enabled && cfg.GetCounterNotificationEnabled() {
		fmt.Println("\n6. 🔢 Отправка уведомлений счетчика...")
		testCounterNotifications(bot, cfg, *testCount, *debugMode)
	} else {
		fmt.Println("\n6. ⏭️  Пропуск уведомлений счетчика (отключено или count=0)")
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎉 ТЕСТИРОВАНИЕ ЗАВЕРШЕНО!")
	fmt.Println("📱 Проверьте уведомления в Telegram чате")
	fmt.Println(strings.Repeat("=", 60))
}

// testGrowthSignals тестирует отправку сигналов роста (ОБНОВЛЕННЫЙ - ПРОСТОЕ СООБЩЕНИЕ)
func testGrowthSignals(bot *telegram.TelegramBot, cfg *config.Config, count int, debug bool) {
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}

	// Ограничиваем количество сообщений
	if count > 3 {
		count = 3
		fmt.Printf("   ⚠️  Ограничено %d сообщениями для теста\n", count)
	}

	for i := 0; i < count && i < len(symbols); i++ {
		symbol := symbols[i]

		signal := types.GrowthSignal{
			Symbol:        symbol,
			Direction:     "growth",
			GrowthPercent: 1.5 + float64(i)*0.5,
			PeriodMinutes: 5,
			Timestamp:     time.Now(),
			Confidence:    75.0,
			Volume24h:     1000000,
		}

		fmt.Printf("   📤 Отправка сигнала роста: %s %.2f%%\n",
			signal.Symbol, signal.GrowthPercent)

		err := bot.SendNotification(signal)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("   ✅ Отправлено\n")
		}

		// Пауза между сообщениями для rate limiting
		if i < count-1 {
			time.Sleep(3 * time.Second)
		}
	}
}

// testFallSignals тестирует отправку сигналов падения (ОБНОВЛЕННЫЙ - ПРОСТОЕ СООБЩЕНИЕ)
func testFallSignals(bot *telegram.TelegramBot, cfg *config.Config, count int, debug bool) {
	symbols := []string{"DOGEUSDT", "MATICUSDT", "XRPUSDT"}

	// Ограничиваем количество сообщений
	if count > 3 {
		count = 3
		fmt.Printf("   ⚠️  Ограничено %d сообщениями для теста\n", count)
	}

	for i := 0; i < count && i < len(symbols); i++ {
		symbol := symbols[i]

		signal := types.GrowthSignal{
			Symbol:        symbol,
			Direction:     "fall",
			FallPercent:   1.0 + float64(i)*0.5,
			PeriodMinutes: 5,
			Timestamp:     time.Now(),
			Confidence:    70.0,
			Volume24h:     500000,
		}

		fmt.Printf("   📤 Отправка сигнала падения: %s %.2f%%\n",
			signal.Symbol, signal.FallPercent)

		err := bot.SendNotification(signal)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("   ✅ Отправлено\n")
		}

		// Пауза между сообщениями для rate limiting
		if i < count-1 {
			time.Sleep(3 * time.Second)
		}
	}
}

// testCounterNotifications тестирует отправку уведомлений счетчика (ОБНОВЛЕННЫЙ)
func testCounterNotifications(bot *telegram.TelegramBot, cfg *config.Config, count int, debug bool) {
	// Только одно сообщение счетчика для теста
	if count > 1 {
		count = 1
		fmt.Println("   ⚠️  Ограничено 1 сообщением счетчика для теста")
	}

	for i := 0; i < count; i++ {
		message := createCounterMessage("BTCUSDT", "15 минут", 3, 8)

		fmt.Printf("   📤 Отправка уведомления счетчика\n")

		err := bot.SendMessage(message)
		if err != nil {
			fmt.Printf("   ❌ Ошибка: %v\n", err)
		} else {
			fmt.Printf("   ✅ Отправлено\n")
		}
	}
}

// createCounterMessage создает сообщение счетчика (УПРОЩЕННОЕ)
func createCounterMessage(symbol string, period string, count int, maxSignals int) string {
	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("2006/01/02 15:04:05")

	return fmt.Sprintf(
		"📊 Счетчик сигналов\n"+
			"⚫ Символ: %s\n"+
			"🕐 Время: %s\n"+
			"⏱️  Период: %s\n"+
			"🟢 Направление: РОСТ\n"+
			"📈 Счетчик: %d/%d (%.0f%%)",
		symbol,
		timeStr,
		period,
		count, maxSignals, percentage,
	)
}
