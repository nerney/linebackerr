package matcher

import (
	"database/sql"
	"errors"
	"testing"

	"linebackerr/db"

	_ "github.com/mattn/go-sqlite3"
)

func setupFinalPipelineTestDB(t *testing.T) *sql.DB {
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

func TestMatchCandidateValidate_FinalPipeline_UnmatchedReturnsCandidateAndError(t *testing.T) {
	testDB := setupFinalPipelineTestDB(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = originalDB })

	candidate := MatchCandidate{
		OriginalInput: "random unmatched game",
		GameType:      GameTypeConference,
		GameDate:      "2024-01-21",
		GameWeek:      "20",
		SeasonYear:    "2023",
		AwayTeam:      "BUF",
		HomeTeam:      "BAL",
	}

	match := candidate.Validate()

	if !errors.Is(match.Error, ErrNoMatchFound) {
		t.Fatalf("expected ErrNoMatchFound, got %v", match.Error)
	}
	if match.NflverseID != "" {
		t.Fatalf("expected empty nflverse id for unmatched game, got %q", match.NflverseID)
	}
	if match.OriginalInput != candidate.OriginalInput ||
		match.GameType != candidate.GameType ||
		match.GameDate != candidate.GameDate ||
		match.GameWeek != candidate.GameWeek ||
		match.SeasonYear != candidate.SeasonYear ||
		match.AwayTeam != candidate.AwayTeam ||
		match.HomeTeam != candidate.HomeTeam {
		t.Fatalf("expected unmatched result to preserve candidate fields, got %#v", match)
	}
	if match.IsResolved() {
		t.Fatalf("expected unmatched game to be unresolved")
	}
}

func TestMatch_MatchedHelper(t *testing.T) {
	matched := Match{NflverseID: "2021_20_KC_NE"}
	if !matched.Matched() {
		t.Fatalf("expected match with nflverse id and no error to be matched")
	}

	unmatchedNoID := Match{}
	if unmatchedNoID.Matched() {
		t.Fatalf("expected empty match to be unmatched")
	}

	unmatchedErr := Match{NflverseID: "2021_20_KC_NE", Error: ErrNoMatchFound}
	if unmatchedErr.Matched() {
		t.Fatalf("expected errored match to be unmatched")
	}
}

func TestMatchCandidateValidate_FinalPipeline_ExistingErrorPreserved(t *testing.T) {
	testDB := setupFinalPipelineTestDB(t)
	defer testDB.Close()

	originalDB := db.DB
	db.DB = testDB
	t.Cleanup(func() { db.DB = originalDB })

	match := MatchCandidate{
		GameType:   GameTypeConference,
		SeasonYear: "2023",
		HomeTeam:   "BAL",
		// AwayTeam intentionally missing to trigger stage validation error.
	}.Validate()

	if !errors.Is(match.Error, ErrMissingFieldsForNflverseLookup) {
		t.Fatalf("expected ErrMissingFieldsForNflverseLookup, got %v", match.Error)
	}
	if errors.Is(match.Error, ErrNoMatchFound) {
		t.Fatalf("did not expect ErrNoMatchFound when a specific validation error exists")
	}
	if match.IsResolved() {
		t.Fatalf("expected invalid match to be unresolved")
	}
}
