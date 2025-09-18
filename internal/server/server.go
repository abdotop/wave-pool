package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/abdotop/wave-pool/internal/db"
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
