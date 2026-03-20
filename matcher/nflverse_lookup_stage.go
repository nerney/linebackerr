package matcher

import (
	"database/sql"
	"fmt"
)

// nflverseLookupStage attempts to find a match in the nflverse_games table.
//
// Lookup paths:
//  1. Exact GameDate + teams (existing regular-season path).
//  2. GameType + SeasonYear + teams (postseason-oriented path).
func nflverseLookupStage(db *sql.DB, match *Match) error {
	if match.NflverseID != "" {
		return nil
	}

	if match.GameDate != "" && match.HomeTeam != "" && match.AwayTeam != "" {
		query := `
			SELECT g.game_id,
			       CAST(g.season AS TEXT),
			       CAST(g.week AS TEXT),
			       g.game_type,
			       g.gameday,
			       away.abbr,
			       home.abbr
			FROM nflverse_games g
			LEFT JOIN nflverse_teams away ON away.id = g.away_team_id
			LEFT JOIN nflverse_teams home ON home.id = g.home_team_id
			WHERE g.gameday = ?
			  AND (
				(g.home_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?) AND g.away_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?))
				OR
				(g.home_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?) AND g.away_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?))
			  )
			LIMIT 1`

		result, err := scanNflverseGameResult(db.QueryRow(query, match.GameDate, match.HomeTeam, match.AwayTeam, match.AwayTeam, match.HomeTeam))
		if err == nil {
			applyNflverseGameResult(match, result)
			return nil
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("nflverse lookup failed: %w", err)
		}
	}

	if match.GameType == GameTypeSuperBowl {
		// Super Bowl lookups can still be handled by the dedicated Roman numeral stage.
		return nil
	}

	if match.GameType == "" || match.SeasonYear == "" || match.HomeTeam == "" || match.AwayTeam == "" {
		return ErrMissingFieldsForNflverseLookup
	}

	query := `
		SELECT g.game_id,
		       CAST(g.season AS TEXT),
		       CAST(g.week AS TEXT),
		       g.game_type,
		       g.gameday,
		       away.abbr,
		       home.abbr
		FROM nflverse_games g
		LEFT JOIN nflverse_teams away ON away.id = g.away_team_id
		LEFT JOIN nflverse_teams home ON home.id = g.home_team_id
		WHERE g.game_type = ?
		  AND CAST(g.season AS TEXT) = ?
		  AND (
			(g.home_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?) AND g.away_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?))
			OR
			(g.home_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?) AND g.away_team_id = (SELECT id FROM nflverse_teams WHERE abbr = ?))
		  )
		LIMIT 1`

	result, err := scanNflverseGameResult(db.QueryRow(query, string(match.GameType), match.SeasonYear, match.HomeTeam, match.AwayTeam, match.AwayTeam, match.HomeTeam))
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("nflverse lookup failed: %w", err)
	}

	applyNflverseGameResult(match, result)
	return nil
}
