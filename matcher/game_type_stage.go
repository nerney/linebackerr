package matcher

// extractGameTypeStage determines the NFL game type from the normalized
// working string. This stage is intentionally non-mutating: it returns the
// original working string unchanged regardless of whether a game type alias is
// found.
func extractGameTypeStage(working string) (gameType GameType, next string, ok bool) {
	tokens := tokenizeForMatching(working)
	for i := 0; i < len(tokens); i++ {
		switch {
		case tokens[i] == "sb":
			return GameTypeSuperBowl, working, true
		case tokens[i] == "superbowl":
			return GameTypeSuperBowl, working, true
		case tokens[i] == "super" && i+1 < len(tokens) && tokens[i+1] == "bowl":
			return GameTypeSuperBowl, working, true
		case tokens[i] == "conference":
			return GameTypeConference, working, true
		case tokens[i] == "con":
			return GameTypeConference, working, true
		case tokens[i] == "championship":
			return GameTypeConference, working, true
		case tokens[i] == "div":
			return GameTypeDivisional, working, true
		case tokens[i] == "division":
			return GameTypeDivisional, working, true
		case tokens[i] == "divisional":
			return GameTypeDivisional, working, true
		case tokens[i] == "wc":
			return GameTypeWildcard, working, true
		case tokens[i] == "wildcard":
			return GameTypeWildcard, working, true
		case tokens[i] == "wild" && i+1 < len(tokens) && tokens[i+1] == "card":
			return GameTypeWildcard, working, true
		}
	}

	return GameTypeRegularSeason, working, false
}
