package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/domain"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
)

const (
	BusinessIDKey   contextKey = "business_id"
	BusinessNameKey contextKey = "business_name"
	UserIDKey       contextKey = "user_id"
)

func returnError(w http.ResponseWriter, payload domain.LastPaymentError, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// APIKeyAuthMiddleware validates the API key provided in the Authorization header.
func (api *API) APIKeyAuthMiddleware(requiredScope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				returnError(w, domain.LastPaymentError{
					Code:    "missing-auth-header",
					Message: "Missing authorization header",
				}, http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				returnError(w, domain.LastPaymentError{
					Code:    "invalid-auth",
					Message: "Missing Bearer authorization header",
				}, http.StatusUnauthorized)
				return
			}

			apiKey := parts[1]
			var secretKey string
			if !strings.HasPrefix(apiKey, "wave-pool_prod_") {
				returnError(w, domain.LastPaymentError{
					Code:    "invalid-api-key-prefix",
					Message: "API key has an invalid prefix",
				}, http.StatusUnauthorized)
				return
			}
			secretKey = apiKey[len("wave-pool_prod_"):]
			salt := []byte(os.Getenv("API_SECRET"))
			hashedSecret := argon2.IDKey([]byte(secretKey), salt, 1, 64*1024, 4, 32)
			hashedSecretStr := base64.RawURLEncoding.EncodeToString(hashedSecret)

			log.Println("Hashed Secret:", hashedSecretStr)
			apiKeyRow, err := api.db.GetAPIKeyByPrefixAndSecret(r.Context(), sqlc.GetAPIKeyByPrefixAndSecretParams{
				Prefix:  "wave-pool_prod_",
				KeyHash: hashedSecretStr,
			})
			if err != nil {
				if err == pgx.ErrNoRows {
					returnError(w, domain.LastPaymentError{
						Code:    "no-matching-api-key",
						Message: fmt.Sprintf("No API key found ending in '%s'", apiKey[len(apiKey)-4:]),
					}, http.StatusUnauthorized)
					return
				}
				returnError(w, domain.LastPaymentError{
					Code: "internal-server-error",
				}, http.StatusInternalServerError)
				return
			}

			if apiKeyRow.Status.Valid && apiKeyRow.Status.String == "revoked" {
				returnError(w, domain.LastPaymentError{
					Code:    "api-key-revoked",
					Message: "API key has been revoked",
				}, http.StatusUnauthorized)
				return
			}

			if !slices.Contains(apiKeyRow.Scopes, requiredScope) {
				returnError(w, domain.LastPaymentError{
					Code:    "invalid-wallet",
					Message: fmt.Sprintf("API key does not have the required scope: %s", requiredScope),
				}, http.StatusForbidden)
				return
			}

			// Add business info to the context
			ctx := context.WithValue(r.Context(), BusinessIDKey, apiKeyRow.BusinessID)
			ctx = context.WithValue(ctx, BusinessNameKey, apiKeyRow.BusinessName)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
