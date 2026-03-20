package server

import (
	"fmt"
	"net/http"
)

const defaultAddr = ":6666"

type Server struct {
	httpServer *http.Server
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func Init() *Server {
	mux := http.NewServeMux()

	// Routes currently live in this package; add/register new HTTP routes here for now.
	mux.HandleFunc("/health", healthHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:    defaultAddr,
			Handler: mux,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}
