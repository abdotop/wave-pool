package domain

import (
	"database/sql/driver"
	"fmt"
)

type WebhookStatus string

const (
	WebhookStatusActive  WebhookStatus = "active"
	WebhookStatusRevoked WebhookStatus = "revoked"
)

func (e WebhookStatus) String() string {
	return string(e)
}

func (e *WebhookStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = WebhookStatus(s)
	case string:
		*e = WebhookStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for WebhookStatus: %T", src)
	}
	return nil
}

func (e WebhookStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type SigningStrategy string

const (
	SigningStrategySharedSecret  SigningStrategy = "shared_secret"
	SigningStrategySigningSecret SigningStrategy = "signing_secret"
)

func (e SigningStrategy) String() string {
	return string(e)
}

func (e *SigningStrategy) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = SigningStrategy(s)
	case string:
		*e = SigningStrategy(s)
	default:
		return fmt.Errorf("unsupported scan type for SigningStrategy: %T", src)
	}
	return nil
}

func (e SigningStrategy) Value() (driver.Value, error) {
	return string(e), nil
}
