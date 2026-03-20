package matcher

import (
	"database/sql"
	"fmt"
)

// nflverseLookupStage attempts to find a match in the nflverse_games table
// using the exact GameDate, HomeTeam, and AwayTeam.
func nflverseLookupStage(db *sql.DB, match *Match) error {
	if match.GameDate == "" || match.HomeTeam == "" || match.AwayTeam == "" {
		return nil
	}

	query := `
		SELECT game_id 
		FROM nflverse_games 
		WHERE gameday = ? 
		  AND (
			(home_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?) AND away_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?))
			OR 
			(home_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?) AND away_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?))
		  )
		LIMIT 1`

	var gameID string
	err := db.QueryRow(query, match.GameDate, match.HomeTeam, match.AwayTeam, match.AwayTeam, match.HomeTeam).Scan(&gameID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("nflverse lookup failed: %w", err)
	}

	match.NflverseID = gameID
	return nil
}
