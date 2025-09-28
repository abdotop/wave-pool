package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abdotop/wave-pool/internal/db"
	"github.com/abdotop/wave-pool/internal/domain/permission"
	"github.com/abdotop/wave-pool/internal/store"
	"github.com/abdotop/wave-pool/internal/utils"
	"github.com/segmentio/ksuid"
)

// Helper function to create API key for testing
func createTestAPIKey(t *testing.T, srv *server) string {
	t.Helper()

	// Create a test user first
	userBody := map[string]string{"phone_number": "+221785626022", "pin": "1234"}
	b, _ := json.Marshal(userBody)
	lreq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	lreq.Header.Set("Content-Type", "application/json")
	lrr := httptest.NewRecorder()
	srv.HandleLogin(lrr, lreq)
	if lrr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d", lrr.Code)
	}

	var loginResp loginResponse
	if err := json.Unmarshal(lrr.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	// Create an API key with CHECKOUT_API permission
	secretBody := createSecretRequest{
		DisplayHint: "Test API Key",
		Permissions: []permission.Permission{permission.CHECKOUT_API},
	}
	sb, _ := json.Marshal(secretBody)
	sreq := httptest.NewRequest(http.MethodPost, "/v1/secrets", bytes.NewReader(sb))
	sreq.Header.Set("Content-Type", "application/json")
	sreq.Header.Set("Authorization", "Bearer "+loginResp.SessionToken)

	// Apply the session middleware for the secret creation
	srr := httptest.NewRecorder()
	middleware := srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleCreateSecret))
	middleware.ServeHTTP(srr, sreq)
	if srr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created for secret, got %d: %s", srr.Code, srr.Body.String())
	}

	var secretResp createSecretResponse
	if err := json.Unmarshal(srr.Body.Bytes(), &secretResp); err != nil {
		t.Fatalf("failed to parse secret response: %v", err)
	}

	return secretResp.APIKey
}

func newTestServer(t *testing.T) *server {
	t.Helper()
	tmp := t.TempDir()
	st, err := store.New(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	if err := st.AutoMigrate(filepath.Join("..", "..", "db", "migrations")); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return NewServer(st)
}

func TestHandleUserExists_FalseThenTrue(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/users/exists?phone_number=221785626022", nil)
	rr := httptest.NewRecorder()
	srv.HandleUserExists(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("false")) {
		t.Fatalf("expected exists=false, got %s", rr.Body.String())
	}

	// Signup user via login endpoint
	body := map[string]string{"phone_number": "+221785626022", "pin": "1234"}
	b, _ := json.Marshal(body)
	lreq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	lreq.Header.Set("Content-Type", "application/json")
	lrr := httptest.NewRecorder()
	srv.HandleLogin(lrr, lreq)
	if lrr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", lrr.Code, lrr.Body.String())
	}

	// Now exists should be true
	rr2 := httptest.NewRecorder()
	srv.HandleUserExists(rr2, req)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}
	if !bytes.Contains(rr2.Body.Bytes(), []byte("true")) {
		t.Fatalf("expected exists=true, got %s", rr2.Body.String())
	}
}

func TestHandleLogin_ExistingUserCorrectPin(t *testing.T) {
	srv := newTestServer(t)

	// Create user first
	body := map[string]string{"phone_number": "+221785626022", "pin": "1234"}
	b, _ := json.Marshal(body)
	lreq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	lreq.Header.Set("Content-Type", "application/json")
	lrr := httptest.NewRecorder()
	srv.HandleLogin(lrr, lreq)
	if lrr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d", lrr.Code)
	}

	// Login with same pin (must create a new request body)
	b2, _ := json.Marshal(body)
	lreq2 := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b2))
	lreq2.Header.Set("Content-Type", "application/json")
	lrr2 := httptest.NewRecorder()
	srv.HandleLogin(lrr2, lreq2)
	if lrr2.Code != http.StatusOK {
		t.Fatalf("expected 200 ok, got %d: %s", lrr2.Code, lrr2.Body.String())
	}
	if !bytes.Contains(lrr2.Body.Bytes(), []byte("session_token")) {
		t.Fatalf("expected session_token in response")
	}
}

func TestHandleLogin_WrongPinTriggersLockout(t *testing.T) {
	srv := newTestServer(t)
	// Signup
	body := map[string]string{"phone_number": "+221785626022", "pin": "1234"}
	b, _ := json.Marshal(body)
	lreq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	lreq.Header.Set("Content-Type", "application/json")
	lrr := httptest.NewRecorder()
	srv.HandleLogin(lrr, lreq)
	if lrr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d", lrr.Code)
	}

	// Wrong PIN multiple times
	bad := map[string]string{"phone_number": "+221785626022", "pin": "0000"}
	bb, _ := json.Marshal(bad)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bb))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.HandleLogin(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 on wrong pin, got %d", rr.Code)
		}
	}
	// Now 429 due to lockout
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(bb))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.HandleLogin(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after lockout, got %d", rr.Code)
	}
}

func TestHandleLogin_NormalizesPhone(t *testing.T) {
	srv := newTestServer(t)
	body := map[string]string{"phone_number": "221785626022", "pin": "1234"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.HandleLogin(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created with plain 221..., got %d: %s", rr.Code, rr.Body.String())
	}
}

// Helper function to create a test user and session for API key tests
func createTestUserAndSession(t *testing.T, srv *server) (string, string) {
	ctx := context.Background()
	userID := ksuid.New().String()

	// Create test user
	err := srv.query.CreateUser(ctx, db.CreateUserParams{
		ID:          userID,
		PhoneNumber: "+221701234567",
		PinHash:     "hashedpin",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create session
	sessionToken, err := utils.NewSessionToken()
	if err != nil {
		t.Fatalf("Failed to create session token: %v", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour).UTC()
	err = srv.query.CreateSession(ctx, db.CreateSessionParams{
		ID:        sessionToken,
		UserID:    userID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	return userID, sessionToken
}

// Helper function to create a test secret
func createTestSecret(t *testing.T, srv *server, userID string) (string, string, string) {
	ctx := context.Background()
	secretID := ksuid.New().String()
	apiKey, err := utils.NewAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}
	keyHash := utils.HashAPIKey(apiKey)

	permissions := []string{"CHECKOUT_API", "BALANCE_API"}
	permissionsJSON, _ := json.Marshal(permissions)

	err = srv.query.CreateSecret(ctx, db.CreateSecretParams{
		ID:          secretID,
		UserID:      userID,
		SecretHash:  keyHash,
		SecretType:  "API_KEY",
		Permissions: string(permissionsJSON),
		DisplayHint: "Test API Key",
	})
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	return secretID, apiKey, keyHash
}

func TestHandleCreateSecret(t *testing.T) {
	srv := newTestServer(t)
	userID, sessionToken := createTestUserAndSession(t, srv)

	tests := []struct {
		name                string
		requestBody         interface{}
		authHeader          string
		expectedStatus      int
		validateResponse    func(t *testing.T, resp *createSecretResponse)
		skipResponseParsing bool
	}{
		{
			name: "successful secret creation",
			requestBody: createSecretRequest{
				DisplayHint: "My API Key",
				Permissions: []permission.Permission{permission.CHECKOUT_API, permission.BALANCE_API},
			},
			authHeader:     "Bearer " + sessionToken,
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, resp *createSecretResponse) {
				if resp.SecretID == "" {
					t.Error("Expected non-empty SecretID")
				}
				if resp.APIKey == "" {
					t.Error("Expected non-empty APIKey")
				}
				if !strings.HasPrefix(resp.APIKey, "wv_") {
					t.Error("Expected API key to have 'wv_' prefix")
				}
			},
		},
		{
			name: "missing authorization header",
			requestBody: createSecretRequest{
				DisplayHint: "My API Key",
				Permissions: []permission.Permission{permission.CHECKOUT_API},
			},
			authHeader:          "",
			expectedStatus:      http.StatusUnauthorized,
			skipResponseParsing: true,
		},
		{
			name: "invalid session token",
			requestBody: createSecretRequest{
				DisplayHint: "My API Key",
				Permissions: []permission.Permission{permission.CHECKOUT_API},
			},
			authHeader:          "Bearer invalid_token",
			expectedStatus:      http.StatusUnauthorized,
			skipResponseParsing: true,
		},
		{
			name:                "invalid JSON body",
			requestBody:         "invalid json",
			authHeader:          "Bearer " + sessionToken,
			expectedStatus:      http.StatusBadRequest,
			skipResponseParsing: true,
		},
		{
			name: "empty permissions",
			requestBody: createSecretRequest{
				DisplayHint: "My API Key",
				Permissions: []permission.Permission{},
			},
			authHeader:          "Bearer " + sessionToken,
			expectedStatus:      http.StatusBadRequest,
			skipResponseParsing: true,
		},
		{
			name: "invalid permission",
			requestBody: createSecretRequest{
				DisplayHint: "My API Key",
				Permissions: []permission.Permission{permission.Permission("INVALID_PERMISSION")},
			},
			authHeader:          "Bearer " + sessionToken,
			expectedStatus:      http.StatusBadRequest,
			skipResponseParsing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/v1/secrets", bytes.NewReader(body))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Add user ID to context (simulating session middleware)
			if strings.Contains(tt.authHeader, sessionToken) && tt.authHeader != "" {
				ctx := context.WithValue(req.Context(), userIDContextKey, userID)
				req = req.WithContext(ctx)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			srv.HandleCreateSecret(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Parse and validate response if not skipping
			if !tt.skipResponseParsing && tt.validateResponse != nil {
				var resp createSecretResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.validateResponse(t, &resp)
			}
		})
	}
}

func TestHandleRevokeSecret(t *testing.T) {
	srv := newTestServer(t)
	userID, sessionToken := createTestUserAndSession(t, srv)
	secretID, _, _ := createTestSecret(t, srv, userID)

	// Create another user and secret for testing ownership
	otherUserID := ksuid.New().String()
	ctx := context.Background()
	err := srv.query.CreateUser(ctx, db.CreateUserParams{
		ID:          otherUserID,
		PhoneNumber: "+221707654321",
		PinHash:     "hashedpin",
	})
	if err != nil {
		t.Fatalf("Failed to create other user: %v", err)
	}
	otherSecretID, _, _ := createTestSecret(t, srv, otherUserID)

	tests := []struct {
		name                string
		secretID            string
		authHeader          string
		userIDInContext     string
		expectedStatus      int
		skipResponseParsing bool
	}{
		{
			name:                "successful secret revocation",
			secretID:            secretID,
			authHeader:          "Bearer " + sessionToken,
			userIDInContext:     userID,
			expectedStatus:      http.StatusNoContent,
			skipResponseParsing: true,
		},
		{
			name:                "missing authorization header",
			secretID:            secretID,
			authHeader:          "",
			userIDInContext:     "",
			expectedStatus:      http.StatusUnauthorized,
			skipResponseParsing: true,
		},
		{
			name:                "secret not found",
			secretID:            "nonexistent",
			authHeader:          "Bearer " + sessionToken,
			userIDInContext:     userID,
			expectedStatus:      http.StatusNotFound,
			skipResponseParsing: true,
		},
		{
			name:                "secret belongs to different user",
			secretID:            otherSecretID,
			authHeader:          "Bearer " + sessionToken,
			userIDInContext:     userID,
			expectedStatus:      http.StatusForbidden,
			skipResponseParsing: true,
		},
		{
			name:                "empty secret ID",
			secretID:            "",
			authHeader:          "Bearer " + sessionToken,
			userIDInContext:     userID,
			expectedStatus:      http.StatusBadRequest,
			skipResponseParsing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			url := "/v1/secrets/" + tt.secretID
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Add user ID to context (simulating session middleware)
			if tt.userIDInContext != "" {
				ctx := context.WithValue(req.Context(), userIDContextKey, tt.userIDInContext)
				req = req.WithContext(ctx)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			srv.HandleRevokeSecret(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// For successful revocation, verify the secret is actually revoked
			if tt.expectedStatus == http.StatusNoContent {
				secret, err := srv.query.GetSecretByID(context.Background(), tt.secretID)
				if err != nil {
					t.Fatalf("Failed to get secret after revocation: %v", err)
				}
				if !secret.RevokedAt.Valid {
					t.Error("Expected secret to be revoked")
				}
			}
		})
	}
}

func TestHandleRevokeSecretAlreadyRevoked(t *testing.T) {
	srv := newTestServer(t)
	userID, sessionToken := createTestUserAndSession(t, srv)
	secretID, _, _ := createTestSecret(t, srv, userID)

	// First revocation
	req1 := httptest.NewRequest(http.MethodDelete, "/v1/secrets/"+secretID, nil)
	req1.Header.Set("Authorization", "Bearer "+sessionToken)
	ctx1 := context.WithValue(req1.Context(), userIDContextKey, userID)
	req1 = req1.WithContext(ctx1)

	w1 := httptest.NewRecorder()
	srv.HandleRevokeSecret(w1, req1)

	if w1.Code != http.StatusNoContent {
		t.Fatalf("First revocation failed with status %d", w1.Code)
	}

	// Second revocation (should fail)
	req2 := httptest.NewRequest(http.MethodDelete, "/v1/secrets/"+secretID, nil)
	req2.Header.Set("Authorization", "Bearer "+sessionToken)
	ctx2 := context.WithValue(req2.Context(), userIDContextKey, userID)
	req2 = req2.WithContext(ctx2)

	w2 := httptest.NewRecorder()
	srv.HandleRevokeSecret(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("Expected status %d for already revoked secret, got %d", http.StatusConflict, w2.Code)
	}
}

func TestCreateAndRevokeSecretIntegration(t *testing.T) {
	srv := newTestServer(t)
	userID, sessionToken := createTestUserAndSession(t, srv)

	// Create secret
	createReq := createSecretRequest{
		DisplayHint: "Integration Test Key",
		Permissions: []permission.Permission{permission.CHECKOUT_API},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/secrets", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	ctx := context.WithValue(req.Context(), userIDContextKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.HandleCreateSecret(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Secret creation failed with status %d", w.Code)
	}

	var createResp createSecretResponse
	if err := json.NewDecoder(w.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	// Verify secret exists and is not revoked
	secret, err := srv.query.GetSecretByID(context.Background(), createResp.SecretID)
	if err != nil {
		t.Fatalf("Failed to get created secret: %v", err)
	}
	if secret.RevokedAt.Valid {
		t.Error("Newly created secret should not be revoked")
	}

	// Revoke secret
	revokeReq := httptest.NewRequest(http.MethodDelete, "/v1/secrets/"+createResp.SecretID, nil)
	revokeReq.Header.Set("Authorization", "Bearer "+sessionToken)
	revokeCtx := context.WithValue(revokeReq.Context(), userIDContextKey, userID)
	revokeReq = revokeReq.WithContext(revokeCtx)

	revokeW := httptest.NewRecorder()
	srv.HandleRevokeSecret(revokeW, revokeReq)

	if revokeW.Code != http.StatusNoContent {
		t.Fatalf("Secret revocation failed with status %d", revokeW.Code)
	}

	// Verify secret is revoked
	revokedSecret, err := srv.query.GetSecretByID(context.Background(), createResp.SecretID)
	if err != nil {
		t.Fatalf("Failed to get revoked secret: %v", err)
	}
	if !revokedSecret.RevokedAt.Valid {
		t.Error("Secret should be revoked")
	}
}

func TestAPIKeyAuthMiddleware(t *testing.T) {
	srv := newTestServer(t)
	userID, _ := createTestUserAndSession(t, srv)
	_, apiKey, _ := createTestSecret(t, srv, userID)

	// Create a simple handler that returns the permissions from context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		permissions, ok := GetPermissionsFromContext(r.Context())
		if !ok {
			http.Error(w, "no permissions in context", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":     "success",
			"permissions": permissions,
		})
	})

	// Wrap with middleware
	handler := srv.APIKeyAuthMiddleware(testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedCode   string
		expectedMsg    string
		checkSuccess   bool
	}{
		{
			name:           "successful authentication",
			authHeader:     "Bearer " + apiKey,
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "missing-auth-header",
			expectedMsg:    "Your request should include an HTTP auth header.",
		},
		{
			name:           "invalid auth header format",
			authHeader:     "InvalidFormat " + apiKey,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "invalid-auth",
			expectedMsg:    "Your HTTP auth header can't be processed.",
		},
		{
			name:           "missing token after Bearer",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "api-key-not-provided",
			expectedMsg:    "Your request should include an API key.",
		},
		{
			name:           "invalid API key",
			authHeader:     "Bearer invalid_key",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "no-matching-api-key",
			expectedMsg:    "The key you provided doesn't exist in our system.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkSuccess {
				// Check that we get a success response with permissions
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode success response: %v", err)
				}
				if resp["message"] != "success" {
					t.Error("Expected success message")
				}
				if perms, ok := resp["permissions"].([]interface{}); !ok || len(perms) == 0 {
					t.Error("Expected permissions in response")
				}
			} else {
				// Check error response
				var apiErr APIError
				if err := json.NewDecoder(w.Body).Decode(&apiErr); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if apiErr.Code != tt.expectedCode {
					t.Errorf("Expected code %s, got %s", tt.expectedCode, apiErr.Code)
				}
				if apiErr.Message != tt.expectedMsg {
					t.Errorf("Expected message %s, got %s", tt.expectedMsg, apiErr.Message)
				}
			}
		})
	}
}

func TestAPIKeyAuthMiddlewareWithRevokedKey(t *testing.T) {
	srv := newTestServer(t)
	userID, sessionToken := createTestUserAndSession(t, srv)
	secretID, apiKey, _ := createTestSecret(t, srv, userID)

	// Revoke the secret first
	req := httptest.NewRequest(http.MethodDelete, "/v1/secrets/"+secretID, nil)
	req.Header.Set("Authorization", "Bearer "+sessionToken)
	ctx := context.WithValue(req.Context(), userIDContextKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	srv.HandleRevokeSecret(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Failed to revoke secret: %d", w.Code)
	}

	// Now test middleware with revoked key
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.APIKeyAuthMiddleware(testHandler)

	authReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	authReq.Header.Set("Authorization", "Bearer "+apiKey)

	authW := httptest.NewRecorder()
	handler.ServeHTTP(authW, authReq)

	if authW.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for revoked key, got %d", http.StatusUnauthorized, authW.Code)
	}

	var apiErr APIError
	if err := json.NewDecoder(authW.Body).Decode(&apiErr); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if apiErr.Code != "api-key-revoked" {
		t.Errorf("Expected code 'api-key-revoked', got %s", apiErr.Code)
	}

	if apiErr.Message != "Your API key has been revoked." {
		t.Errorf("Expected revoked message, got %s", apiErr.Message)
	}
}

func TestHandleCreateCheckoutSession(t *testing.T) {
	srv := newTestServer(t)
	apiKey := createTestAPIKey(t, srv)

	// Test successful checkout session creation
	checkoutBody := CreateCheckoutSessionRequest{
		Amount:          "1000",
		Currency:        "XOF",
		ErrorURL:        "https://example.com/error",
		SuccessURL:      "https://example.com/success",
		ClientReference: &[]string{"test-ref-123"}[0],
	}
	b, _ := json.Marshal(checkoutBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/checkout/sessions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	rr := httptest.NewRecorder()
	middleware := srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleCreateCheckoutSession))
	middleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp CheckoutSession
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify response fields
	if !strings.HasPrefix(resp.ID, "cos-") {
		t.Errorf("Expected ID to start with 'cos-', got %s", resp.ID)
	}
	if len(resp.ID) != 20 {
		t.Errorf("Expected ID length to be 20, got %d", len(resp.ID))
	}
	if resp.Amount != "1000" {
		t.Errorf("Expected amount 1000, got %s", resp.Amount)
	}
	if resp.Currency != "XOF" {
		t.Errorf("Expected currency XOF, got %s", resp.Currency)
	}
	if resp.CheckoutStatus != "open" {
		t.Errorf("Expected checkout status 'open', got %s", resp.CheckoutStatus)
	}
	if resp.PaymentStatus != "processing" {
		t.Errorf("Expected payment status 'processing', got %s", resp.PaymentStatus)
	}
	if resp.BusinessName != "Wave Pool Simulator" {
		t.Errorf("Expected business name 'Wave Pool Simulator', got %s", resp.BusinessName)
	}
	if resp.ClientReference == nil || *resp.ClientReference != "test-ref-123" {
		t.Errorf("Expected client reference 'test-ref-123', got %v", resp.ClientReference)
	}
}

func TestHandleCreateCheckoutSession_ValidationError(t *testing.T) {
	srv := newTestServer(t)
	apiKey := createTestAPIKey(t, srv)

	// Test with missing required fields
	checkoutBody := CreateCheckoutSessionRequest{
		Amount: "1000",
		// Missing currency, error_url, success_url
	}
	b, _ := json.Marshal(checkoutBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/checkout/sessions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	rr := httptest.NewRecorder()
	middleware := srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleCreateCheckoutSession))
	middleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 bad request, got %d: %s", rr.Code, rr.Body.String())
	}

	var apiErr APIError
	if err := json.Unmarshal(rr.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if apiErr.Code != "request-validation-error" {
		t.Errorf("Expected code 'request-validation-error', got %s", apiErr.Code)
	}
}
