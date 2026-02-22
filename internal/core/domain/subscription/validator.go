// internal/core/domain/subscription/validator.go
package subscription

import (
	"context"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// PaymentData структура данных платежа для валидатора
type PaymentData struct {
	ID        int64
	UserID    int
	PlanCode  string
	Amount    int
	CreatedAt time.Time
}

// PaymentRepository интерфейс для получения платежей
type PaymentRepository interface {
	GetSuccessfulPayments(ctx context.Context, days int) ([]*PaymentData, error)
}

// ValidationStats статистика проверки
type ValidationStats struct {
	Total           int
	OK              int
	Missing         int // нет подписки
	FreeWithPayment int // free + платеж
	WrongPlan       int // не тот план
	WrongPeriod     int // не та дата
	Errors          int
}

// SetPaymentRepo сохраняет репозиторий платежей для валидатора.
// Вызывается из фабрики после создания сервиса.
func (s *Service) SetPaymentRepo(paymentRepo PaymentRepository) {
	s.paymentRepo = paymentRepo
}

// RunValidation запускает одну итерацию валидации подписок.
// Вызывается планировщиком задач (scheduler).
func (s *Service) RunValidation(ctx context.Context) error {
	if s.paymentRepo == nil {
		return nil // PaymentRepo не настроен — пропускаем
	}
	s.runValidation(s.paymentRepo)
	return nil
}

// runValidation выполняет одну проверку
func (s *Service) runValidation(paymentRepo PaymentRepository) {
	ctx := context.Background()
	logger.Info("🔍 [VALIDATOR] Начинаем проверку соответствия платежей и подписок")

	// Получаем платежи за последние 30 дней
	payments, err := paymentRepo.GetSuccessfulPayments(ctx, 30)
	if err != nil {
		logger.Error("❌ [VALIDATOR] Ошибка получения платежей: %v", err)
		return
	}

	if len(payments) == 0 {
		logger.Info("📭 [VALIDATOR] Нет платежей для проверки")
		return
	}

	logger.Info("📊 [VALIDATOR] Найдено платежей для проверки: %d", len(payments))

	stats := &ValidationStats{
		Total: len(payments),
	}

	for _, payment := range payments {
		s.validatePayment(ctx, payment, stats)
	}

	// Логируем статистику
	s.logValidationStats(stats)
}

// validatePayment проверяет один платеж
func (s *Service) validatePayment(ctx context.Context, payment *PaymentData, stats *ValidationStats) {
	// Получаем все подписки пользователя
	subscriptions, err := s.subRepo.GetAllByUserID(ctx, payment.UserID)
	if err != nil {
		logger.Error("❌ [VALIDATOR] Ошибка получения подписок для user %d: %v",
			payment.UserID, err)
		stats.Errors++
		return
	}

	// Ищем активную подписку
	activeSub := s.findActiveSubscription(subscriptions)

	// ⭐ КЕЙС 1: Нет активной подписки
	if activeSub == nil {
		s.handleMissingSubscription(ctx, payment, stats)
		return
	}

	// ⭐ КЕЙС 2: Пользователь на free, но есть платный платеж
	if activeSub.PlanCode == models.PlanFree && payment.PlanCode != models.PlanFree {
		s.handleFreeWithPayment(ctx, activeSub, payment, stats)
		return
	}

	// ⭐ КЕЙС 3: План не совпадает (оба платных)
	if activeSub.PlanCode != payment.PlanCode {
		s.handlePlanMismatch(ctx, activeSub, payment, stats)
		return
	}

	// ⭐ КЕЙС 4: План совпадает, проверяем период
	expectedEndDate := s.calculateEndDate(payment)
	if !s.isValidEndDate(activeSub.CurrentPeriodEnd, expectedEndDate) {
		s.handlePeriodMismatch(ctx, activeSub, expectedEndDate, payment, stats)
		return
	}

	// Всё хорошо
	stats.OK++
	logger.Debug("✅ [VALIDATOR] Подписка user %d в порядке (план: %s, до: %v)",
		payment.UserID, activeSub.PlanCode, activeSub.CurrentPeriodEnd)
}

// handleFreeWithPayment обрабатывает случай: пользователь на free, но есть платный платеж
func (s *Service) handleFreeWithPayment(ctx context.Context,
	sub *models.UserSubscription, payment *PaymentData, stats *ValidationStats) {

	logger.Warn("⚠️ [INCIDENT] У user %d активна FREE подписка, но есть платный платеж #%d на план %s",
		payment.UserID, payment.ID, payment.PlanCode)

	// Проверяем, была ли уже попытка апгрейда
	hasUpgradeAttempt := false
	if sub.Metadata != nil {
		if attempt, ok := sub.Metadata["upgrade_attempt"].(bool); ok && attempt {
			hasUpgradeAttempt = true
		}
	}

	if hasUpgradeAttempt {
		logger.Warn("   ⚠️ Предыдущая попытка апгрейда уже была, пробуем снова")
	}

	// Обновляем подписку до платного плана
	if err := s.upgradeFreeToPaid(ctx, sub, payment); err != nil {
		logger.Error("❌ [INCIDENT] Не удалось обновить FREE до %s: %v",
			payment.PlanCode, err)
		stats.Errors++
	} else {
		logger.Info("✅ [INCIDENT] Подписка пользователя %d обновлена с FREE до %s",
			payment.UserID, payment.PlanCode)
		stats.WrongPlan++
	}
}

// upgradeFreeToPaid обновляет FREE подписку до платной
func (s *Service) upgradeFreeToPaid(ctx context.Context,
	sub *models.UserSubscription, payment *PaymentData) error {

	logger.Info("🔄 Обновление FREE -> %s для user %d из платежа #%d",
		payment.PlanCode, payment.UserID, payment.ID)

	// Получаем новый план
	plan, err := s.GetPlan(payment.PlanCode)
	if err != nil {
		return fmt.Errorf("план не найден: %w", err)
	}

	// Получаем период
	period, err := s.GetSubscriptionPeriod(payment.PlanCode)
	if err != nil {
		return err
	}

	// Обновляем подписку
	now := time.Now()
	periodEnd := now.Add(period)

	oldPlan := sub.PlanCode

	sub.PlanID = plan.ID
	sub.PlanCode = payment.PlanCode
	sub.PaymentID = &payment.ID
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &periodEnd
	sub.Status = models.StatusActive

	if sub.Metadata == nil {
		sub.Metadata = make(map[string]interface{})
	}
	sub.Metadata["upgraded_from_free"] = true
	sub.Metadata["upgrade_attempt"] = true
	sub.Metadata["previous_plan"] = oldPlan
	sub.Metadata["upgraded_at"] = now.Format(time.RFC3339)
	sub.Metadata["incident"] = true
	sub.Metadata["payment_id"] = payment.ID

	// Удаляем метки free, если они были
	delete(sub.Metadata, "trial")
	delete(sub.Metadata, "expires_after_hours")

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("ошибка обновления подписки: %w", err)
	}

	// Трекаем событие
	if s.analytics != nil {
		s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_upgraded_from_free",
		UserID:         payment.UserID,
		SubscriptionID: sub.ID,
		PlanCode:       payment.PlanCode,
		OldPlanCode:    oldPlan,
		Status:         models.StatusActive,
		Timestamp:      now,
		Metadata: map[string]interface{}{
			"payment_id": payment.ID,
			"incident":   true,
			},
		})
	}

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(payment.UserID)

	return nil
}

// findActiveSubscription ищет активную подписку в списке
func (s *Service) findActiveSubscription(subscriptions []*models.UserSubscription) *models.UserSubscription {
	for _, sub := range subscriptions {
		if sub.IsActive() {
			return sub
		}
	}
	return nil
}

// handleMissingSubscription обрабатывает отсутствие подписки
func (s *Service) handleMissingSubscription(ctx context.Context, payment *PaymentData, stats *ValidationStats) {
	logger.Warn("⚠️ [INCIDENT] У user %d нет активной подписки, но есть платеж #%d от %s (план: %s)",
		payment.UserID, payment.ID, payment.CreatedAt.Format("02.01.2006"), payment.PlanCode)

	if err := s.restoreSubscription(ctx, payment); err != nil {
		logger.Error("❌ [INCIDENT] Не удалось восстановить подписку: %v", err)
		stats.Errors++
	} else {
		logger.Info("✅ [INCIDENT] Подписка восстановлена для user %d", payment.UserID)
		stats.Missing++
	}
}

// handlePlanMismatch обрабатывает несоответствие плана
func (s *Service) handlePlanMismatch(ctx context.Context, sub *models.UserSubscription,
	payment *PaymentData, stats *ValidationStats) {

	logger.Warn("⚠️ [INCIDENT] У user %d план подписки (%s) не совпадает с платежом (%s)",
		payment.UserID, sub.PlanCode, payment.PlanCode)

	if err := s.fixPlanMismatch(ctx, sub, payment); err != nil {
		logger.Error("❌ [INCIDENT] Не удалось исправить план: %v", err)
		stats.Errors++
	} else {
		logger.Info("✅ [INCIDENT] План подписки исправлен для user %d", payment.UserID)
		stats.WrongPlan++
	}
}

// handlePeriodMismatch обрабатывает несоответствие периода
func (s *Service) handlePeriodMismatch(ctx context.Context, sub *models.UserSubscription,
	expectedEndDate time.Time, payment *PaymentData, stats *ValidationStats) {

	logger.Warn("⚠️ [INCIDENT] У user %d неверная дата окончания подписки", payment.UserID)
	logger.Warn("   Ожидалось: %v, Фактически: %v",
		expectedEndDate, sub.CurrentPeriodEnd)

	if err := s.fixEndDate(ctx, sub, expectedEndDate); err != nil {
		logger.Error("❌ [INCIDENT] Не удалось исправить дату: %v", err)
		stats.Errors++
	} else {
		logger.Info("✅ [INCIDENT] Дата окончания исправлена для user %d", payment.UserID)
		stats.WrongPeriod++
	}
}

// restoreSubscription восстанавливает подписку из платежа
func (s *Service) restoreSubscription(ctx context.Context, payment *PaymentData) error {
	logger.Info("🔄 Восстановление подписки для user %d из платежа #%d",
		payment.UserID, payment.ID)

	// Получаем план
	plan, err := s.GetPlan(payment.PlanCode)
	if err != nil {
		return fmt.Errorf("план не найден: %w", err)
	}

	// Получаем период
	period, err := s.GetSubscriptionPeriod(payment.PlanCode)
	if err != nil {
		return err
	}

	// Создаем подписку
	now := time.Now()
	periodEnd := now.Add(period)

	subscription := &models.UserSubscription{
		UserID:             payment.UserID,
		PlanID:             plan.ID,
		PaymentID:          &payment.ID,
		Status:             models.StatusActive,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
		CancelAtPeriodEnd:  false,
		Metadata: map[string]interface{}{
			"restored":         true,
			"restored_at":      now.Format(time.RFC3339),
			"original_payment": payment.ID,
			"incident":         true,
		},
	}

	if err := s.subRepo.Create(ctx, subscription); err != nil {
		return fmt.Errorf("ошибка создания подписки: %w", err)
	}

	// Трекаем событие
	if s.analytics != nil {
		s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_restored",
		UserID:         payment.UserID,
		SubscriptionID: subscription.ID,
		PlanCode:       payment.PlanCode,
		Status:         models.StatusActive,
		Timestamp:      now,
		Metadata: map[string]interface{}{
			"payment_id": payment.ID,
			"incident":   true,
			},
		})
	}

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(payment.UserID)

	return nil
}

// fixPlanMismatch исправляет несоответствие плана
func (s *Service) fixPlanMismatch(ctx context.Context,
	sub *models.UserSubscription, payment *PaymentData) error {

	logger.Info("🔄 Исправление плана для user %d: %s -> %s",
		payment.UserID, sub.PlanCode, payment.PlanCode)

	// Получаем новый план
	plan, err := s.GetPlan(payment.PlanCode)
	if err != nil {
		return fmt.Errorf("план не найден: %w", err)
	}

	// Получаем период
	period, err := s.GetSubscriptionPeriod(payment.PlanCode)
	if err != nil {
		return err
	}

	// Обновляем подписку
	now := time.Now()
	periodEnd := now.Add(period)

	oldPlan := sub.PlanCode

	sub.PlanID = plan.ID
	sub.PlanCode = payment.PlanCode
	sub.PaymentID = &payment.ID
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &periodEnd

	if sub.Metadata == nil {
		sub.Metadata = make(map[string]interface{})
	}
	sub.Metadata["upgraded_from_payment"] = true
	sub.Metadata["previous_plan"] = oldPlan
	sub.Metadata["upgraded_at"] = now.Format(time.RFC3339)
	sub.Metadata["incident"] = true

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("ошибка обновления подписки: %w", err)
	}

	// Трекаем событие
	if s.analytics != nil {
		s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_fixed_plan",
		UserID:         payment.UserID,
		SubscriptionID: sub.ID,
		PlanCode:       payment.PlanCode,
		OldPlanCode:    oldPlan,
		Status:         models.StatusActive,
		Timestamp:      now,
		Metadata: map[string]interface{}{
			"payment_id": payment.ID,
			"incident":   true,
			},
		})
	}

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(payment.UserID)

	return nil
}

// fixEndDate исправляет дату окончания подписки
func (s *Service) fixEndDate(ctx context.Context, sub *models.UserSubscription, expectedEndDate time.Time) error {
	logger.Info("🔄 Исправление даты окончания для user %d: %v -> %v",
		sub.UserID, sub.CurrentPeriodEnd, expectedEndDate)

	sub.CurrentPeriodEnd = &expectedEndDate

	if sub.Metadata == nil {
		sub.Metadata = make(map[string]interface{})
	}
	sub.Metadata["end_date_fixed"] = true
	sub.Metadata["fixed_at"] = time.Now().Format(time.RFC3339)
	sub.Metadata["incident"] = true

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("ошибка обновления подписки: %w", err)
	}

	// Трекаем событие
	if s.analytics != nil {
		s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_fixed_period",
		UserID:         sub.UserID,
		SubscriptionID: sub.ID,
		PlanCode:       sub.PlanCode,
		Status:         models.StatusActive,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"old_end_date": sub.CurrentPeriodEnd,
			"new_end_date": expectedEndDate,
			"incident":     true,
			},
		})
	}

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(sub.UserID)

	return nil
}

// calculateEndDate вычисляет дату окончания подписки по платежу
func (s *Service) calculateEndDate(payment *PaymentData) time.Time {
	var period time.Duration
	switch payment.PlanCode {
	case "test":
		period = 5 * time.Minute
	case "basic":
		period = 30 * 24 * time.Hour
	case "pro":
		period = 90 * 24 * time.Hour
	case "enterprise":
		period = 365 * 24 * time.Hour
	default:
		return time.Time{} // free и другие — не проверяем
	}
	return payment.CreatedAt.Add(period)
}

// isValidEndDate проверяет корректность даты окончания
func (s *Service) isValidEndDate(actual *time.Time, expected time.Time) bool {
	if actual == nil {
		return false
	}

	// Допустимое отклонение - 1 день
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	return diff <= 24*time.Hour
}

// logValidationStats логирует статистику проверки
func (s *Service) logValidationStats(stats *ValidationStats) {
	logger.Info("📊 [VALIDATOR] Статистика проверки:")
	logger.Info("   • Всего платежей: %d", stats.Total)
	logger.Info("   • ✅ OK: %d", stats.OK)
	logger.Info("   • ⚠️ Отсутствовали: %d", stats.Missing)
	logger.Info("   • ⚠️ Неверный план: %d", stats.WrongPlan)
	logger.Info("   • ⚠️ Неверный период: %d", stats.WrongPeriod)
	logger.Info("   • ❌ Ошибки: %d", stats.Errors)
	logger.Info("   • ⚠️ FREE + платеж: %d", stats.FreeWithPayment)

	// Если были инциденты, пишем отдельный warning
	if stats.Missing+stats.WrongPlan+stats.WrongPeriod > 0 {
		logger.Warn("🚨 [VALIDATOR] Обнаружено инцидентов: %d",
			stats.Missing+stats.WrongPlan+stats.WrongPeriod)
	}
}
