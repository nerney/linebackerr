package prowlarr

import (
	"io"
	"linebackerr/db"
	"linebackerr/matcher"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSearchNFLReleases_UsesAPIKeyAndIndexerFilterAndMapsMatches(t *testing.T) {
	oldDB := db.DB
	db.DB = nil
	t.Cleanup(func() { db.DB = oldDB })

	client := NewClient("http://prowlarr.local", "test-api-key", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", req.Method)
			}
			if req.URL.String() != "http://prowlarr.local/api/v1/search?indexerIds=1&indexerIds=77&query=nfl" {
				t.Fatalf("request url = %s", req.URL.String())
			}
			if req.Header.Get("X-Api-Key") != "test-api-key" {
				t.Fatalf("missing or wrong X-Api-Key header: %q", req.Header.Get("X-Api-Key"))
			}

			body := `[
				{"title":"NFL.2024.09.08.BUF.vs.NE.1080p","guid":"g1","indexerId":1,"indexer":"tracker-a","size":1111,"publishDate":"2024-09-08T20:00:00Z","downloadUrl":"https://dl/1"},
				{"title":"Random.Movie.2024.1080p","guid":"g2","indexerId":77,"indexer":"tracker-b","size":2222,"publishDate":"2024-10-01T20:00:00Z","downloadUrl":"https://dl/2"}
			]`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	})

	results, err := client.SearchNFLReleases("nfl", []int{1, 77})
	if err != nil {
		t.Fatalf("SearchNFLReleases returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected only NFL result to be returned, got %d", len(results))
	}

	result := results[0]
	if result.Raw.GUID != "g1" || result.Raw.Indexer != "tracker-a" {
		t.Fatalf("unexpected raw mapping: %#v", result.Raw)
	}
	if result.Candidate.GameDate != "2024-09-08" {
		t.Fatalf("candidate date = %q, want 2024-09-08", result.Candidate.GameDate)
	}
	if result.Candidate.AwayTeam != "BUF" || result.Candidate.HomeTeam != "NE" {
		t.Fatalf("candidate teams = %s @ %s, want BUF @ NE", result.Candidate.AwayTeam, result.Candidate.HomeTeam)
	}
	if result.Match.Error != matcher.ErrNoMatchFound {
		t.Fatalf("expected unresolved match with ErrNoMatchFound, got %v", result.Match.Error)
	}
}

func TestSearchNFLReleases_RequiresAPIKey(t *testing.T) {
	client := NewClient("http://prowlarr.local", "", &http.Client{})
	_, err := client.SearchNFLReleases("nfl", nil)
	if err == nil || !strings.Contains(err.Error(), "missing prowlarr api key") {
		t.Fatalf("expected missing api key error, got: %v", err)
	}
}

func TestSearchNFLReleases_PropagatesNon200(t *testing.T) {
	client := NewClient("http://prowlarr.local", "test-api-key", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"message":"nope"}`)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	})

	_, err := client.SearchNFLReleases("nfl", nil)
	if err == nil || !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected status error, got: %v", err)
	}
}
