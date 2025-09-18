package permission

import (
	"testing"
)

func TestPermissionConstants(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected string
	}{
		{
			name:     "CHECKOUT_API constant value",
			perm:     CHECKOUT_API,
			expected: "CHECKOUT_API",
		},
		{
			name:     "BALANCE_API constant value",
			perm:     BALANCE_API,
			expected: "BALANCE_API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.perm) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.perm))
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected bool
	}{
		{
			name:     "CHECKOUT_API is valid",
			perm:     CHECKOUT_API,
			expected: true,
		},
		{
			name:     "BALANCE_API is valid",
			perm:     BALANCE_API,
			expected: true,
		},
		{
			name:     "Empty permission is invalid",
			perm:     Permission(""),
			expected: false,
		},
		{
			name:     "Random string is invalid",
			perm:     Permission("INVALID_PERMISSION"),
			expected: false,
		},
		{
			name:     "Case sensitive - lowercase checkout_api is invalid",
			perm:     Permission("checkout_api"),
			expected: false,
		},
		{
			name:     "Case sensitive - lowercase balance_api is invalid",
			perm:     Permission("balance_api"),
			expected: false,
		},
		{
			name:     "Mixed case is invalid",
			perm:     Permission("Checkout_Api"),
			expected: false,
		},
		{
			name:     "Permission with spaces is invalid",
			perm:     Permission("CHECKOUT API"),
			expected: false,
		},
		{
			name:     "Permission with extra characters is invalid",
			perm:     Permission("CHECKOUT_API_EXTRA"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValid(tt.perm)
			if result != tt.expected {
				t.Errorf("IsValid(%q) = %v, expected %v", tt.perm, result, tt.expected)
			}
		})
	}
}

func TestPermissionType(t *testing.T) {
	// Test that Permission is a string type
	var p Permission = "test"
	if string(p) != "test" {
		t.Errorf("Permission type conversion failed")
	}

	// Test that constants can be used as Permission type
	var checkout Permission = CHECKOUT_API
	var balance Permission = BALANCE_API

	if checkout != CHECKOUT_API {
		t.Errorf("CHECKOUT_API constant assignment failed")
	}

	if balance != BALANCE_API {
		t.Errorf("BALANCE_API constant assignment failed")
	}
}

// Benchmark tests for performance
func BenchmarkIsValid(b *testing.B) {
	permissions := []Permission{
		CHECKOUT_API,
		BALANCE_API,
		Permission("INVALID"),
		Permission(""),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, perm := range permissions {
			IsValid(perm)
		}
	}
}
