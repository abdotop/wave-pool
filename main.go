package main

import (
	"cmp"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abdotop/wave-pool/internal/server"
	"github.com/abdotop/wave-pool/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Initialize SQLite store; database file will be created on first run.
	st, err := store.New("wave-pool.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	if err := st.AutoMigrate("db/migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status": "ok"}`))
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to write response", slog.String("error", err.Error()))
		}
	})

	srv := server.NewServer(st)

	// Auth endpoints
	mux.HandleFunc("GET /api/v1/users/exists", srv.HandleUserExists)
	mux.HandleFunc("POST /api/v1/auth/login", srv.HandleLogin)

	// Portal endpoints (session-authenticated)
	mux.Handle("POST /api/v1/portal/secrets", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleCreateSecret)))
	mux.Handle("DELETE /api/v1/portal/secrets/", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleRevokeSecret)))
	mux.Handle("GET /api/v1/portal/secrets", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleListSecrets)))
	mux.Handle("POST /api/v1/portal/webhooks", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleCreateWebhook)))
	mux.Handle("GET /api/v1/portal/checkout-sessions", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleListCheckoutSessions)))

	// Wave API endpoints (API key authenticated)
	mux.Handle("POST /v1/checkout/sessions", srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleCreateCheckoutSession)))
	mux.Handle("GET /v1/checkout/sessions/{id}", srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleGetCheckoutSession)))
	mux.Handle("GET /v1/checkout/sessions", srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleGetCheckoutSessionByTransactionID)))
	mux.Handle("GET /v1/checkout/sessions/search", srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleSearchCheckoutSessions)))
	mux.Handle("POST /v1/checkout/sessions/{id}/expire", srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleExpireCheckoutSession)))
	mux.Handle("POST /v1/checkout/sessions/{id}/refund", srv.APIKeyAuthMiddleware(http.HandlerFunc(srv.HandleRefundCheckoutSession)))

	// Payment simulation endpoint (no authentication)
	mux.HandleFunc("GET /pay/{session_id}", srv.HandlePaymentPageGET)
	mux.HandleFunc("POST /pay/{session_id}", srv.HandlePaymentPagePOST)

	server := &http.Server{
		Addr:         ":" + cmp.Or(os.Getenv("PORT"), "8080"),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("starting server", "port", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	stop()
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(sctx); err != nil {
		slog.Error("server shutdown", "err", err)
	}
	slog.Info("server stopped")
}
