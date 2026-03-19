package matcher

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
