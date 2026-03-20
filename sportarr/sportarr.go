package sportarr

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"linebackerr/nflverse"
	"net/http"
	"net/url"
	"regexp"
)

type Client struct {
	DB       *sql.DB
	NFLverse *nflverse.Client
}

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

func Init(database *sql.DB, nflv *nflverse.Client) *Client {
	fmt.Println("Initializing sportarr package...")

	if err := initDB(database); err != nil {
		panic(fmt.Errorf("failed to initialize sportarr db schema: %w", err))
	}

	client := &Client{DB: database, NFLverse: nflv}
	if err := client.LoadSeasons(); err != nil {
		panic(fmt.Errorf("failed to load seasons from sportarr: %w", err))
	}
	if err := client.LoadTeams(); err != nil {
		panic(fmt.Errorf("failed to load teams from sportarr: %w", err))
	}

	fmt.Println("sportarr initialization complete.")
	return client
}

func initDB(db *sql.DB) error {
	queries := []string{
		`DELETE FROM sportarr_team;`,
		`DELETE FROM sportarr_seasons;`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// LoadSeasons fetches the main poster and scrapes individual season posters.
func (c *Client) LoadSeasons() error {
	fmt.Println("Loading season metadata from Sportarr...")

	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO sportarr_seasons (year, poster_url) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	resp, err := http.Get("https://sportarr.net/api/metadata/plex/series/4391/seasons")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var seasonsData SportarrSeasons
			if err := json.NewDecoder(resp.Body).Decode(&seasonsData); err == nil && len(seasonsData.Seasons) > 0 {
				poster := seasonsData.Seasons[0].PosterURL
				if poster != "" {
					_, _ = stmt.Exec("MAIN", poster)
					fmt.Println("Saved main series poster")
				}
			}
		}
	}

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
						_, _ = stmt.Exec(year, imgMatches[0][1])
					}
				}
			}
		}
	}

	return tx.Commit()
}

// LoadTeams fetches team data from Sportarr by looking up team names from nflverse.
func (c *Client) LoadTeams() error {
	fmt.Println("Loading team data from Sportarr...")

	rows, err := c.DB.Query("SELECT id, full_name FROM nflverse_teams")
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := c.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO sportarr_team (team_id, strLogo, strBadge, strBanner, strFanart1, strDescriptionEN) VALUES (?, ?, ?, ?, ?, ?)")
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
			_, _ = stmt.Exec(teamID, teamData.StrLogo, teamData.StrBadge, teamData.StrBanner, teamData.StrFanart1, teamData.StrDescriptionEN)
		}
		lResp.Body.Close()
	}

	return tx.Commit()
}
