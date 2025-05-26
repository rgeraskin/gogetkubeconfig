package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/joomcode/errorx"
)

func TestNewKubeConfig(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)

	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		wantErr   bool
		validate  func(t *testing.T, kubeConfig *KubeConfig)
	}{
		{
			name: "empty kubeconfig",
			setupFunc: func(t *testing.T) string {
				return "" // Empty file path
			},
			wantErr: false,
			validate: func(t *testing.T, kubeConfig *KubeConfig) {
				if kubeConfig == nil {
					t.Fatal("Expected kubeconfig to be created")
				}
				// Empty kubeconfig should have zero-value fields
				if len(kubeConfig.Clusters) != 0 {
					t.Errorf("Expected 0 clusters, got %d", len(kubeConfig.Clusters))
				}
			},
		},
		{
			name: "valid kubeconfig file",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "test.yaml")

				validConfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: dGVzdC1jZXJ0
    server: https://test.example.com
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
				err := os.WriteFile(filePath, []byte(validConfig), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return filePath
			},
			wantErr: false,
			validate: func(t *testing.T, kubeConfig *KubeConfig) {
				if kubeConfig == nil {
					t.Fatal("Expected kubeconfig to be created")
				}
				if kubeConfig.ApiVersion != "v1" {
					t.Errorf("Expected ApiVersion 'v1', got %s", kubeConfig.ApiVersion)
				}
				if kubeConfig.Kind != "Config" {
					t.Errorf("Expected Kind 'Config', got %s", kubeConfig.Kind)
				}
				if len(kubeConfig.Clusters) != 1 {
					t.Errorf("Expected 1 cluster, got %d", len(kubeConfig.Clusters))
				}
				if kubeConfig.Clusters[0].Name != "test-cluster" {
					t.Errorf(
						"Expected cluster name 'test-cluster', got %s",
						kubeConfig.Clusters[0].Name,
					)
				}
				if len(kubeConfig.Contexts) != 1 {
					t.Errorf("Expected 1 context, got %d", len(kubeConfig.Contexts))
				}
				if len(kubeConfig.Users) != 1 {
					t.Errorf("Expected 1 user, got %d", len(kubeConfig.Users))
				}
				if kubeConfig.CurrentContext != "test-context" {
					t.Errorf(
						"Expected current-context 'test-context', got %s",
						kubeConfig.CurrentContext,
					)
				}
			},
		},
		{
			name: "nonexistent file",
			setupFunc: func(t *testing.T) string {
				return "/nonexistent/file.yaml"
			},
			wantErr: true,
			validate: func(t *testing.T, kubeConfig *KubeConfig) {
				// Should not reach here if wantErr is true
			},
		},
		{
			name: "invalid yaml file",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "invalid.yaml")

				invalidYaml := `invalid: yaml: [content`
				err := os.WriteFile(filePath, []byte(invalidYaml), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return filePath
			},
			wantErr: true,
			validate: func(t *testing.T, kubeConfig *KubeConfig) {
				// Should not reach here if wantErr is true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFunc(t)

			kubeConfig, err := NewKubeConfig(filePath, logger)

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

			tt.validate(t, kubeConfig)
		})
	}
}

func TestMergeKubeConfigs(t *testing.T) {
	tests := []struct {
		name     string
		config1  *KubeConfig
		config2  *KubeConfig
		wantErr  bool
		validate func(t *testing.T, merged *KubeConfig)
	}{
		{
			name: "merge empty with valid config",
			config1: &KubeConfig{
				ApiVersion: "v1",
				Kind:       "Config",
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{},
			},
			config2: &KubeConfig{
				ApiVersion: "v1",
				Kind:       "Config",
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{
						Cluster: struct {
							CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
							Server                   string `yaml:"server" json:"server"`
						}{
							CertificateAuthorityData: "dGVzdA==",
							Server:                   "https://test.example.com",
						},
						Name: "test-cluster",
					},
				},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{
					{
						Context: struct {
							Cluster string `yaml:"cluster" json:"cluster"`
							User    string `yaml:"user" json:"user"`
						}{
							Cluster: "test-cluster",
							User:    "test-user",
						},
						Name: "test-context",
					},
				},
				CurrentContext: "test-context",
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{
					{
						Name: "test-user",
						User: map[string]interface{}{"token": "test-token"},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, merged *KubeConfig) {
				if merged.ApiVersion != kubeConfigApiVersion {
					t.Errorf(
						"Expected ApiVersion %s, got %s",
						kubeConfigApiVersion,
						merged.ApiVersion,
					)
				}
				if merged.Kind != kubeConfigKind {
					t.Errorf("Expected Kind %s, got %s", kubeConfigKind, merged.Kind)
				}
				if len(merged.Clusters) != 1 {
					t.Errorf("Expected 1 cluster, got %d", len(merged.Clusters))
				}
				if len(merged.Contexts) != 1 {
					t.Errorf("Expected 1 context, got %d", len(merged.Contexts))
				}
				if len(merged.Users) != 1 {
					t.Errorf("Expected 1 user, got %d", len(merged.Users))
				}
			},
		},
		{
			name:    "config2 has no clusters",
			config1: &KubeConfig{},
			config2: &KubeConfig{
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{},
			},
			wantErr:  true,
			validate: func(t *testing.T, merged *KubeConfig) {},
		},
		{
			name:    "config2 has no contexts",
			config1: &KubeConfig{},
			config2: &KubeConfig{
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-cluster"},
				},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{},
			},
			wantErr:  true,
			validate: func(t *testing.T, merged *KubeConfig) {},
		},
		{
			name:    "config2 has no users",
			config1: &KubeConfig{},
			config2: &KubeConfig{
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-cluster"},
				},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-context"},
				},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{},
			},
			wantErr:  true,
			validate: func(t *testing.T, merged *KubeConfig) {},
		},
		{
			name: "duplicate cluster names",
			config1: &KubeConfig{
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "duplicate-cluster"},
				},
			},
			config2: &KubeConfig{
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "duplicate-cluster"},
				},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-context"},
				},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-user"},
				},
			},
			wantErr:  true,
			validate: func(t *testing.T, merged *KubeConfig) {},
		},
		{
			name:    "multiple clusters in config2",
			config1: &KubeConfig{},
			config2: &KubeConfig{
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "cluster1"},
					{Name: "cluster2"},
				},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-context"},
				},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-user"},
				},
			},
			wantErr:  true,
			validate: func(t *testing.T, merged *KubeConfig) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := mergeKubeConfigs(tt.config1, tt.config2)

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

			if merged == nil {
				t.Fatal("Expected merged config to be created")
			}

			tt.validate(t, merged)
		})
	}
}

func TestMergeKubeConfigs_CurrentContext(t *testing.T) {
	// Test current context handling
	tests := []struct {
		name               string
		config1CurrentCtx  string
		config2CurrentCtx  string
		expectedCurrentCtx string
	}{
		{
			name:               "config1 has no current context",
			config1CurrentCtx:  "",
			config2CurrentCtx:  "config2-context",
			expectedCurrentCtx: "config2-context",
		},
		{
			name:               "config1 has current context",
			config1CurrentCtx:  "config1-context",
			config2CurrentCtx:  "config2-context",
			expectedCurrentCtx: "config1-context", // Should use config1's context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config1 := &KubeConfig{
				CurrentContext: tt.config1CurrentCtx,
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{},
			}

			config2 := &KubeConfig{
				CurrentContext: tt.config2CurrentCtx,
				Clusters: []struct {
					Cluster struct {
						CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
						Server                   string `yaml:"server" json:"server"`
					} `yaml:"cluster" json:"cluster"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-cluster"},
				},
				Contexts: []struct {
					Context struct {
						Cluster string `yaml:"cluster" json:"cluster"`
						User    string `yaml:"user" json:"user"`
					} `yaml:"context" json:"context"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-context"},
				},
				Users: []struct {
					User any    `yaml:"user" json:"user"`
					Name string `yaml:"name" json:"name"`
				}{
					{Name: "test-user"},
				},
			}

			merged, err := mergeKubeConfigs(config1, config2)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if merged.CurrentContext != tt.expectedCurrentCtx {
				t.Errorf("Expected current context %s, got %s",
					tt.expectedCurrentCtx, merged.CurrentContext)
			}
		})
	}
}

// TestMergeKubeConfigs_DuplicateContexts tests duplicate context name detection
func TestMergeKubeConfigs_DuplicateContexts(t *testing.T) {
	config1 := &KubeConfig{
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "cluster1"},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "duplicate-context"},
		},
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "user1"},
		},
	}

	config2 := &KubeConfig{
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "cluster2"},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "duplicate-context"}, // Same name as config1
		},
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "user2"},
		},
	}

	_, err := mergeKubeConfigs(config1, config2)
	if err == nil {
		t.Error("Expected error for duplicate context names, got nil")
	}
	if !errorx.IsOfType(err, errorx.InternalError) {
		t.Errorf("Expected InternalError, got %T", err)
	}
}

// TestMergeKubeConfigs_DuplicateUsers tests duplicate user name detection
func TestMergeKubeConfigs_DuplicateUsers(t *testing.T) {
	config1 := &KubeConfig{
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "cluster1"},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "context1"},
		},
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "duplicate-user"},
		},
	}

	config2 := &KubeConfig{
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "cluster2"},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "context2"},
		},
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "duplicate-user"}, // Same name as config1
		},
	}

	_, err := mergeKubeConfigs(config1, config2)
	if err == nil {
		t.Error("Expected error for duplicate user names, got nil")
	}
	if !errorx.IsOfType(err, errorx.InternalError) {
		t.Errorf("Expected InternalError, got %T", err)
	}
}

// TestMergeKubeConfigs_MultipleContexts tests multiple contexts in config2
func TestMergeKubeConfigs_MultipleContexts(t *testing.T) {
	config1 := &KubeConfig{}
	config2 := &KubeConfig{
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "test-cluster"},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "context1"},
			{Name: "context2"}, // Multiple contexts
		},
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "test-user"},
		},
	}

	_, err := mergeKubeConfigs(config1, config2)
	if err == nil {
		t.Error("Expected error for multiple contexts in config2, got nil")
	}
	if !errorx.IsOfType(err, errorx.InternalError) {
		t.Errorf("Expected InternalError, got %T", err)
	}
}

// TestMergeKubeConfigs_MultipleUsers tests multiple users in config2
func TestMergeKubeConfigs_MultipleUsers(t *testing.T) {
	config1 := &KubeConfig{}
	config2 := &KubeConfig{
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "test-cluster"},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "test-context"},
		},
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{Name: "user1"},
			{Name: "user2"}, // Multiple users
		},
	}

	_, err := mergeKubeConfigs(config1, config2)
	if err == nil {
		t.Error("Expected error for multiple users in config2, got nil")
	}
	if !errorx.IsOfType(err, errorx.InternalError) {
		t.Errorf("Expected InternalError, got %T", err)
	}
}

// TestMergeKubeConfigs_SuccessfulMerge tests a successful merge with populated config1
func TestMergeKubeConfigs_SuccessfulMerge(t *testing.T) {
	config1 := &KubeConfig{
		ApiVersion: "v1",
		Kind:       "Config",
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Cluster: struct {
					CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
					Server                   string `yaml:"server" json:"server"`
				}{
					CertificateAuthorityData: "Y29uZmlnMQ==",
					Server:                   "https://config1.example.com",
				},
				Name: "config1-cluster",
			},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Context: struct {
					Cluster string `yaml:"cluster" json:"cluster"`
					User    string `yaml:"user" json:"user"`
				}{
					Cluster: "config1-cluster",
					User:    "config1-user",
				},
				Name: "config1-context",
			},
		},
		CurrentContext: "config1-context",
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Name: "config1-user",
				User: map[string]interface{}{"token": "config1-token"},
			},
		},
	}

	config2 := &KubeConfig{
		ApiVersion: "v1",
		Kind:       "Config",
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Cluster: struct {
					CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
					Server                   string `yaml:"server" json:"server"`
				}{
					CertificateAuthorityData: "Y29uZmlnMg==",
					Server:                   "https://config2.example.com",
				},
				Name: "config2-cluster",
			},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Context: struct {
					Cluster string `yaml:"cluster" json:"cluster"`
					User    string `yaml:"user" json:"user"`
				}{
					Cluster: "config2-cluster",
					User:    "config2-user",
				},
				Name: "config2-context",
			},
		},
		CurrentContext: "config2-context",
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Name: "config2-user",
				User: map[string]interface{}{"token": "config2-token"},
			},
		},
	}

	merged, err := mergeKubeConfigs(config1, config2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Validate merged config
	if merged.ApiVersion != kubeConfigApiVersion {
		t.Errorf("Expected ApiVersion %s, got %s", kubeConfigApiVersion, merged.ApiVersion)
	}
	if merged.Kind != kubeConfigKind {
		t.Errorf("Expected Kind %s, got %s", kubeConfigKind, merged.Kind)
	}
	if len(merged.Clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(merged.Clusters))
	}
	if len(merged.Contexts) != 2 {
		t.Errorf("Expected 2 contexts, got %d", len(merged.Contexts))
	}
	if len(merged.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(merged.Users))
	}
	if merged.CurrentContext != "config1-context" {
		t.Errorf(
			"Expected current context %s, got %s",
			"config1-context",
			merged.CurrentContext,
		)
	}

	// Validate that both configs are present
	clusterNames := make([]string, len(merged.Clusters))
	for i, cluster := range merged.Clusters {
		clusterNames[i] = cluster.Name
	}
	if !contains(clusterNames, "config1-cluster") || !contains(clusterNames, "config2-cluster") {
		t.Errorf("Expected both cluster names to be present, got %v", clusterNames)
	}
}

// TestNewKubeConfig_EdgeCases tests additional edge cases for NewKubeConfig
func TestNewKubeConfig_EdgeCases(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)

	t.Run("empty yaml file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "empty.yaml")

		err := os.WriteFile(filePath, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		kubeConfig, err := NewKubeConfig(filePath, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if kubeConfig == nil {
			t.Fatal("Expected kubeconfig to be created")
		}
		// Empty YAML should result in zero-value struct
		if len(kubeConfig.Clusters) != 0 {
			t.Errorf("Expected 0 clusters, got %d", len(kubeConfig.Clusters))
		}
	})

	t.Run("yaml with only comments", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "comments.yaml")

		commentOnlyYaml := `# This is a comment
# Another comment
# More comments`
		err := os.WriteFile(filePath, []byte(commentOnlyYaml), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		kubeConfig, err := NewKubeConfig(filePath, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if kubeConfig == nil {
			t.Fatal("Expected kubeconfig to be created")
		}
	})

	t.Run("complex nested user data", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "complex.yaml")

		complexConfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: dGVzdC1jZXJ0
    server: https://test.example.com
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
      - eks
      - get-token
      - --cluster-name
      - test-cluster
      env:
      - name: AWS_PROFILE
        value: default
`
		err := os.WriteFile(filePath, []byte(complexConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		kubeConfig, err := NewKubeConfig(filePath, logger)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if kubeConfig == nil {
			t.Fatal("Expected kubeconfig to be created")
		}
		if len(kubeConfig.Users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(kubeConfig.Users))
		}
		// Verify complex user data is preserved
		if kubeConfig.Users[0].User == nil {
			t.Error("Expected user data to be preserved")
		}
	})
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// BenchmarkNewKubeConfig benchmarks the NewKubeConfig function
func BenchmarkNewKubeConfig(b *testing.B) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)

	// Create a temporary kubeconfig file for benchmarking
	tempDir := b.TempDir()
	filePath := filepath.Join(tempDir, "benchmark.yaml")

	benchmarkConfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: dGVzdC1jZXJ0aWZpY2F0ZS1kYXRh
    server: https://benchmark.example.com
  name: benchmark-cluster
contexts:
- context:
    cluster: benchmark-cluster
    user: benchmark-user
  name: benchmark-context
current-context: benchmark-context
users:
- name: benchmark-user
  user:
    token: benchmark-token-12345
`
	err := os.WriteFile(filePath, []byte(benchmarkConfig), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewKubeConfig(filePath, logger)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkNewKubeConfig_Empty benchmarks NewKubeConfig with empty file path
func BenchmarkNewKubeConfig_Empty(b *testing.B) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewKubeConfig("", logger)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkMergeKubeConfigs benchmarks the mergeKubeConfigs function
func BenchmarkMergeKubeConfigs(b *testing.B) {
	config1 := &KubeConfig{
		ApiVersion: "v1",
		Kind:       "Config",
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Cluster: struct {
					CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
					Server                   string `yaml:"server" json:"server"`
				}{
					CertificateAuthorityData: "Y29uZmlnMQ==",
					Server:                   "https://config1.example.com",
				},
				Name: "config1-cluster",
			},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Context: struct {
					Cluster string `yaml:"cluster" json:"cluster"`
					User    string `yaml:"user" json:"user"`
				}{
					Cluster: "config1-cluster",
					User:    "config1-user",
				},
				Name: "config1-context",
			},
		},
		CurrentContext: "config1-context",
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Name: "config1-user",
				User: map[string]interface{}{"token": "config1-token"},
			},
		},
	}

	config2 := &KubeConfig{
		ApiVersion: "v1",
		Kind:       "Config",
		Clusters: []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
				Server                   string `yaml:"server" json:"server"`
			} `yaml:"cluster" json:"cluster"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Cluster: struct {
					CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
					Server                   string `yaml:"server" json:"server"`
				}{
					CertificateAuthorityData: "Y29uZmlnMg==",
					Server:                   "https://config2.example.com",
				},
				Name: "config2-cluster",
			},
		},
		Contexts: []struct {
			Context struct {
				Cluster string `yaml:"cluster" json:"cluster"`
				User    string `yaml:"user" json:"user"`
			} `yaml:"context" json:"context"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Context: struct {
					Cluster string `yaml:"cluster" json:"cluster"`
					User    string `yaml:"user" json:"user"`
				}{
					Cluster: "config2-cluster",
					User:    "config2-user",
				},
				Name: "config2-context",
			},
		},
		CurrentContext: "config2-context",
		Users: []struct {
			User any    `yaml:"user" json:"user"`
			Name string `yaml:"name" json:"name"`
		}{
			{
				Name: "config2-user",
				User: map[string]interface{}{"token": "config2-token"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mergeKubeConfigs(config1, config2)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
