package sabnzbd

import (
	"context"
	"linebackerr/prowlarr"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestDownloadFromProwlarrReleases_AddAndWaitForCompletion(t *testing.T) {
	var (
		mu            sync.Mutex
		queueCalls    int
		historyCalls  int
		addCalls      int
		receivedAPI   []string
		receivedModes []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := r.URL.Query().Get("mode")
		apiKey := r.URL.Query().Get("apikey")

		mu.Lock()
		receivedAPI = append(receivedAPI, apiKey)
		receivedModes = append(receivedModes, mode)
		mu.Unlock()

		if apiKey != "sab-key" {
			t.Fatalf("unexpected api key: %q", apiKey)
		}

		switch mode {
		case "queue":
			mu.Lock()
			queueCalls++
			call := queueCalls
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if call < 2 {
				_, _ = w.Write([]byte(`{"queue":{"slots":[]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"queue":{"slots":[{"nzo_id":"SABnzbd_nzo_abc","filename":"NFL.2024.09.08.BUF.vs.NE.nzb","status":"Downloading","cat":"linebackerr","percentage":"62"}]}}`))
		case "addurl":
			if got := r.URL.Query().Get("name"); got != "https://indexer.local/getnzb/1" {
				t.Fatalf("unexpected addurl name: %q", got)
			}
			if got := r.URL.Query().Get("cat"); got != "linebackerr" {
				t.Fatalf("unexpected category: %q", got)
			}
			mu.Lock()
			addCalls++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"nzo_ids":["SABnzbd_nzo_abc"]}`))
		case "history":
			mu.Lock()
			historyCalls++
			call := historyCalls
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if call < 2 {
				_, _ = w.Write([]byte(`{"history":{"slots":[]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"history":{"slots":[{"nzo_id":"SABnzbd_nzo_abc","name":"NFL.2024.09.08.BUF.vs.NE","status":"Completed","category":"linebackerr"}]}}`))
		default:
			t.Fatalf("unexpected mode: %s", mode)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "sab-key", ts.Client())
	releases := []prowlarr.NFLRelease{{
		Raw: prowlarr.SearchResult{
			Title:       "NFL.2024.09.08.BUF.vs.NE.1080p",
			DownloadURL: "https://indexer.local/getnzb/1",
		},
	}}

	jobs, err := client.DownloadFromProwlarrReleases(context.Background(), releases, DownloadOptions{
		Category:          "linebackerr",
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
	if jobs[0].NZOID != "SABnzbd_nzo_abc" {
		t.Fatalf("nzo id = %q, want SABnzbd_nzo_abc", jobs[0].NZOID)
	}
	if jobs[0].Queue.NZOID != "SABnzbd_nzo_abc" {
		t.Fatalf("queue nzo id = %q, want SABnzbd_nzo_abc", jobs[0].Queue.NZOID)
	}
	if jobs[0].History.Status != "Completed" {
		t.Fatalf("history status = %q, want Completed", jobs[0].History.Status)
	}
	if !jobs[0].IsSuccessful {
		t.Fatalf("expected IsSuccessful=true")
	}

	mu.Lock()
	defer mu.Unlock()
	if addCalls != 1 {
		t.Fatalf("add calls = %d, want 1", addCalls)
	}
	if queueCalls < 2 {
		t.Fatalf("queue calls = %d, want >= 2", queueCalls)
	}
	if historyCalls < 2 {
		t.Fatalf("history calls = %d, want >= 2", historyCalls)
	}
}

func TestDownloadFromProwlarrReleases_RequiresAPIKey(t *testing.T) {
	client := NewClient("http://sab.local", "", &http.Client{})
	_, err := client.DownloadFromProwlarrReleases(context.Background(), nil, DownloadOptions{})
	if err == nil || err.Error() != "missing sabnzbd api key" {
		t.Fatalf("expected missing api key error, got: %v", err)
	}
}

func TestDownloadFromProwlarrReleases_SkipsEmptyDownloadURLs(t *testing.T) {
	var addCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("mode") {
		case "queue":
			_, _ = w.Write([]byte(`{"queue":{"slots":[]}}`))
		case "addurl":
			addCalled = true
			_, _ = w.Write([]byte(`{"status":true,"nzo_ids":["SABnzbd_nzo_x"]}`))
		default:
			t.Fatalf("unexpected mode: %s", r.URL.Query().Get("mode"))
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "sab-key", ts.Client())
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
	if addCalled {
		t.Fatalf("did not expect addurl to be called")
	}
}
