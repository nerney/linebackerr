package matcher

import "database/sql"

type nflverseGameResult struct {
	GameID     string
	SeasonYear string
	GameWeek   string
	GameType   string
	GameDate   string
	AwayTeam   string
	HomeTeam   string
}

func scanNflverseGameResult(scanner interface{ Scan(dest ...any) error }) (nflverseGameResult, error) {
	var result nflverseGameResult
	var seasonYear sql.NullString
	var gameWeek sql.NullString
	var gameType sql.NullString
	var gameDate sql.NullString
	var awayTeam sql.NullString
	var homeTeam sql.NullString

	err := scanner.Scan(
		&result.GameID,
		&seasonYear,
		&gameWeek,
		&gameType,
		&gameDate,
		&awayTeam,
		&homeTeam,
	)
	if err != nil {
		return nflverseGameResult{}, err
	}

	if seasonYear.Valid {
		result.SeasonYear = seasonYear.String
	}
	if gameWeek.Valid {
		result.GameWeek = gameWeek.String
	}
	if gameType.Valid {
		result.GameType = gameType.String
	}
	if gameDate.Valid {
		result.GameDate = gameDate.String
	}
	if awayTeam.Valid {
		result.AwayTeam = awayTeam.String
	}
	if homeTeam.Valid {
		result.HomeTeam = homeTeam.String
	}

	return result, nil
}

func applyNflverseGameResult(match *Match, result nflverseGameResult) {
	match.NflverseID = result.GameID
	if result.GameType != "" {
		match.GameType = GameType(result.GameType)
	}
	if result.SeasonYear != "" {
		match.SeasonYear = result.SeasonYear
	}
	if result.GameWeek != "" {
		match.GameWeek = result.GameWeek
	}
	if result.GameDate != "" {
		match.GameDate = result.GameDate
	}
	if result.AwayTeam != "" {
		match.AwayTeam = result.AwayTeam
	}
	if result.HomeTeam != "" {
		match.HomeTeam = result.HomeTeam
	}
}
