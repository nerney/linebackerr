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

// ParseReleases takes a list of release strings and returns an array of strings
// containing either the extracted YEAR.POSTSEASONSUBSTR, a formatted DATE (YYYY-MM-DD),
// or the original release string if no match could be found.
func ParseReleases(releases []string) []string {
	results := make([]string, 0, len(releases))

	for _, release := range releases {
		tokens := tokenizeForMatching(release)
		if !hasNormalizedToken(tokens, "nfl") {
			results = append(results, release)
			continue
		}

		normalized := normalizeForMatching(release)

		if postseason, ok := extractPostseasonMatch(normalized); ok {
			results = append(results, postseason)
			continue
		}

		if date, ok := extractDateMatch(normalized); ok {
			results = append(results, date)
			continue
		}

		results = append(results, release)
	}

	return results
}
