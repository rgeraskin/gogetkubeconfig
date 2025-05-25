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
	"github.com/rgeraskin/gogetkubeconfig/internal/server"
	"github.com/rgeraskin/gogetkubeconfig/internal/testutil"
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

	// Create web templates directory for template testing
	tempDir := t.TempDir()
	webDir := filepath.Join(tempDir, "web", "templates")
	err := os.MkdirAll(webDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create web templates directory: %v", err)
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

	// Create logger
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Reduce test noise

	// Create app config
	appConfig, err := newAppConfig(logger)
	if err != nil {
		t.Fatalf("Failed to create app config: %v", err)
	}

	// Create server
	srv, err := server.NewServer(&appConfig.Server)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create HTTP test server instead of starting a real server
	mux := http.NewServeMux()
	mux.HandleFunc("/json/list", srv.HandleListConfigsJson)
	mux.HandleFunc("/yaml/list", srv.HandleListConfigsYaml)
	mux.HandleFunc("/json/get", srv.HandleGetKubeConfigsJson)
	mux.HandleFunc("/yaml/get", srv.HandleGetKubeConfigsYaml)
	mux.HandleFunc("/", srv.HandleIndex)

	testServer := httptest.NewServer(mux)
	defer testServer.Close() // This ensures the server is properly shut down

	baseURL := testServer.URL

	// Helper functions for content checks
	checkConfigList := func(format string, unmarshal func([]byte, interface{}) error) func(t *testing.T, body []byte) {
		return func(t *testing.T, body []byte) {
			var configs []string
			err := unmarshal(body, &configs)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", format, err)
				return
			}
			if len(configs) != 5 { // Now using valid-configs directory with 5 files
				t.Errorf("Expected 5 configs, got %d", len(configs))
			}
			// Check if expected configs are present (only for detailed check)
			if format == "JSON" {
				expectedConfigs := map[string]bool{
					"dev":              false,
					"prod":             false,
					"integration-dev":  false,
					"integration-prod": false,
					"valid-test":       false,
				}
				for _, config := range configs {
					if _, exists := expectedConfigs[config]; exists {
						expectedConfigs[config] = true
					}
				}
				for config, found := range expectedConfigs {
					if !found {
						t.Errorf("Expected config %s not found", config)
					}
				}
			}
		}
	}

	checkSpecificConfig := func(format string, unmarshal func([]byte, interface{}) error, expectedCluster string) func(t *testing.T, body []byte) {
		return func(t *testing.T, body []byte) {
			var kubeConfig server.KubeConfig
			err := unmarshal(body, &kubeConfig)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", format, err)
				return
			}
			if len(kubeConfig.Clusters) != 1 {
				t.Errorf("Expected 1 cluster, got %d", len(kubeConfig.Clusters))
			}
			if kubeConfig.Clusters[0].Name != expectedCluster {
				t.Errorf(
					"Expected cluster name '%s', got %s",
					expectedCluster,
					kubeConfig.Clusters[0].Name,
				)
			}
		}
	}

	checkAllConfigs := func(format string, unmarshal func([]byte, interface{}) error) func(t *testing.T, body []byte) {
		return func(t *testing.T, body []byte) {
			var kubeConfig server.KubeConfig
			err := unmarshal(body, &kubeConfig)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", format, err)
				return
			}
			if len(kubeConfig.Clusters) != 5 {
				t.Errorf("Expected 5 clusters, got %d", len(kubeConfig.Clusters))
			}
			if len(kubeConfig.Contexts) != 5 {
				t.Errorf("Expected 5 contexts, got %d", len(kubeConfig.Contexts))
			}
			if len(kubeConfig.Users) != 5 {
				t.Errorf("Expected 5 users, got %d", len(kubeConfig.Users))
			}
		}
	}

	// Test cases
	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
		contentCheck   func(t *testing.T, body []byte)
	}{
		{
			name:           "list configs JSON",
			endpoint:       "/json/list",
			expectedStatus: http.StatusOK,
			contentCheck:   checkConfigList("JSON", json.Unmarshal),
		},
		{
			name:           "list configs YAML",
			endpoint:       "/yaml/list",
			expectedStatus: http.StatusOK,
			contentCheck:   checkConfigList("YAML", yaml.Unmarshal),
		},
		{
			name:           "get specific config JSON",
			endpoint:       "/json/get?name=integration-dev",
			expectedStatus: http.StatusOK,
			contentCheck:   checkSpecificConfig("JSON", json.Unmarshal, "integration-dev-cluster"),
		},
		{
			name:           "get all configs JSON",
			endpoint:       "/json/get",
			expectedStatus: http.StatusOK,
			contentCheck:   checkAllConfigs("JSON", json.Unmarshal),
		},
		{
			name:           "get nonexistent config",
			endpoint:       "/json/get?name=nonexistent",
			expectedStatus: http.StatusNotFound,
			contentCheck: func(t *testing.T, body []byte) {
				// Just check that we got an error response
				if len(body) == 0 {
					t.Error("Expected error message in response body")
				}
			},
		},
		{
			name:           "index page",
			endpoint:       "/",
			expectedStatus: http.StatusOK,
			contentCheck: func(t *testing.T, body []byte) {
				content := string(body)
				if !strings.Contains(content, "Available Configs") {
					t.Error("Expected index page to contain 'Available Configs'")
				}
				if !strings.Contains(content, "integration-dev") {
					t.Error("Expected index page to contain 'integration-dev'")
				}
				if !strings.Contains(content, "integration-prod") {
					t.Error("Expected index page to contain 'integration-prod'")
				}
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := baseURL + tc.endpoint
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to make request to %s: %v", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			body := make([]byte, 0)
			buffer := make([]byte, 1024)
			for {
				n, err := resp.Body.Read(buffer)
				if n > 0 {
					body = append(body, buffer[:n]...)
				}
				if err != nil {
					break
				}
			}

			if tc.contentCheck != nil {
				tc.contentCheck(t, body)
			}
		})
	}
}
