package main

import (
	"fmt"
	"linebackerr/db"
	"linebackerr/nflverse"
	"log"
	"net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	if err := nflverse.Init(); err != nil {
		log.Fatalf("Failed to initialize nflverse: %v", err)
	}

	if err := db.Init(nflverse.DB); err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}

	http.HandleFunc("/health", healthHandler)
	fmt.Println("Server listening on port 6666...")
	err := http.ListenAndServe(":6666", nil)
	if err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
