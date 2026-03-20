package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const defaultAddr = ":6666"

type Server struct {
	e    *echo.Echo
	addr string
}

func Init() *Server {
	e := echo.New()

	// Middleware currently applied globally for all routes.
	e.Use(middleware.Recover())

	// Routes currently live in this package; add/register new HTTP routes here for now.
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	distDir := "dist"
	indexPath := filepath.Join(distDir, "index.html")

	e.GET("/", func(c echo.Context) error {
		return c.File(indexPath)
	})

	e.GET("/*", func(c echo.Context) error {
		reqPath := filepath.Clean(c.Param("*"))
		if reqPath == "." || strings.HasPrefix(reqPath, "..") {
			return c.File(indexPath)
		}

		assetPath := filepath.Join(distDir, reqPath)
		if stat, err := os.Stat(assetPath); err == nil && !stat.IsDir() {
			return c.File(assetPath)
		}

		return c.File(indexPath)
	})

	return &Server{
		e:    e,
		addr: defaultAddr,
	}
}

func (s *Server) Start() error {
	return s.e.Start(s.addr)
}
