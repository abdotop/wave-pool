package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/domain"
	"github.com/segmentio/ksuid"
	"github.com/skip2/go-qrcode"
)

func (api *API) PaymentPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawID := r.PathValue("session_id")
	if rawID == "" || !strings.HasPrefix(rawID, "cos_") {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-found", Message: "Invalid session id"}, http.StatusNotFound)
		return
	}
	sessionID := strings.TrimPrefix(rawID, "cos_")

	session, err := api.db.GetCheckoutSessionByID(ctx, sessionID)
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-found", Message: "Checkout session not found"}, http.StatusNotFound)
		return
	}

	if session.ExpiresAt.Time.Before(time.Now()) {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-expired", Message: "Checkout session has expired"}, http.StatusConflict)
		return
	}

	if session.Status != "open" {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-open", Message: "Checkout session is not open"}, http.StatusConflict)
		return
	}

	business, err := api.db.GetBusinessByID(ctx, session.BusinessID)
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "business-not-found", Message: "Business not found"}, http.StatusNotFound)
		return
	}

	baseURL := "http://" + r.Host
	successURL := baseURL + "/c/" + rawID + "/succeed"
	failURL := baseURL + "/c/" + rawID + "/fail"

	qrContent := fmt.Sprintf(`{"success_url":"%s", "fail_url":"%s"}`, successURL, failURL)
	qrCode, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}
	qrCodeBase64 := base64.StdEncoding.EncodeToString(qrCode)

	html := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Wave Pool Payment</title>
		<style>
			body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background-color: #f4f7f6; }
			.container { text-align: center; padding: 40px; border-radius: 10px; background-color: white; box-shadow: 0 4px 8px rgba(0,0,0,0.1); }
			.logo { width: 150px; margin-bottom: 20px; }
			.qr-code { margin-top: 20px; margin-bottom: 20px; }
			.btn { padding: 10px 20px; border: none; border-radius: 5px; cursor: pointer; color: white; margin: 5px; }
			.btn-success { background-color: #4CAF50; }
			.btn-fail { background-color: #f44336; }
		</style>
	</head>
	<body>
		<div class="container">
			<img src="/logo.png" alt="Wave Pool Logo" class="logo">
			<h2>Payment to ` + business.Name + `</h2>
			<p>Amount: ` + session.Amount + ` ` + session.Currency + `</p>
			<div class="qr-code">
				<img src="data:image/png;base64,` + qrCodeBase64 + `" alt="QR Code">
			</div>
			<form action="` + successURL + `" method="post" style="display: inline;">
				<button type="submit" class="btn btn-success">Simulate Success</button>
			</form>
			<form action="` + failURL + `" method="post" style="display: inline;">
				<button type="submit" class="btn btn-fail">Simulate Fail</button>
			</form>
		</div>
	</body>
	</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (api *API) SucceedPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawID := r.PathValue("session_id")
	if rawID == "" || !strings.HasPrefix(rawID, "cos_") {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-found", Message: "Invalid session id"}, http.StatusNotFound)
		return
	}
	sessionID := strings.TrimPrefix(rawID, "cos_")

	session, err := api.db.GetCheckoutSessionByID(ctx, sessionID)
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-found", Message: "Checkout session not found"}, http.StatusNotFound)
		return
	}

	if session.ExpiresAt.Time.Before(time.Now()) {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-expired", Message: "Checkout session has expired"}, http.StatusConflict)
		return
	}

	if session.Status != "open" {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-open", Message: "Checkout session is not open"}, http.StatusConflict)
		return
	}

	_, err = api.db.CreatePayment(ctx, sqlc.CreatePaymentParams{
		ID:        ksuid.New().String(),
		SessionID: session.ID,
		Amount:    session.Amount,
		Currency:  session.Currency,
		Status:    "succeeded",
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "internal-server-error", Message: "Failed to create payment"}, http.StatusInternalServerError)
		return
	}

	updatedSession, err := api.db.SucceedCheckoutSession(ctx, sessionID)
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "internal-server-error", Message: "Failed to update session"}, http.StatusInternalServerError)
		return
	}

	api.webhookSender.SendWebhook(context.Background(), "checkout.session.completed", updatedSession)

	http.Redirect(w, r, updatedSession.SuccessUrl, http.StatusSeeOther)
}

func (api *API) FailPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawID := r.PathValue("session_id")
	if rawID == "" || !strings.HasPrefix(rawID, "cos_") {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-found", Message: "Invalid session id"}, http.StatusNotFound)
		return
	}
	sessionID := strings.TrimPrefix(rawID, "cos_")

	session, err := api.db.GetCheckoutSessionByID(ctx, sessionID)
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-found", Message: "Checkout session not found"}, http.StatusNotFound)
		return
	}

	if session.ExpiresAt.Time.Before(time.Now()) {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-expired", Message: "Checkout session has expired"}, http.StatusConflict)
		return
	}

	if session.Status != "open" {
		returnError(w, domain.LastPaymentError{Code: "checkout-session-not-open", Message: "Checkout session is not open"}, http.StatusConflict)
		return
	}

	// Select a random realistic error
	errors := []string{
		`{"code": "insufficient-funds", "message": "The user did not have enough account balance."}`,
		`{"code": "blocked-account", "message": "The customer used a blocked account to try and pay for the checkout."}`,
		`{"code": "payment-failure", "message": "A technical error has occurred in Wave's system."}`,
	}
	paymentError := errors[time.Now().UnixNano()%int64(len(errors))]

	_, err = api.db.CreatePayment(ctx, sqlc.CreatePaymentParams{
		ID:            ksuid.New().String(),
		SessionID:     session.ID,
		Amount:        session.Amount,
		Currency:      session.Currency,
		Status:        "failed",
		FailureReason: nullString(paymentError),
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "internal-server-error", Message: "Failed to create payment"}, http.StatusInternalServerError)
		return
	}

	updatedSession, err := api.db.FailCheckoutSession(ctx, sqlc.FailCheckoutSessionParams{
		ID:               sessionID,
		LastPaymentError: []byte(paymentError),
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{Code: "internal-server-error", Message: "Failed to update session"}, http.StatusInternalServerError)
		return
	}

	api.webhookSender.SendWebhook(context.Background(), "checkout.session.payment_failed", updatedSession)

	http.Redirect(w, r, updatedSession.ErrorUrl, http.StatusSeeOther)
}
