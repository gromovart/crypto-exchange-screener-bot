// internal/infrastructure/persistence/postgres/models/api_key_types.go
package models

// ExchangeType тип биржи
type ExchangeType string

const (
	ExchangeBybit   ExchangeType = "bybit"
	ExchangeBinance ExchangeType = "binance"
	ExchangeKucoin  ExchangeType = "kucoin"
	ExchangeOKX     ExchangeType = "okx"
	ExchangeGateIO  ExchangeType = "gateio"
)

// PermissionType тип разрешения
type PermissionType string

const (
	PermissionReadOnly   PermissionType = "read_only"
	PermissionTrade      PermissionType = "trade"
	PermissionWithdraw   PermissionType = "withdraw"
	PermissionMargin     PermissionType = "margin"
	PermissionFutures    PermissionType = "futures"
	PermissionSpot       PermissionType = "spot"
	PermissionWallet     PermissionType = "wallet"
	PermissionSubAccount PermissionType = "sub_account"
)

// APIKeyAction действие с API ключом
type APIKeyAction string

const (
	APIKeyActionCreated           APIKeyAction = "key_created"
	APIKeyActionUsed              APIKeyAction = "key_used"
	APIKeyActionDeactivated       APIKeyAction = "key_deactivated"
	APIKeyActionRotated           APIKeyAction = "key_rotated"
	APIKeyActionPermissionGranted APIKeyAction = "permission_granted"
	APIKeyActionPermissionRevoked APIKeyAction = "permission_revoked"
)

// RotationReason причина ротации ключа
type RotationReason string

const (
	RotationReasonSecurity    RotationReason = "security_rotation"
	RotationReasonExpired     RotationReason = "key_expired"
	RotationReasonCompromised RotationReason = "key_compromised"
	RotationReasonRegular     RotationReason = "regular_rotation"
	RotationReasonUserRequest RotationReason = "user_request"
)

// IsValidExchange проверяет, является ли биржа валидной
func IsValidExchange(exchange string) bool {
	switch ExchangeType(exchange) {
	case ExchangeBybit,
		ExchangeBinance,
		ExchangeKucoin,
		ExchangeOKX,
		ExchangeGateIO:
		return true
	default:
		return false
	}
}

// IsValidPermission проверяет, является ли разрешение валидным
func IsValidPermission(permission string) bool {
	switch PermissionType(permission) {
	case PermissionReadOnly,
		PermissionTrade,
		PermissionWithdraw,
		PermissionMargin,
		PermissionFutures,
		PermissionSpot,
		PermissionWallet,
		PermissionSubAccount:
		return true
	default:
		return false
	}
}

// ExchangeDisplayName возвращает отображаемое имя биржи
func ExchangeDisplayName(exchange ExchangeType) string {
	switch exchange {
	case ExchangeBybit:
		return "Bybit"
	case ExchangeBinance:
		return "Binance"
	case ExchangeKucoin:
		return "KuCoin"
	case ExchangeOKX:
		return "OKX"
	case ExchangeGateIO:
		return "Gate.io"
	default:
		return string(exchange)
	}
}

// PermissionDisplayName возвращает отображаемое имя разрешения
func PermissionDisplayName(permission PermissionType) string {
	switch permission {
	case PermissionReadOnly:
		return "Read Only"
	case PermissionTrade:
		return "Trade"
	case PermissionWithdraw:
		return "Withdraw"
	case PermissionMargin:
		return "Margin"
	case PermissionFutures:
		return "Futures"
	case PermissionSpot:
		return "Spot"
	case PermissionWallet:
		return "Wallet"
	case PermissionSubAccount:
		return "Sub Account"
	default:
		return string(permission)
	}
}
