package matcher

import (
	"database/sql"
	"errors"
	"testing"

	"linebackerr/db"

	_ "github.com/mattn/go-sqlite3"
)

func setupSuperBowlTestDB(t *testing.T) *sql.DB {
	t.Helper()

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE nflverse_teams (
			id TEXT PRIMARY KEY,
			abbr TEXT
		);

		CREATE TABLE nflverse_games (
			game_id TEXT PRIMARY KEY,
			season TEXT,
			week TEXT,
			game_type TEXT,
			gameday TEXT,
			away_team_id TEXT,
			home_team_id TEXT
		);
	`)
	if err != nil {
		testDB.Close()
		t.Fatalf("create nflverse_games table: %v", err)
	}

	return testDB
}

func TestMatchCandidateValidate_SuperBowlRomanNumeralLookup_FindsMatch(t *testing.T) {
	testDB := setupSuperBowlTestDB(t)
	defer testDB.Close()

	_, err := testDB.Exec(`
		INSERT INTO nflverse_teams (id, abbr) VALUES ('KC', 'KC'), ('SF', 'SF');
		INSERT INTO nflverse_games (game_id, season, week, game_type, gameday, away_team_id, home_team_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "SB_LVIII_2023", "2023", "22", "SB", "2024-02-11", "SF", "KC")
	if err != nil {
		t.Fatalf("insert test game: %v", err)
	}

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	match := MatchCandidate{
		GameType: GameTypeSuperBowl,
		GameWeek: "LVIII",
	}.Validate()

	if match.Error != nil {
		t.Fatalf("expected no error, got %v", match.Error)
	}
	if match.NflverseID != "SB_LVIII_2023" {
		t.Fatalf("expected nflverse id SB_LVIII_2023, got %q", match.NflverseID)
	}
	if match.SeasonYear != "2023" {
		t.Fatalf("expected season year 2023, got %q", match.SeasonYear)
	}
	if match.GameDate != "2024-02-11" {
		t.Fatalf("expected game date 2024-02-11, got %q", match.GameDate)
	}
	if match.HomeTeam != "KC" || match.AwayTeam != "SF" {
		t.Fatalf("expected teams SF@KC from nflverse, got away=%q home=%q", match.AwayTeam, match.HomeTeam)
	}
}

func TestMatchCandidateValidate_SuperBowlRomanNumeralLookup_NoRomanNumeral_NoMatch(t *testing.T) {
	testDB := setupSuperBowlTestDB(t)
	defer testDB.Close()

	_, err := testDB.Exec(`
		INSERT INTO nflverse_teams (id, abbr) VALUES ('KC', 'KC'), ('SF', 'SF');
		INSERT INTO nflverse_games (game_id, season, week, game_type, gameday, away_team_id, home_team_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "SB_LVIII_2023", "2023", "22", "SB", "2024-02-11", "SF", "KC")
	if err != nil {
		t.Fatalf("insert test game: %v", err)
	}

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	match := MatchCandidate{
		GameType: GameTypeSuperBowl,
		GameWeek: "week 18",
	}.Validate()

	if !errors.Is(match.Error, ErrNoMatchFound) {
		t.Fatalf("expected ErrNoMatchFound, got %v", match.Error)
	}
	if match.NflverseID != "" {
		t.Fatalf("expected no nflverse id, got %q", match.NflverseID)
	}
	if match.IsResolved() {
		t.Fatalf("expected match to be unresolved")
	}
}
