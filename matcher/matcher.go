package matcher

import (
	"regexp"
	"strings"
)

// ParseReleases takes a list of release strings and returns an array of strings
// containing either the extracted YEAR.POSTSEASONSUBSTR, a formatted DATE (YYYY-MM-DD),
// or the original release string if no match could be found.
func ParseReleases(releases []string) []string {
	var results []string

	// Postseason regex: looking for a year (19xx or 20xx) followed by a separator and then the postseason string
	// or the postseason string followed by a separator and a year.
	// We'll use a few regexes to capture the variations.
	postseasonPattern := `(?i)(Super\.Bowl|Divisional|Wildcard|Championship)`
	yearPattern := `((?:19|20)\d{2})`
	sepPattern := `[\s\-\.]`

	// Matches: Year - Substring OR Substring - Year
	postseasonRegex := regexp.MustCompile(yearPattern + sepPattern + `*` + postseasonPattern + `|` + postseasonPattern + sepPattern + `*` + yearPattern)

	// Date regex: YYYY-MM-DD or YYYY.MM.DD or YYYY MM DD or YYYYMMDD
	// We capture the groups to format them later.
	dateRegex := regexp.MustCompile(`((?:19|20)\d{2})[\s\-\.]?([0-1]\d)[\s\-\.]?([0-3]\d)`)

	for _, release := range releases {
		// Must contain "NFL" (case sensitive or insensitive? Let's assume case-insensitive for safety, but prompt said "NFL". Let's use strings.Contains upper)
		if !strings.Contains(strings.ToUpper(release), "NFL") {
			results = append(results, release)
			continue
		}

		// 1. Check for Postseason substrings
		if match := postseasonRegex.FindStringSubmatch(release); match != nil {
			// FindStringSubmatch returns the full match, then the capture groups.
			// Because of the OR in the regex, either group 1 & 2 are filled, or 3 & 4 are filled.
			var year, sub string
			if match[1] != "" {
				year = match[1]
				sub = match[2]
			} else {
				sub = match[3]
				year = match[4]
			}
			
			// Normalize substring to title case for consistency if desired, or keep as matched.
			// Let's keep as matched but make sure we have YEAR.SUBSTRING
			results = append(results, year+"."+sub)
			continue
		}

		// 2. Check for Full Date
		if match := dateRegex.FindStringSubmatch(release); match != nil {
			// match[1] = YYYY, match[2] = MM, match[3] = DD
			dateStr := match[1] + "-" + match[2] + "-" + match[3]
			results = append(results, dateStr)
			continue
		}

		// 3. Fallback to original
		results = append(results, release)
	}

	return results
}
