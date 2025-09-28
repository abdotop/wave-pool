package server

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/abdotop/wave-pool/internal/db"
	"github.com/abdotop/wave-pool/internal/domain/permission"
	"github.com/abdotop/wave-pool/internal/store"
	"github.com/abdotop/wave-pool/internal/utils"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	query *db.Queries

	mu sync.Mutex
	rl map[string]*lockout
}

type lockout struct {
	fails int
	until time.Time
}

var (
	e164Re = regexp.MustCompile(`^\+221(70|71|75|76|77|78)\d{7}$`)
)

func NewServer(st *store.Store) *server {
	return &server{
		query: db.New(st.DB),
		rl:    make(map[string]*lockout),
	}
}

// handleUserExists GET /v1/users/exists?phone_number=+15551234567
func (s *server) HandleUserExists(w http.ResponseWriter, r *http.Request) {
	phone := strings.TrimSpace(r.URL.Query().Get("phone_number"))
	phone = normalizeToSN(phone)
	slog.InfoContext(r.Context(), "Check user exists", slog.String("phone_number", phone))
	if !e164Re.MatchString(phone) {
		http.Error(w, "invalid phone_number", http.StatusBadRequest)
		return
	}
	// Query existence
	ctx := r.Context()
	_, err := s.query.GetUserByPhone(ctx, phone)
	exists := (err == nil)
	type resp struct {
		Exists bool `json:"exists"`
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp{Exists: exists})
}

type loginRequest struct {
	Phone string `json:"phone_number"`
	PIN   string `json:"pin"`
}

type loginResponse struct {
	SessionToken string `json:"session_token"`
	User         struct {
		ID          string `json:"id"`
		PhoneNumber string `json:"phone_number"`
	} `json:"user"`
}

// handleLogin POST /v1/auth/login
func (s *server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Phone = normalizeToSN(strings.TrimSpace(req.Phone))
	if !e164Re.MatchString(req.Phone) {
		http.Error(w, "invalid phone_number", http.StatusBadRequest)
		return
	}
	if err := utils.ValidatePIN(req.PIN); err != nil {
		http.Error(w, "invalid pin", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Rate limiting / lockout check
	if err := s.checkLockout(req.Phone); err != nil {
		http.Error(w, "too many attempts, try later", http.StatusTooManyRequests)
		return
	}

	u, err := s.query.GetUserByPhone(ctx, req.Phone)
	if err == nil {
		// Existing user: verify PIN
		if bcrypt.CompareHashAndPassword([]byte(u.PinHash), []byte(req.PIN)) != nil {
			s.registerFail(req.Phone)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.resetFail(req.Phone)
		// Create session
		token, exp, err := s.createSession(ctx, u.ID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = exp // reserved for future response use
		resp := loginResponse{SessionToken: token}
		resp.User.ID = u.ID
		resp.User.PhoneNumber = u.PhoneNumber
		utils.WriteJSON(w, http.StatusOK, resp)
		return
	}

	// User not found: sign-up
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	userID := ksuid.New().String()
	if err := s.query.CreateUser(ctx, db.CreateUserParams{
		ID:          userID,
		PhoneNumber: req.Phone,
		PinHash:     string(hashed),
	}); err != nil {
		// Possible race on unique phone_number
		http.Error(w, "conflict", http.StatusConflict)
		return
	}
	token, exp, err := s.createSession(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = exp
	resp := loginResponse{SessionToken: token}
	resp.User.ID = userID
	resp.User.PhoneNumber = req.Phone
	utils.WriteJSON(w, http.StatusCreated, resp)
}

func (s *server) createSession(ctx context.Context, userID string) (token string, expiresAt time.Time, err error) {
	token, err = utils.NewSessionToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().Add(24 * time.Hour).UTC()
	if err := s.query.CreateSession(ctx, db.CreateSessionParams{
		ID:        token,
		UserID:    userID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// Rate limiter helpers
func (s *server) checkLockout(phone string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.rl[phone]
	if st == nil {
		return nil
	}
	if st.until.After(time.Now()) {
		return errors.New("locked")
	}
	return nil
}

func (s *server) registerFail(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.rl[phone]
	if st == nil {
		st = &lockout{}
		s.rl[phone] = st
	}
	st.fails++
	if st.fails >= 5 {
		st.until = time.Now().Add(15 * time.Minute)
		st.fails = 0 // reset after lockout period set
	}
}

func (s *server) resetFail(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rl, phone)
}

// normalizeToSN normalizes common Senegal phone inputs to E.164 +221 format
func normalizeToSN(in string) string {
	in = strings.TrimSpace(in)
	// keep only digits and plus
	var b []rune
	for _, r := range in {
		if (r >= '0' && r <= '9') || r == '+' {
			b = append(b, r)
		}
	}
	s := string(b)
	if strings.HasPrefix(s, "00") {
		s = "+" + s[2:]
	}
	if strings.HasPrefix(s, "+221") {
		return s
	}
	if strings.HasPrefix(s, "221") {
		return "+" + s
	}
	return s
}

// Context key for storing user ID in request context
type contextKey string

const userIDContextKey contextKey = "userID"

// APIError represents a standard error response for API endpoints
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeAPIError writes a standard API error response
func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(APIError{
		Code:    code,
		Message: message,
	})
	if err != nil {
		slog.Error("Failed to write API error response", slog.String("error", err.Error()))
	}
}

// SessionAuthMiddleware validates session tokens and adds user ID to context
func (s *server) SessionAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			http.Error(w, "missing session token", http.StatusUnauthorized)
			return
		}

		// Validate session
		ctx := r.Context()
		session, err := s.query.GetSession(ctx, token)
		if err != nil {
			http.Error(w, "invalid session token", http.StatusUnauthorized)
			return
		}

		// Check if session is expired
		expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil || expiresAt.Before(time.Now()) {
			http.Error(w, "session expired", http.StatusUnauthorized)
			return
		}

		// Add user ID to context
		ctx = context.WithValue(ctx, userIDContextKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

type createSecretRequest struct {
	DisplayHint string                  `json:"display_hint"`
	Permissions []permission.Permission `json:"permissions"`
}

type createSecretResponse struct {
	SecretID string `json:"secret_id"`
	APIKey   string `json:"api_key"`
}

// HandleCreateSecret POST /v1/secrets
func (s *server) HandleCreateSecret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by session middleware)
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Validate permissions
	if len(req.Permissions) == 0 {
		http.Error(w, "at least one permission is required", http.StatusBadRequest)
		return
	}

	var validPermissions []string
	for _, perm := range req.Permissions {
		if !permission.IsValid(perm) {
			http.Error(w, "invalid permission: "+string(perm), http.StatusBadRequest)
			return
		}
		validPermissions = append(validPermissions, string(perm))
	}

	// Generate API key
	apiKey, err := utils.NewAPIKey()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Hash the API key
	keyHash := utils.HashAPIKey(apiKey)

	// Create secret record
	secretID := ksuid.New().String()
	permissionsJSON, err := json.Marshal(validPermissions)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := s.query.CreateSecret(ctx, db.CreateSecretParams{
		ID:                      secretID,
		UserID:                  userID,
		SecretHash:              keyHash,
		SecretType:              "API_KEY",
		Permissions:             string(permissionsJSON),
		DisplayHint:             req.DisplayHint,
		WebhookUrl:              sql.NullString{}, // empty for API keys
		WebhookSecurityStrategy: sql.NullString{}, // empty for API keys
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return the plaintext API key (only time it's shown)
	resp := createSecretResponse{
		SecretID: secretID,
		APIKey:   apiKey,
	}
	utils.WriteJSON(w, http.StatusCreated, resp)
}

// HandleRevokeSecret DELETE /v1/secrets/{secret_id}
func (s *server) HandleRevokeSecret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by session middleware)
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract secret ID from URL path
	secretID := strings.TrimPrefix(r.URL.Path, "/v1/secrets/")
	if secretID == "" || secretID == r.URL.Path {
		http.Error(w, "invalid secret ID", http.StatusBadRequest)
		return
	}

	// Get the secret to verify ownership
	secret, err := s.query.GetSecretByID(ctx, secretID)
	if err != nil {
		http.Error(w, "secret not found", http.StatusNotFound)
		return
	}

	// Verify the secret belongs to the logged-in user
	if secret.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Check if already revoked
	if secret.RevokedAt.Valid {
		http.Error(w, "secret already revoked", http.StatusConflict)
		return
	}

	// Revoke the secret
	if err := s.query.RevokeSecret(ctx, db.RevokeSecretParams{
		RevokedAt: sql.NullString{
			String: time.Now().UTC().Format(time.RFC3339),
			Valid:  true,
		},
		ID: secretID,
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// Context key for storing permissions in request context
const permissionsContextKey contextKey = "permissions"

// Checkout Session Data Models

// LastPaymentError represents the nested `last_payment_error` object.
type LastPaymentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CheckoutSession represents the full Checkout Session object.
type CheckoutSession struct {
	ID                   string            `json:"id"`
	Amount               string            `json:"amount"`
	CheckoutStatus       string            `json:"checkout_status"` // "open", "complete", or "expired"
	ClientReference      *string           `json:"client_reference,omitempty"`
	Currency             string            `json:"currency"`
	ErrorURL             string            `json:"error_url"`
	LastPaymentError     *LastPaymentError `json:"last_payment_error,omitempty"`
	BusinessName         string            `json:"business_name"`
	PaymentStatus        string            `json:"payment_status"` // "processing", "cancelled", or "succeeded"
	TransactionID        *string           `json:"transaction_id,omitempty"`
	AggregatedMerchantID *string           `json:"aggregated_merchant_id,omitempty"`
	SuccessURL           string            `json:"success_url"`
	WaveLaunchURL        string            `json:"wave_launch_url"`
	WhenCompleted        *string           `json:"when_completed,omitempty"`
	WhenCreated          string            `json:"when_created"`
	WhenExpires          string            `json:"when_expires"`
}

// CreateCheckoutSessionRequest represents the POST request body for creating checkout sessions
type CreateCheckoutSessionRequest struct {
	Amount               string  `json:"amount"`
	ClientReference      *string `json:"client_reference,omitempty"`
	Currency             string  `json:"currency"`
	ErrorURL             string  `json:"error_url"`
	SuccessURL           string  `json:"success_url"`
	RestrictPayerMobile  *string `json:"restrict_payer_mobile,omitempty"`
	EnforcePayerMobile   *string `json:"enforce_payer_mobile,omitempty"` // For backward compatibility
	AggregatedMerchantID *string `json:"aggregated_merchant_id,omitempty"`
}

// APIKeyAuthMiddleware validates API keys and adds permissions to context
func (s *server) APIKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check for Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAPIError(w, http.StatusUnauthorized, "missing-auth-header", "Your request should include an HTTP auth header.")
			return
		}

		// Check header format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeAPIError(w, http.StatusUnauthorized, "invalid-auth", "Your HTTP auth header can't be processed.")
			return
		}

		// Check for the key itself
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			writeAPIError(w, http.StatusUnauthorized, "api-key-not-provided", "Your request should include an API key.")
			return
		}

		// Validate the key - compute SHA-256 hash
		keyHash := utils.HashAPIKey(token)

		// Query the secrets table for a record with matching secret_hash
		secret, err := s.query.GetSecretByHash(ctx, keyHash)
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "no-matching-api-key", "The key you provided doesn't exist in our system.")
			return
		}

		// Check if the key is revoked
		if secret.RevokedAt.Valid {
			writeAPIError(w, http.StatusUnauthorized, "api-key-revoked", "Your API key has been revoked.")
			return
		}

		// Internal validation - verify secret_type is 'API_KEY'
		if secret.SecretType != "API_KEY" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// Parse permissions from database and add to context
		var permissions []string
		if err := json.Unmarshal([]byte(secret.Permissions), &permissions); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Add permissions to context
		ctx = context.WithValue(ctx, permissionsContextKey, permissions)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetPermissionsFromContext extracts the permissions from the request context
func GetPermissionsFromContext(ctx context.Context) ([]string, bool) {
	permissions, ok := ctx.Value(permissionsContextKey).([]string)
	return permissions, ok
}

// Helper function to check if user has specific permission
func hasPermission(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == required {
			return true
		}
	}
	return false
}

// convertToAPICheckoutSession converts database model to API response
func convertToAPICheckoutSession(dbSession db.CheckoutSession) CheckoutSession {
	session := CheckoutSession{
		ID:             dbSession.ID,
		Amount:         dbSession.Amount,
		CheckoutStatus: dbSession.CheckoutStatus,
		Currency:       dbSession.Currency,
		ErrorURL:       dbSession.ErrorUrl,
		BusinessName:   dbSession.BusinessName.String,
		PaymentStatus:  dbSession.PaymentStatus,
		SuccessURL:     dbSession.SuccessUrl,
		WaveLaunchURL:  dbSession.WaveLaunchUrl,
		WhenCreated:    dbSession.WhenCreated,
		WhenExpires:    dbSession.WhenExpires,
	}

	// Handle optional fields
	if dbSession.ClientReference.Valid {
		session.ClientReference = &dbSession.ClientReference.String
	}
	if dbSession.TransactionID.Valid {
		session.TransactionID = &dbSession.TransactionID.String
	}
	if dbSession.AggregatedMerchantID.Valid {
		session.AggregatedMerchantID = &dbSession.AggregatedMerchantID.String
	}
	if dbSession.WhenCompleted.Valid {
		session.WhenCompleted = &dbSession.WhenCompleted.String
	}

	// Handle last payment error
	if dbSession.LastPaymentErrorCode.Valid && dbSession.LastPaymentErrorMessage.Valid {
		session.LastPaymentError = &LastPaymentError{
			Code:    dbSession.LastPaymentErrorCode.String,
			Message: dbSession.LastPaymentErrorMessage.String,
		}
	}

	return session
}

// HandleCreateCheckoutSession POST /v1/checkout/sessions
func (s *server) HandleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get permissions from context (set by API key middleware)
	permissions, ok := GetPermissionsFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No permissions found in context")
		return
	}

	// Check if user has CHECKOUT_API permission
	if !hasPermission(permissions, "CHECKOUT_API") {
		writeAPIError(w, http.StatusForbidden, "insufficient-permissions", "This API key does not have permission to access checkout operations")
		return
	}

	// Parse request body
	var req CreateCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "request-validation-error", "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if req.Amount == "" {
		writeAPIError(w, http.StatusBadRequest, "request-validation-error", "amount is required")
		return
	}
	if req.Currency == "" {
		writeAPIError(w, http.StatusBadRequest, "request-validation-error", "currency is required")
		return
	}
	if req.ErrorURL == "" {
		writeAPIError(w, http.StatusBadRequest, "request-validation-error", "error_url is required")
		return
	}
	if req.SuccessURL == "" {
		writeAPIError(w, http.StatusBadRequest, "request-validation-error", "success_url is required")
		return
	}

	// Handle backward compatibility for enforce_payer_mobile
	var restrictPayerMobile *string
	if req.RestrictPayerMobile != nil {
		restrictPayerMobile = req.RestrictPayerMobile
	} else if req.EnforcePayerMobile != nil {
		restrictPayerMobile = req.EnforcePayerMobile
	}

	// Generate unique ID with cos_ prefix (20 characters total)
	// Format: cos-xxxxxxxxxxxx (20 characters)
	sessionID := "cos-" + utils.GenerateRandomID(16) // cos- (4) + 16 random chars = 20 total

	// Generate transaction ID if not provided
	transactionID := ksuid.New().String()

	// Set timestamps
	now := time.Now().UTC()
	whenCreated := now.Format(time.RFC3339)
	whenExpires := now.Add(30 * time.Minute).Format(time.RFC3339)

		// Construct Wave launch URL (simulated)
		waveLaunchURL := cmp.Or(os.Getenv("WAVE_LAUNCH_URL"), "local.wave.pool") + "/pay/" + sessionID

	// Create database record
	createParams := db.CreateCheckoutSessionParams{
		ID:             sessionID,
		Amount:         req.Amount,
		CheckoutStatus: "open",
		Currency:       req.Currency,
		ErrorUrl:       req.ErrorURL,
		SuccessUrl:     req.SuccessURL,
		PaymentStatus:  "processing",
		TransactionID:  sql.NullString{String: transactionID, Valid: true},
		WaveLaunchUrl:  waveLaunchURL,
		WhenCreated:    whenCreated,
		WhenExpires:    whenExpires,
		BusinessName:   sql.NullString{String: "Wave Pool Simulator", Valid: true},
	}

	// Handle optional fields
	if req.ClientReference != nil {
		createParams.ClientReference = sql.NullString{String: *req.ClientReference, Valid: true}
	}
	if req.AggregatedMerchantID != nil {
		createParams.AggregatedMerchantID = sql.NullString{String: *req.AggregatedMerchantID, Valid: true}
	}
	if restrictPayerMobile != nil {
		createParams.RestrictPayerMobile = sql.NullString{String: *restrictPayerMobile, Valid: true}
		// Also set enforce_payer_mobile for backward compatibility
		createParams.EnforcePayerMobile = sql.NullString{String: *restrictPayerMobile, Valid: true}
	}

	// Save to database
	if err := s.query.CreateCheckoutSession(ctx, createParams); err != nil {
		slog.ErrorContext(ctx, "Failed to create checkout session", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to create checkout session")
		return
	}

	// Return the created session
	response := CheckoutSession{
		ID:             sessionID,
		Amount:         req.Amount,
		CheckoutStatus: "open",
		Currency:       req.Currency,
		ErrorURL:       req.ErrorURL,
		BusinessName:   "Wave Pool Simulator",
		PaymentStatus:  "processing",
		TransactionID:  &transactionID,
		SuccessURL:     req.SuccessURL,
		WaveLaunchURL:  waveLaunchURL,
		WhenCreated:    whenCreated,
		WhenExpires:    whenExpires,
	}

	if req.ClientReference != nil {
		response.ClientReference = req.ClientReference
	}
	if req.AggregatedMerchantID != nil {
		response.AggregatedMerchantID = req.AggregatedMerchantID
	}

	utils.WriteJSON(w, http.StatusCreated, response)
}

// HandleGetCheckoutSession GET /v1/checkout/sessions/:id
func (s *server) HandleGetCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get permissions from context (set by API key middleware)
	permissions, ok := GetPermissionsFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No permissions found in context")
		return
	}

	// Check if user has CHECKOUT_API permission
	if !hasPermission(permissions, "CHECKOUT_API") {
		writeAPIError(w, http.StatusForbidden, "insufficient-permissions", "This API key does not have permission to access checkout operations")
		return
	}

	// Extract session ID from URL path
	// Expected format: /v1/checkout/sessions/cos-xxxxxxxxxxxx
	sessionID := r.PathValue("id")
	if sessionID == "" || sessionID == r.URL.Path {
		writeAPIError(w, http.StatusBadRequest, "invalid-session-id", "Invalid session ID in URL")
		return
	}

	// Get session from database
	dbSession, err := s.query.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusNotFound, "checkout-session-not-found", "The checkout session was not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to get checkout session", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to retrieve checkout session")
		return
	}

	// Convert to API response format
	response := convertToAPICheckoutSession(dbSession)
	utils.WriteJSON(w, http.StatusOK, response)
}

// HandleGetCheckoutSessionByTransactionID GET /v1/checkout/sessions?transaction_id=xxx
func (s *server) HandleGetCheckoutSessionByTransactionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get permissions from context (set by API key middleware)
	permissions, ok := GetPermissionsFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No permissions found in context")
		return
	}

	// Check if user has CHECKOUT_API permission
	if !hasPermission(permissions, "CHECKOUT_API") {
		writeAPIError(w, http.StatusForbidden, "insufficient-permissions", "This API key does not have permission to access checkout operations")
		return
	}

	// Get transaction_id from query parameters
	transactionID := strings.TrimSpace(r.URL.Query().Get("transaction_id"))
	if transactionID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing-transaction-id", "transaction_id query parameter is required")
		return
	}

	// Get session from database
	dbSession, err := s.query.GetCheckoutSessionByTransactionID(ctx, sql.NullString{String: transactionID, Valid: true})
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusNotFound, "checkout-session-not-found", "The checkout session was not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to get checkout session by transaction ID", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to retrieve checkout session")
		return
	}

	// Convert to API response format
	response := convertToAPICheckoutSession(dbSession)
	utils.WriteJSON(w, http.StatusOK, response)
}

// HandleSearchCheckoutSessions GET /v1/checkout/sessions/search?client_reference=xxx
func (s *server) HandleSearchCheckoutSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get permissions from context (set by API key middleware)
	permissions, ok := GetPermissionsFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No permissions found in context")
		return
	}

	// Check if user has CHECKOUT_API permission
	if !hasPermission(permissions, "CHECKOUT_API") {
		writeAPIError(w, http.StatusForbidden, "insufficient-permissions", "This API key does not have permission to access checkout operations")
		return
	}

	// Get client_reference from query parameters
	clientReference := strings.TrimSpace(r.URL.Query().Get("client_reference"))
	if clientReference == "" {
		writeAPIError(w, http.StatusBadRequest, "missing-client-reference", "client_reference query parameter is required")
		return
	}

	// Get sessions from database
	dbSessions, err := s.query.GetCheckoutSessionsByClientReference(ctx, sql.NullString{String: clientReference, Valid: true})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to search checkout sessions", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to search checkout sessions")
		return
	}

	// Convert to API response format
	var results []CheckoutSession
	for _, dbSession := range dbSessions {
		results = append(results, convertToAPICheckoutSession(dbSession))
	}

	// Return in the specified format: {"result": [...]}
	response := map[string][]CheckoutSession{
		"result": results,
	}
	utils.WriteJSON(w, http.StatusOK, response)
}

// HandleExpireCheckoutSession POST /v1/checkout/sessions/:id/expire
func (s *server) HandleExpireCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get permissions from context (set by API key middleware)
	permissions, ok := GetPermissionsFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No permissions found in context")
		return
	}

	// Check if user has CHECKOUT_API permission
	if !hasPermission(permissions, "CHECKOUT_API") {
		writeAPIError(w, http.StatusForbidden, "insufficient-permissions", "This API key does not have permission to access checkout operations")
		return
	}

	// Extract session ID from URL path
	// Expected format: /v1/checkout/sessions/cos-xxxxxxxxxxxx/expire
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid-session-id", "Invalid session ID in URL")
		return
	}

	// Get current session to check status
	dbSession, err := s.query.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusNotFound, "checkout-session-not-found", "The checkout session was not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to get checkout session", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to retrieve checkout session")
		return
	}

	// Check if session is already completed or expired
	if dbSession.CheckoutStatus == "complete" || dbSession.CheckoutStatus == "expired" {
		writeAPIError(w, http.StatusConflict, "session-already-finalized", "The checkout session has already been completed or expired")
		return
	}

	// Update session to expired status
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.query.UpdateCheckoutSessionStatus(ctx, db.UpdateCheckoutSessionStatusParams{
		CheckoutStatus: "expired",
		WhenCompleted:  sql.NullString{String: now, Valid: true},
		ID:             sessionID,
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to expire checkout session", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to expire checkout session")
		return
	}

	// Return 200 OK with empty body
	w.WriteHeader(http.StatusOK)
}

// HandleRefundCheckoutSession POST /v1/checkout/sessions/:id/refund
func (s *server) HandleRefundCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get permissions from context (set by API key middleware)
	permissions, ok := GetPermissionsFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No permissions found in context")
		return
	}

	// Check if user has CHECKOUT_API permission
	if !hasPermission(permissions, "CHECKOUT_API") {
		writeAPIError(w, http.StatusForbidden, "insufficient-permissions", "This API key does not have permission to access checkout operations")
		return
	}

	// Extract session ID from URL path
	// Expected format: /v1/checkout/sessions/cos-xxxxxxxxxxxx/refund
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid-session-id", "Invalid session ID in URL")
		return
	}

	// Get current session to check status
	dbSession, err := s.query.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeAPIError(w, http.StatusNotFound, "checkout-session-not-found", "The checkout session was not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to get checkout session", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to retrieve checkout session")
		return
	}

	// Check for idempotency - if already refunded, return success
	if dbSession.WhenRefunded.Valid {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check if payment status is succeeded
	if dbSession.PaymentStatus != "succeeded" {
		writeAPIError(w, http.StatusBadRequest, "checkout-refund-failed", "Can only refund payments that have succeeded")
		return
	}

	// Update session to mark as refunded
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.query.UpdateCheckoutSessionRefund(ctx, db.UpdateCheckoutSessionRefundParams{
		WhenRefunded: sql.NullString{String: now, Valid: true},
		ID:           sessionID,
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to refund checkout session", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to refund checkout session")
		return
	}

	// Return 200 OK with empty body
	w.WriteHeader(http.StatusOK)
}

// HandleListSecrets GET /api/v1/portal/secrets  
func (s *server) HandleListSecrets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context (set by session middleware)
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No user ID found in context")
		return
	}

	// Get secrets from database (limit to 100 for now)
	secrets, err := s.query.ListSecretsByUser(ctx, db.ListSecretsByUserParams{
		UserID: userID,
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list secrets", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to list secrets")
		return
	}

	// Convert to response format (exclude sensitive data)
	type secretResponse struct {
		ID          string `json:"id"`
		SecretType  string `json:"secret_type"`
		DisplayHint string `json:"display_hint"`
		Permissions string `json:"permissions,omitempty"`
		WebhookURL  string `json:"webhook_url,omitempty"`
		CreatedAt   string `json:"created_at"`
		RevokedAt   string `json:"revoked_at,omitempty"`
	}

	var response []secretResponse
	for _, secret := range secrets {
		resp := secretResponse{
			ID:          secret.ID,
			SecretType:  secret.SecretType,
			DisplayHint: secret.DisplayHint,
			CreatedAt:   secret.CreatedAt,
		}
		
		if secret.SecretType == "API_KEY" {
			resp.Permissions = secret.Permissions
		} else if secret.SecretType == "WEBHOOK_SECRET" && secret.WebhookUrl.Valid {
			resp.WebhookURL = secret.WebhookUrl.String
		}
		
		if secret.RevokedAt.Valid {
			resp.RevokedAt = secret.RevokedAt.String
		}
		
		response = append(response, resp)
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

// CreateWebhookRequest represents the POST request body for creating webhooks
type CreateWebhookRequest struct {
	URL              string   `json:"url"`
	Events           []string `json:"events"`
	SecurityStrategy string   `json:"security_strategy"` // "shared_secret" or "signing_secret"
	DisplayHint      string   `json:"display_hint"`
}

type CreateWebhookResponse struct {
	WebhookID string `json:"webhook_id"`
	Secret    string `json:"webhook_secret"`
}

// HandleCreateWebhook POST /api/v1/portal/webhooks
func (s *server) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context (set by session middleware)
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No user ID found in context")
		return
	}

	// Parse request body
	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid-request", "Invalid JSON request body")
		return
	}

	// Validate required fields
	if req.URL == "" {
		writeAPIError(w, http.StatusBadRequest, "missing-url", "webhook URL is required")
		return
	}
	if req.SecurityStrategy == "" {
		req.SecurityStrategy = "shared_secret" // default
	}
	if req.SecurityStrategy != "shared_secret" && req.SecurityStrategy != "signing_secret" {
		writeAPIError(w, http.StatusBadRequest, "invalid-security-strategy", "security_strategy must be 'shared_secret' or 'signing_secret'")
		return
	}
	if req.DisplayHint == "" {
		req.DisplayHint = "Webhook for " + req.URL
	}

	// Generate webhook secret and hash it
	webhookSecret, err := utils.NewSessionToken() // reuse session token generation for webhook secrets
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate webhook secret", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to generate webhook secret")
		return
	}
	secretHash := utils.HashAPIKey(webhookSecret)
	webhookID := ksuid.New().String()

	// Create webhook record
	err = s.query.CreateSecret(ctx, db.CreateSecretParams{
		ID:                      webhookID,
		UserID:                  userID,
		SecretHash:              secretHash,
		SecretType:              "WEBHOOK_SECRET",
		Permissions:             "[]", // webhooks don't use permissions
		DisplayHint:             req.DisplayHint,
		WebhookUrl:              sql.NullString{String: req.URL, Valid: true},
		WebhookSecurityStrategy: sql.NullString{String: req.SecurityStrategy, Valid: true},
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create webhook", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to create webhook")
		return
	}

	// Return response with the webhook secret (only shown once)
	resp := CreateWebhookResponse{
		WebhookID: webhookID,
		Secret:    webhookSecret,
	}
	utils.WriteJSON(w, http.StatusCreated, resp)
}

// HandleListCheckoutSessions GET /api/v1/portal/checkout-sessions
func (s *server) HandleListCheckoutSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context (set by session middleware)
	_, ok := GetUserIDFromContext(ctx)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "No user ID found in context")
		return
	}

	// Get checkout sessions from database (limit to 100 for now)
	// In a real implementation, this would filter by the user's merchant account
	dbSessions, err := s.query.ListCheckoutSessionsByUser(ctx, db.ListCheckoutSessionsByUserParams{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list checkout sessions", slog.String("error", err.Error()))
		writeAPIError(w, http.StatusInternalServerError, "internal-error", "Failed to list checkout sessions")
		return
	}

	// Convert to API response format
	var sessions []CheckoutSession
	for _, dbSession := range dbSessions {
		sessions = append(sessions, convertToAPICheckoutSession(dbSession))
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// HandlePaymentPageGET GET /pay/{session_id}
func (s *server) HandlePaymentPageGET(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Get session from database
	ctx := r.Context()
	dbSession, err := s.query.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "Failed to get checkout session", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if session is still valid
	if dbSession.CheckoutStatus != "open" {
		http.Error(w, "Session is no longer valid", http.StatusBadRequest)
		return
	}

	// Generate QR code content (deep link for the Flutter app)
	deepLink := "wavepool://pay/" + sessionID

	// Serve HTML page with payment details and QR code
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Wave Payment - ` + sessionID + `</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; text-align: center; }
        .payment-details { background: #f5f5f5; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .amount { font-size: 2em; color: #2196F3; font-weight: bold; }
        .qr-code { margin: 20px 0; }
        .instructions { color: #666; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Complete Your Payment</h1>
    <div class="payment-details">
        <div class="amount">` + dbSession.Amount + ` ` + dbSession.Currency + `</div>
        <div class="merchant">` + dbSession.BusinessName.String + `</div>
    </div>
    
    <div class="qr-code">
        <div id="qrcode"></div>
    </div>
    
    <div class="instructions">
        Scan this QR code with the Wave Pool mobile app to simulate the payment
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/qrcode@1.5.3/build/qrcode.min.js"></script>
    <script>
        QRCode.toCanvas(document.getElementById('qrcode'), '` + deepLink + `', {
            width: 256,
            margin: 2,
            color: {
                dark: '#000000',
                light: '#FFFFFF'
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// PaymentStatusRequest represents the POST request body for payment status updates
type PaymentStatusRequest struct {
	Status string `json:"status"` // "succeeded" or "failed"
}

// HandlePaymentPagePOST POST /pay/{session_id}
func (s *server) HandlePaymentPagePOST(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Parse request body
	var req PaymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate status
	if req.Status != "succeeded" && req.Status != "failed" {
		http.Error(w, "Status must be 'succeeded' or 'failed'", http.StatusBadRequest)
		return
	}

	// Get current session
	dbSession, err := s.query.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "Failed to get checkout session", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if session is still valid for payment
	if dbSession.CheckoutStatus != "open" {
		http.Error(w, "Session is no longer valid for payment", http.StatusBadRequest)
		return
	}

	// Update session status
	now := time.Now().UTC().Format(time.RFC3339)
	var checkoutStatus string
	
	if req.Status == "succeeded" {
		checkoutStatus = "complete"
	} else {
		checkoutStatus = "complete" // both failed and succeeded mark as complete
	}

	err = s.query.UpdateCheckoutSessionStatus(ctx, db.UpdateCheckoutSessionStatusParams{
		CheckoutStatus: checkoutStatus,
		WhenCompleted:  sql.NullString{String: now, Valid: true},
		ID:             sessionID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update checkout session", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// TODO: Trigger webhook delivery here
	// For now, just log the event
	slog.InfoContext(ctx, "Payment simulation completed", 
		slog.String("session_id", sessionID),
		slog.String("status", req.Status))

	// Return success
	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Payment status updated",
	})
}
