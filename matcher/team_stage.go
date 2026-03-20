package matcher

// extractTeamsStage determines away/home teams from the normalized working
// string by running the existing team matcher twice. The first matched team is
// treated as the away team and the second matched team as the home team. This
// stage is intentionally non-mutating: it returns the original working string
// unchanged regardless of whether two teams are found.
func extractTeamsStage(working string) (awayTeam string, homeTeam string, next string, ok bool) {
	tokens := tokenizeForMatching(working)
	firstTeam, start, end, found := matchTeamAlias(tokens)
	if !found {
		return "", "", working, false
	}

	remaining := append([]string{}, tokens[:start]...)
	remaining = append(remaining, tokens[end:]...)

	secondTeam, _, _, found := matchTeamAlias(remaining)
	if !found {
		return "", "", working, false
	}

	return firstTeam, secondTeam, working, true
}
