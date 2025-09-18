package permission

// Permission represents a specific API permission
type Permission string

// Valid API permissions
const (
	CHECKOUT_API Permission = "CHECKOUT_API"
	BALANCE_API  Permission = "BALANCE_API"
	WEBHOOK_API  Permission = "WEBHOOK_API"
)

// Security strategies for webhook secrets
const (
	StrategySharedSecret  = "SHARED_SECRET"
	StrategySigningSecret = "SIGNING_SECRET"
)

// IsValid checks if the given permission is a valid permission constant
func IsValid(p Permission) bool {
	switch p {
	case CHECKOUT_API, BALANCE_API, WEBHOOK_API:
		return true
	default:
		return false
	}
}

// IsValidSecurityStrategy checks if the given strategy is valid
func IsValidSecurityStrategy(strategy string) bool {
	switch strategy {
	case StrategySharedSecret, StrategySigningSecret:
		return true
	default:
		return false
	}
}
