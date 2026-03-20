package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultDataDir = "/config"
	dbFileName     = "linebackerr.db"
)

var DB *sql.DB

func Init() *sql.DB {
	fmt.Println("Initializing linebackerr DB package...")

	dataDir := DataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		panic(fmt.Errorf("failed to create data directory: %w", err))
	}

	dbPath := filepath.Join(dataDir, dbFileName)
	fmt.Printf("Initializing SQLite database at %s...\n", dbPath)

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(fmt.Errorf("failed to open linebackerr db: %w", err))
	}

	if err := DB.Ping(); err != nil {
		panic(fmt.Errorf("failed to ping linebackerr db: %w", err))
	}

	if err := initSchema(DB); err != nil {
		panic(fmt.Errorf("failed to init schema: %w", err))
	}

	fmt.Println("linebackerr DB initialization complete.")
	return DB
}

func DataDir() string {
	if dir := os.Getenv("LINEBACKERR_DATA_DIR"); dir != "" {
		return dir
	}
	return defaultDataDir
}

func DBPath() string {
	return filepath.Join(DataDir(), dbFileName)
}

func initSchema(db *sql.DB) error {
	fmt.Println("Creating linebackerr DB tables...")
	queries := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS nflverse_teams (
			id TEXT PRIMARY KEY,
			abbr TEXT,
			full_name TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS nflverse_games (
			game_id TEXT PRIMARY KEY,
			season INTEGER,
			week INTEGER,
			game_type TEXT,
			gameday TEXT,
			away_team_id TEXT,
			home_team_id TEXT,
			away_score INTEGER,
			home_score INTEGER,
			FOREIGN KEY (away_team_id) REFERENCES nflverse_teams(id),
			FOREIGN KEY (home_team_id) REFERENCES nflverse_teams(id)
		);`,
		`CREATE TABLE IF NOT EXISTS nflverse_team_games (
			team_id TEXT,
			game_id TEXT,
			PRIMARY KEY (team_id, game_id),
			FOREIGN KEY (team_id) REFERENCES nflverse_teams(id),
			FOREIGN KEY (game_id) REFERENCES nflverse_games(game_id)
		);`,
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
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	fmt.Println("linebackerr tables created successfully.")
	return nil
}
