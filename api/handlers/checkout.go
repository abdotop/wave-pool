package handlers

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/domain"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/segmentio/ksuid"
)

// validate is a singleton instance of the validator.
var validate *validator.Validate

func init() {
	validate = validator.New()
	// Register custom validation functions
	validate.RegisterValidation("iso4217", validateISO4217)
	validate.RegisterValidation("e164", validateE164)
}

// validateISO4217 implements validator.Func for ISO 4217 currency codes.
func validateISO4217(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^[A-Z]{3}$`)
	return re.MatchString(fl.Field().String())
}

// validateE164 implements validator.Func for E.164 phone numbers.
func validateE164(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	re := regexp.MustCompile(`^(\+221)?(77|78|75|71|70|76)[0-9]{7}$`)
	return re.MatchString(fl.Field().String())
}

// CreateCheckoutSession handles the creation of a new checkout session.
func (api *API) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.CreateCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "request-validation-error",
			Message: "Invalid JSON body",
		}, http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "request-validation-error",
			Message: err.Error(),
		}, http.StatusBadRequest)
		return
	}

	businessID, ok := ctx.Value(BusinessIDKey).(string)
	if !ok {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "internal server error",
		}, http.StatusInternalServerError)
		return
	}
	businessName, _ := ctx.Value(BusinessNameKey).(string)

	// ---------- 4. Construire la session ----------
	sessionID := ksuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.Add(30 * time.Minute)

	baseLaunch := cmp.Or(os.Getenv("WAVE_LAUNCH_URL"), "http://localhost:"+cmp.Or(os.Getenv("PORT"), "8080"))
	waveLaunchURL := fmt.Sprintf("%s/c/cos_%s", baseLaunch, sessionID)

	arg := sqlc.CreateCheckoutSessionParams{
		ID:                   sessionID,
		BusinessID:           businessID,
		AggregatedMerchantID: nullString(req.AggregatedMerchantID),
		Amount:               req.Amount,
		Currency:             req.Currency,
		ClientReference:      nullString(req.ClientReference),
		Status:               "open",
		ErrorUrl:             req.ErrorURL,
		SuccessUrl:           req.SuccessURL,
		RestrictPayerMobile:  nullString(req.RestrictPayerMobile),
		WaveLaunchUrl:        pgtype.Text{String: waveLaunchURL, Valid: true},
		PaymentStatus:        pgtype.Text{String: "processing", Valid: true},
		ExpiresAt:            pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}
	session, err := api.db.CreateCheckoutSession(ctx, arg)
	if err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "Failed to create checkout session",
			Details: err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	resp := domain.CheckoutSessionResponse{
		ID:                   "cos_" + session.ID,
		Amount:               session.Amount,
		CheckoutStatus:       session.Status,
		ClientReference:      nullableToPtr(session.ClientReference),
		Currency:             session.Currency,
		ErrorURL:             session.ErrorUrl,
		BusinessName:         businessName,
		PaymentStatus:        session.PaymentStatus.String,
		SuccessURL:           session.SuccessUrl,
		WaveLaunchURL:        session.WaveLaunchUrl.String,
		WhenCreated:          session.WhenCreated.Time,
		WhenExpires:          session.ExpiresAt.Time,
		AggregatedMerchantID: nullableToPtr(session.AggregatedMerchantID),
		RestrictPayerMobile:  nullableToPtr(session.RestrictPayerMobile),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Returns the complete Checkout-Session object or the documented error.
func (api *API) GetCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawID := r.PathValue("session_id")
	if rawID == "" || !strings.HasPrefix(rawID, "cos_") {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "Invalid session id",
		}, http.StatusNotFound)
		return
	}

	sessionID := strings.TrimPrefix(rawID, "cos_")

	businessID, ok := ctx.Value(BusinessIDKey).(string)
	if !ok {
		returnError(w, domain.LastPaymentError{
			Code:    "unauthorized-wallet",
			Message: "Session does not belong to your wallet",
		}, http.StatusForbidden)
		return
	}

	session, err := api.db.GetCheckoutSession(ctx, sqlc.GetCheckoutSessionParams{
		ID:         sessionID,
		BusinessID: businessID,
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "Checkout session not found",
		}, http.StatusNotFound)
		return
	}

	businessName, _ := ctx.Value(BusinessNameKey).(string)

	resp := domain.CheckoutSessionResponse{
		ID:                   "cos_" + session.ID,
		Amount:               session.Amount,
		CheckoutStatus:       session.Status,
		ClientReference:      nullableToPtr(session.ClientReference),
		Currency:             session.Currency,
		ErrorURL:             session.ErrorUrl,
		BusinessName:         businessName,
		PaymentStatus:        session.PaymentStatus.String,
		SuccessURL:           session.SuccessUrl,
		WaveLaunchURL:        session.WaveLaunchUrl.String,
		WhenCreated:          session.WhenCreated.Time,
		WhenExpires:          session.ExpiresAt.Time,
		AggregatedMerchantID: nullableToPtr(session.AggregatedMerchantID),
		RestrictPayerMobile:  nullableToPtr(session.RestrictPayerMobile),
	}

	if session.TransactionID.Valid {
		resp.TransactionID = &session.TransactionID.String
	}
	if session.WhenCompleted.Valid {
		resp.WhenCompleted = &session.WhenCompleted.Time
	}
	_ = json.Unmarshal(session.LastPaymentError, &resp.LastPaymentError)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Returns the unique Checkout-Session that owns this Wave transaction id.
func (api *API) GetCheckoutSessionByTxID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	txID := r.URL.Query().Get("transaction_id")
	if txID == "" || !strings.HasPrefix(txID, "T_") {
		returnError(w, domain.LastPaymentError{
			Code:    "request-validation-error",
			Message: "transaction_id is required and must start with T_",
		}, http.StatusBadRequest)
		return
	}

	businessID, ok := ctx.Value(BusinessIDKey).(string)
	if !ok {
		returnError(w, domain.LastPaymentError{
			Code:    "unauthorized-wallet",
			Message: "Session does not belong to your wallet",
		}, http.StatusForbidden)
		return
	}
	session, err := api.db.GetCheckoutSessionByTxID(ctx, sqlc.GetCheckoutSessionByTxIDParams{
		TransactionID: pgtype.Text{String: txID, Valid: true},
		BusinessID:    businessID,
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "No checkout session found for this transaction id",
		}, http.StatusNotFound)
		return
	}

	businessName, _ := ctx.Value(BusinessNameKey).(string)

	resp := domain.CheckoutSessionResponse{
		ID:                   "cos_" + session.ID,
		Amount:               session.Amount,
		CheckoutStatus:       session.Status,
		ClientReference:      nullableToPtr(session.ClientReference),
		Currency:             session.Currency,
		ErrorURL:             session.ErrorUrl,
		BusinessName:         businessName,
		PaymentStatus:        session.PaymentStatus.String,
		SuccessURL:           session.SuccessUrl,
		WaveLaunchURL:        session.WaveLaunchUrl.String,
		WhenCreated:          session.WhenCreated.Time,
		WhenExpires:          session.ExpiresAt.Time,
		AggregatedMerchantID: nullableToPtr(session.AggregatedMerchantID),
		RestrictPayerMobile:  nullableToPtr(session.RestrictPayerMobile),
	}
	if session.TransactionID.Valid {
		resp.TransactionID = &session.TransactionID.String
	}
	if session.WhenCompleted.Valid {
		resp.WhenCompleted = &session.WhenCompleted.Time
	}
	_ = json.Unmarshal(session.LastPaymentError, &resp.LastPaymentError)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Only the client_reference filter is documented for now.
func (api *API) SearchCheckoutSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ref := r.URL.Query().Get("client_reference")
	if ref == "" {
		returnError(w, domain.LastPaymentError{
			Code:    "request-validation-error",
			Message: "client_reference is required",
		}, http.StatusBadRequest)
		return
	}

	businessID, ok := ctx.Value(BusinessIDKey).(string)
	if !ok {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "missing business_id in context",
		}, http.StatusInternalServerError)
		return
	}
	businessName, _ := ctx.Value(BusinessNameKey).(string)

	rows, err := api.db.SearchCheckoutSessions(ctx, sqlc.SearchCheckoutSessionsParams{
		BusinessID:      businessID,
		ClientReference: pgtype.Text{String: ref, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			returnError(w, domain.LastPaymentError{
				Code:    "checkout-session-not-found",
				Message: "No checkout session found",
			}, http.StatusNotFound)
			return
		}
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "Failed to search sessions",
			Details: err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	result := make([]domain.CheckoutSessionResponse, 0, len(rows))
	for _, s := range rows {
		res := domain.CheckoutSessionResponse{
			ID:                   "cos_" + s.ID,
			Amount:               s.Amount,
			CheckoutStatus:       s.Status,
			ClientReference:      nullableToPtr(s.ClientReference),
			Currency:             s.Currency,
			ErrorURL:             s.ErrorUrl,
			BusinessName:         businessName,
			PaymentStatus:        s.PaymentStatus.String,
			SuccessURL:           s.SuccessUrl,
			WaveLaunchURL:        s.WaveLaunchUrl.String,
			WhenCreated:          s.WhenCreated.Time,
			WhenExpires:          s.ExpiresAt.Time,
			AggregatedMerchantID: nullableToPtr(s.AggregatedMerchantID),
			RestrictPayerMobile:  nullableToPtr(s.RestrictPayerMobile),
		}
		if s.TransactionID.Valid {
			res.TransactionID = &s.TransactionID.String
		}
		if s.WhenCompleted.Valid {
			res.WhenCompleted = &s.WhenCompleted.Time
		}
		_ = json.Unmarshal(s.LastPaymentError, &res.LastPaymentError)

		result = append(result, res)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"result": result})
}

// RefundCheckoutSession
// POST /v1/checkout/sessions/:id/refund
func (api *API) RefundCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawID := r.PathValue("session_id")
	if rawID == "" || !strings.HasPrefix(rawID, "cos_") {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "Invalid session id",
		}, http.StatusNotFound)
		return
	}
	sessionID := strings.TrimPrefix(rawID, "cos_")

	businessID, ok := ctx.Value(BusinessIDKey).(string)
	if !ok {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "missing business_id in context",
		}, http.StatusInternalServerError)
		return
	}

	session, err := api.db.GetCheckoutSession(ctx, sqlc.GetCheckoutSessionParams{
		ID:         sessionID,
		BusinessID: businessID,
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "Checkout session not found",
		}, http.StatusNotFound)
		return
	}
	if session.BusinessID != businessID {
		returnError(w, domain.LastPaymentError{
			Code:    "unauthorized-wallet",
			Message: "Session does not belong to your wallet",
		}, http.StatusForbidden)
		return
	}

	if session.PaymentStatus.String == "cancelled" {
		w.WriteHeader(http.StatusOK) // doc : 200 vide
		return
	}

	if session.PaymentStatus.String != "succeeded" {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-refund-failed",
			Message: "Payment is not in a refundable state",
		}, http.StatusBadRequest)
		return
	}

	err = api.db.UpdateCheckoutPaymentStatus(ctx, sqlc.UpdateCheckoutPaymentStatusParams{
		ID:            session.ID,
		BusinessID:    businessID,
		PaymentStatus: pgtype.Text{String: "cancelled", Valid: true},
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "Failed to refund checkout session",
			Details: err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Returns 200 + empty body on success, documented errors otherwise.
func (api *API) ExpireCheckoutSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rawID := r.PathValue("session_id")
	if rawID == "" || !strings.HasPrefix(rawID, "cos_") {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "Invalid session id",
		}, http.StatusNotFound)
		return
	}
	sessionID := strings.TrimPrefix(rawID, "cos_")

	businessID, ok := ctx.Value(BusinessIDKey).(string)
	if !ok {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "missing business_id in context",
		}, http.StatusInternalServerError)
		return
	}

	session, err := api.db.GetCheckoutSession(ctx, sqlc.GetCheckoutSessionParams{
		ID:         sessionID,
		BusinessID: businessID,
	})
	if err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-not-found",
			Message: "Checkout session not found",
		}, http.StatusNotFound)
		return
	}
	if session.BusinessID != businessID {
		returnError(w, domain.LastPaymentError{
			Code:    "unauthorized-wallet",
			Message: "Session does not belong to your wallet",
		}, http.StatusForbidden)
		return
	}

	switch session.Status {
	case "expired":
		w.WriteHeader(http.StatusOK)
		return
	case "complete":
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-conflict",
			Message: "Session already completed",
		}, http.StatusConflict)
		return
	case "open":
		// continue
	default:
		returnError(w, domain.LastPaymentError{
			Code:    "checkout-session-conflict",
			Message: "Session not in expire-able state",
		}, http.StatusConflict)
		return
	}

	now := time.Now().UTC()
	if err := api.db.ExpireCheckoutSession(ctx, sqlc.ExpireCheckoutSessionParams{
		ID:            sessionID,
		Status:        "expired",
		WhenCompleted: pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		returnError(w, domain.LastPaymentError{
			Code:    "internal-server-error",
			Message: "Failed to expire session",
			Details: err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func nullString(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func nullableToPtr(n pgtype.Text) *string {
	if n.Valid {
		return &n.String
	}
	return nil
}
