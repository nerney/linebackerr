package nflverse

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	teamsURL = "https://raw.githubusercontent.com/nflverse/nfldata/refs/heads/master/data/teams.csv"
	gamesURL = "https://raw.githubusercontent.com/nflverse/nfldata/refs/heads/master/data/games.csv"
	dataDir  = "/config"
)

var DB *sql.DB

func Init() error {
	fmt.Println("Initializing nflverse package...")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	teamsPath := filepath.Join(dataDir, "teams.csv")
	if err := downloadFile(teamsURL, teamsPath); err != nil {
		return err
	}

	gamesPath := filepath.Join(dataDir, "games.csv")
	if err := downloadFile(gamesURL, gamesPath); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "nflverse.db")
	fmt.Printf("Connecting to SQLite database at %s...\n", dbPath)
	
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite db: %w", err)
	}
	
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping sqlite db: %w", err)
	}
	
	fmt.Println("Successfully connected to SQLite database.")

	if err := initDB(DB); err != nil {
		return fmt.Errorf("failed to initialize db schema: %w", err)
	}

	teamMap, err := loadTeams(DB, teamsPath)
	if err != nil {
		return fmt.Errorf("failed to load teams: %w", err)
	}

	if err := loadGames(DB, gamesPath, teamMap); err != nil {
		return fmt.Errorf("failed to load games: %w", err)
	}

	fmt.Println("nflverse initialization complete.")
	return nil
}

func initDB(db *sql.DB) error {
	fmt.Println("Creating database tables...")
	
	queries := []string{
		`CREATE TABLE IF NOT EXISTS teams (
			id TEXT PRIMARY KEY,
			abbr TEXT,
			full_name TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS games (
			game_id TEXT PRIMARY KEY,
			season INTEGER,
			week INTEGER,
			gameday TEXT,
			away_team_id TEXT,
			home_team_id TEXT,
			away_score INTEGER,
			home_score INTEGER,
			FOREIGN KEY (away_team_id) REFERENCES teams(id),
			FOREIGN KEY (home_team_id) REFERENCES teams(id)
		);`,
		`CREATE TABLE IF NOT EXISTS team_games (
			team_id TEXT,
			game_id TEXT,
			PRIMARY KEY (team_id, game_id),
			FOREIGN KEY (team_id) REFERENCES teams(id),
			FOREIGN KEY (game_id) REFERENCES games(game_id)
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	fmt.Println("Tables created successfully.")
	return nil
}

func loadTeams(db *sql.DB, path string) (map[string]string, error) {
	fmt.Println("Loading teams data into database...")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	
	teamIdx, idIdx, fullIdx := -1, -1, -1
	for i, h := range headers {
		if h == "team" { teamIdx = i }
		if h == "nfl_team_id" { idIdx = i }
		if h == "full" { fullIdx = i }
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	
	stmt, err := tx.Prepare("REPLACE INTO teams (id, abbr, full_name) VALUES (?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	teamMap := make(map[string]string)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		if teamIdx != -1 && idIdx != -1 && fullIdx != -1 {
			abbr := record[teamIdx]
			teamID := record[idIdx]
			fullName := record[fullIdx]
			
			if teamID == "" || abbr == "" {
				continue
			}

			// Keep mapping for games lookup
			teamMap[abbr] = teamID

			_, err = stmt.Exec(teamID, abbr, fullName)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}
	
	return teamMap, tx.Commit()
}

func loadGames(db *sql.DB, path string, teamMap map[string]string) error {
	fmt.Println("Loading games data into database...")
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 
	
	headers, err := reader.Read()
	if err != nil {
		return err
	}
	
	idx := make(map[string]int)
	for i, h := range headers {
		idx[h] = i
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	
	gameStmt, err := tx.Prepare("INSERT OR IGNORE INTO games (game_id, season, week, gameday, away_team_id, home_team_id, away_score, home_score) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer gameStmt.Close()

	linkStmt, err := tx.Prepare("INSERT OR IGNORE INTO team_games (team_id, game_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer linkStmt.Close()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		getCol := func(col string) string {
			if i, ok := idx[col]; ok && i < len(record) {
				return record[i]
			}
			return ""
		}

		gameID := getCol("game_id")
		awayAbbr := getCol("away_team")
		homeAbbr := getCol("home_team")
		
		if gameID == "" {
			continue
		}

		awayID := teamMap[awayAbbr]
		homeID := teamMap[homeAbbr]
		
		_, err = gameStmt.Exec(gameID, getCol("season"), getCol("week"), getCol("gameday"), awayID, homeID, getCol("away_score"), getCol("home_score"))
		if err != nil {
			tx.Rollback()
			return err
		}

		if awayID != "" {
			_, err = linkStmt.Exec(awayID, gameID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		if homeID != "" {
			_, err = linkStmt.Exec(homeID, gameID)
			if err != nil {
				tx.Rollback()
				return err
			}
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
