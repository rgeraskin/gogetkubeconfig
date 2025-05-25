package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/gogetkubeconfig/internal/testutil"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name     string
		debug    string
		expected log.Level
	}{
		{
			name:     "default log level",
			debug:    "",
			expected: log.InfoLevel,
		},
		{
			name:     "debug log level",
			debug:    "true",
			expected: log.DebugLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.debug != "" {
				os.Setenv("DEBUG", tt.debug)
				defer os.Unsetenv("DEBUG")
			}

			logger := NewLogger()
			if logger == nil {
				t.Fatal("Expected logger to be created, got nil")
			}

			if logger.GetLevel() != tt.expected {
				t.Errorf("Expected log level %v, got %v", tt.expected, logger.GetLevel())
			}
		})
	}
}

func TestNewAppConfig(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Reduce noise in tests

	// Create a temporary directory structure for testing
	tempDir := t.TempDir()
	configsDir := filepath.Join(tempDir, "configs")
	err := os.MkdirAll(configsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test configs directory: %v", err)
	}

	// Copy a valid kubeconfig file for testing from testdata
	testConfigs := map[string]string{
		"test.yaml": "valid-test.yaml",
	}
	testutil.CopyTestKubeConfigs(t, configsDir, testConfigs)

	tests := []struct {
		name         string
		configsDir   string
		port         string
		wantErr      bool
		expectedDir  string
		expectedPort string
	}{
		{
			name:         "default configuration",
			configsDir:   configsDir, // Use the created test directory instead of default
			port:         "",
			wantErr:      false,
			expectedDir:  configsDir,
			expectedPort: defaultPort,
		},
		{
			name:         "custom configuration",
			configsDir:   configsDir,
			port:         "9090",
			wantErr:      false,
			expectedDir:  configsDir,
			expectedPort: "9090",
		},
		{
			name:         "invalid configs directory",
			configsDir:   "/nonexistent/directory",
			port:         "8080",
			wantErr:      false, // App config creation no longer validates directory
			expectedDir:  "/nonexistent/directory",
			expectedPort: "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.configsDir != "" {
				os.Setenv("CONFIGS_DIR", tt.configsDir)
				defer os.Unsetenv("CONFIGS_DIR")
			}
			if tt.port != "" {
				os.Setenv("PORT", tt.port)
				defer os.Unsetenv("PORT")
			}

			config, err := newAppConfig(logger)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config.ConfigsDir != tt.expectedDir {
				t.Errorf("Expected ConfigsDir %q, got %q", tt.expectedDir, config.ConfigsDir)
			}

			if config.Port != tt.expectedPort {
				t.Errorf("Expected Port %q, got %q", tt.expectedPort, config.Port)
			}

			if config.Logger == nil {
				t.Error("Expected Logger to be set")
			}
		})
	}
}
