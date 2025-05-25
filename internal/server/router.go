package server

import (
	"net/http"

	"github.com/joomcode/errorx"
)

// setupRoutes configures all HTTP routes for the server
func (s *Server) setupRoutes() {
	http.HandleFunc("/json/list", s.HandleListConfigsJson)
	http.HandleFunc("/yaml/list", s.HandleListConfigsYaml)
	http.HandleFunc("/json/get", s.HandleGetKubeConfigsJson)
	http.HandleFunc("/yaml/get", s.HandleGetKubeConfigsYaml)
	http.HandleFunc("/", s.HandleIndex)
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	s.setupRoutes()

	s.Logger.Info("Server starting", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		return errorx.Decorate(err, "failed to start server")
	}

	return nil
}
