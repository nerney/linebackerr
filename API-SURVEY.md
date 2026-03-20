# Linebackerr Package / Structure Report

> Snapshot report captured before the later `server` package refactor. This reflects the repo state at the time the report was generated, not necessarily the exact current tree.

## `package main`
**Files:** `main.go`

- **Types/structs**
  - none

- **Vars/constants**
  - none

- **Functions**
  - `func healthHandler(w http.ResponseWriter, r *http.Request)`
  - `func main()`

- **Methods**
  - none

### `main.go` usage summary
**Internal packages used**
- `linebackerr/db`
- `linebackerr/nflverse`
- `linebackerr/sportarr`

**Stdlib used**
- `fmt`
- `log`
- `net/http`

**Functions/types it calls or uses**
- `db.Init() error`
- `db.NeedsSync(module string) bool`
- `db.UpdateSync(module string) error`
- `db.DB *sql.DB`
- `nflverse.Init(db *sql.DB) error`
- `sportarr.LoadSeasons(db *sql.DB) error`
- `sportarr.LoadTeams(db *sql.DB) error`
- `http.ResponseWriter`
- `*http.Request`
- `http.StatusOK`
- `http.HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))`
- `http.ListenAndServe(addr string, handler http.Handler) error`
- `fmt.Fprintf(...)`
- `fmt.Println(...)`
- `fmt.Printf(...)`
- `log.Fatalf(...)`
- `log.Printf(...)`

---

## `package db`
**Files:** `db/db.go`

- **Types/structs**
  - none

- **Vars/constants**
  - `const dataDir = "/config"`
  - `var DB *sql.DB`

- **Functions**
  - `func Init() error`
  - `func initSchema() error`
  - `func NeedsSync(module string) bool`
  - `func UpdateSync(module string) error`

- **Methods**
  - none

---

## `package nflverse`
**Files:** `nflverse/nflverse.go`

- **Types/structs**
  - none

- **Vars/constants**
  - `const teamsURL = "https://raw.githubusercontent.com/nflverse/nfldata/refs/heads/master/data/teams.csv"`
  - `const gamesURL = "https://raw.githubusercontent.com/nflverse/nfldata/refs/heads/master/data/games.csv"`
  - `const dataDir = "/config"`

- **Functions**
  - `func Init(db *sql.DB) error`
  - `func initDB(db *sql.DB) error`
  - `func loadTeams(db *sql.DB, path string) (map[string]string, error)`
  - `func loadGames(db *sql.DB, path string, teamMap map[string]string) error`
  - `func downloadFile(url, filepath string) error`

- **Methods**
  - none

---

## `package sportarr`
**Files:** `sportarr/sportarr.go`

- **Types/structs**
  - `type SportarrSeasons struct { Seasons []struct { PosterURL string \`json:"poster_url"\` } \`json:"seasons"\` }`
  - `type SportarrSearch struct { Data struct { Search []struct { IdTeam interface{} \`json:"idTeam"\` } \`json:"search"\` } \`json:"data"\` }`
  - `type SportarrLookup struct { Data struct { Lookup []struct { StrLogo string; StrBadge string; StrBanner string; StrFanart1 string; StrDescriptionEN string } \`json:"lookup"\` } \`json:"data"\` }`

- **Vars/constants**
  - none

- **Functions**
  - `func LoadSeasons(db *sql.DB) error`
  - `func LoadTeams(db *sql.DB) error`

- **Methods**
  - none

---

## `package matcher`
**Files:**  
`matcher/match_candidate.go`  
`matcher/matcher.go`  
`matcher/normalize.go`  
`matcher/game_date_stage.go`  
`matcher/game_type_stage.go`  
`matcher/game_week_stage.go`  
`matcher/season_year_stage.go`  
`matcher/team_aliases.go`  
`matcher/team_stage.go`

- **Types/structs**
  - `type GameType string`
  - `type MatchCandidate struct { OriginalInput string; GameType GameType; GameDate string; GameWeek string; SeasonYear string; AwayTeam string; HomeTeam string }`
  - `type Match struct { OriginalInput string; GameType GameType; GameDate string; GameWeek string; SeasonYear string; AwayTeam string; HomeTeam string; NflverseID string; Error error }`
  - `type AmbiguousTeamAliasError struct { Alias string; Teams []string }`
  - `type teamAliasPattern struct { team string; alias string; tokens []string }`

- **Vars/constants**
  - `const ( GameTypeSuperBowl GameType = "SB"; GameTypeConference GameType = "CON"; GameTypeDivisional GameType = "DIV"; GameTypeWildcard GameType = "WC"; GameTypeRegularSeason GameType = "RS" )`
  - `var nonAlphanumericRunRegex = regexp.MustCompile(...)`
  - `var gameDateStageRegex = regexp.MustCompile(...)`
  - `var gameWeekNumericStageRegex = regexp.MustCompile(...)`
  - `var seasonYearStageRegex = regexp.MustCompile(...)`
  - `var ErrAmbiguousTeamAlias = errors.New("ambiguous team alias")`
  - `var teamAliasInventory = map[string][]string{...}`
  - `var ambiguousTeamAliases = map[string][]string{...}`
  - `var teamAliasLookup = buildTeamAliasLookup(teamAliasInventory)`

- **Functions**
  - `func ParseReleases(releases []string) []string`
  - `func NormalizeForMatch(input string) string`
  - `func TokenizeForMatch(input string) []string`
  - `func normalizeForMatching(input string) string`
  - `func tokenizeForMatching(input string) []string`
  - `func hasNormalizedToken(tokens []string, want string) bool`
  - `func canonicalPostseasonLabel(raw string) (string, bool)`
  - `func extractPostseasonMatch(normalized string) (string, bool)`
  - `func postseasonLabelFromTokens(tokens []string) (string, bool)`
  - `func isSeasonYearToken(token string) bool`
  - `func extractDateMatch(normalized string) (string, bool)`
  - `func extractGameDateStage(working string) (gameDate string, next string, ok bool)`
  - `func extractGameTypeStage(working string) (gameType GameType, next string, ok bool)`
  - `func extractGameWeekStage(working string, gameType GameType) (gameWeek string, next string, ok bool)`
  - `func hasSuperBowlAliasBefore(tokens []string, idx int) bool`
  - `func isValidRomanNumeralToken(token string) bool`
  - `func toRomanNumeral(value int) string`
  - `func extractSeasonYearStage(working string, gameDate string) (seasonYear string, next string, ok bool)`
  - `func decrementYear(year string) string`
  - `func buildTeamAliasLookup(inventory map[string][]string) []teamAliasPattern`
  - `func TeamAliases() map[string][]string`
  - `func matchTeamAlias(tokens []string) (team string, start int, end int, found bool)`
  - `func detectAmbiguousTeamAlias(tokens []string) error`
  - `func extractTeamsStage(working string) (awayTeam string, homeTeam string, next string, ok bool, err error)`

- **Methods**
  - `func (e *AmbiguousTeamAliasError) Error() string`
  - `func (e *AmbiguousTeamAliasError) Unwrap() error`
