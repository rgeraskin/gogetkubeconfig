package server

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/joomcode/errorx"
	"gopkg.in/yaml.v3"
)

// Server represents the API server
type Server struct {
	ConfigsDir string
	Logger     *log.Logger
}

// NewServer creates a new server instance
func NewServer(appConfig *Server) (*Server, error) {
	return &Server{
		ConfigsDir: appConfig.ConfigsDir,
		Logger:     appConfig.Logger,
	}, nil
}

// Start starts the http server
func (s *Server) Start(port string) error {
	// Setup routes
	http.HandleFunc("/json/list", s.HandleListConfigsJson)
	http.HandleFunc("/yaml/list", s.HandleListConfigsYaml)
	http.HandleFunc("/json/get", s.HandleGetKubeConfigsJson)
	http.HandleFunc("/yaml/get", s.HandleGetKubeConfigsYaml)
	http.HandleFunc("/", s.HandleIndex)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		return errorx.Decorate(err, "failed to start server")
	}

	return nil
}

// Index handles the root route
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// html template with list of available configs
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	names, err := s.ListConfigs()
	if err != nil {
		s.Logger.Error("Failed to list configs in dir", "error", err)
		http.Error(w, "Failed to list configs in dir", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]any{
		"Configs": names,
	})
}

// ListConfigsYaml lists all available kubeconfigs in YAML format
func (s *Server) HandleListConfigsYaml(w http.ResponseWriter, r *http.Request) {
	s.HandleListConfigs(w, r, func(w io.Writer) Encoder {
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		return enc
	})
}

// ListConfigsJson lists all available kubeconfigs in JSON format
func (s *Server) HandleListConfigsJson(w http.ResponseWriter, r *http.Request) {
	s.HandleListConfigs(w, r, func(w io.Writer) Encoder {
		return json.NewEncoder(w)
	})
}

// GetKubeConfigsYaml returns a merged kubeconfig in YAML format
func (s *Server) HandleGetKubeConfigsYaml(w http.ResponseWriter, r *http.Request) {
	s.HandleGetKubeConfigs(w, r, func(w io.Writer) Encoder {
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		return enc
	})
}

// GetKubeConfigsJson returns a merged kubeconfig in JSON format
func (s *Server) HandleGetKubeConfigsJson(w http.ResponseWriter, r *http.Request) {
	s.HandleGetKubeConfigs(w, r, func(w io.Writer) Encoder {
		return json.NewEncoder(w)
	})
}

// Define an Encoder interface
type Encoder interface {
	Encode(v interface{}) error
}

func (s *Server) ListConfigs() ([]string, error) {
	s.Logger.Info("Listing configs")

	s.Logger.Debug("Checking if configs in dir is readable", "path", s.ConfigsDir)
	files, err := os.ReadDir(s.ConfigsDir)
	if err != nil {
		return nil, errorx.Decorate(err, "failed to list configs in dir")
	}

	names := []string{}
	for _, file := range files {
		names = append(names, strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())))
	}

	return names, nil
}

// ListConfigs returns all available kubeconfigs
func (s *Server) HandleListConfigs(
	w http.ResponseWriter,
	r *http.Request,
	encoder func(io.Writer) Encoder,
) {
	s.Logger.Info("HandleListConfigs")
	names, err := s.ListConfigs()
	if err != nil {
		s.Logger.Error("Failed to list configs in dir", "error", err)
		http.Error(w, "Failed to list configs in dir", http.StatusInternalServerError)
		return
	}

	// w.Header().Set("Content-Type", "application/json")
	err = encoder(w).Encode(names)
	if err != nil {
		s.Logger.Error("Failed to encode configs list", "error", err)
		http.Error(
			w,
			"Failed to encode configs list: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	s.Logger.Debug("Listed configs", "names", names)
}

// GetKubeConfigs returns multiple kubeconfigs
func (s *Server) HandleGetKubeConfigs(
	w http.ResponseWriter,
	r *http.Request,
	encoder func(io.Writer) Encoder,
) {
	// Get list of all kubeconfig files in the configs directory
	s.Logger.Debug("Checking if configs in dir is readable", "path", s.ConfigsDir)
	fileNames, err := os.ReadDir(s.ConfigsDir)
	if err != nil {
		s.Logger.Error("Failed to read configs directory", "error", err)
		http.Error(
			w,
			"Failed to read configs directory: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	// Config name is the filename without the .yaml extension
	configNames := []string{}
	for _, file := range fileNames {
		configNames = append(
			configNames,
			strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		)
	}

	// Create empty kubeconfig
	kubeConfig, err := NewKubeConfig("", s.Logger)
	if err != nil {
		s.Logger.Error("Failed to create empty kubeconfig", "error", err)
		http.Error(
			w,
			"Failed to create empty kubeconfig: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	// Get list of requested config names from the query parameter
	names := r.URL.Query()["name"]
	if len(names) == 0 {
		s.Logger.Info("No config names provided, getting all configs")
		names = configNames
	} else {
		s.Logger.Info("Getting configs", "names", names)
		// kubeConfig.CurrentContext = names[0]
	}

	s.Logger.Debug("Empty kubeconfig", "kubeconfig", kubeConfig)

	// For each requested config
	for _, name := range names {
		// If name is not in files, return error
		if !slices.Contains(configNames, name) {
			s.Logger.Error("Kubeconfig not found", "name", name)
			http.Error(w, "Kubeconfig not found", http.StatusNotFound)
			return
		}

		filePath := filepath.Join(s.ConfigsDir, name+".yaml")
		s.Logger.Debug("Reading kubeconfig", "path", filePath)
		kubeConfigNew, err := NewKubeConfig(filePath, s.Logger)
		if err != nil {
			s.Logger.Error("Failed to read kubeconfig", "path", filePath, "error", err)
			http.Error(
				w,
				"Failed to read kubeconfig: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		kubeConfig, err = mergeKubeConfigs(kubeConfig, kubeConfigNew)
		if err != nil {
			s.Logger.Error("Failed to create kubeconfig bundle with", "name", name, "error", err)
			http.Error(
				w,
				"Failed to create kubeconfig bundle: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}

	// Return the merged config
	// w.Header().Set("Content-Type", "application/json")
	err = encoder(w).Encode(kubeConfig)
	if err != nil {
		s.Logger.Error("Failed to serialize kubeconfig", "error", err)
		http.Error(
			w,
			"Failed to serialize kubeconfig: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
}
