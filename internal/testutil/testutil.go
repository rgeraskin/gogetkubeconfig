package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// GetProjectRoot finds the project root by looking for go.mod
func GetProjectRoot(t *testing.T) string {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Walk up the directory tree to find go.mod
	projectRoot := currentDir
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatalf("Could not find project root (go.mod not found)")
		}
		projectRoot = parent
	}
	return projectRoot
}

// GetTestDataDir returns the path to the testdata directory
func GetTestDataDir(t *testing.T) string {
	return filepath.Join(GetProjectRoot(t), "testdata")
}

// GetValidKubeConfigsDir returns the path to valid kubeconfigs (for "get all" operations)
func GetValidKubeConfigsDir(t *testing.T) string {
	return filepath.Join(GetTestDataDir(t), "valid-configs")
}

// GetInvalidKubeConfigsDir returns the path to invalid kubeconfigs (for error testing)
func GetInvalidKubeConfigsDir(t *testing.T) string {
	return filepath.Join(GetTestDataDir(t), "invalid-configs")
}

// GetMixedKubeConfigsDir returns the path to mixed kubeconfigs (for comprehensive listing)
func GetMixedKubeConfigsDir(t *testing.T) string {
	return filepath.Join(GetTestDataDir(t), "mixed-configs")
}

// GetTestTemplatesDir returns the path to the testdata templates directory
func GetTestTemplatesDir(t *testing.T) string {
	return filepath.Join(GetTestDataDir(t), "templates")
}

// LoadTestData loads a file from the testdata directory relative to the project root
func LoadTestData(t *testing.T, filename string) []byte {
	testDataPath := filepath.Join(GetTestDataDir(t), filename)
	data, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to read test data file %s: %v", filename, err)
	}
	return data
}

// CopyTestKubeConfigs copies kubeconfig test files to a target directory
func CopyTestKubeConfigs(t *testing.T, targetDir string, configs map[string]string) {
	for filename, testDataFile := range configs {
		data := LoadTestData(t, filepath.Join("kubeconfigs", testDataFile))
		err := os.WriteFile(filepath.Join(targetDir, filename), data, 0644)
		if err != nil {
			t.Fatalf("Failed to write test kubeconfig %s: %v", filename, err)
		}
	}
}

// CopyTestTemplate copies a template file to a target directory
func CopyTestTemplate(t *testing.T, targetDir, filename, testDataFile string) {
	data := LoadTestData(t, filepath.Join("templates", testDataFile))
	err := os.WriteFile(filepath.Join(targetDir, filename), data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test template %s: %v", filename, err)
	}
}
