// cmd/launcher/main.go
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	mode := strings.ToLower(os.Args[1])

	switch mode {
	case "full", "--full", "-f":
		fmt.Println("🚀 Запуск полной версии бота...")
		fmt.Println("────────────────────────────────────")
		// Здесь можно было бы вызвать RunMainBot(), но т.к. это отдельная программа,
		// просто выводим инструкцию
		fmt.Println("Для запуска полной версии выполните:")
		fmt.Println("  go run cmd/bot/main.go")
		fmt.Println()
		fmt.Println("Или скомпилируйте и запустите:")
		fmt.Println("  go build -o bin/bot cmd/bot/main.go")
		fmt.Println("  ./bin/bot")

	case "signals", "--signals", "-s":
		fmt.Println("📈 Запуск режима только сигналов...")
		fmt.Println("────────────────────────────────────")
		fmt.Println("Для запуска режима сигналов выполните:")
		fmt.Println("  go run cmd/signals/main.go")
		fmt.Println()
		fmt.Println("Или скомпилируйте и запустите:")
		fmt.Println("  go build -o bin/signals cmd/signals/main.go")
		fmt.Println("  ./bin/signals")

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Printf("❌ Неизвестный режим: %s\n\n", mode)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Crypto Exchange Screener Bot - Лаунчер")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("Использование:")
	fmt.Println("  go run cmd/launcher/main.go [режим]")
	fmt.Println()
	fmt.Println("Режимы:")
	fmt.Println("  full     - Полный бот с мониторингом и API")
	fmt.Println("  signals  - Только мониторинг сигналов")
	fmt.Println("  help     - Эта справка")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  go run cmd/launcher/main.go full")
	fmt.Println("  go run cmd/launcher/main.go signals")
	fmt.Println("  go run cmd/launcher/main.go help")
	fmt.Println()
	fmt.Println("Прямой запуск (без лаунчера):")
	fmt.Println("  go run cmd/bot/main.go      - Полная версия")
	fmt.Println("  go run cmd/signals/main.go  - Только сигналы")
}

func printHelp() {
	printUsage()
	fmt.Println()
	fmt.Println("📚 Описание режимов:")
	fmt.Println()
	fmt.Println("1. FULL MODE - Полный бот")
	fmt.Println("   • Мониторинг всех USDT пар")
	fmt.Println("   • HTTP API сервер")
	fmt.Println("   • Статистика и логирование")
	fmt.Println("   • Настройка через .env файл")
	fmt.Println()
	fmt.Println("2. SIGNALS ONLY - Только сигналы")
	fmt.Println("   • Фокус на сигналах ценовых изменений")
	fmt.Println("   • Оптимизировано для быстрого реагирования")
	fmt.Println("   • Минимальные требования к ресурсам")
	fmt.Println("   • Форматированный вывод в терминал")
	fmt.Println()
	fmt.Println("⚙️  Настройка:")
	fmt.Println("   • Отредактируйте файл .env для настройки API ключей")
	fmt.Println("   • Порог сигнала: ALERT_THRESHOLD=0.2 (0.2%)")
	fmt.Println("   • Интервал обновления: UPDATE_INTERVAL=10 (секунд)")
	fmt.Println()
	fmt.Println("🔗 Ссылки:")
	fmt.Println("   • Bybit API: https://bybit-exchange.github.io/docs/")
	fmt.Println("   • GitHub: https://github.com/your-repo")
}
