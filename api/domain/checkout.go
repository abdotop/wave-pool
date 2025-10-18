package domain

import "time"

// CreateCheckoutSessionRequest represents the request body for creating a new checkout session.
type CreateCheckoutSessionRequest struct {
	Amount               string `json:"amount" validate:"required,numeric"`
	Currency             string `json:"currency" validate:"required,iso4217"`
	ClientReference      string `json:"client_reference,omitempty" validate:"max=255"`
	RestrictPayerMobile  string `json:"restrict_payer_mobile,omitempty" validate:"e164"`
	ErrorURL             string `json:"error_url" validate:"required,url,startswith=http"`
	SuccessURL           string `json:"success_url" validate:"required,url,startswith=http"`
	AggregatedMerchantID string `json:"aggregated_merchant_id,omitempty"`
}

// LastPaymentError represents the error details for the last failed payment attempt.
type LastPaymentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// CheckoutSessionResponse represents the data returned after creating a checkout session.
type CheckoutSessionResponse struct {
	ID                   string            `json:"id"`
	Amount               string            `json:"amount"`
	CheckoutStatus       string            `json:"checkout_status"`
	ClientReference      *string           `json:"client_reference,omitempty"`
	Currency             string            `json:"currency"`
	ErrorURL             string            `json:"error_url"`
	LastPaymentError     *LastPaymentError `json:"last_payment_error,omitempty"`
	BusinessName         string            `json:"business_name"`
	PaymentStatus        string            `json:"payment_status"`
	TransactionID        *string           `json:"transaction_id,omitempty"`
	AggregatedMerchantID *string           `json:"aggregated_merchant_id,omitempty"`
	SuccessURL           string            `json:"success_url"`
	WaveLaunchURL        string            `json:"wave_launch_url"`
	WhenCompleted        *time.Time        `json:"when_completed,omitempty"`
	WhenCreated          time.Time         `json:"when_created"`
	WhenExpires          time.Time         `json:"when_expires"`
	RestrictPayerMobile  *string           `json:"restrict_payer_mobile,omitempty"`
}
