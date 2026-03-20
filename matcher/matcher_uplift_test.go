package matcher

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAndTokenHelpers(t *testing.T) {
	assert.Equal(t, "nfl chiefs bills 2024", NormalizeForMatch("NFL: Chiefs@Bills (2024)!"))
	assert.Equal(t, []string{"nfl", "chiefs", "bills"}, TokenizeForMatch("NFL Chiefs -- Bills"))
	assert.True(t, hasNormalizedToken([]string{"n", "f", "l"}, "nfl"))
	assert.True(t, hasNormalizedToken([]string{"nfl", "week", "1"}, "nfl"))
	assert.False(t, hasNormalizedToken([]string{"nba"}, "nfl"))
}

func TestDateAndSeasonStages(t *testing.T) {
	date, next, ok := extractGameDateStage("nfl 2024 09 08 bills patriots")
	require.True(t, ok)
	assert.Equal(t, "2024-09-08", date)
	assert.Equal(t, "nfl bills patriots", next)

	date, _, ok = extractGameDateStage("nfl 20240908 chiefs bills")
	require.True(t, ok)
	assert.Equal(t, "2024-09-08", date)

	year, next, ok := extractSeasonYearStage("nfl 2023 conference chiefs ravens", "")
	require.True(t, ok)
	assert.Equal(t, "2023", year)
	assert.Equal(t, "nfl conference chiefs ravens", next)

	year, _, ok = extractSeasonYearStage("anything", "2024-01-21")
	require.True(t, ok)
	assert.Equal(t, "2023", year)
	assert.Equal(t, "2023", decrementYear("2024"))
	assert.Equal(t, "0999", decrementYear("1000"))
}

func TestGameTypeAndWeekStages(t *testing.T) {
	gt, next, ok := extractGameTypeStage("nfl super bowl chiefs niners")
	require.True(t, ok)
	assert.Equal(t, GameTypeSuperBowl, gt)
	assert.Equal(t, "nfl super bowl chiefs niners", next)

	gt, _, ok = extractGameTypeStage("nfl week 1 chiefs bills")
	assert.False(t, ok)
	assert.Equal(t, GameTypeRegularSeason, gt)

	week, next, ok := extractGameWeekStage("nfl week 18 bills patriots", GameTypeRegularSeason)
	require.True(t, ok)
	assert.Equal(t, "18", week)
	assert.Equal(t, "nfl bills patriots", next)

	week, next, ok = extractGameWeekStage("nfl super bowl lviii chiefs niners", GameTypeSuperBowl)
	require.True(t, ok)
	assert.Equal(t, "LVIII", week)
	assert.Equal(t, "nfl super bowl chiefs niners", next)

	assert.True(t, hasSuperBowlAliasBefore([]string{"super", "bowl", "lviii"}, 2))
	assert.True(t, isValidRomanNumeralToken("lviii"))
	assert.False(t, isValidRomanNumeralToken("IIX"))
	assert.Equal(t, "LVIII", toRomanNumeral(58))
}

func TestTeamAliasStages(t *testing.T) {
	aliases := TeamAliases()
	require.Contains(t, aliases, "KC")
	require.NotEmpty(t, aliases["KC"])

	team, start, end, found := matchTeamAlias([]string{"kansas", "city", "chiefs", "vs", "bills"})
	require.True(t, found)
	assert.Equal(t, "KC", team)
	assert.Equal(t, 0, start)
	assert.Equal(t, 3, end)

	err := detectAmbiguousTeamAlias([]string{"los", "angeles"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAmbiguousTeamAlias)
	var amb *AmbiguousTeamAliasError
	require.True(t, errors.As(err, &amb))
	assert.Equal(t, "los angeles", amb.Alias)

	away, home, _, ok, err := extractTeamsStage("chiefs bills")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "KC", away)
	assert.Equal(t, "BUF", home)
}

func TestPipelineAndParseReleases(t *testing.T) {
	cand := Pipeline("NFL 2024-09-08 week 1 BUF @ NE")
	assert.Equal(t, "2024-09-08", cand.GameDate)
	assert.Equal(t, "2024", cand.SeasonYear)
	assert.Equal(t, "1", cand.GameWeek)
	assert.Equal(t, "BUF", cand.AwayTeam)
	assert.Equal(t, "NE", cand.HomeTeam)

	results := ParseReleases([]string{
		"NFL 2024-09-08 Week 1 BUF@NE",
		"NFL 2023 Conference Chiefs vs Ravens",
		"Some.Non.NFL.Release",
	})
	require.Len(t, results, 3)
	assert.Equal(t, "2024-09-08", results[0])
	assert.Equal(t, "2023.Championship", results[1])
	assert.Equal(t, "Some.Non.NFL.Release", results[2])
}

func TestPostseasonExtractionHelpers(t *testing.T) {
	match, ok := extractPostseasonMatch("nfl 2023 super bowl chiefs niners")
	require.True(t, ok)
	assert.Equal(t, "2023.Super.Bowl", match)

	match, ok = extractPostseasonMatch("wild card nfl 2021")
	require.True(t, ok)
	assert.Equal(t, "2021.Wildcard", match)

	date, ok := extractDateMatch("nfl 2024 10 14 jets bills")
	require.True(t, ok)
	assert.Equal(t, "2024-10-14", date)
}
