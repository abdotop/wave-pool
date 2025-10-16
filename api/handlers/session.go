package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const userContextKey = contextKey("user")

// Middleware to check for a valid access token and add the user to the context
func (api *API) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			slog.ErrorContext(r.Context(), "Authorization header missing")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			slog.ErrorContext(r.Context(), "Invalid Authorization header format", "header", authHeader)
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := tokenParts[1]
		claims := &jwt.MapClaims{}

		jwtSecret := []byte(os.Getenv("API_SECRET"))
		if string(jwtSecret) == "" {
			jwtSecret = []byte("default-secret")
		}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				slog.ErrorContext(r.Context(), "Invalid token signature", "error", err)
				http.Error(w, "Invalid token signature", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			slog.ErrorContext(r.Context(), "Token is not valid")
			http.Error(w, "Token is not valid", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, (*claims)["sub"])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Refresh token endpoint
func (api *API) Refresh(w http.ResponseWriter, r *http.Request) {
	type request struct {
		RefreshToken string `json:"refresh_token"`
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

	claims := &jwt.MapClaims{}
	jwtSecret := []byte(os.Getenv("API_SECRET"))
	if string(jwtSecret) == "" {
		jwtSecret = []byte("default-secret")
	}

	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		slog.ErrorContext(r.Context(), "Invalid refresh token", "error", err)
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	if !token.Valid {
		http.Error(w, "Refresh token is not valid", http.StatusUnauthorized)
		return
	}

	userID, ok := (*claims)["sub"].(string)
	if !ok {
		slog.ErrorContext(r.Context(), "Invalid user ID in token")
		http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
		return
	}

	// Check if the refresh token is in Redis
	_, err = api.redis.Get(r.Context(), "refresh_token:"+req.RefreshToken).Result()
	if err == redis.Nil {
		slog.ErrorContext(r.Context(), "Refresh token not found in Redis")
		http.Error(w, "Refresh token not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		slog.ErrorContext(r.Context(), "Failed to get refresh token from Redis", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate new access and refresh tokens
	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	})
	newAccessTokenString, err := newAccessToken.SignedString(jwtSecret)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to generate access token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	newRefreshTokenString, err := newRefreshToken.SignedString(jwtSecret)
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to generate refresh token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Invalidate the old refresh token and store the new one
	if err := api.redis.Del(r.Context(), "refresh_token:"+req.RefreshToken).Err(); err != nil {
		slog.ErrorContext(r.Context(), "Failed to delete old refresh token from Redis", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := api.redis.Set(r.Context(), "refresh_token:"+newRefreshTokenString, userID, 7*24*time.Hour).Err(); err != nil {
		slog.ErrorContext(r.Context(), "Failed to store new refresh token in Redis", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := response{
		AccessToken:  newAccessTokenString,
		RefreshToken: newRefreshTokenString,
		ExpiresIn:    900,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Logout endpoint
func (api *API) Logout(w http.ResponseWriter, r *http.Request) {
	type request struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(r.Context(), "Failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Delete the refresh token from Redis
	err := api.redis.Del(r.Context(), "refresh_token:"+req.RefreshToken).Err()
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to delete refresh token from Redis", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out successfully"))
}

func GetUserFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userContextKey).(string)
	if !ok {
		return "", errors.New("user not found in context")
	}
	return userID, nil
}
