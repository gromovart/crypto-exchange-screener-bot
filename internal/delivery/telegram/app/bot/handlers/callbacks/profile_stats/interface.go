package profile_stats

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// ProfileStatsHandler интерфейс обработчика статистики профиля
type ProfileStatsHandler interface {
	handlers.Handler
}
