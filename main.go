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

	mux.HandleFunc("GET /v1/users/exists", srv.HandleUserExists)
	mux.HandleFunc("POST /v1/auth/login", srv.HandleLogin)

	// Apply session authentication middleware to API key management endpoints
	mux.Handle("POST /v1/secrets", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleCreateSecret)))
	mux.Handle("DELETE /v1/secrets/", srv.SessionAuthMiddleware(http.HandlerFunc(srv.HandleRevokeSecret)))

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
