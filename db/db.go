package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	teamsAPIURL = "https://www.thesportsdb.com/api/v1/json/3/search_all_teams.php?l=NFL"
	dataDir     = "/config"
)

var DB *sql.DB

type SportsDBResponse struct {
	Teams []struct {
		StrTeam  string `json:"strTeam"`
		StrBadge string `json:"strBadge"`
		StrLogo  string `json:"strLogo"`
	} `json:"teams"`
}

func Init(nflverseDB *sql.DB) error {
	fmt.Println("Initializing linebackerr DB package...")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	teamsJSONPath := filepath.Join(dataDir, "sportsdb_teams.json")
	if err := downloadFile(teamsAPIURL, teamsJSONPath); err != nil {
		return err
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

	if err := loadTeamArt(nflverseDB, teamsJSONPath); err != nil {
		return fmt.Errorf("failed to load team art: %w", err)
	}

	fmt.Println("linebackerr DB initialization complete.")
	return nil
}

func initSchema() error {
	fmt.Println("Creating linebackerr DB tables...")
	queries := []string{
		`CREATE TABLE IF NOT EXISTS team (
			team_id TEXT PRIMARY KEY,
			badge_url TEXT,
			logo_url TEXT
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

func loadTeamArt(nflverseDB *sql.DB, jsonPath string) error {
	fmt.Println("Loading team art data into database...")

	file, err := os.Open(jsonPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var data SportsDBResponse
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO team (team_id, badge_url, logo_url) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, team := range data.Teams {
		var teamID string
		err := nflverseDB.QueryRow("SELECT id FROM teams WHERE full_name = ?", team.StrTeam).Scan(&teamID)
		if err == sql.ErrNoRows {
			errFallback := nflverseDB.QueryRow("SELECT id FROM teams WHERE full_name LIKE ?", "%"+team.StrTeam+"%").Scan(&teamID)
			if errFallback != nil {
				fmt.Printf("Warning: Could not find team ID for %s in nflverse DB\n", team.StrTeam)
				continue
			}
		} else if err != nil {
			tx.Rollback()
			return err
		}

		if _, err := stmt.Exec(teamID, team.StrBadge, team.StrLogo); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func downloadFile(url, filepath string) error {
	if _, err := os.Stat(filepath); err == nil {
		fmt.Printf("File %s already exists, skipping download.\n", filepath)
		return nil
	}

	fmt.Printf("Downloading %s to %s...\n", url, filepath)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file %s: %w", filepath, err)
	}

	fmt.Printf("Successfully downloaded %s\n", filepath)
	return nil
}
