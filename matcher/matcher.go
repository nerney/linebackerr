package matcher

import (
	"database/sql"
	"fmt"
)

// MatchResult represents the outcome of a match attempt for a release string.
type MatchResult struct {
	ReleaseName string
	GameID      string // Empty if no match found
	Matched     bool
	Confidence  float64 // 0.0 to 1.0
}

// MatchReleases takes a list of release strings and an nflverse database connection,
// and attempts to match each string to a specific game in the nflverse DB.
func MatchReleases(db *sql.DB, releases []string) []MatchResult {
	var results []MatchResult

	for _, release := range releases {
		// TODO: Implement parsing and matching logic here.
		// For now, just return unmatched results.
		fmt.Printf("Analyzing release: %s\n", release)

		results = append(results, MatchResult{
			ReleaseName: release,
			GameID:      "",
			Matched:     false,
			Confidence:  0.0,
		})
	}

	return results
}
