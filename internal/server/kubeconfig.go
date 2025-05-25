package server

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/joomcode/errorx"
	"gopkg.in/yaml.v3"
)

const (
	kubeConfigApiVersion     = "v1"
	kubeConfigKind           = "Config"
	kubeConfigCurrentContext = "pp-dev"
)

// KubeConfig represents a kubeconfig file
type KubeConfig struct {
	ApiVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Clusters   []struct {
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
			Server                   string `yaml:"server" json:"server"`
		} `yaml:"cluster" json:"cluster"`
		Name string `yaml:"name" json:"name"`
	} `yaml:"clusters" json:"clusters"`
	Contexts []struct {
		Context struct {
			Cluster string `yaml:"cluster" json:"cluster"`
			User    string `yaml:"user" json:"user"`
		} `yaml:"context" json:"context"`
		Name string `yaml:"name" json:"name"`
	} `yaml:"contexts" json:"contexts"`
	CurrentContext string `yaml:"current-context" json:"current-context"`
	Users          []struct {
		User any    `yaml:"user" json:"user"`
		Name string `yaml:"name" json:"name"`
	} `yaml:"users" json:"users"`
}

// NewKubeConfig creates a new KubeConfig with default values
func NewKubeConfig(filePath string, logger *log.Logger) (*KubeConfig, error) {
	kubeConfig := &KubeConfig{}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, errorx.Decorate(err, "can't read kubeconfig file")
		}
		err = yaml.Unmarshal(data, &kubeConfig)
		if err != nil {
			return nil, errorx.Decorate(err, "can't parse kubeconfig file")
		}
	} else {
		logger.Debug("No kubeconfig file provided, using empty kubeconfig")
	}

	return kubeConfig, nil
}

// Validate checks if the kubeconfig has required fields
func (k *KubeConfig) Validate() error {
	if len(k.Clusters) == 0 {
		return errorx.InternalError.New("kubeconfig has no clusters")
	}
	if len(k.Contexts) == 0 {
		return errorx.InternalError.New("kubeconfig has no contexts")
	}
	if len(k.Users) == 0 {
		return errorx.InternalError.New("kubeconfig has no users")
	}
	return nil
}

// HasDuplicateNames checks if this config has duplicate names with another config
func (k *KubeConfig) HasDuplicateNames(other *KubeConfig) error {
	// Check cluster name duplicates
	if len(k.Clusters) > 0 && len(other.Clusters) > 0 &&
		other.Clusters[0].Name == k.Clusters[0].Name {
		return errorx.InternalError.New("kubeconfig has duplicate cluster name")
	}

	// Check context name duplicates
	if len(k.Contexts) > 0 && len(other.Contexts) > 0 &&
		other.Contexts[0].Name == k.Contexts[0].Name {
		return errorx.InternalError.New("kubeconfig has duplicate context name")
	}

	// Check user name duplicates
	if len(k.Users) > 0 && len(other.Users) > 0 &&
		other.Users[0].Name == k.Users[0].Name {
		return errorx.InternalError.New("kubeconfig has duplicate user name")
	}

	return nil
}

// HasMultipleEntries checks if the config has more than one cluster, context, or user
func (k *KubeConfig) HasMultipleEntries() error {
	if len(k.Clusters) > 1 {
		return errorx.InternalError.New("kubeconfig has more than one cluster")
	}
	if len(k.Contexts) > 1 {
		return errorx.InternalError.New("kubeconfig has more than one context")
	}
	if len(k.Users) > 1 {
		return errorx.InternalError.New("kubeconfig has more than one user")
	}
	return nil
}

// mergeKubeConfigs merges two kubeconfigs into a new one
func mergeKubeConfigs(config1 *KubeConfig, config2 *KubeConfig) (*KubeConfig, error) {
	merged := &KubeConfig{
		ApiVersion: kubeConfigApiVersion,
		Kind:       kubeConfigKind,
	}

	// Validate config2 has required fields
	if err := config2.Validate(); err != nil {
		return nil, err
	}

	// Check for duplicates
	if err := config1.HasDuplicateNames(config2); err != nil {
		return nil, err
	}

	// Check for multiple entries in config2
	if err := config2.HasMultipleEntries(); err != nil {
		return nil, err
	}

	// Merge the configs
	merged.Clusters = append(config1.Clusters, config2.Clusters...)
	merged.Contexts = append(config1.Contexts, config2.Contexts...)
	merged.Users = append(config1.Users, config2.Users...)

	// Set current context
	if config1.CurrentContext == "" {
		merged.CurrentContext = config2.CurrentContext
	} else {
		merged.CurrentContext = kubeConfigCurrentContext // default value
	}

	return merged, nil
}
