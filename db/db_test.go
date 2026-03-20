package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestInitResetsDatabaseAndBuildsSchema(t *testing.T) {
	t.Setenv("LINEBACKERR_DATA_DIR", t.TempDir())

	first := Init()
	if first == nil {
		t.Fatal("Init returned nil db")
	}
	t.Cleanup(func() { _ = first.Close() })

	if _, err := first.Exec(`INSERT INTO sportarr_seasons (year, poster_url) VALUES ('2024', 'poster')`); err != nil {
		t.Fatalf("seed row: %v", err)
	}

	var count int
	if err := first.QueryRow(`SELECT COUNT(*) FROM sportarr_seasons`).Scan(&count); err != nil {
		t.Fatalf("count seeded rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 seeded row, got %d", count)
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close first db: %v", err)
	}

	second := Init()
	if second == nil {
		t.Fatal("second Init returned nil db")
	}

	for _, table := range []string{"sportarr_team", "sportarr_seasons"} {
		var name string
		if err := second.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing after reset: %v", table, err)
		}
	}

	if err := second.QueryRow(`SELECT COUNT(*) FROM sportarr_seasons`).Scan(&count); err != nil {
		t.Fatalf("count rows after reset: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected clean-slate sportarr_seasons after reset, got %d rows", count)
	}

	if _, err := os.Stat(DBPath()); err != nil {
		t.Fatalf("expected db file at %s: %v", DBPath(), err)
	}
}

func TestInitSchemaBuildsTablesOnProvidedDB(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	if err := initSchema(db); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	for _, table := range []string{"sportarr_team", "sportarr_seasons"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}
