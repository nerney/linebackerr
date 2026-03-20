package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitCreatesDatabaseFileAndAllOwnedTables(t *testing.T) {
	t.Setenv("LINEBACKERR_DATA_DIR", t.TempDir())

	database := Init()
	if database == nil {
		t.Fatal("Init returned nil db")
	}
	defer database.Close()

	for _, table := range []string{
		"nflverse_teams",
		"nflverse_games",
		"nflverse_team_games",
		"sportarr_team",
		"sportarr_seasons",
	} {
		var name string
		if err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing after Init: %v", table, err)
		}
	}

	if _, err := os.Stat(DBPath()); err != nil {
		t.Fatalf("expected db file at %s: %v", DBPath(), err)
	}
}

func TestInitSchemaBuildsAllTablesOnProvidedDB(t *testing.T) {
	database, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer database.Close()

	if err := initSchema(database); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	for _, table := range []string{
		"nflverse_teams",
		"nflverse_games",
		"nflverse_team_games",
		"sportarr_team",
		"sportarr_seasons",
	} {
		var name string
		if err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}
