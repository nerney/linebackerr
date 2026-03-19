package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dataDir = "/config"
)

var DB *sql.DB

func Init() error {
	fmt.Println("Initializing linebackerr DB package...")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "linebackerr.db")
	fmt.Printf("Connecting to SQLite database at %s...\n", dbPath)

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open linebackerr db: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping linebackerr db: %w", err)
	}

	fmt.Println("Successfully connected to linebackerr DB.")

	if err := initSchema(); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	fmt.Println("linebackerr DB initialization complete.")
	return nil
}

func initSchema() error {
	fmt.Println("Creating linebackerr DB tables...")
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sportarr_team (
			team_id TEXT PRIMARY KEY,
			strLogo TEXT,
			strBadge TEXT,
			strBanner TEXT,
			strFanart1 TEXT,
			strDescriptionEN TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS sportarr_seasons (
			year TEXT PRIMARY KEY,
			poster_url TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS sync_state (
			module TEXT PRIMARY KEY,
			last_sync DATETIME
		);`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return err
		}
	}
	fmt.Println("linebackerr tables created successfully.")
	return nil
}

// NeedsSync returns true if the module has never been synced, or if the last sync was over 48 hours ago.
func NeedsSync(module string) bool {
	var lastSync time.Time
	err := DB.QueryRow("SELECT last_sync FROM sync_state WHERE module = ?", module).Scan(&lastSync)
	if err != nil {
		if err == sql.ErrNoRows {
			return true // Never synced
		}
		fmt.Printf("Error checking sync state for %s: %v\n", module, err)
		return true // Default to true on error just in case
	}

	return time.Since(lastSync) > 48*time.Hour
}

// UpdateSync updates the sync state for the given module to the current time.
func UpdateSync(module string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO sync_state (module, last_sync) VALUES (?, ?)", module, time.Now())
	return err
}
