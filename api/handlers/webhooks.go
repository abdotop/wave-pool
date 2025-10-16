package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/domain"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/argon2"
)

type webhookPayload struct {
	URL             string                 `json:"url"`
	SigningStrategy domain.SigningStrategy `json:"signing_strategy"`
	Events          []string               `json:"events"`
}

type webhookUpdatePayload struct {
	URL             string                 `json:"url"`
	SigningStrategy domain.SigningStrategy `json:"signing_strategy"`
	Events          []string               `json:"events"`
	Status          domain.WebhookStatus   `json:"status"`
}

type webhookResponse struct {
	ID              string    `json:"id"`
	BusinessID      string    `json:"business_id"`
	URL             string    `json:"url"`
	SigningStrategy string    `json:"signing_strategy"`
	Secret          string    `json:"secret,omitempty"`
	Events          []string  `json:"events"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// IsURL checks if a string is a valid URL.
func IsURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() {
			return false
		}
	}

	return true
}

func (api *API) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
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

	if !IsURL(payload.URL) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		http.Error(w, "Failed to generate secret key", http.StatusInternalServerError)
		return
	}
	secretKey := base64.RawURLEncoding.EncodeToString(secretBytes)

	webhoockID := ksuid.New().String()
	// Hash the secret key
	salt := []byte(webhoockID)
	hashedSecret := argon2.IDKey([]byte(secretKey), salt, 1, 64*1024, 4, 32)
	hashedSecretStr := base64.RawURLEncoding.EncodeToString(hashedSecret)

	webhook, err := api.db.CreateWebhook(r.Context(), sqlc.CreateWebhookParams{
		ID:              webhoockID,
		BusinessID:      business.ID,
		Url:             payload.URL,
		SigningStrategy: payload.SigningStrategy.String(),
		Secret:          hashedSecretStr,
		Events:          payload.Events,
		Status:          domain.WebhookStatusActive.String(),
	})
	if err != nil {
		http.Error(w, "failed to create webhook", http.StatusInternalServerError)
		return
	}

	resp := webhookResponse{
		ID:              webhook.ID,
		BusinessID:      webhook.BusinessID,
		URL:             webhook.Url,
		SigningStrategy: webhook.SigningStrategy,
		Secret:          webhook.Secret,
		Events:          webhook.Events,
		Status:          webhook.Status,
		CreatedAt:       webhook.CreatedAt.Time,
		UpdatedAt:       webhook.UpdatedAt.Time,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (api *API) ListWebhooks(w http.ResponseWriter, r *http.Request) {
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
	webhooks, err := api.db.ListWebhooksByBusinessID(r.Context(), business.ID)
	if err != nil {
		http.Error(w, "failed to list webhooks", http.StatusInternalServerError)
		return
	}

	var response []webhookResponse
	for _, webhook := range webhooks {
		response = append(response, webhookResponse{
			ID:              webhook.ID,
			BusinessID:      webhook.BusinessID,
			URL:             webhook.Url,
			SigningStrategy: webhook.SigningStrategy,
			Events:          webhook.Events,
			Status:          webhook.Status,
			CreatedAt:       webhook.CreatedAt.Time,
			UpdatedAt:       webhook.UpdatedAt.Time,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (api *API) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	webhookIDStr := r.PathValue("webhook_id")
	webhookID, err := ksuid.Parse(webhookIDStr)
	if err != nil {
		http.Error(w, "invalid webhook_id", http.StatusBadRequest)
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

	var payload webhookUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "failed to parse request body", http.StatusBadRequest)
		return
	}

	if !IsURL(payload.URL) {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	webhook, err := api.db.UpdateWebhook(r.Context(), sqlc.UpdateWebhookParams{
		ID:              webhookID.String(),
		Url:             payload.URL,
		SigningStrategy: payload.SigningStrategy.String(),
		Events:          payload.Events,
		Status:          payload.Status.String(),
	})
	if err != nil {
		http.Error(w, "failed to update webhook", http.StatusInternalServerError)
		return
	}

	resp := webhookResponse{
		ID:              webhook.ID,
		BusinessID:      webhook.BusinessID,
		URL:             webhook.Url,
		SigningStrategy: webhook.SigningStrategy,
		Events:          webhook.Events,
		Status:          webhook.Status,
		CreatedAt:       webhook.CreatedAt.Time,
		UpdatedAt:       webhook.UpdatedAt.Time,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (api *API) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	webhookIDStr := r.PathValue("webhook_id")
	webhookID, err := ksuid.Parse(webhookIDStr)
	if err != nil {
		http.Error(w, "invalid webhook_id", http.StatusBadRequest)
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

	err = api.db.DeleteWebhook(r.Context(), webhookID.String())
	if err != nil {
		http.Error(w, "failed to delete webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
