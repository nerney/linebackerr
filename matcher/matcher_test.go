package matcher

import (
	"errors"
	"reflect"
	"testing"
)

func TestMatchCandidateFields(t *testing.T) {
	candidate := MatchCandidate{
		OriginalInput: "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv",
		GameType:      GameTypeRegularSeason,
		GameDate:      "2021-09-19",
		GameWeek:      "2",
		SeasonYear:    "2021",
		AwayTeam:      "NE",
		HomeTeam:      "NYJ",
	}

	if candidate.OriginalInput != "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv" {
		t.Fatalf("expected original input NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv, got %q", candidate.OriginalInput)
	}
	if candidate.GameType != GameTypeRegularSeason {
		t.Fatalf("expected game type %q, got %q", GameTypeRegularSeason, candidate.GameType)
	}
	if candidate.GameDate != "2021-09-19" {
		t.Fatalf("expected game date 2021-09-19, got %q", candidate.GameDate)
	}
	if candidate.GameWeek != "2" {
		t.Fatalf("expected game week 2, got %q", candidate.GameWeek)
	}
	if candidate.SeasonYear != "2021" {
		t.Fatalf("expected season year 2021, got %q", candidate.SeasonYear)
	}
	if candidate.AwayTeam != "NE" {
		t.Fatalf("expected away team NE, got %q", candidate.AwayTeam)
	}
	if candidate.HomeTeam != "NYJ" {
		t.Fatalf("expected home team NYJ, got %q", candidate.HomeTeam)
	}
}

func TestMatchFields(t *testing.T) {
	matchErr := errors.New("no nflverse match")
	match := Match{
		OriginalInput: "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv",
		GameType:      GameTypeRegularSeason,
		GameDate:      "2021-09-19",
		GameWeek:      "2",
		SeasonYear:    "2021",
		AwayTeam:      "NE",
		HomeTeam:      "NYJ",
		NflverseID:    "2021_02_NE_NYJ",
		Error:         matchErr,
	}

	if match.OriginalInput != "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv" {
		t.Fatalf("expected original input NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv, got %q", match.OriginalInput)
	}
	if match.GameType != GameTypeRegularSeason {
		t.Fatalf("expected game type %q, got %q", GameTypeRegularSeason, match.GameType)
	}
	if match.GameDate != "2021-09-19" {
		t.Fatalf("expected game date 2021-09-19, got %q", match.GameDate)
	}
	if match.GameWeek != "2" {
		t.Fatalf("expected game week 2, got %q", match.GameWeek)
	}
	if match.SeasonYear != "2021" {
		t.Fatalf("expected season year 2021, got %q", match.SeasonYear)
	}
	if match.AwayTeam != "NE" {
		t.Fatalf("expected away team NE, got %q", match.AwayTeam)
	}
	if match.HomeTeam != "NYJ" {
		t.Fatalf("expected home team NYJ, got %q", match.HomeTeam)
	}
	if match.NflverseID != "2021_02_NE_NYJ" {
		t.Fatalf("expected nflverse ID 2021_02_NE_NYJ, got %q", match.NflverseID)
	}
	if !errors.Is(match.Error, matchErr) {
		t.Fatalf("expected match error %v, got %v", matchErr, match.Error)
	}
}

func TestNormalizeForMatching(t *testing.T) {
	input := " NFL---2018..Super__Bowl@@Patriots\tvs\nRams "
	got := normalizeForMatching(input)
	want := "nfl 2018 super bowl patriots vs rams"

	if got != want {
		t.Fatalf("expected normalized string %q, got %q", want, got)
	}
}

func TestTokenizeForMatching(t *testing.T) {
	got := tokenizeForMatching("Patriots.vs---Jets")
	want := []string{"patriots", "vs", "jets"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected tokens %v, got %v", want, got)
	}
}

func TestHasNormalizedToken(t *testing.T) {
	tokens := tokenizeForMatching("N.F.L. Films")
	if !hasNormalizedToken(tokens, "nfl") {
		t.Fatalf("expected normalized NFL token to be found in %v", tokens)
	}
	if hasNormalizedToken(tokens, "nba") {
		t.Fatalf("did not expect NBA token to be found in %v", tokens)
	}
}

func TestCanonicalPostseasonLabel(t *testing.T) {
	tests := map[string]string{
		"Super.Bowl":   "Super.Bowl",
		"SUPER BOWL":   "Super.Bowl",
		"divisional":   "Divisional",
		"Wild-Card":    "Wildcard",
		"Championship": "Championship",
	}

	for input, want := range tests {
		got, ok := canonicalPostseasonLabel(input)
		if want == "" {
			if ok {
				t.Fatalf("expected %q to be unsupported, got %q", input, got)
			}
			continue
		}
		if !ok {
			t.Fatalf("expected %q to normalize successfully", input)
		}
		if got != want {
			t.Fatalf("expected %q to normalize to %q, got %q", input, want, got)
		}
	}
}

func TestExtractPostseasonMatch(t *testing.T) {
	tests := map[string]string{
		"nfl 2001 super bowl s2001e021 stl at ne mkv": "2001.Super.Bowl",
		"nfl divisional 2025 hou at ne":               "2025.Divisional",
		"nfl 2015 championship":                       "2015.Championship",
		"nfl 2019 regular season":                     "",
	}

	for normalized, want := range tests {
		got, ok := extractPostseasonMatch(normalized)
		if want == "" {
			if ok {
				t.Fatalf("expected %q to have no postseason match, got %q", normalized, got)
			}
			continue
		}
		if !ok {
			t.Fatalf("expected %q to produce a postseason match", normalized)
		}
		if got != want {
			t.Fatalf("expected postseason match %q, got %q", want, got)
		}
	}
}

func TestExtractDateMatch(t *testing.T) {
	tests := map[string]string{
		"nfl 2021 09 19 s2021e002 ne at nyj mkv": "2021-09-19",
		"nfl 20181216 patriots vs steelers":      "2018-12-16",
		"nfl s2017e2 mkv":                        "",
	}

	for normalized, want := range tests {
		got, ok := extractDateMatch(normalized)
		if want == "" {
			if ok {
				t.Fatalf("expected %q to have no date match, got %q", normalized, got)
			}
			continue
		}
		if !ok {
			t.Fatalf("expected %q to produce a date match", normalized)
		}
		if got != want {
			t.Fatalf("expected date match %q, got %q", want, got)
		}
	}
}

func TestExtractGameDateStage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDate  string
		wantNext  string
		wantFound bool
	}{
		{
			name:      "hyphen separated date",
			input:     normalizeForMatching("NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv"),
			wantDate:  "2021-09-19",
			wantNext:  "nfl s2021e002 ne at nyj mkv",
			wantFound: true,
		},
		{
			name:      "dot separated date",
			input:     normalizeForMatching("NFL.2021.09.19.S2021E002.NE.at.NYJ.mkv"),
			wantDate:  "2021-09-19",
			wantNext:  "nfl s2021e002 ne at nyj mkv",
			wantFound: true,
		},
		{
			name:      "slash separated date",
			input:     normalizeForMatching("NFL/2021/09/19/NE/at/NYJ"),
			wantDate:  "2021-09-19",
			wantNext:  "nfl ne at nyj",
			wantFound: true,
		},
		{
			name:      "compact date",
			input:     normalizeForMatching("NFL_20210919_Patriots_vs_Jets"),
			wantDate:  "2021-09-19",
			wantNext:  "nfl patriots vs jets",
			wantFound: true,
		},
		{
			name:      "no date leaves working string unchanged",
			input:     normalizeForMatching("NFL.S2021E002.NE.at.NYJ.mkv"),
			wantNext:  "nfl s2021e002 ne at nyj mkv",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotNext, gotFound := extractGameDateStage(tt.input)
			if gotFound != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, gotFound)
			}
			if gotDate != tt.wantDate {
				t.Fatalf("expected date %q, got %q", tt.wantDate, gotDate)
			}
			if gotNext != tt.wantNext {
				t.Fatalf("expected next working string %q, got %q", tt.wantNext, gotNext)
			}
		})
	}
}

func TestExtractSeasonYearStage(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		gameDate       string
		wantSeasonYear string
		wantNext       string
		wantFound      bool
	}{
		{
			name:           "derive same season year from in-season game date",
			input:          normalizeForMatching("NFL.S2021E002.NE.at.NYJ.mkv"),
			gameDate:       "2021-09-19",
			wantSeasonYear: "2021",
			wantNext:       "nfl s2021e002 ne at nyj mkv",
			wantFound:      true,
		},
		{
			name:           "derive previous season year for january game date",
			input:          normalizeForMatching("NFL.Wild.Card.BAL.at.BUF"),
			gameDate:       "2022-01-15",
			wantSeasonYear: "2021",
			wantNext:       "nfl wild card bal at buf",
			wantFound:      true,
		},
		{
			name:           "extract standalone season year and remove it from working string",
			input:          normalizeForMatching("NFL 2018 Patriots vs Steelers"),
			wantSeasonYear: "2018",
			wantNext:       "nfl patriots vs steelers",
			wantFound:      true,
		},
		{
			name:      "no date or standalone season year leaves working string unchanged",
			input:     normalizeForMatching("NFL S2021E002 Patriots vs Steelers"),
			wantNext:  "nfl s2021e002 patriots vs steelers",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSeasonYear, gotNext, gotFound := extractSeasonYearStage(tt.input, tt.gameDate)
			if gotFound != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, gotFound)
			}
			if gotSeasonYear != tt.wantSeasonYear {
				t.Fatalf("expected season year %q, got %q", tt.wantSeasonYear, gotSeasonYear)
			}
			if gotNext != tt.wantNext {
				t.Fatalf("expected next working string %q, got %q", tt.wantNext, gotNext)
			}
		})
	}
}

func TestExtractGameTypeStage(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantGameType GameType
		wantNext     string
		wantFound    bool
	}{
		{
			name:         "super bowl aliases map to sb",
			input:        normalizeForMatching("NFL.super.bowl.SB.SuperBowl"),
			wantGameType: GameTypeSuperBowl,
			wantNext:     "nfl super bowl sb superbowl",
			wantFound:    true,
		},
		{
			name:         "conference aliases map to con",
			input:        normalizeForMatching("NFL.conference.CON.championship"),
			wantGameType: GameTypeConference,
			wantNext:     "nfl conference con championship",
			wantFound:    true,
		},
		{
			name:         "divisional aliases map to div",
			input:        normalizeForMatching("NFL.div.division.divisional"),
			wantGameType: GameTypeDivisional,
			wantNext:     "nfl div division divisional",
			wantFound:    true,
		},
		{
			name:         "wildcard aliases map to wc",
			input:        normalizeForMatching("NFL.wc.wildcard.wild.card"),
			wantGameType: GameTypeWildcard,
			wantNext:     "nfl wc wildcard wild card",
			wantFound:    true,
		},
		{
			name:         "unknown game type defaults to regular season without mutating",
			input:        normalizeForMatching("NFL Patriots at Jets"),
			wantGameType: GameTypeRegularSeason,
			wantNext:     "nfl patriots at jets",
			wantFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGameType, gotNext, gotFound := extractGameTypeStage(tt.input)
			if gotFound != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, gotFound)
			}
			if gotGameType != tt.wantGameType {
				t.Fatalf("expected game type %q, got %q", tt.wantGameType, gotGameType)
			}
			if gotNext != tt.wantNext {
				t.Fatalf("expected next working string %q, got %q", tt.wantNext, gotNext)
			}
		})
	}
}

func TestExtractGameWeekStage(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		gameType     GameType
		wantGameWeek string
		wantNext     string
		wantFound    bool
	}{
		{
			name:         "extract week number from separated week token",
			input:        normalizeForMatching("NFL Week 2 Patriots at Jets"),
			wantGameWeek: "2",
			wantNext:     "nfl patriots at jets",
			wantFound:    true,
		},
		{
			name:         "extract week number from compact week token",
			input:        normalizeForMatching("NFL Week18 Bills at Dolphins"),
			wantGameWeek: "18",
			wantNext:     "nfl bills at dolphins",
			wantFound:    true,
		},
		{
			name:         "extract week number from compact wk token",
			input:        normalizeForMatching("NFL.WK18.Bills.at.Dolphins"),
			wantGameWeek: "18",
			wantNext:     "nfl bills at dolphins",
			wantFound:    true,
		},
		{
			name:         "extract week number from compact w token",
			input:        normalizeForMatching("NFL w7 Chiefs at Raiders"),
			wantGameWeek: "7",
			wantNext:     "nfl chiefs at raiders",
			wantFound:    true,
		},
		{
			name:         "extract week number from double digit compact w token",
			input:        normalizeForMatching("NFL w18 Ravens at Bengals"),
			wantGameWeek: "18",
			wantNext:     "nfl ravens at bengals",
			wantFound:    true,
		},
		{
			name:         "extract super bowl roman numeral after super bowl alias",
			input:        normalizeForMatching("NFL Super Bowl lix Chiefs vs Eagles"),
			gameType:     GameTypeSuperBowl,
			wantGameWeek: "LIX",
			wantNext:     "nfl super bowl chiefs vs eagles",
			wantFound:    true,
		},
		{
			name:         "extract super bowl roman numeral after sb alias",
			input:        normalizeForMatching("NFL SB xlviii Seahawks vs Broncos"),
			gameType:     GameTypeSuperBowl,
			wantGameWeek: "XLVIII",
			wantNext:     "nfl sb seahawks vs broncos",
			wantFound:    true,
		},
		{
			name:      "invalid super bowl roman numeral is ignored",
			input:     normalizeForMatching("NFL Super Bowl iix Chiefs vs Eagles"),
			gameType:  GameTypeSuperBowl,
			wantNext:  "nfl super bowl iix chiefs vs eagles",
			wantFound: false,
		},
		{
			name:      "roman numeral without super bowl alias is ignored",
			input:     normalizeForMatching("NFL IX Chiefs at Raiders"),
			gameType:  GameTypeRegularSeason,
			wantNext:  "nfl ix chiefs at raiders",
			wantFound: false,
		},
		{
			name:      "no week token leaves working string unchanged",
			input:     normalizeForMatching("NFL Patriots at Jets"),
			wantNext:  "nfl patriots at jets",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGameWeek, gotNext, gotFound := extractGameWeekStage(tt.input, tt.gameType)
			if gotFound != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, gotFound)
			}
			if gotGameWeek != tt.wantGameWeek {
				t.Fatalf("expected game week %q, got %q", tt.wantGameWeek, gotGameWeek)
			}
			if gotNext != tt.wantNext {
				t.Fatalf("expected next working string %q, got %q", tt.wantNext, gotNext)
			}
		})
	}
}

func TestNormalizeForMatch(t *testing.T) {
	input := "NFL__2018...Super---Bowl__Patriots-vs-Rams"
	got := NormalizeForMatch(input)
	want := "nfl 2018 super bowl patriots vs rams"
	if got != want {
		t.Fatalf("expected normalized string %q, got %q", want, got)
	}
}

func TestTokenizeForMatch(t *testing.T) {
	input := "Patriots...at__Jets"
	got := TokenizeForMatch(input)
	want := []string{"patriots", "at", "jets"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected tokens %v, got %v", want, got)
	}
}

func TestParseReleasesHandlesGeneralNonAlphanumericSeparators(t *testing.T) {
	files := []string{
		"NFL__2018__Super---Bowl__S2018E021__NE__at__LA.mkv",
		"NFL__2021__09__19__S2021E002__NE__at__NYJ.mkv",
	}

	got := ParseReleases(files)
	want := []string{
		"2018.Super.Bowl",
		"2021-09-19",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestParseReleases(t *testing.T) {
	files := []string{
		"N.F.L.2001.Super.Bowl.S2001E021.STL.at.NE.mkv",
		"nfl-2014 divisional S2014E019 BAL at NE mkv",
		"NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv",
		"NFL_2018_12_16_Patriots_vs_Steelers_S2018E13.mkv",
		"CFL.2021-09-19.TOR.at.OTT.mkv",
	}

	got := ParseReleases(files)
	want := []string{
		"2001.Super.Bowl",
		"2014.Divisional",
		"2021-09-19",
		"2018-12-16",
		"CFL.2021-09-19.TOR.at.OTT.mkv",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected parsed releases %v, got %v", want, got)
	}
}
