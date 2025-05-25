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
	ApiVersion string `yaml:"apiVersion"      json:"apiVersion"`
	Kind       string `yaml:"kind"            json:"kind"`
	Clusters   []struct {
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data" json:"certificate-authority-data"`
			Server                   string `yaml:"server" json:"server"`
		} `yaml:"cluster" json:"cluster"`
		Name string `yaml:"name" json:"name"`
	} `yaml:"clusters"        json:"clusters"`
	Contexts []struct {
		Context struct {
			Cluster string `yaml:"cluster" json:"cluster"`
			User    string `yaml:"user" json:"user"`
		} `yaml:"context" json:"context"`
		Name string `yaml:"name" json:"name"`
	} `yaml:"contexts"        json:"contexts"`
	CurrentContext string `yaml:"current-context" json:"current-context"`
	Users          []struct {
		User any    `yaml:"user" json:"user"`
		Name string `yaml:"name" json:"name"`
	} `yaml:"users"           json:"users"`
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

func mergeKubeConfigs(config1 *KubeConfig, config2 *KubeConfig) (*KubeConfig, error) {
	merged := &KubeConfig{
		ApiVersion: kubeConfigApiVersion,
		Kind:       kubeConfigKind,
	}

	// check nil
	if len(config2.Clusters) == 0 {
		return nil, errorx.InternalError.New("kubeconfig has no clusters")
	}
	if len(config2.Contexts) == 0 {
		return nil, errorx.InternalError.New("kubeconfig has no contexts")
	}
	if len(config2.Users) == 0 {
		return nil, errorx.InternalError.New("kubeconfig has no users")
	}

	// check duplicates
	if len(config1.Clusters) > 0 && len(config2.Clusters) > 0 && config2.Clusters[0].Name == config1.Clusters[0].Name {
		return nil, errorx.InternalError.New("kubeconfig has duplicate cluster name")
	}
	if len(config1.Contexts) > 0 && len(config2.Contexts) > 0 && config2.Contexts[0].Name == config1.Contexts[0].Name {
		return nil, errorx.InternalError.New("kubeconfig has duplicate context name")
	}
	if len(config1.Users) > 0 && len(config2.Users) > 0 && config2.Users[0].Name == config1.Users[0].Name {
		return nil, errorx.InternalError.New("kubeconfig has duplicate user name")
	}

	// check len
	if len(config2.Clusters) > 1 {
		return nil, errorx.InternalError.New("kubeconfig has more than one cluster")
	}
	if len(config2.Contexts) > 1 {
		return nil, errorx.InternalError.New("kubeconfig has more than one context")
	}
	if len(config2.Users) > 1 {
		return nil, errorx.InternalError.New("kubeconfig has more than one user")
	}

	// append
	merged.Clusters = append(config1.Clusters, config2.Clusters...)
	merged.Contexts = append(config1.Contexts, config2.Contexts...)
	merged.Users = append(config1.Users, config2.Users...)

	if config1.CurrentContext == "" {
		merged.CurrentContext = config2.CurrentContext
	} else {
		merged.CurrentContext = kubeConfigCurrentContext // default value
	}

	return merged, nil
}
