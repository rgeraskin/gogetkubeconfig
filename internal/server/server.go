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
	WebDir     string
	Logger     *log.Logger
}

// NewServer creates a new server instance
func NewServer(appConfig *Server) (*Server, error) {
	server := &Server{
		ConfigsDir: appConfig.ConfigsDir,
		WebDir:     appConfig.WebDir,
		Logger:     appConfig.Logger,
	}

	// Check that index can be generated
	err := server.TemplateIndex(nil)
	if err != nil {
		return nil, errorx.Decorate(err, "can't generate index page")
	}
	return server, nil
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

func (s *Server) TemplateIndex(w http.ResponseWriter) error {
	// html template with list of available configs
	templatePath := filepath.Join(s.WebDir, "index.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return errorx.Decorate(err, "failed to parse index template file")
	}
	names, err := s.ListConfigs()
	if err != nil {
		return errorx.Decorate(err, "failed to list configs in dir")
	}
	vals := map[string][]string{
		"names": names,
	}

	// Only execute the template if the writer is not nil
	if w != nil {
		err = tmpl.Execute(w, vals)
		if err != nil {
			return errorx.Decorate(err, "failed to execute index template")
		}
	}

	return nil
}

// Index handles the root route
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	err := s.TemplateIndex(w)
	if err != nil {
		s.Logger.Error(err)
		http.Error(w, "Failed to template index", http.StatusInternalServerError)
	}
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

// getConfigNames returns all available config names from the configs directory
func (s *Server) getConfigNames() ([]string, error) {
	s.Logger.Debug("Checking if configs in dir is readable", "path", s.ConfigsDir)
	fileNames, err := os.ReadDir(s.ConfigsDir)
	if err != nil {
		return nil, errorx.Decorate(err, "failed to read configs directory")
	}

	configNames := []string{}
	for _, file := range fileNames {
		configNames = append(
			configNames,
			strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		)
	}
	return configNames, nil
}

// getRequestedConfigNames extracts requested config names from query parameters
func (s *Server) getRequestedConfigNames(r *http.Request, allConfigNames []string) []string {
	names := r.URL.Query()["name"]
	if len(names) == 0 {
		s.Logger.Info("No config names provided, getting all configs")
		return allConfigNames
	}
	s.Logger.Info("Getting configs", "names", names)
	return names
}

// validateConfigExists checks if a config name exists in the available configs
func (s *Server) validateConfigExists(name string, configNames []string) error {
	if !slices.Contains(configNames, name) {
		return errorx.InternalError.New("kubeconfig not found: %s", name)
	}
	return nil
}

// loadAndMergeConfigs loads and merges multiple kubeconfigs
func (s *Server) loadAndMergeConfigs(names []string, configNames []string) (interface{}, error) {
	// Create empty kubeconfig
	kubeConfig, err := NewKubeConfig("", s.Logger)
	if err != nil {
		return nil, errorx.Decorate(err, "failed to create empty kubeconfig")
	}

	s.Logger.Debug("Empty kubeconfig", "kubeconfig", kubeConfig)

	// For each requested config
	for _, name := range names {
		// Validate config exists
		if err := s.validateConfigExists(name, configNames); err != nil {
			return nil, err
		}

		filePath := filepath.Join(s.ConfigsDir, name+".yaml")
		s.Logger.Debug("Reading kubeconfig", "path", filePath)
		kubeConfigNew, err := NewKubeConfig(filePath, s.Logger)
		if err != nil {
			return nil, errorx.Decorate(err, "failed to read kubeconfig: %s", filePath)
		}

		kubeConfig, err = mergeKubeConfigs(kubeConfig, kubeConfigNew)
		if err != nil {
			return nil, errorx.Decorate(err, "failed to merge kubeconfig: %s", filePath)
		}
	}

	return kubeConfig, nil
}

// GetKubeConfigs returns multiple kubeconfigs
func (s *Server) HandleGetKubeConfigs(
	w http.ResponseWriter,
	r *http.Request,
	encoder func(io.Writer) Encoder,
) {
	// Get all available config names
	configNames, err := s.getConfigNames()
	if err != nil {
		s.Logger.Error("Failed to get config names", "error", err)
		http.Error(
			w,
			"Failed to read configs directory: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	// Get requested config names from query parameters
	requestedNames := s.getRequestedConfigNames(r, configNames)

	// Load and merge the requested configs
	kubeConfig, err := s.loadAndMergeConfigs(requestedNames, configNames)
	if err != nil {
		s.Logger.Error("Failed to load and merge configs", "error", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return the merged config
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
