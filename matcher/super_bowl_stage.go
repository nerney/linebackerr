package matcher

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var romanNumeralRegex = regexp.MustCompile(`\b[IVXLCDM]+\b`)

// superBowlLookupStage attempts to find a match for Super Bowls using a Roman numeral
// signal in GameWeek when GameType is SB.
func superBowlLookupStage(db *sql.DB, match *Match) error {
	if match.GameType != GameTypeSuperBowl || match.GameWeek == "" {
		return nil
	}

	rawRoman := romanNumeralRegex.FindString(match.GameWeek)
	if rawRoman == "" || !isValidRomanNumeralToken(strings.ToLower(rawRoman)) {
		return nil
	}
	roman := strings.ToUpper(rawRoman)

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
		WHERE g.game_type = 'SB'
		  AND g.game_id LIKE ?
		LIMIT 1`

	result, err := scanNflverseGameResult(db.QueryRow(query, "%"+roman+"%"))
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("super bowl lookup failed: %w", err)
	}

	applyNflverseGameResult(match, result)
	return nil
}
