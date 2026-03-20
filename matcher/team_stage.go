package matcher

// extractTeamsStage determines away/home teams from the normalized working
// string by running the existing team matcher twice. The first matched team is
// treated as the away team and the second matched team as the home team. This
// stage is intentionally non-mutating: it returns the original working string
// unchanged regardless of whether two teams are found.
//
// If the remaining input contains a bare city/location alias shared by multiple
// franchises (for example "los angeles" or "new york"), the stage returns an
// explicit ambiguity error instead of silently choosing one team.
func extractTeamsStage(working string) (awayTeam string, homeTeam string, next string, ok bool, err error) {
	tokens := tokenizeForMatching(working)
	firstTeam, start, end, found := matchTeamAlias(tokens)
	if !found {
		if err := detectAmbiguousTeamAlias(tokens); err != nil {
			return "", "", working, false, err
		}
		return "", "", working, false, nil
	}

	remaining := append([]string{}, tokens[:start]...)
	remaining = append(remaining, tokens[end:]...)

	secondTeam, _, _, found := matchTeamAlias(remaining)
	if !found {
		if err := detectAmbiguousTeamAlias(remaining); err != nil {
			return "", "", working, false, err
		}
		return "", "", working, false, nil
	}

	return firstTeam, secondTeam, working, true, nil
}
