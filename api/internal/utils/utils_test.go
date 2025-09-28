package utils

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestNewSessionToken_LengthAndHex(t *testing.T) {
	tok, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken err: %v", err)
	}
	if len(tok) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(tok))
	}
	for _, r := range tok {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			t.Fatalf("non-hex character in token: %q", r)
		}
	}
}

func TestValidatePIN(t *testing.T) {
	if err := ValidatePIN("1234"); err != nil {
		t.Fatalf("valid pin rejected: %v", err)
	}
	if err := ValidatePIN("12a4"); err == nil {
		t.Fatalf("expected error for non-digit pin")
	}
	if err := ValidatePIN("123"); err == nil {
		t.Fatalf("expected error for short pin")
	}
	if err := ValidatePIN("12345"); err == nil {
		t.Fatalf("expected error for long pin")
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	payload := map[string]string{"ok": "1"}
	WriteJSON(rr, 201, payload)
	if rr.Code != 201 {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", ct)
	}
	var got map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got["ok"] != "1" {
		t.Fatalf("unexpected body: %v", got)
	}
}
