package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kubeconfig is a minimal struct for parsing kubeconfig files
type Kubeconfig struct {
	CurrentContext string `yaml:"current-context"`
	Contexts       []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	Clusters []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
}

// Load loads and parses the kubeconfig file
// It checks $KUBECONFIG first, then falls back to ~/.kube/config
func Load() (*Kubeconfig, error) {
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, ".kube", "config")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig at %s: %w", path, err)
	}

	var kc Kubeconfig
	if err := yaml.Unmarshal(data, &kc); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	return &kc, nil
}

// normalizeServerURL removes trailing slashes and normalizes the URL for comparison
func normalizeServerURL(url string) string {
	return strings.TrimRight(url, "/")
}

// FindContextByServerURL finds a kubeconfig context whose cluster matches the server URL
// Returns the context name and nil error if found, or empty string and error if not found
func FindContextByServerURL(serverURL string) (string, error) {
	kc, err := Load()
	if err != nil {
		return "", err
	}

	return kc.FindContextByServerURL(serverURL)
}

// FindContextByServerURL finds a context by server URL within the loaded kubeconfig
func (kc *Kubeconfig) FindContextByServerURL(serverURL string) (string, error) {
	// Handle in-cluster special case - use current context
	if serverURL == "https://kubernetes.default.svc" || serverURL == "https://kubernetes.default.svc/" {
		if kc.CurrentContext != "" {
			return kc.CurrentContext, nil
		}
		return "", fmt.Errorf("in-cluster detected but no current context set")
	}

	normalizedTarget := normalizeServerURL(serverURL)

	// Build a map of cluster name -> server URL
	clusterServers := make(map[string]string)
	for _, cluster := range kc.Clusters {
		clusterServers[cluster.Name] = normalizeServerURL(cluster.Cluster.Server)
	}

	// Find context whose cluster's server URL matches
	for _, ctx := range kc.Contexts {
		clusterServer, ok := clusterServers[ctx.Context.Cluster]
		if !ok {
			continue
		}
		if clusterServer == normalizedTarget {
			return ctx.Name, nil
		}
	}

	return "", fmt.Errorf("no context found for server URL: %s", serverURL)
}

// FindContextByName tries to find a context by its name (exact match)
func (kc *Kubeconfig) FindContextByName(name string) (string, bool) {
	for _, ctx := range kc.Contexts {
		if ctx.Name == name {
			return ctx.Name, true
		}
	}
	return "", false
}

// GetCurrentContext returns the current context name
func (kc *Kubeconfig) GetCurrentContext() string {
	return kc.CurrentContext
}

// ListContexts returns a list of all context names
func (kc *Kubeconfig) ListContexts() []string {
	contexts := make([]string, 0, len(kc.Contexts))
	for _, ctx := range kc.Contexts {
		contexts = append(contexts, ctx.Name)
	}
	return contexts
}

// ListContextNames loads kubeconfig and returns all context names
func ListContextNames() ([]string, error) {
	kc, err := Load()
	if err != nil {
		return nil, err
	}
	return kc.ListContexts(), nil
}
