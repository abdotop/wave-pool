package main

import (
	"cmp"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// Simple HTTP server with a health check endpoint
	router := http.NewServeMux()
	router.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         ":" + cmp.Or(os.Getenv("PORT"), "8080"),
		Handler:      router,
		ReadTimeout:  5 * 600000000,  // 5 minutes
		WriteTimeout: 10 * 600000000, // 10 minutes
	}
	go func() {
		slog.Info("Starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down server")

	if err := server.Shutdown(context.Background()); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	} else {
		slog.Info("Server gracefully stopped")
	}
}
