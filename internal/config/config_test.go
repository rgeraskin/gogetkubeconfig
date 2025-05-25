package config

import (
	"os"
	"testing"

	"github.com/charmbracelet/log"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectedPort  string
		expectedDir   string
		expectedWeb   string
		expectedDebug bool
		wantErr       bool
	}{
		{
			name:          "default configuration",
			envVars:       map[string]string{},
			expectedPort:  DefaultPort,
			expectedDir:   DefaultConfigsDir,
			expectedWeb:   DefaultWebDir,
			expectedDebug: false,
			wantErr:       false,
		},
		{
			name: "custom configuration",
			envVars: map[string]string{
				"PORT":        "9090",
				"CONFIGS_DIR": "/custom/configs",
				"WEB_DIR":     "/custom/web",
				"DEBUG":       "true",
			},
			expectedPort:  "9090",
			expectedDir:   "/custom/configs",
			expectedWeb:   "/custom/web",
			expectedDebug: true,
			wantErr:       false,
		},
		{
			name: "partial custom configuration",
			envVars: map[string]string{
				"PORT":  "8888",
				"DEBUG": "false",
			},
			expectedPort:  "8888",
			expectedDir:   DefaultConfigsDir,
			expectedWeb:   DefaultWebDir,
			expectedDebug: false,
			wantErr:       false,
		},
		{
			name: "invalid debug value",
			envVars: map[string]string{
				"DEBUG": "invalid",
			},
			expectedPort:  DefaultPort,
			expectedDir:   DefaultConfigsDir,
			expectedWeb:   DefaultWebDir,
			expectedDebug: false, // Should default to false for invalid values
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			envKeys := []string{"PORT", "CONFIGS_DIR", "WEB_DIR", "DEBUG"}
			originalValues := make(map[string]string)
			for _, key := range envKeys {
				originalValues[key] = os.Getenv(key)
				os.Unsetenv(key)
			}
			defer func() {
				// Restore original environment
				for _, key := range envKeys {
					if val, exists := originalValues[key]; exists && val != "" {
						os.Setenv(key, val)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := NewConfig()

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

			if cfg.Port != tt.expectedPort {
				t.Errorf("Expected Port %q, got %q", tt.expectedPort, cfg.Port)
			}

			if cfg.ConfigsDir != tt.expectedDir {
				t.Errorf("Expected ConfigsDir %q, got %q", tt.expectedDir, cfg.ConfigsDir)
			}

			if cfg.WebDir != tt.expectedWeb {
				t.Errorf("Expected WebDir %q, got %q", tt.expectedWeb, cfg.WebDir)
			}

			if cfg.Debug != tt.expectedDebug {
				t.Errorf("Expected Debug %v, got %v", tt.expectedDebug, cfg.Debug)
			}

			if cfg.Logger == nil {
				t.Error("Expected Logger to be set")
			}
		})
	}
}

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name     string
		debug    bool
		expected log.Level
	}{
		{
			name:     "debug logger",
			debug:    true,
			expected: log.DebugLevel,
		},
		{
			name:     "info logger",
			debug:    false,
			expected: log.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := createLogger(tt.debug)

			if logger == nil {
				t.Fatal("Expected logger to be created, got nil")
			}

			if logger.GetLevel() != tt.expected {
				t.Errorf("Expected log level %v, got %v", tt.expected, logger.GetLevel())
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "environment variable not set",
			key:          "TEST_VAR_UNSET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "environment variable empty",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the environment variable
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "1 value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "0 value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "invalid value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			expected:     true, // Should return default
		},
		{
			name:         "empty value",
			key:          "TEST_BOOL_EMPTY",
			defaultValue: true,
			envValue:     "",
			expected:     true, // Should return default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the environment variable
			originalValue := os.Getenv(tt.key)
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.key, originalValue)
				} else {
					os.Unsetenv(tt.key)
				}
			}()

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultValue)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
