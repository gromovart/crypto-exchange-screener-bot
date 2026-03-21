// internal/delivery/telegram/services/tbank/service.go
package tbank

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	message_sender "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	tbank_client "crypto-exchange-screener-bot/internal/infrastructure/http/tbank"
	"crypto-exchange-screener-bot/pkg/logger"
)

// planPrices цены планов в копейках
var planPrices = map[string]int64{
	"test":       1000,   // 10 ₽ (минимум для СБП)
	"basic":      149000, // 1490 ₽
	"pro":        249000, // 2490 ₽
	"enterprise": 599000, // 5990 ₽
}

// planNames названия планов (для описания платежа и уведомлений)
var planNames = map[string]string{
	"test":       "🧪 Тестовый доступ",
	"basic":      "📱 Доступ на 1 месяц",
	"pro":        "🚀 Доступ на 3 месяца",
	"enterprise": "🏢 Доступ на 12 месяцев",
}

// PaymentResult результат создания платежа
type PaymentResult struct {
	OrderId    string // наш уникальный ID заказа
	PaymentURL string // URL формы оплаты Т-Банк
	Amount     int64  // сумма в копейках
	PlanID     string
	UserID     int
}

// Service интерфейс сервиса Т-Банк платежей
type Service interface {
	CreatePayment(ctx context.Context, userID int, planID string) (*PaymentResult, error)
	HandleNotification(ctx context.Context, params map[string]string) error
}

// Dependencies зависимости сервиса
type Dependencies struct {
	TBankClient         *tbank_client.Client
	SubscriptionService *subscription.Service
	UserService         *users.Service
	MessageSender       message_sender.MessageSender
	Password            string
	NotifyURL           string
	SuccessURL          string
	FailURL             string
}

type serviceImpl struct {
	client              *tbank_client.Client
	subscriptionService *subscription.Service
	userService         *users.Service
	messageSender       message_sender.MessageSender
	password            string
	notifyURL           string
	successURL          string
	failURL             string
}

// NewService создает новый сервис Т-Банк платежей
func NewService(deps Dependencies) Service {
	return &serviceImpl{
		client:              deps.TBankClient,
		subscriptionService: deps.SubscriptionService,
		userService:         deps.UserService,
		messageSender:       deps.MessageSender,
		password:            deps.Password,
		notifyURL:           deps.NotifyURL,
		successURL:          deps.SuccessURL,
		failURL:             deps.FailURL,
	}
}

// CreatePayment создаёт платёж через Т-Банк и возвращает ссылку на форму оплаты
func (s *serviceImpl) CreatePayment(ctx context.Context, userID int, planID string) (*PaymentResult, error) {
	amount, ok := planPrices[planID]
	if !ok {
		return nil, fmt.Errorf("неизвестный план: %s", planID)
	}

	planName := planNames[planID]
	orderId := fmt.Sprintf("tbank_%s_%d_%d", planID, userID, time.Now().Unix())

	req := tbank_client.InitRequest{
		Amount:      amount,
		OrderId:     orderId,
		Description: fmt.Sprintf("Подписка: %s", planName),
		PayType:     "O",
		Language:    "ru",
	}
	if s.notifyURL != "" {
		req.NotificationURL = s.notifyURL
	}
	if s.successURL != "" {
		req.SuccessURL = s.successURL
	}
	if s.failURL != "" {
		req.FailURL = s.failURL
	}

	resp, err := s.client.Init(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации платежа: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("Т-Банк отклонил платёж: [%s] %s", resp.ErrorCode, resp.Message)
	}

	logger.Info("✅ Платёж Т-Банк создан: OrderId=%s, PaymentId=%s", orderId, resp.PaymentId)

	return &PaymentResult{
		OrderId:    orderId,
		PaymentURL: resp.PaymentURL,
		Amount:     amount,
		PlanID:     planID,
		UserID:     userID,
	}, nil
}

// HandleNotification обрабатывает уведомление от Т-Банк о статусе платежа
func (s *serviceImpl) HandleNotification(ctx context.Context, params map[string]string) error {
	// Проверяем подпись
	if !tbank_client.VerifyToken(params, s.password) {
		logger.Warn("⚠️ Неверный токен уведомления от Т-Банк")
		return fmt.Errorf("неверный токен уведомления")
	}

	orderId := params["OrderId"]
	status := params["Status"]
	successStr := params["Success"]

	logger.Info("📩 Уведомление Т-Банк: OrderId=%s, Status=%s, Success=%s", orderId, status, successStr)

	// Обрабатываем отклонённые платежи
	if status == "REJECTED" {
		planID, userID, err := parseOrderId(orderId)
		if err == nil {
			go s.notifyPaymentFailed(userID, planID, params["ErrorCode"])
		}
		return nil
	}

	// Нас интересуют только подтверждённые платежи
	if successStr != "true" || status != "CONFIRMED" {
		logger.Info("ℹ️ Платёж %s не подтверждён (статус: %s)", orderId, status)
		return nil
	}

	// Парсим OrderId: tbank_{planID}_{userID}_{timestamp}
	planID, userID, err := parseOrderId(orderId)
	if err != nil {
		return fmt.Errorf("ошибка парсинга OrderId %s: %w", orderId, err)
	}

	// Активируем подписку
	logger.Info("🔑 Активация подписки: план=%s, пользователь=%d", planID, userID)

	if s.subscriptionService == nil {
		return fmt.Errorf("SubscriptionService не настроен")
	}

	_, err = s.subscriptionService.CreateSubscription(ctx, userID, planID, nil, false)
	if err != nil {
		if strings.Contains(err.Error(), "уже есть активная подписка") {
			// Подписка уже есть — накапливаем время поверх текущего срока
			logger.Info("➕ Накопление подписки: план=%s, пользователь=%d", planID, userID)
			_, err = s.subscriptionService.ExtendSubscription(ctx, userID, planID, nil)
			if err != nil {
				return fmt.Errorf("ошибка продления подписки: %w", err)
			}
		} else {
			return fmt.Errorf("ошибка активации подписки: %w", err)
		}
	}

	logger.Info("✅ Подписка активирована: план=%s, пользователь=%d", planID, userID)

	// Уведомляем пользователя в Telegram
	go s.notifyUser(userID, planID)

	return nil
}

// parseOrderId разбирает OrderId вида tbank_{planID}_{userID}_{timestamp}
func parseOrderId(orderId string) (planID string, userID int, err error) {
	// Минимальный формат: tbank_basic_12345_1700000000
	parts := strings.SplitN(orderId, "_", 4)
	if len(parts) < 4 || parts[0] != "tbank" {
		return "", 0, fmt.Errorf("неверный формат OrderId: %s", orderId)
	}

	planID = parts[1]
	uid, parseErr := strconv.Atoi(parts[2])
	if parseErr != nil {
		return "", 0, fmt.Errorf("неверный userID в OrderId %s: %w", orderId, parseErr)
	}

	return planID, uid, nil
}

// notifyUser отправляет уведомление пользователю в Telegram после успешной оплаты
func (s *serviceImpl) notifyUser(userID int, planID string) {
	if s.messageSender == nil || s.userService == nil {
		return
	}

	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		logger.Error("❌ Не удалось найти пользователя %d для уведомления: %v", userID, err)
		return
	}
	if user == nil {
		logger.Error("❌ Пользователь %d не найден для уведомления", userID)
		return
	}

	chatID := int64(0)
	if user.ChatID != "" {
		chatID, _ = strconv.ParseInt(user.ChatID, 10, 64)
	}
	if chatID == 0 {
		chatID = int64(userID)
	}

	planName := planNames[planID]
	if planName == "" {
		planName = planID
	}

	text := "✅ *Платёж успешно получен!*\n\n"
	text += fmt.Sprintf("📋 Тариф: *%s*\n", planName)
	text += "🎉 Ваша подписка активирована!\n\n"
	text += "Теперь вам доступны все функции выбранного тарифа."

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": "👤 Мой профиль", "callback_data": "profile_main"}},
			{{"text": "🏠 Главное меню", "callback_data": "menu_main"}},
		},
	}

	if err := s.messageSender.SendTextMessage(chatID, text, keyboard); err != nil {
		logger.Error("❌ Не удалось отправить уведомление об оплате пользователю %d: %v", userID, err)
	}
}

// notifyPaymentFailed отправляет уведомление об отклонённом платеже
func (s *serviceImpl) notifyPaymentFailed(userID int, planID string, errorCode string) {
	if s.messageSender == nil || s.userService == nil {
		return
	}

	user, err := s.userService.GetUserByID(userID)
	if err != nil || user == nil {
		return
	}

	chatID := int64(0)
	if user.ChatID != "" {
		chatID, _ = strconv.ParseInt(user.ChatID, 10, 64)
	}
	if chatID == 0 {
		chatID = int64(userID)
	}

	reason := rejectionReason(errorCode)
	planName := planNames[planID]
	if planName == "" {
		planName = planID
	}

	text := "❌ *Не получилось оплатить*\n\n"
	text += fmt.Sprintf("📋 Тариф: *%s*\n", planName)
	text += fmt.Sprintf("💬 Причина: %s\n\n", reason)
	text += "Попробуйте ещё раз или выберите другой способ оплаты."

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": "🔄 Попробовать снова", "callback_data": "payment_plan:" + planID}},
			{{"text": "🏠 Главное меню", "callback_data": "menu_main"}},
		},
	}

	if err := s.messageSender.SendTextMessage(chatID, text, keyboard); err != nil {
		logger.Error("❌ Не удалось отправить уведомление об ошибке оплаты пользователю %d: %v", userID, err)
	}
}

// rejectionReason возвращает понятную причину отклонения по коду ошибки Т-Банк
func rejectionReason(errorCode string) string {
	switch errorCode {
	case "1051":
		return "На карте недостаточно средств"
	case "1033", "1054":
		return "Срок действия карты истёк"
	case "1041":
		return "Карта утеряна"
	case "1043":
		return "Карта украдена"
	case "1082":
		return "Неверный CVV"
	case "1006", "1012", "1057":
		return "Операция не разрешена для данной карты"
	case "1089":
		return "Ошибка аутентификации 3DS"
	case "1091":
		return "Банк-эмитент недоступен"
	default:
		return "Банк отклонил платёж"
	}
}
