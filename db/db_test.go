package db

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestDBInitAndSync(t *testing.T) {
	var err error
	DB, err = sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer DB.Close()

	if err := initSchema(); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	if !NeedsSync("test_module") {
		t.Error("Expected true for new module")
	}

	if err := UpdateSync("test_module"); err != nil {
		t.Errorf("Failed to update sync: %v", err)
	}

	if NeedsSync("test_module") {
		t.Error("Expected false after update")
	}

	oldTime := time.Now().Add(-50 * time.Hour)
	_, err = DB.Exec("UPDATE sync_state SET last_sync = ? WHERE module = ?", oldTime, "test_module")
	if err != nil {
		t.Fatalf("Failed to update time: %v", err)
	}

	if !NeedsSync("test_module") {
		t.Error("Expected true after 50 hours")
	}
}
