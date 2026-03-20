package matcher

// Pipeline executes the extraction stages progressively to build a MatchCandidate
// from a raw input string.
func Pipeline(input string) MatchCandidate {
	candidate := MatchCandidate{
		OriginalInput: input,
	}

	working := normalizeForMatching(input)

	// 1. GameDate
	date, next, _ := extractGameDateStage(working)
	candidate.GameDate = date
	working = next

	// 2. SeasonYear
	year, next, _ := extractSeasonYearStage(working, candidate.GameDate)
	candidate.SeasonYear = year
	working = next

	// 3. GameType
	gameType, next, _ := extractGameTypeStage(working)
	candidate.GameType = gameType
	working = next

	// 4. GameWeek
	week, next, _ := extractGameWeekStage(working, candidate.GameType)
	candidate.GameWeek = week
	working = next

	// 5. Away/Home team extraction
	away, home, _, _, _ := extractTeamsStage(working)
	candidate.AwayTeam = away
	candidate.HomeTeam = home

	return candidate
}

// ParseReleases takes a list of release strings and uses the Pipeline to extract
// MatchCandidate information for each release. It returns a list of formatted strings:
// YEAR.POSTSEASONSUBSTR for postseason games, formatted DATE (YYYY-MM-DD) for others,
// or the original release string if no NFL match or date/season could be identified.
func ParseReleases(releases []string) []string {
	results := make([]string, 0, len(releases))

	for _, release := range releases {
		tokens := tokenizeForMatching(release)
		if !hasNormalizedToken(tokens, "nfl") {
			results = append(results, release)
			continue
		}

		candidate := Pipeline(release)

		// 1. Date match (highest precedence for regular season or when date is explicit)
		// Note: we prefer date for regular season games. Postseason games often prefer the Season.Type format.
		if candidate.GameType == GameTypeRegularSeason && candidate.GameDate != "" {
			results = append(results, candidate.GameDate)
			continue
		}

		// 2. Postseason games (Year.Type)
		if candidate.GameType != GameTypeRegularSeason && candidate.SeasonYear != "" {
			var label string
			switch candidate.GameType {
			case GameTypeSuperBowl:
				label = "Super.Bowl"
			case GameTypeConference:
				label = "Championship"
			case GameTypeDivisional:
				label = "Divisional"
			case GameTypeWildcard:
				label = "Wildcard"
			}
			if label != "" {
				results = append(results, candidate.SeasonYear+"."+label)
				continue
			}
		}

		// 3. Fallback to date if not a postseason game with a season year
		if candidate.GameDate != "" {
			results = append(results, candidate.GameDate)
			continue
		}

		// Fallback to original
		results = append(results, release)
	}

	return results
}
