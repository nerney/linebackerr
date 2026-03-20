package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"linebackerr/server/api"
)

const defaultAddr = ":6666"

type Server struct {
	httpServer *http.Server
	addr       string
}

type handler struct{}

func (handler) HealthCheck(ctx context.Context) (*api.HealthCheckOK, error) {
	_ = ctx
	return &api.HealthCheckOK{Status: "ok"}, nil
}

func Init() *Server {
	apiServer, err := api.NewServer(handler{})
	if err != nil {
		panic(fmt.Errorf("create ogen server: %w", err))
	}

	spaHandler := newSPAHandler("dist")
	rootHandler := withRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := apiServer.FindPath(r.Method, r.URL); ok {
			apiServer.ServeHTTP(w, r)
			return
		}
		spaHandler.ServeHTTP(w, r)
	}))

	return &Server{
		httpServer: &http.Server{
			Addr:    defaultAddr,
			Handler: rootHandler,
		},
		addr: defaultAddr,
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

type spaHandler struct {
	distDir   string
	indexPath string
}

func newSPAHandler(distDir string) http.Handler {
	return &spaHandler{
		distDir:   distDir,
		indexPath: filepath.Join(distDir, "index.html"),
	}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, h.indexPath)
		return
	}

	reqPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if reqPath == "." || strings.HasPrefix(reqPath, "..") {
		http.ServeFile(w, r, h.indexPath)
		return
	}

	assetPath := filepath.Join(h.distDir, reqPath)
	if stat, err := os.Stat(assetPath); err == nil && !stat.IsDir() {
		http.ServeFile(w, r, assetPath)
		return
	}

	http.ServeFile(w, r, h.indexPath)
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
