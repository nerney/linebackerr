package matcher

import (
	"regexp"
	"strings"
)

var gameWeekNumericStageRegex = regexp.MustCompile(`(?:^| )((?:week|wk) ?(\d{1,2})|w(\d{1,2}))(?: |$)`)

// extractGameWeekStage determines the game week from the normalized working
// string. It supports numeric week markers like "week 2", "week2", "wk 18",
// "wk18", "w7", and "w18". When the game type is Super Bowl, it also
// extracts a valid Roman numeral token that follows a Super Bowl alias and
// separator. Any matched week token is removed from the working string before
// the next stage runs.
func extractGameWeekStage(working string, gameType GameType) (gameWeek string, next string, ok bool) {
	matchIndexes := gameWeekNumericStageRegex.FindStringSubmatchIndex(working)
	if matchIndexes != nil {
		if matchIndexes[4] != -1 {
			gameWeek = working[matchIndexes[4]:matchIndexes[5]]
		} else {
			gameWeek = working[matchIndexes[6]:matchIndexes[7]]
		}
		next = normalizeForMatching(working[:matchIndexes[0]] + " " + working[matchIndexes[1]:])
		return gameWeek, next, true
	}

	if gameType != GameTypeSuperBowl {
		return "", working, false
	}

	tokens := tokenizeForMatching(working)
	for i, token := range tokens {
		if !isValidRomanNumeralToken(token) {
			continue
		}
		if !hasSuperBowlAliasBefore(tokens, i) {
			continue
		}

		nextTokens := append([]string{}, tokens[:i]...)
		nextTokens = append(nextTokens, tokens[i+1:]...)
		return strings.ToUpper(token), strings.Join(nextTokens, " "), true
	}

	return "", working, false
}

func hasSuperBowlAliasBefore(tokens []string, idx int) bool {
	if idx <= 0 {
		return false
	}
	if tokens[idx-1] == "sb" || tokens[idx-1] == "superbowl" {
		return true
	}
	return idx >= 2 && tokens[idx-2] == "super" && tokens[idx-1] == "bowl"
}

func isValidRomanNumeralToken(token string) bool {
	if token == "" {
		return false
	}

	values := map[rune]int{
		'i': 1,
		'v': 5,
		'x': 10,
		'l': 50,
		'c': 100,
		'd': 500,
		'm': 1000,
	}

	total := 0
	prev := 0
	for i := len(token) - 1; i >= 0; i-- {
		value, ok := values[rune(token[i])]
		if !ok {
			return false
		}
		if value < prev {
			total -= value
		} else {
			total += value
			prev = value
		}
	}

	return total > 0 && toRomanNumeral(total) == strings.ToUpper(token)
}

func toRomanNumeral(value int) string {
	if value <= 0 {
		return ""
	}

	numerals := []struct {
		value int
		sym   string
	}{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}

	var builder strings.Builder
	for _, numeral := range numerals {
		for value >= numeral.value {
			builder.WriteString(numeral.sym)
			value -= numeral.value
		}
	}

	return builder.String()
}
