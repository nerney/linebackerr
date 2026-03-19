package main

import (
	"fmt"
	"linebackerr/db"
	"linebackerr/nflverse"
	"linebackerr/sportarr"
	"log"
	"net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	// Initialize the shared DB first
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}

	// Check if nflverse needs syncing
	if db.NeedsSync("nflverse") {
		fmt.Println("nflverse data is stale or missing. Syncing...")
		if err := nflverse.Init(db.DB); err != nil {
			log.Fatalf("Failed to initialize nflverse: %v", err)
		}
		if err := db.UpdateSync("nflverse"); err != nil {
			log.Printf("Failed to update sync state for nflverse: %v", err)
		}
	} else {
		fmt.Println("nflverse data is up to date.")
	}

	// Check if sportarr needs syncing
	if db.NeedsSync("sportarr") {
		fmt.Println("sportarr data is stale or missing. Syncing...")
		if err := sportarr.LoadSeasons(db.DB); err != nil {
			log.Printf("Failed to load seasons from sportarr: %v", err)
		}
		if err := sportarr.LoadTeams(db.DB); err != nil {
			log.Printf("Failed to load teams from sportarr: %v", err)
		}
		if err := db.UpdateSync("sportarr"); err != nil {
			log.Printf("Failed to update sync state for sportarr: %v", err)
		}
	} else {
		fmt.Println("sportarr data is up to date.")
	}

	http.HandleFunc("/health", healthHandler)
	fmt.Println("Server listening on port 6666...")
	err := http.ListenAndServe(":6666", nil)
	if err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
