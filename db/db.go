package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dataDir = "/config"
)

var DB *sql.DB

type SportarrSeasons struct {
	Seasons []struct {
		PosterURL string `json:"poster_url"`
	} `json:"seasons"`
}

type SportarrSearch struct {
	Data struct {
		Search []struct {
			IdTeam interface{} `json:"idTeam"`
		} `json:"search"`
	} `json:"data"`
}

type SportarrLookup struct {
	Data struct {
		Lookup []struct {
			StrLogo          string `json:"strLogo"`
			StrBadge         string `json:"strBadge"`
			StrBanner        string `json:"strBanner"`
			StrFanart1       string `json:"strFanart1"`
			StrDescriptionEN string `json:"strDescriptionEN"`
		} `json:"lookup"`
	} `json:"data"`
}

func Init(nflverseDB *sql.DB) error {
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

	if err := loadSeasons(); err != nil {
		return fmt.Errorf("failed to load season posters: %w", err)
	}

	if err := loadTeams(nflverseDB); err != nil {
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
			strLogo TEXT,
			strBadge TEXT,
			strBanner TEXT,
			strFanart1 TEXT,
			strDescriptionEN TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS seasons (
			year TEXT PRIMARY KEY,
			poster_url TEXT
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

func loadSeasons() error {
	fmt.Println("Loading season metadata...")

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO seasons (year, poster_url) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 1. Fetch main season poster
	resp, err := http.Get("https://sportarr.net/api/metadata/plex/series/4391/seasons")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var seasonsData SportarrSeasons
			if err := json.NewDecoder(resp.Body).Decode(&seasonsData); err == nil && len(seasonsData.Seasons) > 0 {
				poster := seasonsData.Seasons[0].PosterURL
				if poster != "" {
					stmt.Exec("MAIN", poster)
					fmt.Println("Saved main series poster")
				}
			}
		}
	}

	// 2. Scrape individual seasons
	resp, err = http.Get("https://www.thesportsdb.com/league/4391-nfl")
	if err == nil {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		body := string(bodyBytes)

		re := regexp.MustCompile(`href=['"](/season/4391-[^/]+/(\d{4}))['"]`)
		matches := re.FindAllStringSubmatch(body, -1)

		visited := make(map[string]bool)
		for _, m := range matches {
			link := m[1]
			year := m[2]
			if !visited[year] {
				visited[year] = true
				seasonURL := "https://www.thesportsdb.com" + link
				fmt.Printf("Scraping poster for season %s...\n", year)
				
				sResp, sErr := http.Get(seasonURL)
				if sErr == nil {
					sBytes, _ := io.ReadAll(sResp.Body)
					sResp.Body.Close()
					
					reImg := regexp.MustCompile(`(https://[^\s'"]+poster[^\s'"]+\.jpg)`)
					imgMatches := reImg.FindAllStringSubmatch(string(sBytes), 1)
					if len(imgMatches) > 0 {
						stmt.Exec(year, imgMatches[0][1])
					}
				}
			}
		}
	}

	return tx.Commit()
}

func loadTeams(nflverseDB *sql.DB) error {
	fmt.Println("Loading team data from Sportarr...")

	// Get all teams from nflverse
	rows, err := nflverseDB.Query("SELECT id, full_name FROM teams")
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO team (team_id, strLogo, strBadge, strBanner, strFanart1, strDescriptionEN) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for rows.Next() {
		var teamID, fullName string
		if err := rows.Scan(&teamID, &fullName); err != nil {
			continue
		}

		fmt.Printf("Fetching Sportarr data for %s...\n", fullName)
		searchURL := "https://sportarr.net/api/v2/json/search/team/" + url.PathEscape(fullName)
		sResp, sErr := http.Get(searchURL)
		if sErr != nil {
			continue
		}
		
		var searchData SportarrSearch
		if err := json.NewDecoder(sResp.Body).Decode(&searchData); err != nil || len(searchData.Data.Search) == 0 {
			sResp.Body.Close()
			continue
		}
		sResp.Body.Close()

		idTeam := fmt.Sprintf("%v", searchData.Data.Search[0].IdTeam)
		
		lookupURL := "https://sportarr.net/api/v2/json/lookup/team/" + url.PathEscape(idTeam)
		lResp, lErr := http.Get(lookupURL)
		if lErr != nil {
			continue
		}

		var lookupData SportarrLookup
		if err := json.NewDecoder(lResp.Body).Decode(&lookupData); err == nil && len(lookupData.Data.Lookup) > 0 {
			teamData := lookupData.Data.Lookup[0]
			stmt.Exec(teamID, teamData.StrLogo, teamData.StrBadge, teamData.StrBanner, teamData.StrFanart1, teamData.StrDescriptionEN)
		}
		lResp.Body.Close()
	}

	return tx.Commit()
}
