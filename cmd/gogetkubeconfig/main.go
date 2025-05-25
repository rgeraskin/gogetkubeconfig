package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/gogetkubeconfig/internal/server"
)

// default values
const (
	defaultPort               = "8080"
	defaultConfigsDir         = "./configs"
	defaultWebDir             = "./web"
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

	config.WebDir = os.Getenv("WEB_DIR")
	if config.WebDir == "" {
		config.WebDir = defaultWebDir
	}

	config.Port = os.Getenv("PORT")
	if config.Port == "" {
		config.Port = defaultPort
	}

	return config, nil
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
