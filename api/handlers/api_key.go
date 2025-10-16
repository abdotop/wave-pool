package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/argon2"
)

type CreateAPIKeyRequest struct {
	Scopes []string `json:"scopes"`
	Env    string   `json:"env"`
}

type CreateAPIKeyResponse struct {
	ID        string   `json:"id"`
	SecretKey string   `json:"secret_key"`
	Prefix    string   `json:"prefix"`
	Scopes    []string `json:"scopes"`
	Env       string   `json:"env"`
}

func (api *API) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := api.db.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	business, err := api.db.GetBusinessByOwnerID(r.Context(), user.ID)
	if err != nil || business.OwnerID != user.ID {
		http.Error(w, "Business not found", http.StatusNotFound)
		return
	}

	// Generate a new API key
	keyID := ksuid.New().String()
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		http.Error(w, "Failed to generate secret key", http.StatusInternalServerError)
		return
	}
	secretKey := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Hash the secret key
	salt := []byte(keyID)
	hashedSecret := argon2.IDKey([]byte(secretKey), salt, 1, 64*1024, 4, 32)
	hashedSecretStr := base64.RawURLEncoding.EncodeToString(hashedSecret)

	prefix := "wave-pool_" + req.Env[:3] + "_"

	params := sqlc.CreateAPIKeyParams{
		ID:         keyID,
		BusinessID: business.ID,
		Prefix:     prefix,
		KeyHash:    hashedSecretStr,
		Scopes:     req.Scopes,
		Env:        req.Env,
	}

	apiKey, err := api.db.CreateAPIKey(r.Context(), params)
	if err != nil {
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	resp := CreateAPIKeyResponse{
		ID:        apiKey.ID,
		SecretKey: prefix + secretKey,
		Prefix:    apiKey.Prefix,
		Scopes:    apiKey.Scopes,
		Env:       apiKey.Env,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (api *API) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := api.db.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	business, err := api.db.GetBusinessByOwnerID(r.Context(), user.ID)
	if err != nil || business.OwnerID != user.ID {
		http.Error(w, "Business not found", http.StatusNotFound)
		return
	}

	keys, err := api.db.ListAPIKeys(r.Context(), business.ID)
	if err != nil {
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(keys)
}

func (api *API) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	keyID := r.PathValue("key_id")
	if keyID == "" {
		http.Error(w, "API Key ID is required", http.StatusBadRequest)
		return
	}

	user, err := api.db.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	business, err := api.db.GetBusinessByOwnerID(r.Context(), user.ID)
	if err != nil || business.OwnerID != user.ID {
		http.Error(w, "Business not found", http.StatusNotFound)
		return
	}

	key, err := api.db.GetAPIKeyByID(r.Context(), keyID)
	if err != nil {
		http.Error(w, "API Key not found", http.StatusNotFound)
		return
	}

	if key.BusinessID != business.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	err = api.db.RevokeAPIKey(r.Context(), sqlc.RevokeAPIKeyParams{
		ID:         keyID,
		BusinessID: business.ID,
	})
	if err != nil {
		http.Error(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
