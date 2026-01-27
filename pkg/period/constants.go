package period

import "time"

// Поддерживаемые периоды в минутах
const (
	Minutes5    = 5
	Minutes15   = 15
	Minutes30   = 30
	Minutes60   = 60   // 1 час
	Minutes240  = 240  // 4 часа
	Minutes1440 = 1440 // 1 день
)

// Поддерживаемые строковые представления
const (
	Period5m  = "5m"
	Period15m = "15m"
	Period30m = "30m"
	Period1h  = "1h"
	Period4h  = "4h"
	Period1d  = "1d"
)

// Все поддерживаемые периоды
var AllPeriods = []string{
	Period5m,
	Period15m,
	Period30m,
	Period1h,
	Period4h,
	Period1d,
}

// Все поддерживаемые периоды в минутах
var AllMinutes = []int{
	Minutes5,
	Minutes15,
	Minutes30,
	Minutes60,
	Minutes240,
	Minutes1440,
}

// Дефолтные значения
const (
	DefaultPeriod   = Period15m
	DefaultMinutes  = Minutes15
	DefaultDuration = 15 * time.Minute
)
