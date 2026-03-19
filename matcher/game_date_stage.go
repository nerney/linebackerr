package matcher

import "regexp"

var gameDateStageRegex = regexp.MustCompile(`(?:^| )((?:19|20)\d{2})(?: ([0-1]\d) ([0-3]\d)|([0-1]\d)([0-3]\d))(?: |$)`)

// extractGameDateStage pulls a supported game date from the normalized working
// string, standardizes it to YYYY-MM-DD, and removes the matched date before
// the next extraction stage runs.
func extractGameDateStage(working string) (gameDate string, next string, ok bool) {
	matchIndexes := gameDateStageRegex.FindStringSubmatchIndex(working)
	if matchIndexes == nil {
		return "", working, false
	}

	match := working[matchIndexes[0]:matchIndexes[1]]
	groups := gameDateStageRegex.FindStringSubmatch(match)
	if groups == nil {
		return "", working, false
	}

	month, day := groups[2], groups[3]
	if month == "" || day == "" {
		month, day = groups[4], groups[5]
	}

	gameDate = groups[1] + "-" + month + "-" + day
	next = normalizeForMatching(working[:matchIndexes[0]] + " " + working[matchIndexes[1]:])
	return gameDate, next, true
}
