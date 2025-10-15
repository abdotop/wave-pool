package main

import (
	"cmp"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/abdotop/wave-pool/db/sqlc"
	"github.com/abdotop/wave-pool/handlers"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Rate Limiter
var (
	clients = make(map[string]*client)
	mu      sync.Mutex
)

type client struct {
	lastSeen time.Time
	requests int
}

func rateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ip := r.RemoteAddr
		if c, found := clients[ip]; found {
			if time.Since(c.lastSeen) > 1*time.Minute {
				c.requests = 1
				c.lastSeen = time.Now()
			} else {
				c.requests++
			}
			if c.requests > 10 {
				mu.Unlock()
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
		} else {
			clients[ip] = &client{lastSeen: time.Now(), requests: 1}
		}
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// Database connection
	dbpool, err := pgxpool.New(ctx, os.Getenv("DB_SOURCE"))
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	defer dbpool.Close()

	db := sqlc.New(dbpool)

	api := handlers.NewAPI(db)

	// Simple HTTP server with a health check endpoint
	router := http.NewServeMux()
	router.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	router.Handle("POST /api/v1/auth", rateLimiter(http.HandlerFunc(api.Auth)))

	server := &http.Server{
		Addr:         ":" + cmp.Or(os.Getenv("PORT"), "8080"),
		Handler:      router,
		ReadTimeout:  5 * time.Minute,  // 5 minutes
		WriteTimeout: 10 * time.Minute, // 10 minutes
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
