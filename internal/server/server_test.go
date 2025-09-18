package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/abdotop/wave-pool/internal/store"
)

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
