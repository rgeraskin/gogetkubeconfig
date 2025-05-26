package server

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/joomcode/errorx"
	"gopkg.in/yaml.v3"
)

// Server represents the API server
type Server struct {
	ConfigsDir    string
	WebDir        string
	Logger        *log.Logger
	LoadedConfigs map[string]*KubeConfig // Pre-loaded configs to avoid file system changes affecting runtime
	EmbeddedFiles *embed.FS              // Optional embedded files for container deployment
}

// NewServer creates a new server instance
func NewServer(appConfig *Server) (*Server, error) {
	server := &Server{
		ConfigsDir:    appConfig.ConfigsDir,
		WebDir:        appConfig.WebDir,
		Logger:        appConfig.Logger,
		LoadedConfigs: make(map[string]*KubeConfig),
		EmbeddedFiles: appConfig.EmbeddedFiles,
	}

	// Load all configs on startup
	if err := server.loadAllConfigs(); err != nil {
		return nil, errorx.Decorate(err, "failed to load configs on startup")
	}

	// Test that all configs can be merged together
	if err := server.validateAllConfigsMergeable(); err != nil {
		return nil, errorx.Decorate(err, "configs cannot be merged together")
	}

	// Check that index can be generated
	err := server.TemplateIndex(nil)
	if err != nil {
		return nil, errorx.Decorate(err, "can't generate index page")
	}
	return server, nil
}

// Note: Start method moved to router.go for better separation of concerns

func (s *Server) TemplateIndex(w http.ResponseWriter) error {
	var tmpl *template.Template
	var err error

	// Try to load from WebDir first (for development)
	templatePath := filepath.Join(s.WebDir, "index.html")
	if _, err := os.Stat(templatePath); err == nil {
		tmpl, err = template.ParseFiles(templatePath)
		if err != nil {
			return errorx.Decorate(err, "failed to parse index template file from WebDir")
		}
	} else if s.EmbeddedFiles != nil {
		// Fall back to embedded files (for production/container)
		templateContent, err := s.EmbeddedFiles.ReadFile("kodata/web/index.html")
		if err != nil {
			return errorx.Decorate(err, "failed to read embedded index template")
		}
		tmpl, err = template.New("index.html").Parse(string(templateContent))
		if err != nil {
			return errorx.Decorate(err, "failed to parse embedded index template")
		}
	} else {
		return errorx.InternalError.New("neither WebDir nor EmbeddedFiles available for template")
	}

	names, err := s.listConfigs()
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
		s.handleHTTPError(w, err, "Failed to template index", http.StatusInternalServerError)
	}
}

// ListConfigsYaml lists all available kubeconfigs in YAML format
func (s *Server) HandleListConfigsYaml(w http.ResponseWriter, r *http.Request) {
	s.HandleListConfigs(w, r, createYAMLEncoder)
}

// ListConfigsJson lists all available kubeconfigs in JSON format
func (s *Server) HandleListConfigsJson(w http.ResponseWriter, r *http.Request) {
	s.HandleListConfigs(w, r, createJSONEncoder)
}

// GetKubeConfigsYaml returns a merged kubeconfig in YAML format
func (s *Server) HandleGetKubeConfigsYaml(w http.ResponseWriter, r *http.Request) {
	s.HandleGetKubeConfigs(w, r, createYAMLEncoder)
}

// GetKubeConfigsJson returns a merged kubeconfig in JSON format
func (s *Server) HandleGetKubeConfigsJson(w http.ResponseWriter, r *http.Request) {
	s.HandleGetKubeConfigs(w, r, createJSONEncoder)
}

// Define an Encoder interface
type Encoder interface {
	Encode(v interface{}) error
}

// Note: handleHTTPError moved to errors.go for better organization

// createYAMLEncoder creates a YAML encoder with consistent formatting
func createYAMLEncoder(w io.Writer) Encoder {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc
}

// createJSONEncoder creates a JSON encoder
func createJSONEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

// listConfigs returns all available config names from the loaded configs
func (s *Server) listConfigs() ([]string, error) {
	s.Logger.Info("Listing configs")
	configNames := make([]string, 0, len(s.LoadedConfigs))
	for name := range s.LoadedConfigs {
		configNames = append(configNames, name)
	}
	return configNames, nil
}

// HandleListConfigs returns all available kubeconfigs
func (s *Server) HandleListConfigs(
	w http.ResponseWriter,
	r *http.Request,
	encoder func(io.Writer) Encoder,
) {
	s.Logger.Info("HandleListConfigs")
	names, err := s.listConfigs()
	if err != nil {
		s.handleHTTPError(w, err, "Failed to list configs in dir", http.StatusInternalServerError)
		return
	}

	// w.Header().Set("Content-Type", "application/json")
	err = encoder(w).Encode(names)
	if err != nil {
		s.handleHTTPError(w, err, "Failed to encode configs list", http.StatusInternalServerError)
		return
	}

	s.Logger.Debug("Listed configs", "names", names)
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

// validateConfigExists checks if a config name exists in the loaded configs
func (s *Server) validateConfigExists(name string) error {
	if _, exists := s.LoadedConfigs[name]; !exists {
		return errorx.InternalError.New("kubeconfig not found: %s", name)
	}
	return nil
}

// loadAndMergeConfigs loads and merges multiple kubeconfigs from pre-loaded configs
func (s *Server) loadAndMergeConfigs(names []string) (interface{}, error) {
	// Create empty kubeconfig
	kubeConfig, err := NewKubeConfig("", s.Logger)
	if err != nil {
		return nil, errorx.Decorate(err, "failed to create empty kubeconfig")
	}

	s.Logger.Debug("Empty kubeconfig", "kubeconfig", kubeConfig)

	// For each requested config
	for _, name := range names {
		// Validate config exists
		if err := s.validateConfigExists(name); err != nil {
			return nil, err
		}

		s.Logger.Debug("Using pre-loaded kubeconfig", "name", name)
		kubeConfigNew := s.LoadedConfigs[name]

		kubeConfig, err = mergeKubeConfigs(kubeConfig, kubeConfigNew)
		if err != nil {
			return nil, errorx.Decorate(err, "failed to merge kubeconfig: %s", name)
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
	configNames, err := s.listConfigs()
	if err != nil {
		s.handleHTTPError(
			w,
			err,
			"Failed to read configs directory",
			http.StatusInternalServerError,
		)
		return
	}

	// Get requested config names from query parameters
	requestedNames := s.getRequestedConfigNames(r, configNames)

	// Load and merge the requested configs
	kubeConfig, err := s.loadAndMergeConfigs(requestedNames)
	if err != nil {
		s.handleError(w, err, "Failed to load and merge configs")
		return
	}

	// Return the merged config
	err = encoder(w).Encode(kubeConfig)
	if err != nil {
		s.handleHTTPError(w, err, "Failed to serialize kubeconfig", http.StatusInternalServerError)
		return
	}
}

// validateConfigsDirectory validates that the configs directory exists and is a directory
func (s *Server) validateConfigsDirectory() error {
	info, err := os.Stat(s.ConfigsDir)
	if err != nil && os.IsNotExist(err) {
		return errorx.InternalError.New("config directory does not exist: %s", s.ConfigsDir)
	}
	if err != nil {
		return errorx.Decorate(err, "unexpected error checking config directory")
	}
	if !info.IsDir() {
		return errorx.InternalError.New("config directory is not a directory: %s", s.ConfigsDir)
	}
	return nil
}

// readConfigFiles reads all files from the configs directory
func (s *Server) readConfigFiles() ([]os.DirEntry, error) {
	files, err := os.ReadDir(s.ConfigsDir)
	if err != nil {
		return nil, errorx.Decorate(err, "failed to read configs directory")
	}
	return files, nil
}

// loadSingleConfig loads a single config file and stores it in LoadedConfigs
func (s *Server) loadSingleConfig(file os.DirEntry) error {
	// Skip directories
	if file.IsDir() {
		s.Logger.Debug("Skipping directory", "file", file.Name())
		return nil
	}

	filePath := filepath.Join(s.ConfigsDir, file.Name())

	// Skip hidden files and Kubernetes ConfigMap metadata files
	fileName := file.Name()
	if strings.HasPrefix(fileName, "..") {
		s.Logger.Debug("Skipping Kubernetes ConfigMap metadata file", "file", fileName)
		return nil
	}

	// Additional check: verify the file path is actually a regular file
	// This handles cases where symlinks might not be detected properly by IsDir()
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		s.Logger.Debug("Skipping file due to stat error", "file", fileName, "error", err)
		return nil
	}
	if fileInfo.IsDir() {
		s.Logger.Debug("Skipping directory", "file", fileName)
		return nil
	}

	configName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	s.Logger.Debug("Loading config file", "path", filePath, "name", configName)

	kubeConfig, err := NewKubeConfig(filePath, s.Logger)
	if err != nil {
		return errorx.Decorate(err, "failed to load kubeconfig: %s", filePath)
	}

	s.LoadedConfigs[configName] = kubeConfig
	s.Logger.Debug("Successfully loaded config", "name", configName)
	return nil
}

// loadAllConfigs loads all config files from the configs directory into memory
func (s *Server) loadAllConfigs() error {
	s.Logger.Info("Loading all configs on startup", "configsDir", s.ConfigsDir)

	// Validate configs directory exists and is a directory
	if err := s.validateConfigsDirectory(); err != nil {
		return err
	}

	// Read all files from the configs directory
	files, err := s.readConfigFiles()
	if err != nil {
		return err
	}

	// Load each config file
	for _, file := range files {
		if err := s.loadSingleConfig(file); err != nil {
			return err
		}
	}

	s.Logger.Info("Successfully loaded all configs", "count", len(s.LoadedConfigs))
	return nil
}

// createEmptyKubeConfigForValidation creates an empty kubeconfig for merge validation
func (s *Server) createEmptyKubeConfigForValidation() (*KubeConfig, error) {
	mergedConfig, err := NewKubeConfig("", s.Logger)
	if err != nil {
		return nil, errorx.Decorate(err, "failed to create empty kubeconfig for merge test")
	}
	return mergedConfig, nil
}

// getAllConfigNames returns a slice of all loaded config names
func (s *Server) getAllConfigNames() []string {
	configNames := make([]string, 0, len(s.LoadedConfigs))
	for name := range s.LoadedConfigs {
		configNames = append(configNames, name)
	}
	return configNames
}

// mergeAllConfigsForValidation attempts to merge all loaded configs to test compatibility
func (s *Server) mergeAllConfigsForValidation(
	mergedConfig *KubeConfig,
	configNames []string,
) error {
	s.Logger.Debug("Testing merge of all configs", "configs", configNames)

	for name, config := range s.LoadedConfigs {
		s.Logger.Debug("Merging config for validation", "name", name)
		var err error
		mergedConfig, err = mergeKubeConfigs(mergedConfig, config)
		if err != nil {
			return errorx.Decorate(err, "failed to merge config '%s' during validation", name)
		}
	}
	return nil
}

// validateAllConfigsMergeable tests that all loaded configs can be merged together
func (s *Server) validateAllConfigsMergeable() error {
	s.Logger.Info("Validating that all configs can be merged together")

	if len(s.LoadedConfigs) == 0 {
		s.Logger.Warn("No configs loaded, skipping merge validation")
		return nil
	}

	// Create empty kubeconfig for merging
	mergedConfig, err := s.createEmptyKubeConfigForValidation()
	if err != nil {
		return err
	}

	// Get all config names for logging
	configNames := s.getAllConfigNames()

	// Try to merge all configs
	if err := s.mergeAllConfigsForValidation(mergedConfig, configNames); err != nil {
		return err
	}

	s.Logger.Info("Successfully validated that all configs can be merged together")
	return nil
}
