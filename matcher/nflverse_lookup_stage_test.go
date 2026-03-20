package matcher

import (
	"database/sql"
	"errors"
	"testing"

	"linebackerr/db"

	_ "github.com/mattn/go-sqlite3"
)

func setupNflverseLookupTestDB(t *testing.T) *sql.DB {
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
		t.Fatalf("create nflverse tables: %v", err)
	}

	return testDB
}

func TestMatchCandidateValidate_NflverseLookup_ByGameTypeSeasonYearTeams_FindsMatch(t *testing.T) {
	testDB := setupNflverseLookupTestDB(t)
	defer testDB.Close()

	_, err := testDB.Exec(`
		INSERT INTO nflverse_teams (id, abbr) VALUES
		('NE', 'NE'),
		('KC', 'KC');

		INSERT INTO nflverse_games (game_id, season, week, game_type, gameday, away_team_id, home_team_id)
		VALUES ('2021_20_KC_NE', '2021', '20', 'CON', '2022-01-30', 'NE', 'KC');
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = originalDB })

	match := MatchCandidate{
		GameType:   GameTypeConference,
		SeasonYear: "2021",
		HomeTeam:   "KC",
		AwayTeam:   "NE",
	}.Validate()

	if match.Error != nil {
		t.Fatalf("expected no error, got %v", match.Error)
	}
	if match.NflverseID != "2021_20_KC_NE" {
		t.Fatalf("expected nflverse id 2021_20_KC_NE, got %q", match.NflverseID)
	}
	if !match.IsResolved() {
		t.Fatalf("expected match to be resolved")
	}
}

func TestMatchCandidateValidate_NflverseLookup_PopulatesMatchFieldsFromGameResult(t *testing.T) {
	testDB := setupNflverseLookupTestDB(t)
	defer testDB.Close()

	_, err := testDB.Exec(`
		INSERT INTO nflverse_teams (id, abbr) VALUES
		('NE', 'NE'),
		('KC', 'KC');

		INSERT INTO nflverse_games (game_id, season, week, game_type, gameday, away_team_id, home_team_id)
		VALUES ('2021_20_KC_NE', '2021', '20', 'CON', '2022-01-30', 'NE', 'KC');
	`)
	if err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = originalDB })

	match := MatchCandidate{
		GameType:   GameTypeConference,
		GameDate:   "2022-01-30",
		SeasonYear: "1999", // intentionally wrong to verify population from nflverse
		HomeTeam:   "NE",   // intentionally swapped
		AwayTeam:   "KC",
	}.Validate()

	if match.Error != nil {
		t.Fatalf("expected no error, got %v", match.Error)
	}
	if match.NflverseID != "2021_20_KC_NE" {
		t.Fatalf("expected nflverse id 2021_20_KC_NE, got %q", match.NflverseID)
	}
	if match.GameType != GameTypeConference {
		t.Fatalf("expected game type CON, got %q", match.GameType)
	}
	if match.SeasonYear != "2021" {
		t.Fatalf("expected season year 2021, got %q", match.SeasonYear)
	}
	if match.GameWeek != "20" {
		t.Fatalf("expected game week 20, got %q", match.GameWeek)
	}
	if match.GameDate != "2022-01-30" {
		t.Fatalf("expected game date 2022-01-30, got %q", match.GameDate)
	}
	if match.HomeTeam != "KC" || match.AwayTeam != "NE" {
		t.Fatalf("expected teams NE@KC from nflverse, got away=%q home=%q", match.AwayTeam, match.HomeTeam)
	}
	if !match.IsResolved() {
		t.Fatalf("expected match to be resolved")
	}
}

func TestMatchCandidateValidate_NflverseLookup_MissingRequiredField_ReturnsError(t *testing.T) {
	testDB := setupNflverseLookupTestDB(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = originalDB })

	match := MatchCandidate{
		GameType:   GameTypeConference,
		SeasonYear: "2021",
		HomeTeam:   "KC",
		// AwayTeam intentionally missing.
	}.Validate()

	if !errors.Is(match.Error, ErrMissingFieldsForNflverseLookup) {
		t.Fatalf("expected ErrMissingFieldsForNflverseLookup, got %v", match.Error)
	}
	if match.NflverseID != "" {
		t.Fatalf("expected no nflverse id, got %q", match.NflverseID)
	}
	if match.IsResolved() {
		t.Fatalf("expected match to be unresolved")
	}
}
