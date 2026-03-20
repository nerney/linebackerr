package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"linebackerr/server/api"
)

func newTestHandler(t *testing.T, distDir string) http.Handler {
	t.Helper()

	apiServer, err := api.NewServer(handler{})
	if err != nil {
		t.Fatalf("new ogen server: %v", err)
	}

	spa := newSPAHandler(distDir)
	return withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := apiServer.FindPath(r.Method, r.URL); ok {
			apiServer.ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	}))
}

func TestHealthRoute(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	h := newTestHandler(t, dist)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestSPAFallbackAndAssetServing(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("INDEX"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dist, "app.js"), []byte("ASSET"), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}

	h := newTestHandler(t, dist)

	t.Run("serves static asset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if rr.Body.String() != "ASSET" {
			t.Fatalf("expected asset content, got %q", rr.Body.String())
		}
	})

	t.Run("falls back to index for unknown route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/some/frontend/route", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		if rr.Body.String() != "INDEX" {
			t.Fatalf("expected index fallback, got %q", rr.Body.String())
		}
	})
}
