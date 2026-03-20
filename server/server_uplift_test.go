package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitUsesDefaultAddr(t *testing.T) {
	s := Init()
	require.NotNil(t, s)
	assert.Equal(t, defaultAddr, s.addr)
	require.NotNil(t, s.httpServer)
	assert.Equal(t, defaultAddr, s.httpServer.Addr)
}

func TestWithRecoveryHandlesPanics(t *testing.T) {
	h := withRecovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal server error")
}

func TestSPAHandlerPathTraversalIsRejected(t *testing.T) {
	dist := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dist, "index.html"), []byte("INDEX"), 0o644))

	h := newSPAHandler(dist)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/safe", nil)
	req.URL.Path = "/../secret.txt"
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid URL path")
}
