package permission

// Permission represents a specific API permission
type Permission string

// Valid API permissions
const (
	CHECKOUT_API Permission = "CHECKOUT_API"
	BALANCE_API  Permission = "BALANCE_API"
)

// IsValid checks if the given permission is a valid permission constant
func IsValid(p Permission) bool {
	switch p {
	case CHECKOUT_API, BALANCE_API:
		return true
	default:
		return false
	}
}
