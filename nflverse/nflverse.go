package nflverse

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"linebackerr/db"
)

const (
	teamsURL      = "https://raw.githubusercontent.com/nflverse/nfldata/refs/heads/master/data/teams.csv"
	gamesURL      = "https://raw.githubusercontent.com/nflverse/nfldata/refs/heads/master/data/games.csv"
	teamsFileName = "teams.csv"
	gamesFileName = "games.csv"
)

type Client struct {
	DB      *sql.DB
	TeamMap map[string]string
}

// Init downloads the nflverse CSV data, rebuilds nflverse tables, and returns the initialized client.
func Init(database *sql.DB) *Client {
	fmt.Println("Initializing nflverse package...")

	dataDir := db.DataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		panic(fmt.Errorf("failed to create data directory: %w", err))
	}

	teamsPath := filepath.Join(dataDir, teamsFileName)
	gamesPath := filepath.Join(dataDir, gamesFileName)

	for _, path := range []string{teamsPath, gamesPath} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			panic(fmt.Errorf("failed to reset cached nflverse file %s: %w", path, err))
		}
	}

	if err := downloadFile(teamsURL, teamsPath); err != nil {
		panic(err)
	}
	if err := downloadFile(gamesURL, gamesPath); err != nil {
		panic(err)
	}

	if err := initDB(database); err != nil {
		panic(fmt.Errorf("failed to initialize db schema: %w", err))
	}

	teamMap, err := loadTeams(database, teamsPath)
	if err != nil {
		panic(fmt.Errorf("failed to load teams: %w", err))
	}

	if err := loadGames(database, gamesPath, teamMap); err != nil {
		panic(fmt.Errorf("failed to load games: %w", err))
	}

	fmt.Println("nflverse initialization complete.")
	return &Client{DB: database, TeamMap: teamMap}
}

func initDB(db *sql.DB) error {
	fmt.Println("Creating database tables for nflverse...")

	queries := []string{
		`DROP TABLE IF EXISTS nflverse_team_games;`,
		`DROP TABLE IF EXISTS nflverse_games;`,
		`DROP TABLE IF EXISTS nflverse_teams;`,
		`CREATE TABLE nflverse_teams (
			id TEXT PRIMARY KEY,
			abbr TEXT,
			full_name TEXT
		);`,
		`CREATE TABLE nflverse_games (
			game_id TEXT PRIMARY KEY,
			season INTEGER,
			week INTEGER,
			gameday TEXT,
			away_team_id TEXT,
			home_team_id TEXT,
			away_score INTEGER,
			home_score INTEGER,
			FOREIGN KEY (away_team_id) REFERENCES nflverse_teams(id),
			FOREIGN KEY (home_team_id) REFERENCES nflverse_teams(id)
		);`,
		`CREATE TABLE nflverse_team_games (
			team_id TEXT,
			game_id TEXT,
			PRIMARY KEY (team_id, game_id),
			FOREIGN KEY (team_id) REFERENCES nflverse_teams(id),
			FOREIGN KEY (game_id) REFERENCES nflverse_games(game_id)
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
		if h == "team" {
			teamIdx = i
		}
		if h == "nfl_team_id" {
			idIdx = i
		}
		if h == "full" {
			fullIdx = i
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	stmt, err := tx.Prepare("REPLACE INTO nflverse_teams (id, abbr, full_name) VALUES (?, ?, ?)")
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

	gameStmt, err := tx.Prepare("INSERT OR IGNORE INTO nflverse_games (game_id, season, week, gameday, away_team_id, home_team_id, away_score, home_score) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer gameStmt.Close()

	linkStmt, err := tx.Prepare("INSERT OR IGNORE INTO nflverse_team_games (team_id, game_id) VALUES (?, ?)")
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
