package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/bcrypt"
)

// registerUser handles user registration
func (api *API) Auth(w http.ResponseWriter, r *http.Request) {
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
	if matched, _ := regexp.MatchString(`^(\+221)?(77|78|75|71|70|76)[0-9]{7}$`, req.Phone); !matched {
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
	user, err := api.db.GetUserByPhone(r.Context(), req.Phone)
	if err != nil {
		if err == pgx.ErrNoRows {
			pinHash, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
			if err != nil {
				slog.ErrorContext(r.Context(), "Failed to hash PIN", "error", err)
				http.Error(w, "Failed to process PIN", http.StatusInternalServerError)
				return
			}

			// Create user
			user, err = api.db.CreateUser(r.Context(), sqlc.CreateUserParams{
				ID:      ksuid.New().String(),
				Phone:   req.Phone,
				PinHash: string(pinHash),
			})
			if err != nil {
				slog.ErrorContext(r.Context(), "Failed to create user", "error", err)
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}

			// Create default business for the user
			_, err = api.db.CreateBusiness(r.Context(), sqlc.CreateBusinessParams{
				ID:       ksuid.New().String(),
				Name:     gofakeit.Company(),
				OwnerID:  user.ID,
				Country:  "SN", // Default country Senegal
				Currency: "XOF",
			})
			if err != nil {
				slog.ErrorContext(r.Context(), "Failed to create default business", "error", err)
				http.Error(w, "Failed to create default business", http.StatusInternalServerError)
				return
			}
		} else {

			slog.ErrorContext(r.Context(), "Failed to check user existence", "error", err)
			http.Error(w, "Failed to check user existence", http.StatusInternalServerError)
			return
		}
	}

	// Verify PIN
	if err := bcrypt.CompareHashAndPassword([]byte(user.PinHash), []byte(req.Pin)); err != nil {
		slog.ErrorContext(r.Context(), "Invalid PIN", "phone", req.Phone, "error", err)
		http.Error(w, "Invalid phone or PIN", http.StatusUnauthorized)
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

	// Store the refresh token in Redis
	err = api.redis.Set(r.Context(), "refresh_token:"+refreshTokenString, user.ID, 7*24*time.Hour).Err()
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to store refresh token in Redis", "error", err)
		http.Error(w, "Failed to store refresh token", http.StatusInternalServerError)
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
