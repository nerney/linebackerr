package downloader

import (
	"context"
	"encoding/json"
	"linebackerr/prowlarr"
	"linebackerr/qbit"
	"linebackerr/sabnzbd"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestService_SubmitAndMonitorQBit(t *testing.T) {
	var (
		mu       sync.Mutex
		added    bool
		infoCall int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			mu.Lock()
			added = true
			mu.Unlock()
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			mu.Lock()
			infoCall++
			call := infoCall
			isAdded := added
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			if !isAdded {
				_, _ = w.Write([]byte(`[]`))
				return
			}

			state := "downloading"
			progress := 0.25
			if call >= 3 {
				state = "uploading"
				progress = 1
			}
			_ = json.NewEncoder(w).Encode([]qbit.TorrentInfo{{
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

	svc := NewService(qbit.NewClient(ts.URL, "admin", "adminpass", ts.Client()), nil)
	releases := []prowlarr.NFLRelease{{
		Raw: prowlarr.SearchResult{Title: "NFL.2024.09.08.BUF.vs.NE", DownloadURL: "magnet:?xt=urn:btih:abc123"},
	}}

	jobs, err := svc.Submit(context.Background(), ClientQBit, releases, SubmitOptions{QBit: qbit.DownloadOptions{
		PollInterval:      time.Millisecond,
		DetectTimeout:     250 * time.Millisecond,
		WaitForCompletion: false,
	}})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}
	if jobs[0].Status != StatusActive {
		t.Fatalf("initial status = %s, want %s", jobs[0].Status, StatusActive)
	}

	active, err := svc.Monitor(context.Background())
	if err != nil {
		t.Fatalf("Monitor returned error: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active len = %d, want 0 after completion", len(active))
	}

	all := svc.AllJobs()
	if len(all) != 1 {
		t.Fatalf("all jobs len = %d, want 1", len(all))
	}
	if all[0].Status != StatusCompleted {
		t.Fatalf("final status = %s, want %s", all[0].Status, StatusCompleted)
	}
}

func TestService_SubmitAndMonitorSABnzbd(t *testing.T) {
	var (
		mu           sync.Mutex
		queueCalls   int
		historyCalls int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("mode") {
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
			_, _ = w.Write([]byte(`{"queue":{"slots":[{"nzo_id":"nzo-1","filename":"NFL.2024.09.08.BUF.vs.NE.nzb","status":"Downloading","percentage":"42"}]}}`))
		case "addurl":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"nzo_ids":["nzo-1"]}`))
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
			_, _ = w.Write([]byte(`{"history":{"slots":[{"nzo_id":"nzo-1","name":"NFL.2024.09.08.BUF.vs.NE","status":"Completed"}]}}`))
		default:
			t.Fatalf("unexpected mode: %s", r.URL.Query().Get("mode"))
		}
	}))
	defer ts.Close()

	svc := NewService(nil, sabnzbd.NewClient(ts.URL, "sab-key", ts.Client()))
	releases := []prowlarr.NFLRelease{{
		Raw: prowlarr.SearchResult{Title: "NFL.2024.09.08.BUF.vs.NE", DownloadURL: "https://idx/getnzb/1"},
	}}

	jobs, err := svc.Submit(context.Background(), ClientSABnzbd, releases, SubmitOptions{SABnzbd: sabnzbd.DownloadOptions{
		PollInterval:      time.Millisecond,
		DetectTimeout:     250 * time.Millisecond,
		WaitForCompletion: false,
	}})
	if err != nil {
		t.Fatalf("Submit returned error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}
	if jobs[0].Status != StatusActive {
		t.Fatalf("initial status = %s, want %s", jobs[0].Status, StatusActive)
	}

	active, err := svc.Monitor(context.Background())
	if err != nil {
		t.Fatalf("Monitor returned error: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("active len = %d, want 1 while history has not finalized", len(active))
	}

	active, err = svc.Monitor(context.Background())
	if err != nil {
		t.Fatalf("Monitor second pass returned error: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active len = %d, want 0 after completion", len(active))
	}

	all := svc.AllJobs()
	if len(all) != 1 {
		t.Fatalf("all jobs len = %d, want 1", len(all))
	}
	if all[0].Status != StatusCompleted {
		t.Fatalf("final status = %s, want %s", all[0].Status, StatusCompleted)
	}
}

func TestService_SubmitRejectsUnsupportedClient(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.Submit(context.Background(), ClientType("other"), nil, SubmitOptions{})
	if err == nil {
		t.Fatalf("expected error for unsupported client")
	}
}
