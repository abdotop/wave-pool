package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Wave Pool API Simulator - %s", r.URL.Path)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok"}`)
	})

	port := "8080"
	fmt.Printf("Wave Pool API simulator starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
