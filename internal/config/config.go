package config

import (
	"os"
	"strconv"

	"github.com/charmbracelet/log"
)

// Config represents the application configuration
type Config struct {
	Port       string
	ConfigsDir string
	WebDir     string
	Debug      bool
	Logger     *log.Logger
}

// Default values
const (
	DefaultPort       = "8080"
	DefaultConfigsDir = "./configs"
	DefaultWebDir     = "./web"
)

// NewConfig creates a new configuration from environment variables
func NewConfig() (*Config, error) {
	config := &Config{
		Port:       getEnvOrDefault("PORT", DefaultPort),
		ConfigsDir: getEnvOrDefault("CONFIGS_DIR", DefaultConfigsDir),
		WebDir:     getEnvOrDefault("WEB_DIR", DefaultWebDir),
		Debug:      getEnvBool("DEBUG", false),
	}

	// Create logger based on configuration
	config.Logger = createLogger(config.Debug)

	return config, nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns environment variable as boolean or default
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// createLogger creates a logger with appropriate level
func createLogger(debug bool) *log.Logger {
	logger := log.New(os.Stderr)
	if debug {
		logger.SetLevel(log.DebugLevel)
	}
	logger.SetReportTimestamp(true)
	return logger
}
