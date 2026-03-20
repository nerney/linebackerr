package search

import (
	"context"
	"errors"
	"linebackerr/matcher"
	"linebackerr/prowlarr"
	"reflect"
	"testing"
)

type fakeSearcher struct {
	calls   []string
	results map[string][]prowlarr.NFLRelease
	err     error
}

func (f *fakeSearcher) SearchNFLReleases(query string, _ []int) ([]prowlarr.NFLRelease, error) {
	f.calls = append(f.calls, query)
	if f.err != nil {
		return nil, f.err
	}
	return append([]prowlarr.NFLRelease(nil), f.results[query]...), nil
}

func TestBuildQueries(t *testing.T) {
	queries := BuildQueries(matcher.Match{
		GameDate:   "2024-09-08",
		SeasonYear: "2024",
		GameType:   matcher.GameTypeRegularSeason,
		GameWeek:   "1",
		AwayTeam:   "BUF",
		HomeTeam:   "NE",
	})

	want := []string{
		"nfl 2024-09-08 buf vs ne",
		"nfl 2024 week 1 buf ne",
		"nfl 2024 buf ne",
		"nfl buf ne",
		"nfl",
	}
	if !reflect.DeepEqual(queries, want) {
		t.Fatalf("queries = %#v, want %#v", queries, want)
	}
}

func TestSearchAndDownload_RunsLifecycleFromMatchToDownload(t *testing.T) {
	resolved := matcher.Match{
		OriginalInput: "NFL.2024.09.08.BUF.vs.NE.1080p",
		GameDate:      "2024-09-08",
		SeasonYear:    "2024",
		GameType:      matcher.GameTypeRegularSeason,
		GameWeek:      "1",
		AwayTeam:      "BUF",
		HomeTeam:      "NE",
		NflverseID:    "2024_01_BUF_NE",
	}

	releaseMatch := resolved
	releases := []prowlarr.NFLRelease{
		{
			Raw: prowlarr.SearchResult{
				Title:       "NFL.2024.09.08.BUF.vs.NE.1080p.WEB-DL",
				GUID:        "guid-1",
				DownloadURL: "https://dl/1",
			},
			Match: releaseMatch,
		},
		{
			Raw: prowlarr.SearchResult{
				Title:       "NFL.2024.09.08.BUF.vs.NE.720p.WEB-DL",
				GUID:        "guid-2",
				DownloadURL: "https://dl/2",
			},
			Match: releaseMatch,
		},
	}

	searcher := &fakeSearcher{results: map[string][]prowlarr.NFLRelease{
		"nfl 2024-09-08 buf vs ne": releases,
	}}

	var downloaded []prowlarr.NFLRelease
	svc := NewService(searcher, func(_ context.Context, selected []prowlarr.NFLRelease) error {
		downloaded = append(downloaded, selected...)
		return nil
	}, []int{1, 2})
	svc.MaxDownloads = 1

	result, err := svc.SearchAndDownload(context.Background(), resolved)
	if err != nil {
		t.Fatalf("SearchAndDownload returned error: %v", err)
	}

	if len(searcher.calls) == 0 {
		t.Fatalf("expected search calls")
	}
	if len(result.ReleasesFound) != 2 {
		t.Fatalf("releases found = %d, want 2", len(result.ReleasesFound))
	}
	if len(result.SelectedReleases) != 1 {
		t.Fatalf("selected releases = %d, want 1", len(result.SelectedReleases))
	}
	if len(downloaded) != 1 {
		t.Fatalf("downloaded releases = %d, want 1", len(downloaded))
	}
	if downloaded[0].Raw.GUID != "guid-1" {
		t.Fatalf("downloaded guid = %q, want guid-1", downloaded[0].Raw.GUID)
	}
}

func TestSearchAndDownload_FiltersToTargetNflverseMatch(t *testing.T) {
	resolved := matcher.Match{
		GameDate:   "2024-09-08",
		SeasonYear: "2024",
		GameType:   matcher.GameTypeRegularSeason,
		GameWeek:   "1",
		AwayTeam:   "BUF",
		HomeTeam:   "NE",
		NflverseID: "2024_01_BUF_NE",
	}

	wrong := resolved
	wrong.NflverseID = "2024_01_DAL_NYG"

	searcher := &fakeSearcher{results: map[string][]prowlarr.NFLRelease{
		"nfl 2024-09-08 buf vs ne": {
			{Raw: prowlarr.SearchResult{Title: "wrong", GUID: "bad", DownloadURL: "https://dl/bad"}, Match: wrong},
			{Raw: prowlarr.SearchResult{Title: "right", GUID: "good", DownloadURL: "https://dl/good"}, Match: resolved},
		},
	}}

	var downloaded []prowlarr.NFLRelease
	svc := NewService(searcher, func(_ context.Context, selected []prowlarr.NFLRelease) error {
		downloaded = append(downloaded, selected...)
		return nil
	}, nil)

	result, err := svc.SearchAndDownload(context.Background(), resolved)
	if err != nil {
		t.Fatalf("SearchAndDownload returned error: %v", err)
	}
	if len(result.ReleasesFound) != 1 || result.ReleasesFound[0].Raw.GUID != "good" {
		t.Fatalf("expected only matching nflverse release, got %#v", result.ReleasesFound)
	}
	if len(downloaded) != 1 || downloaded[0].Raw.GUID != "good" {
		t.Fatalf("downloaded = %#v, want only good guid", downloaded)
	}
}

func TestSearchAndDownload_UnresolvedMatch(t *testing.T) {
	svc := NewService(&fakeSearcher{}, func(context.Context, []prowlarr.NFLRelease) error { return nil }, nil)
	_, err := svc.SearchAndDownload(context.Background(), matcher.Match{})
	if !errors.Is(err, ErrUnresolvedMatch) {
		t.Fatalf("expected ErrUnresolvedMatch, got %v", err)
	}
}
