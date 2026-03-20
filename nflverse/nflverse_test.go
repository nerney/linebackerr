package nflverse

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	linebackerrdb "linebackerr/db"

	_ "github.com/mattn/go-sqlite3"
)

func newNFLverseTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func TestResetDataRequiresSchemaOwnedByDBPackage(t *testing.T) {
	database := newNFLverseTestDB(t)

	if err := resetData(database); err == nil {
		t.Fatal("expected resetData to fail without pre-created schema")
	}
}

func TestResetDataClearsExistingNFLverseTables(t *testing.T) {
	t.Setenv("LINEBACKERR_DATA_DIR", t.TempDir())
	database := linebackerrdb.Init()
	defer database.Close()

	stmts := []string{
		`INSERT INTO nflverse_teams (id, abbr, full_name) VALUES ('1', 'NE', 'New England Patriots')`,
		`INSERT INTO nflverse_games (game_id, season, week, game_type, gameday, away_team_id, home_team_id, away_score, home_score) VALUES ('g1', 2024, 1, 'RS', '2024-09-08', '1', '1', 1, 2)`,
		`INSERT INTO nflverse_team_games (team_id, game_id) VALUES ('1', 'g1')`,
	}
	for _, stmt := range stmts {
		if _, err := database.Exec(stmt); err != nil {
			t.Fatalf("seed nflverse tables: %v", err)
		}
	}

	if err := resetData(database); err != nil {
		t.Fatalf("resetData: %v", err)
	}

	for _, table := range []string{"nflverse_teams", "nflverse_games", "nflverse_team_games"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("expected %s to be empty after reset, got %d rows", table, count)
		}
	}
}

func TestLoadTeamsAndGamesPopulateDatabase(t *testing.T) {
	t.Setenv("LINEBACKERR_DATA_DIR", t.TempDir())
	database := linebackerrdb.Init()
	defer database.Close()

	if err := resetData(database); err != nil {
		t.Fatalf("resetData: %v", err)
	}

	dir := t.TempDir()
	teamsPath := filepath.Join(dir, "teams.csv")
	gamesPath := filepath.Join(dir, "games.csv")

	teamsCSV := "team,nfl_team_id,full\nNE,1,New England Patriots\nBUF,2,Buffalo Bills\nXXX,,Missing ID\n"
	gamesCSV := "game_id,season,week,game_type,gameday,away_team,home_team,away_score,home_score\n2024_01_BUF_NE,2024,1,RS,2024-09-08,BUF,NE,17,24\n2024_02_NE_MIA,2024,2,RS,2024-09-15,NE,MIA,21,14\n,2024,3,RS,2024-09-22,BUF,NE,10,7\n"

	if err := os.WriteFile(teamsPath, []byte(teamsCSV), 0644); err != nil {
		t.Fatalf("write teams csv: %v", err)
	}
	if err := os.WriteFile(gamesPath, []byte(gamesCSV), 0644); err != nil {
		t.Fatalf("write games csv: %v", err)
	}

	teamMap, err := loadTeams(database, teamsPath)
	if err != nil {
		t.Fatalf("loadTeams: %v", err)
	}
	if len(teamMap) != 2 || teamMap["NE"] != "1" || teamMap["BUF"] != "2" {
		t.Fatalf("unexpected team map: %#v", teamMap)
	}

	if err := loadGames(database, gamesPath, teamMap); err != nil {
		t.Fatalf("loadGames: %v", err)
	}

	var awayID, homeID sql.NullString
	var gameType string
	var awayScore, homeScore int
	if err := database.QueryRow(`SELECT away_team_id, home_team_id, game_type, away_score, home_score FROM nflverse_games WHERE game_id = '2024_01_BUF_NE'`).Scan(&awayID, &homeID, &gameType, &awayScore, &homeScore); err != nil {
		t.Fatalf("query first game: %v", err)
	}
	if !awayID.Valid || awayID.String != "2" || !homeID.Valid || homeID.String != "1" || gameType != "RS" || awayScore != 17 || homeScore != 24 {
		t.Fatalf("unexpected first game row: away=%#v home=%#v type=%q score=%d-%d", awayID, homeID, gameType, awayScore, homeScore)
	}

	if err := database.QueryRow(`SELECT away_team_id, home_team_id FROM nflverse_games WHERE game_id = '2024_02_NE_MIA'`).Scan(&awayID, &homeID); err != nil {
		t.Fatalf("query second game: %v", err)
	}
	if !awayID.Valid || awayID.String != "1" || homeID.Valid {
		t.Fatalf("unexpected second game teams: away=%#v home=%#v", awayID, homeID)
	}

	var linksCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM nflverse_team_games`).Scan(&linksCount); err != nil {
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
