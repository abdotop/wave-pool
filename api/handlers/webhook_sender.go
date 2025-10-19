package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/domain"
)

type WebhookSender struct {
	db *sqlc.Queries
}

func NewWebhookSender(db *sqlc.Queries) *WebhookSender {
	return &WebhookSender{db: db}
}

func (s *WebhookSender) SendWebhook(ctx context.Context, eventType string, session sqlc.CheckoutSession) {
	webhooks, err := s.db.ListWebhooksByBusinessID(ctx, session.BusinessID)
	if err != nil {
		log.Printf("Failed to list webhooks for business %s: %v", session.BusinessID, err)
		return
	}

	slog.Info("Sending webhooks", "hooks", webhooks, "business_id", session.BusinessID, "event_type", eventType)

	for _, webhook := range webhooks {
		go s.send(ctx, webhook, eventType, session)
	}
}

func (s *WebhookSender) send(ctx context.Context, webhook sqlc.Webhook, eventType string, session sqlc.CheckoutSession) {
	event := domain.Event{
		ID:   "EV_" + session.ID,
		Type: eventType,
		Data: session,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal webhook payload: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhook.Url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to create webhook request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	if webhook.SigningStrategy == domain.SigningStrategySigningSecret.String() {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		mac := hmac.New(sha256.New, []byte(webhook.Secret))
		mac.Write([]byte(timestamp + "." + string(payload)))
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("Wave-Signature", fmt.Sprintf("t=%s,v1=%s", timestamp, signature))
	} else {
		req.Header.Set("Authorization", "Bearer "+webhook.Secret)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Webhook sent to %s, status: %s", webhook.Url, resp.Status)
}
