package matcher

// GameType identifies the NFL game stage for matching.
type GameType string

const (
	GameTypeSuperBowl     GameType = "SB"
	GameTypeConference    GameType = "CON"
	GameTypeDivisional    GameType = "DIV"
	GameTypeWildcard      GameType = "WC"
	GameTypeRegularSeason GameType = "RS"
)

// MatchCandidate is the normalized set of fields the matcher can use to
// identify a game before resolving it to a final match record.
type MatchCandidate struct {
	OriginalInput string
	GameType      GameType
	GameDate      string
	GameWeek      string
	SeasonYear    string
	AwayTeam      string
	HomeTeam      string
}

// Validate executes the validation pipeline to resolve the MatchCandidate
// into a final Match record.
func (mc MatchCandidate) Validate() Match {
	return mc.validatePipeline()
}

func (mc MatchCandidate) validatePipeline() Match {
	// For now, this is just a pass-through that copies fields.
	// Future tasks will add stages to this pipeline.
	return Match{
		OriginalInput: mc.OriginalInput,
		GameType:      mc.GameType,
		GameDate:      mc.GameDate,
		GameWeek:      mc.GameWeek,
		SeasonYear:    mc.SeasonYear,
		AwayTeam:      mc.AwayTeam,
		HomeTeam:      mc.HomeTeam,
	}
}

// Match is a resolved game match record. It carries all candidate fields plus
// the resolved nflverse identifier and any match error encountered.
type Match struct {
	OriginalInput string
	GameType      GameType
	GameDate      string
	GameWeek      string
	SeasonYear    string
	AwayTeam      string
	HomeTeam      string
	NflverseID    string
	Error         error
}
