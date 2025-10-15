package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/argon2"
)

// registerUser handles user registration
func (api *API) RegisterUser(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Phone string `json:"phone"`
		Pin   string `json:"pin"`
	}

	type response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(r.Context(), "Failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate phone number (E.164 format)
	if matched, _ := regexp.MatchString(`^\+[1-9]\d{1,14}$`, req.Phone); !matched {
		slog.ErrorContext(r.Context(), "Invalid phone number format", "phone", req.Phone)
		http.Error(w, "Invalid phone number format", http.StatusBadRequest)
		return
	}

	// Validate PIN (4 digits)
	if matched, _ := regexp.MatchString(`^\d{4}$`, req.Pin); !matched {
		slog.ErrorContext(r.Context(), "Invalid PIN format", "pin", req.Pin)
		http.Error(w, "PIN must be 4 digits", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	_, err := api.db.GetUserByPhone(r.Context(), req.Phone)
	if err == nil {
		slog.ErrorContext(r.Context(), "User with this phone number already exists", "phone", req.Phone)
		http.Error(w, "User with this phone number already exists", http.StatusConflict)
		return
	}
	if err != pgx.ErrNoRows {
		slog.ErrorContext(r.Context(), "Failed to check user existence", "error", err)
		http.Error(w, "Failed to check user existence", http.StatusInternalServerError)
		return
	}
	id := ksuid.New().String()
	// Hash the PIN using Argon2id
	salt := []byte(id) // In a real app, use a unique salt for each user
	pinHash := argon2.IDKey([]byte(req.Pin), salt, 1, 64*1024, 4, 32)

	// Create user
	user, err := api.db.CreateUser(r.Context(), sqlc.CreateUserParams{
		ID:      id,
		Phone:   req.Phone,
		PinHash: fmt.Sprintf("%x", pinHash),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to create user", "error", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT tokens
	jwtSecret := os.Getenv("API_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-secret"
	}

	// Create access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	accessTokenString, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to generate access token", "error", err)
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// Create refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtSecret))
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to generate refresh token", "error", err)
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	resp := response{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    900,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
