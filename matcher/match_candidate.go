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
	GameType   GameType
	GameDate   string
	GameWeek   string
	SeasonYear string
	AwayTeam   string
	HomeTeam   string
}
