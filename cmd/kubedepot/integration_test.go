package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/kubedepot/internal/config"
	"github.com/rgeraskin/kubedepot/internal/server"
	"github.com/rgeraskin/kubedepot/internal/testutil"
	"gopkg.in/yaml.v3"
)

// TestIntegration_ServerEndpoints tests the full server integration
func TestIntegration_ServerEndpoints(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test environment using the valid configs directory (no file copying!)
	configsDir := testutil.GetValidKubeConfigsDir(t)

	// Create web directory for template testing
	tempDir := t.TempDir()
	webDir := filepath.Join(tempDir, "web")
	err := os.MkdirAll(webDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create web directory: %v", err)
	}

	// Copy index template from testdata for template testing
	testutil.CopyTestTemplate(t, webDir, "index.html", "index.html")

	// Set environment variables for the test
	originalConfigsDir := os.Getenv("CONFIGS_DIR")
	originalPort := os.Getenv("PORT")
	defer func() {
		if originalConfigsDir != "" {
			os.Setenv("CONFIGS_DIR", originalConfigsDir)
		} else {
			os.Unsetenv("CONFIGS_DIR")
		}
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	os.Setenv("CONFIGS_DIR", configsDir)
	os.Setenv("PORT", "0") // Use port 0 for testing (will be assigned automatically)

	// Change working directory to make templates accessible
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Create app config using the new config package
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("Failed to create app config: %v", err)
	}

	// Set log level to reduce test noise
	cfg.Logger.SetLevel(log.ErrorLevel)

	// Create server configuration
	serverConfig := &server.Server{
		ConfigsDir: cfg.ConfigsDir,
		WebDir:     cfg.WebDir,
		Logger:     cfg.Logger,
	}

	// Create server
	srv, err := server.NewServer(serverConfig)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test endpoints using httptest
	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		checkContent   func(t *testing.T, body string)
	}{
		{
			name:           "JSON list endpoint",
			endpoint:       "/json/list",
			expectedStatus: http.StatusOK,
			checkContent: func(t *testing.T, body string) {
				var configs []string
				if err := json.Unmarshal([]byte(body), &configs); err != nil {
					t.Errorf("Failed to parse JSON response: %v", err)
				}
				if len(configs) == 0 {
					t.Error("Expected at least one config in response")
				}
			},
		},
		{
			name:           "YAML list endpoint",
			endpoint:       "/yaml/list",
			expectedStatus: http.StatusOK,
			checkContent: func(t *testing.T, body string) {
				var configs []string
				if err := yaml.Unmarshal([]byte(body), &configs); err != nil {
					t.Errorf("Failed to parse YAML response: %v", err)
				}
				if len(configs) == 0 {
					t.Error("Expected at least one config in response")
				}
			},
		},
		{
			name:           "JSON get endpoint",
			endpoint:       "/json/get",
			expectedStatus: http.StatusOK,
			checkContent: func(t *testing.T, body string) {
				var kubeConfig map[string]interface{}
				if err := json.Unmarshal([]byte(body), &kubeConfig); err != nil {
					t.Errorf("Failed to parse JSON kubeconfig: %v", err)
				}
				if kubeConfig["apiVersion"] != "v1" {
					t.Error("Expected apiVersion to be v1")
				}
			},
		},
		{
			name:           "Index endpoint",
			endpoint:       "/",
			expectedStatus: http.StatusOK,
			checkContent: func(t *testing.T, body string) {
				if !strings.Contains(body, "Available Configs") {
					t.Error("Expected index page to contain 'Available Configs'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			// Route the request to the appropriate handler
			switch tt.endpoint {
			case "/json/list":
				srv.HandleListConfigsJson(w, req)
			case "/yaml/list":
				srv.HandleListConfigsYaml(w, req)
			case "/json/get":
				srv.HandleGetKubeConfigsJson(w, req)
			case "/":
				srv.HandleIndex(w, req)
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkContent != nil {
				tt.checkContent(t, w.Body.String())
			}
		})
	}
}
