package qbit

import (
	"context"
	"encoding/json"
	"linebackerr/prowlarr"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDownloadFromProwlarrReleases_AddsTorrentAndWaitsForCompletion(t *testing.T) {
	var (
		mu             sync.Mutex
		loginCalls     int
		addCalls       int
		lastAddValues  url.Values
		infoCalls      int
		torrentAdded   bool
		completionPass int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse login form: %v", err)
			}
			if r.Form.Get("username") != "admin" || r.Form.Get("password") != "adminpass" {
				t.Fatalf("unexpected creds: %q / %q", r.Form.Get("username"), r.Form.Get("password"))
			}
			mu.Lock()
			loginCalls++
			mu.Unlock()
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse add form: %v", err)
			}
			mu.Lock()
			addCalls++
			torrentAdded = true
			lastAddValues = r.PostForm
			mu.Unlock()
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			mu.Lock()
			infoCalls++
			added := torrentAdded
			completionPass++
			pass := completionPass
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if !added {
				_, _ = w.Write([]byte(`[]`))
				return
			}

			state := "downloading"
			progress := 0.5
			if pass > 1 {
				state = "uploading"
				progress = 1
			}
			_ = json.NewEncoder(w).Encode([]TorrentInfo{{
				Hash:     "abc123",
				Name:     "NFL.2024.09.08.BUF.vs.NE",
				State:    state,
				Progress: progress,
			}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "admin", "adminpass", ts.Client())
	releases := []prowlarr.NFLRelease{{
		Raw: prowlarr.SearchResult{
			Title:       "NFL.2024.09.08.BUF.vs.NE.1080p",
			DownloadURL: "https://tracker.local/download/1",
		},
	}}

	jobs, err := client.DownloadFromProwlarrReleases(context.Background(), releases, DownloadOptions{
		SavePath:          "/data/downloads/torrents",
		Category:          "linebackerr",
		Tags:              []string{"nfl", "linebackerr"},
		PollInterval:      1 * time.Millisecond,
		DetectTimeout:     200 * time.Millisecond,
		CompletionTimeout: 200 * time.Millisecond,
		WaitForCompletion: true,
	})
	if err != nil {
		t.Fatalf("DownloadFromProwlarrReleases returned error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}
	if jobs[0].Torrent.Hash != "abc123" {
		t.Fatalf("torrent hash = %q, want abc123", jobs[0].Torrent.Hash)
	}
	if jobs[0].Torrent.State != "uploading" {
		t.Fatalf("final torrent state = %q, want uploading", jobs[0].Torrent.State)
	}

	mu.Lock()
	defer mu.Unlock()
	if loginCalls != 1 {
		t.Fatalf("login calls = %d, want 1", loginCalls)
	}
	if addCalls != 1 {
		t.Fatalf("add calls = %d, want 1", addCalls)
	}
	if infoCalls < 2 {
		t.Fatalf("info calls = %d, want >=2 to observe lifecycle", infoCalls)
	}
	if got := lastAddValues.Get("urls"); got != "https://tracker.local/download/1" {
		t.Fatalf("add urls = %q, want download URL", got)
	}
	if got := lastAddValues.Get("category"); got != "linebackerr" {
		t.Fatalf("add category = %q, want linebackerr", got)
	}
	if got := lastAddValues.Get("savepath"); got != "/data/downloads/torrents" {
		t.Fatalf("add savepath = %q, want /data/downloads/torrents", got)
	}
	if got := lastAddValues.Get("tags"); got != "nfl,linebackerr" {
		t.Fatalf("add tags = %q, want nfl,linebackerr", got)
	}
}

func TestLogin_RequiresCredentials(t *testing.T) {
	client := NewClient("http://qbit.local", "", "", &http.Client{})
	err := client.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing qbit credentials") {
		t.Fatalf("expected missing creds error, got: %v", err)
	}
}

func TestDownloadFromProwlarrReleases_SkipsEmptyDownloadURLs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			_, _ = w.Write([]byte(`[]`))
		case "/api/v2/torrents/add":
			t.Fatalf("did not expect /torrents/add to be called")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "admin", "adminpass", ts.Client())
	releases := []prowlarr.NFLRelease{{
		Raw: prowlarr.SearchResult{Title: "NFL.2024.09.08.BUF.vs.NE"},
	}}

	jobs, err := client.DownloadFromProwlarrReleases(context.Background(), releases, DownloadOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("jobs len = %d, want 0", len(jobs))
	}
}
