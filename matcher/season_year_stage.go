package matcher

import "regexp"

var seasonYearStageRegex = regexp.MustCompile(`(?:^| )((?:19|20)\d{2})(?: |$)`)

// extractSeasonYearStage determines the NFL season year for the normalized
// working string. When a game date is already known, Jan/Feb games roll back to
// the previous season year and all other months use the game date year. When no
// game date is present, the stage extracts a standalone season year token from
// the working string and removes it before the next extraction stage runs.
func extractSeasonYearStage(working string, gameDate string) (seasonYear string, next string, ok bool) {
	if gameDate != "" {
		year := gameDate[:4]
		month := gameDate[5:7]
		if month == "01" || month == "02" {
			return decrementYear(year), working, true
		}
		return year, working, true
	}

	matchIndexes := seasonYearStageRegex.FindStringSubmatchIndex(working)
	if matchIndexes == nil {
		return "", working, false
	}

	seasonYear = working[matchIndexes[2]:matchIndexes[3]]
	next = normalizeForMatching(working[:matchIndexes[0]] + " " + working[matchIndexes[1]:])
	return seasonYear, next, true
}

func decrementYear(year string) string {
	if len(year) != 4 {
		return year
	}

	bytes := []byte(year)
	for i := len(bytes) - 1; i >= 0; i-- {
		if bytes[i] > '0' {
			bytes[i]--
			break
		}
		bytes[i] = '9'
	}

	return string(bytes)
}
