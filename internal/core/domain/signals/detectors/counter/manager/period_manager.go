// internal/core/domain/signals/detectors/counter/manager/period_manager.go
package manager

import (
	"fmt"
	"time"
)

// PeriodManager - менеджер для работы с периодами счетчиков
type PeriodManager struct{}

func NewPeriodManager() *PeriodManager {
	return &PeriodManager{}
}

func (pm *PeriodManager) CheckAndResetPeriod(
	cnt *counterSignalCounter,
	period string,
	maxSignals int,
) bool {
	cnt.Lock()
	defer cnt.Unlock()

	now := time.Now()
	periodDuration := getPeriodDuration(period)
	shouldReset := false

	if now.Sub(cnt.PeriodStartTime) >= periodDuration {
		shouldReset = true
	} else if cnt.SignalCount >= maxSignals {
		shouldReset = true
	} else if cnt.SelectedPeriod != period {
		shouldReset = true
	}

	if shouldReset {
		cnt.BasePeriodCount = 0
		cnt.SignalCount = 0
		cnt.GrowthCount = 0
		cnt.FallCount = 0
		cnt.PeriodStartTime = now
		cnt.PeriodEndTime = now.Add(periodDuration)
		cnt.SelectedPeriod = period
		cnt.Settings.SelectedPeriod = period
		return true
	}

	return false
}

func (pm *PeriodManager) CalculateMaxSignals(period string, basePeriodMinutes int) int {
	if basePeriodMinutes <= 0 {
		basePeriodMinutes = 1
	}

	totalPossibleSignals := getPeriodMinutes(period) / basePeriodMinutes

	switch {
	case totalPossibleSignals < 5:
		return 5
	case totalPossibleSignals > 15:
		return 15
	default:
		return totalPossibleSignals
	}
}

func (pm *PeriodManager) GetPeriodProgress(cnt *counterSignalCounter) float64 {
	cnt.RLock()
	defer cnt.RUnlock()

	now := time.Now()
	periodStart := cnt.PeriodStartTime
	periodEnd := cnt.PeriodEndTime

	if now.Before(periodStart) {
		return 0.0
	}
	if now.After(periodEnd) {
		return 100.0
	}

	totalDuration := periodEnd.Sub(periodStart)
	elapsed := now.Sub(periodStart)

	return (float64(elapsed) / float64(totalDuration)) * 100.0
}

func (pm *PeriodManager) GetTimeUntilReset(cnt *counterSignalCounter) time.Duration {
	cnt.RLock()
	defer cnt.RUnlock()

	now := time.Now()
	if now.After(cnt.PeriodEndTime) {
		return 0
	}

	return cnt.PeriodEndTime.Sub(now)
}

func (pm *PeriodManager) GetRemainingSignals(cnt *counterSignalCounter, maxSignals int) int {
	cnt.RLock()
	defer cnt.RUnlock()

	remaining := maxSignals - cnt.SignalCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (pm *PeriodManager) FormatResetReason(cnt *counterSignalCounter, maxSignals int) string {
	cnt.RLock()
	defer cnt.RUnlock()

	now := time.Now()
	periodDuration := getPeriodDuration(cnt.SelectedPeriod)

	if now.Sub(cnt.PeriodStartTime) >= periodDuration {
		return fmt.Sprintf("истек период (%s)", cnt.SelectedPeriod)
	} else if cnt.SignalCount >= maxSignals {
		return fmt.Sprintf("достигнут максимум сигналов (%d/%d)", cnt.SignalCount, maxSignals)
	}

	return "изменение конфигурации"
}
