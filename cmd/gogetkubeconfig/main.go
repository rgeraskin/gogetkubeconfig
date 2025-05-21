package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/joomcode/errorx"
	"github.com/rgeraskin/gogetkubeconfig/internal/server"
)

// default values
const (
	defaultPort               = "8080"
	defaultConfigsDir         = "./configs"
	defaultDefaultsConfigName = "defaults.yaml"
)

// AppConfig represents the application configuration
type AppConfig struct {
	server.Server
	Port string
}

// newConfig creates a new app Config
func newAppConfig(logger *log.Logger) (*AppConfig, error) {
	config := &AppConfig{
		Server: server.Server{
			Logger: logger,
		},
	}

	config.ConfigsDir = os.Getenv("CONFIGS_DIR")
	if config.ConfigsDir == "" {
		config.ConfigsDir = defaultConfigsDir
	}

	defaultsConfigFileName := os.Getenv("DEFAULTS_CONFIG")
	if defaultsConfigFileName == "" {
		defaultsConfigFileName = defaultDefaultsConfigName
	}

	config.Port = os.Getenv("PORT")
	if config.Port == "" {
		config.Port = defaultPort
	}

	// validate config
	if err := validateKubeConfigs(config); err != nil {
		return nil, errorx.Decorate(err, "invalid config")
	}

	return config, nil
}

func validateKubeConfigs(c *AppConfig) error {
	info, err := os.Stat(c.ConfigsDir)

	c.Logger.Debug("Checking if config directory exists", "path", c.ConfigsDir)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", c.ConfigsDir)
	}

	// unexpected error
	if err != nil {
		return errorx.Decorate(err, "unexpected error")
	}

	c.Logger.Debug("Checking if config directory is a directory", "path", c.ConfigsDir)
	if !info.IsDir() {
		return fmt.Errorf("config directory is not a directory: %s", c.ConfigsDir)
	}

	c.Logger.Debug("Checking if configs in dir is readable", "path", c.ConfigsDir)
	files, err := os.ReadDir(c.ConfigsDir)
	if err != nil {
		return errorx.Decorate(err, "can't read configs directory")
	}

	for _, file := range files {
		filePath := filepath.Join(c.ConfigsDir, file.Name())
		c.Logger.Debug("Checking if config file is valid kubeconfig", "path", filePath)

		_, err := server.NewKubeConfig(filePath, c.Logger)
		if err != nil {
			return errorx.Decorate(err, "kubeconfig file is invalid: path=%s", filePath)
		}
	}

	return nil
}

func NewLogger() *log.Logger {
	logger := log.New(os.Stderr)
	logLevel := os.Getenv("DEBUG")
	if logLevel != "" {
		logger.SetLevel(log.DebugLevel)
	}
	logger.SetReportTimestamp(true)
	return logger
}

func main() {
	logger := NewLogger()

	// Configuration
	logger.Info("Creating config")
	appConfig, err := newAppConfig(logger)
	if err != nil {
		logger.Fatalf("Failed to configure application: %+v", err)
	}

	// Effective config
	logger.Info("Effective application config", "config", appConfig)

	// Create server
	logger.Info("Creating server")
	server, err := server.NewServer(&appConfig.Server)
	if err != nil {
		logger.Fatalf("Failed to initialize server: %+v", err)
	}

	// Start server
	logger.Info("Starting server", "port", appConfig.Port)
	if err := server.Start(appConfig.Port); err != nil {
		logger.Fatalf("Server failed: %+v", err)
	}
}
