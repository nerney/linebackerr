package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestInitCreatesDatabaseFileAndAllOwnedTables(t *testing.T) {
	t.Setenv("LINEBACKERR_DATA_DIR", t.TempDir())

	database := Init()
	require.NotNil(t, database)
	defer database.Close()

	for _, table := range []string{
		"nflverse_teams",
		"nflverse_games",
		"nflverse_team_games",
		"sportarr_team",
		"sportarr_seasons",
	} {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		require.NoErrorf(t, err, "table %s missing after Init", table)
	}

	_, err := os.Stat(DBPath())
	require.NoErrorf(t, err, "expected db file at %s", DBPath())
}

func TestInitSchemaBuildsAllTablesOnProvidedDB(t *testing.T) {
	database, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer database.Close()

	require.NoError(t, initSchema(database))

	for _, table := range []string{
		"nflverse_teams",
		"nflverse_games",
		"nflverse_team_games",
		"sportarr_team",
		"sportarr_seasons",
	} {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		require.NoErrorf(t, err, "table %s missing", table)
	}
}
