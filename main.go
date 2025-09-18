package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/abdotop/wave-pool/internal/store"
)

func main() {
	// Initialize SQLite store; database file will be created on first run.
	st, err := store.New("wave-pool.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	if err := st.AutoMigrate("db/migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	// Keep st for future handlers; ensure it's not optimized away
	_ = st
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Wave Pool API Simulator - %s", r.URL.Path)
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to write response", slog.String("error", err.Error()))
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status": "ok"}`))
		if err != nil {
			slog.ErrorContext(r.Context(), "Failed to write response", slog.String("error", err.Error()))
		}
	})

	port := "8080"
	fmt.Printf("Wave Pool API simulator starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
