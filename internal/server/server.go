package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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
		ID:          secretID,
		UserID:      userID,
		SecretHash:  keyHash,
		SecretType:  "API_KEY",
		Permissions: string(permissionsJSON),
		DisplayHint: req.DisplayHint,
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
