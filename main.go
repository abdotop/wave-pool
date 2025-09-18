package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Wave Pool API Simulator - %s", r.URL.Path)
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
