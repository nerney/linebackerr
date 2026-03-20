package sportarr

import (
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type rewriteTransport struct {
	base  http.RoundTripper
	route map[string]func(*http.Request) (*http.Response, error)
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if handler, ok := t.route[req.URL.String()]; ok {
		return handler(req)
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("not found")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func withPatchedTransport(tb testing.TB, route map[string]func(*http.Request) (*http.Response, error)) {
	tb.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = rewriteTransport{base: original, route: route}
	tb.Cleanup(func() {
		http.DefaultTransport = original
	})
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE sportarr_team (team_id TEXT PRIMARY KEY, strLogo TEXT, strBadge TEXT, strBanner TEXT, strFanart1 TEXT, strDescriptionEN TEXT);`,
		`CREATE TABLE sportarr_seasons (year TEXT PRIMARY KEY, poster_url TEXT);`,
		`CREATE TABLE nflverse_teams (id TEXT PRIMARY KEY, abbr TEXT, full_name TEXT);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}

	return db
}

func TestLoadTeamsStoresLookupData(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.Exec(`INSERT INTO nflverse_teams (id, abbr, full_name) VALUES ('NE', 'NE', 'New England Patriots'), ('BUF', 'BUF', 'Buffalo Bills')`); err != nil {
		t.Fatalf("seed teams: %v", err)
	}

	withPatchedTransport(t, map[string]func(*http.Request) (*http.Response, error){
		"https://sportarr.net/api/v2/json/search/team/New%20England%20Patriots": func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, `{"data":{"search":[{"idTeam":1570}]}}`), nil
		},
		"https://sportarr.net/api/v2/json/lookup/team/1570": func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, `{"data":{"lookup":[{"strLogo":"logo-ne","strBadge":"badge-ne","strBanner":"banner-ne","strFanart1":"fanart-ne","strDescriptionEN":"Patriots desc"}]}}`), nil
		},
		"https://sportarr.net/api/v2/json/search/team/Buffalo%20Bills": func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, `{"data":{"search":[]}}`), nil
		},
	})

	client := &Client{DB: db}
	if err := client.LoadTeams(); err != nil {
		t.Fatalf("LoadTeams: %v", err)
	}

	var gotID, gotLogo, gotBadge, gotBanner, gotFanart, gotDesc string
	if err := db.QueryRow(`SELECT team_id, strLogo, strBadge, strBanner, strFanart1, strDescriptionEN FROM sportarr_team WHERE team_id = 'NE'`).Scan(&gotID, &gotLogo, &gotBadge, &gotBanner, &gotFanart, &gotDesc); err != nil {
		t.Fatalf("query team row: %v", err)
	}

	if gotID != "NE" || gotLogo != "logo-ne" || gotBadge != "badge-ne" || gotBanner != "banner-ne" || gotFanart != "fanart-ne" || gotDesc != "Patriots desc" {
		t.Fatalf("unexpected team row: %q %q %q %q %q %q", gotID, gotLogo, gotBadge, gotBanner, gotFanart, gotDesc)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sportarr_team`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 saved team, got %d", count)
	}
}

func TestLoadSeasonsStoresMainAndScrapedPosters(t *testing.T) {
	db := newTestDB(t)

	withPatchedTransport(t, map[string]func(*http.Request) (*http.Response, error){
		"https://sportarr.net/api/metadata/plex/series/4391/seasons": func(req *http.Request) (*http.Response, error) {
			return jsonResponse(req, `{"seasons":[{"poster_url":"https://img.example/main.jpg"}]}`), nil
		},
		"https://www.thesportsdb.com/league/4391-nfl": func(req *http.Request) (*http.Response, error) {
			body := `<a href="/season/4391-nfl/2024">2024</a><a href="/season/4391-nfl/2024">dup</a><a href="/season/4391-nfl/2023">2023</a>`
			return textResponse(req, body), nil
		},
		"https://www.thesportsdb.com/season/4391-nfl/2024": func(req *http.Request) (*http.Response, error) {
			return textResponse(req, `https://cdn.example.com/poster/season-2024-poster.jpg`), nil
		},
		"https://www.thesportsdb.com/season/4391-nfl/2023": func(req *http.Request) (*http.Response, error) {
			return textResponse(req, `https://cdn.example.com/poster/season-2023-poster.jpg`), nil
		},
	})

	client := &Client{DB: db}
	if err := client.LoadSeasons(); err != nil {
		t.Fatalf("LoadSeasons: %v", err)
	}

	rows, err := db.Query(`SELECT year, poster_url FROM sportarr_seasons ORDER BY year`)
	if err != nil {
		t.Fatalf("query seasons: %v", err)
	}
	defer rows.Close()

	got := map[string]string{}
	for rows.Next() {
		var year, poster string
		if err := rows.Scan(&year, &poster); err != nil {
			t.Fatalf("scan season: %v", err)
		}
		got[year] = poster
	}

	expected := map[string]string{
		"2023": "https://cdn.example.com/poster/season-2023-poster.jpg",
		"2024": "https://cdn.example.com/poster/season-2024-poster.jpg",
		"MAIN": "https://img.example/main.jpg",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d season rows, got %d: %#v", len(expected), len(got), got)
	}
	for year, poster := range expected {
		if got[year] != poster {
			t.Fatalf("poster for %s = %q, want %q", year, got[year], poster)
		}
	}
}

func jsonResponse(req *http.Request, body string) *http.Response {
	resp := textResponse(req, body)
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

func textResponse(req *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}
}
