// cmd/debug/telegram_integration/main.go
package main

import (
	"fmt"
	"strings"
	"time"
)

func main() {
	fmt.Println("🧪 ТЕСТ ИНТЕГРАЦИИ COUNTER ANALYZER С TELEGRAM")
	fmt.Println(strings.Repeat("=", 70))

	// Создаем мок Telegram бота
	mockBot := NewMockTelegramBot()
	fmt.Println("✅ Mock Telegram Bot создан")

	// Регистрируем callback обработчики
	mockBot.RegisterCallback("counter_settings", func() string {
		return mockBot.ShowCounterSettings()
	})

	mockBot.RegisterCallback("counter_period_15m", func() string {
		return "✅ Период изменен на 15 минут"
	})

	// Тест 1: Отправка уведомлений счетчика
	fmt.Println("\n1️⃣  ТЕСТ ОТПРАВКИ УВЕДОМЛЕНИЙ:")
	testCounterNotifications(mockBot)

	// Тест 2: Проверка rate limiting
	fmt.Println("\n2️⃣  ТЕСТ RATE LIMITING:")
	testRateLimiting(mockBot)

	// Тест 3: Обработка callback
	fmt.Println("\n3️⃣  ТЕСТ CALLBACK ОБРАБОТКИ:")
	testCallbackHandling(mockBot)

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("✅ ВСЕ ТЕСТЫ ИНТЕГРАЦИИ С TELEGRAM ЗАВЕРШЕНЫ")
}

// testCounterNotifications тестирует отправку уведомлений
func testCounterNotifications(mockBot *MockTelegramBot) {
	fmt.Println("   📨 Тестирование отправки уведомлений...")

	// Очищаем предыдущие сообщения
	mockBot.ClearMessages()

	// Тестовые уведомления
	testNotifications := []MockCounterNotification{
		{
			Symbol:          "BTCUSDT",
			SignalType:      "growth",
			CurrentCount:    1,
			Period:          "15 минут",
			PeriodStartTime: time.Now(),
			Timestamp:       time.Now(),
			MaxSignals:      8,
			Percentage:      12.5,
		},
		{
			Symbol:          "ETHUSDT",
			SignalType:      "fall",
			CurrentCount:    2,
			Period:          "15 минут",
			PeriodStartTime: time.Now(),
			Timestamp:       time.Now(),
			MaxSignals:      8,
			Percentage:      25.0,
		},
		{
			Symbol:          "SOLUSDT",
			SignalType:      "growth",
			CurrentCount:    8,
			Period:          "15 минут",
			PeriodStartTime: time.Now(),
			Timestamp:       time.Now(),
			MaxSignals:      8,
			Percentage:      100.0,
		},
	}

	// Отправляем уведомления
	for i, notification := range testNotifications {
		err := mockBot.SendCounterNotification(notification)
		if err != nil {
			fmt.Printf("   ❌ Ошибка отправки уведомления %d: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✅ Уведомление %d отправлено: %s\n", i+1, notification.Symbol)
		}
	}

	// Проверяем отправленные сообщения
	sentMessages := mockBot.GetSentMessages()
	fmt.Printf("   📊 Отправлено сообщений: %d\n", len(sentMessages))

	if len(sentMessages) == len(testNotifications) {
		fmt.Println("   ✅ Все уведомления отправлены успешно")

		// Показываем формат сообщений
		fmt.Println("   📋 Формат сообщений:")
		for i, msg := range sentMessages {
			lines := strings.Split(msg, "\n")
			if len(lines) > 0 {
				fmt.Printf("      %d. %s\n", i+1, lines[0])
				if len(lines) > 1 {
					fmt.Printf("         %s\n", lines[1])
				}
			}
		}
	} else {
		fmt.Printf("   ❌ Ожидалось %d сообщений, получено %d\n",
			len(testNotifications), len(sentMessages))
	}
}

// testRateLimiting тестирует ограничение частоты
func testRateLimiting(mockBot *MockTelegramBot) {
	fmt.Println("   ⏱️  Тестирование rate limiting...")

	// Очищаем сообщения
	mockBot.ClearMessages()

	// Быстрая отправка нескольких уведомлений
	notification := MockCounterNotification{
		Symbol:          "TESTUSDT",
		SignalType:      "growth",
		CurrentCount:    1,
		Period:          "15 минут",
		PeriodStartTime: time.Now(),
		Timestamp:       time.Now(),
		MaxSignals:      8,
		Percentage:      12.5,
	}

	// Отправляем дважды быстро
	mockBot.SendCounterNotification(notification)
	mockBot.SendCounterNotification(notification) // Должно быть пропущено

	// Ждем и отправляем снова
	time.Sleep(3 * time.Second)
	mockBot.SendCounterNotification(notification) // Должно отправиться

	sentMessages := mockBot.GetSentMessages()
	fmt.Printf("   📊 Отправлено сообщений: %d (ожидалось 2)\n", len(sentMessages))

	if len(sentMessages) == 2 {
		fmt.Println("   ✅ Rate limiting работает корректно")
	} else {
		fmt.Println("   ❌ Проблемы с rate limiting")
	}
}

// testCallbackHandling тестирует обработку callback
func testCallbackHandling(mockBot *MockTelegramBot) {
	fmt.Println("   🔘 Тестирование обработки callback...")

	testCases := []struct {
		name     string
		callback string
		expected string
	}{
		{
			name:     "Настройки счетчика",
			callback: "counter_settings",
			expected: "Настройки счетчика",
		},
		{
			name:     "Включение уведомлений",
			callback: "counter_notify_on",
			expected: "включены",
		},
		{
			name:     "Выключение уведомлений",
			callback: "counter_notify_off",
			expected: "выключены",
		},
		{
			name:     "Изменение периода",
			callback: "counter_period_15m",
			expected: "15 минут",
		},
		{
			name:     "Неизвестный callback",
			callback: "unknown_callback",
			expected: "Неизвестный",
		},
	}

	for _, tc := range testCases {
		result := mockBot.HandleCallback(tc.callback)
		if strings.Contains(result, tc.expected) {
			fmt.Printf("   ✅ %s: обработка корректна\n", tc.name)
		} else {
			fmt.Printf("   ❌ %s: ожидалось '%s', получено '%s'\n",
				tc.name, tc.expected, result)
		}
	}

	// Проверка формата уведомлений
	fmt.Println("\n   📱 Проверка формата уведомлений:")
	mockBot.ClearMessages()

	notification := MockCounterNotification{
		Symbol:          "BTCUSDT",
		SignalType:      "growth",
		CurrentCount:    3,
		Period:          "15 минут",
		PeriodStartTime: time.Now(),
		Timestamp:       time.Now(),
		MaxSignals:      8,
		Percentage:      37.5,
	}

	mockBot.SendCounterNotification(notification)
	messages := mockBot.GetSentMessages()

	if len(messages) > 0 {
		msg := messages[0]
		requiredElements := []string{
			"Счетчик сигналов",
			"BTCUSDT",
			"РОСТ",
			"3/8",
			"37%",
			"Базовый период",
		}

		missing := []string{}
		for _, element := range requiredElements {
			if !strings.Contains(msg, element) {
				missing = append(missing, element)
			}
		}

		if len(missing) == 0 {
			fmt.Println("   ✅ Формат уведомления корректный")
		} else {
			fmt.Printf("   ❌ Отсутствуют элементы: %v\n", missing)
		}
	}
}
