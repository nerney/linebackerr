package matcher

import "strings"

// teamAliasInventory is a matcher-focused alias set derived from nflverse team
// names/abbreviations/history where practical.
//
// Tradeoffs / ambiguity notes:
//   - We intentionally skip bare city/location aliases that collide across teams,
//     such as "new york", "los angeles", and plain "la".
//   - We keep broad mascot aliases like "giants", "jets", "raiders", etc.
//     because they are highly representative in release names and currently do
//     not collide within the NFL.
//   - We include historical franchise names/abbreviations where nflverse team
//     history exposes relocations or naming changes (for example STL/LA Rams,
//     SD/LAC Chargers, OAK/LV Raiders, WFT/Commanders).
//   - We prefer current nflverse-style canonical abbreviations as the matcher
//     output keys: LA, LAC, LV, JAX, WAS, etc.
var teamAliasInventory = map[string][]string{
	"ARI": {"ari", "arizona", "arizona cardinals", "cardinals"},
	"ATL": {"atl", "atlanta", "atlanta falcons", "falcons"},
	"BAL": {"bal", "baltimore", "baltimore ravens", "ravens"},
	"BUF": {"buf", "buffalo", "buffalo bills", "bills"},
	"CAR": {"car", "carolina", "carolina panthers", "panthers"},
	"CHI": {"chi", "chicago", "chicago bears", "bears"},
	"CIN": {"cin", "cincinnati", "cincinnati bengals", "bengals"},
	"CLE": {"cle", "cleveland", "cleveland browns", "browns"},
	"DAL": {"dal", "dallas", "dallas cowboys", "cowboys"},
	"DEN": {"den", "denver", "denver broncos", "broncos"},
	"DET": {"det", "detroit", "detroit lions", "lions"},
	"GB":  {"gb", "green bay", "green bay packers", "packers"},
	"HOU": {"hou", "houston", "houston texans", "texans"},
	"IND": {"ind", "indianapolis", "indianapolis colts", "colts"},
	"JAX": {"jax", "jac", "jacksonville", "jacksonville jaguars", "jaguars"},
	"KC":  {"kc", "kan", "kansas city", "kansas city chiefs", "chiefs"},
	"LA":  {"la rams", "los angeles rams", "st louis rams", "st. louis rams", "st louis", "stl", "rams"},
	"LAC": {"lac", "la chargers", "los angeles chargers", "san diego chargers", "san diego", "sd", "sdg", "chargers"},
	"LV":  {"lv", "las vegas raiders", "oakland raiders", "oakland", "oak", "raiders"},
	"MIA": {"mia", "miami", "miami dolphins", "dolphins"},
	"MIN": {"min", "minnesota", "minnesota vikings", "vikings"},
	"NE":  {"ne", "new england", "new england patriots", "patriots"},
	"NO":  {"no", "new orleans", "new orleans saints", "saints"},
	"NYG": {"nyg", "new york giants", "ny giants", "giants"},
	"NYJ": {"nyj", "new york jets", "ny jets", "jets"},
	"PHI": {"phi", "philadelphia", "philadelphia eagles", "eagles"},
	"PIT": {"pit", "pittsburgh", "pittsburgh steelers", "steelers"},
	"SEA": {"sea", "seattle", "seattle seahawks", "seahawks"},
	"SF":  {"sf", "san francisco", "san francisco 49ers", "49ers", "forty niners"},
	"TB":  {"tb", "tampa bay", "tampa bay buccaneers", "bucs", "buccaneers"},
	"TEN": {"ten", "tennessee", "tennessee titans", "titans"},
	"WAS": {"was", "wsh", "washington", "washington commanders", "washington football team", "washington redskins", "commanders", "redskins"},
}

var teamAliasLookup = buildTeamAliasLookup(teamAliasInventory)

type teamAliasPattern struct {
	team   string
	alias  string
	tokens []string
}

func buildTeamAliasLookup(inventory map[string][]string) []teamAliasPattern {
	patterns := make([]teamAliasPattern, 0)
	for team, aliases := range inventory {
		seen := make(map[string]struct{}, len(aliases))
		for _, alias := range aliases {
			normalized := NormalizeForMatch(alias)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			patterns = append(patterns, teamAliasPattern{
				team:   team,
				alias:  normalized,
				tokens: strings.Fields(normalized),
			})
		}
	}

	return patterns
}

// TeamAliases returns a copy of the matcher alias inventory keyed by canonical
// nflverse-style team abbreviation.
func TeamAliases() map[string][]string {
	out := make(map[string][]string, len(teamAliasInventory))
	for team, aliases := range teamAliasInventory {
		out[team] = append([]string(nil), aliases...)
	}
	return out
}

// matchTeamAlias finds the longest normalized alias span inside the provided
// token slice and returns the canonical team abbreviation plus the alias span.
// When multiple aliases share the same length, the earliest token span wins.
func matchTeamAlias(tokens []string) (team string, start int, end int, found bool) {
	bestLen := 0
	bestStart := len(tokens)

	for _, pattern := range teamAliasLookup {
		plen := len(pattern.tokens)
		if plen == 0 || plen > len(tokens) {
			continue
		}

		for i := 0; i+plen <= len(tokens); i++ {
			matched := true
			for j := 0; j < plen; j++ {
				if tokens[i+j] != pattern.tokens[j] {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}

			if plen > bestLen || (plen == bestLen && i < bestStart) {
				team = pattern.team
				start = i
				end = i + plen
				bestLen = plen
				bestStart = i
				found = true
			}
		}
	}

	return team, start, end, found
}
