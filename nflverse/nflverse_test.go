package nflverse

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newNFLverseTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestInitDBCreatesTables(t *testing.T) {
	db := newNFLverseTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatalf("initDB: %v", err)
	}

	for _, table := range []string{"nflverse_teams", "nflverse_games", "nflverse_team_games"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestLoadTeamsAndGamesPopulateDatabase(t *testing.T) {
	db := newNFLverseTestDB(t)
	if err := initDB(db); err != nil {
		t.Fatalf("initDB: %v", err)
	}

	dir := t.TempDir()
	teamsPath := filepath.Join(dir, "teams.csv")
	gamesPath := filepath.Join(dir, "games.csv")

	teamsCSV := "team,nfl_team_id,full\nNE,1,New England Patriots\nBUF,2,Buffalo Bills\nXXX,,Missing ID\n"
	gamesCSV := "game_id,season,week,gameday,away_team,home_team,away_score,home_score\n2024_01_BUF_NE,2024,1,2024-09-08,BUF,NE,17,24\n2024_02_NE_MIA,2024,2,2024-09-15,NE,MIA,21,14\n,2024,3,2024-09-22,BUF,NE,10,7\n"

	if err := os.WriteFile(teamsPath, []byte(teamsCSV), 0644); err != nil {
		t.Fatalf("write teams csv: %v", err)
	}
	if err := os.WriteFile(gamesPath, []byte(gamesCSV), 0644); err != nil {
		t.Fatalf("write games csv: %v", err)
	}

	teamMap, err := loadTeams(db, teamsPath)
	if err != nil {
		t.Fatalf("loadTeams: %v", err)
	}
	if len(teamMap) != 2 || teamMap["NE"] != "1" || teamMap["BUF"] != "2" {
		t.Fatalf("unexpected team map: %#v", teamMap)
	}

	if err := loadGames(db, gamesPath, teamMap); err != nil {
		t.Fatalf("loadGames: %v", err)
	}

	var teamsCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nflverse_teams`).Scan(&teamsCount); err != nil {
		t.Fatalf("count teams: %v", err)
	}
	if teamsCount != 2 {
		t.Fatalf("expected 2 teams, got %d", teamsCount)
	}

	var awayID, homeID string
	var awayScore, homeScore int
	if err := db.QueryRow(`SELECT away_team_id, home_team_id, away_score, home_score FROM nflverse_games WHERE game_id = '2024_01_BUF_NE'`).Scan(&awayID, &homeID, &awayScore, &homeScore); err != nil {
		t.Fatalf("query first game: %v", err)
	}
	if awayID != "2" || homeID != "1" || awayScore != 17 || homeScore != 24 {
		t.Fatalf("unexpected first game row: away=%q home=%q score=%d-%d", awayID, homeID, awayScore, homeScore)
	}

	if err := db.QueryRow(`SELECT away_team_id, home_team_id FROM nflverse_games WHERE game_id = '2024_02_NE_MIA'`).Scan(&awayID, &homeID); err != nil {
		t.Fatalf("query second game: %v", err)
	}
	if awayID != "1" || homeID != "" {
		t.Fatalf("unexpected second game teams: away=%q home=%q", awayID, homeID)
	}

	var linksCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nflverse_team_games`).Scan(&linksCount); err != nil {
		t.Fatalf("count links: %v", err)
	}
	if linksCount != 3 {
		t.Fatalf("expected 3 team-game links, got %d", linksCount)
	}
}

func TestDownloadFileDownloadsAndSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "teams.csv")
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("team,nfl_team_id,full\nNE,1,New England Patriots\n"))
	}))
	defer server.Close()

	if err := downloadFile(server.URL, target); err != nil {
		t.Fatalf("downloadFile first run: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 HTTP hit after first download, got %d", hits)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(content) == "" {
		t.Fatal("downloaded file was empty")
	}

	if err := downloadFile(server.URL, target); err != nil {
		t.Fatalf("downloadFile second run: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected existing file to skip second download, got %d HTTP hits", hits)
	}
}
