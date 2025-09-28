package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
)

func NewSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// NewAPIKey generates a new API key with a "wv_" prefix
func NewAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "wv_" + hex.EncodeToString(b), nil
}

// HashAPIKey computes the SHA-256 hash of an API key
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GenerateRandomID generates a random ID of specified length using alphanumeric characters
func GenerateRandomID(length int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple method if crypto/rand fails
		return hex.EncodeToString(b)[:length]
	}

	result := make([]byte, length)
	for i := range result {
		result[i] = chars[b[i]%byte(len(chars))]
	}
	return string(result)
}

func ValidatePIN(pin string) error {
	if len(pin) != 4 {
		return errors.New("pin must be 4 digits")
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return errors.New("pin must be digits")
		}
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
