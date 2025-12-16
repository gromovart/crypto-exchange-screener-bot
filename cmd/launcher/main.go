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
		fmt.Println("🚀 Запуск полной версии бота с Монитором Роста...")
		fmt.Println("────────────────────────────────────")
		fmt.Println("Для запуска выполните:")
		fmt.Println("  go run cmd/bot/main.go")
		fmt.Println()
		fmt.Println("Или скомпилируйте и запустите:")
		fmt.Println("  go build -o bin/bot cmd/bot/main.go")
		fmt.Println("  ./bin/bot")

	case "growth", "--growth", "-g":
		fmt.Println("📈 Запуск режима только Монитора Роста...")
		fmt.Println("────────────────────────────────────")
		fmt.Println("Для запуска выполните:")
		fmt.Println("  go run cmd/signals/main.go")
		fmt.Println()
		fmt.Println("Или скомпилируйте и запустите:")
		fmt.Println("  go build -o bin/growth cmd/signals/main.go")
		fmt.Println("  ./bin/growth")

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
	fmt.Println("  full     - Полный бот с мониторингом роста")
	fmt.Println("  growth   - Только мониторинг роста/падения")
	fmt.Println("  help     - Эта справка")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  go run cmd/launcher/main.go full")
	fmt.Println("  go run cmd/launcher/main.go growth")
	fmt.Println("  go run cmd/launcher/main.go help")
}

func printHelp() {
	printUsage()
	fmt.Println()
	fmt.Println("📚 Описание режимов:")
	fmt.Println()
	fmt.Println("1. ПОЛНЫЙ РЕЖИМ - Полный бот")
	fmt.Println("   • Мониторинг всех USDT фьючерсных пар")
	fmt.Println("   • Обнаружение непрерывного роста/падения")
	fmt.Println("   • Статистика и логирование")
	fmt.Println("   • Настройка через .env файл")
	fmt.Println()
	fmt.Println("2. РЕЖИМ РОСТА - Только рост/падение")
	fmt.Println("   • Фокус на сигналах роста/падения")
	fmt.Println("   • Оптимизировано для быстрого реагирования")
	fmt.Println("   • Форматированный вывод в терминал")
	fmt.Println()
	fmt.Println("⚙️  Настройка роста:")
	fmt.Println("   • GROWTH_THRESHOLD=0.05 (0.05%)")
	fmt.Println("   • FALL_THRESHOLD=0.05 (0.05%)")
	fmt.Println("   • GROWTH_PERIODS=5,15,30")
	fmt.Println("   • CHECK_CONTINUITY=false")
}
