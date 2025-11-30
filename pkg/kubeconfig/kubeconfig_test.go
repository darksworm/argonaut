package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindContextByServerURL(t *testing.T) {
	// Create a temporary kubeconfig file
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: prod-context
clusters:
  - name: prod-cluster
    cluster:
      server: https://api.prod.example.com:6443
  - name: staging-cluster
    cluster:
      server: https://api.staging.example.com:6443
  - name: in-cluster
    cluster:
      server: https://kubernetes.default.svc
contexts:
  - name: prod-context
    context:
      cluster: prod-cluster
  - name: staging-context
    context:
      cluster: staging-cluster
  - name: in-cluster-context
    context:
      cluster: in-cluster
`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	// Set KUBECONFIG to our temp file
	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Setenv("KUBECONFIG", oldKubeconfig)

	tests := []struct {
		name        string
		serverURL   string
		wantContext string
		wantErr     bool
	}{
		{
			name:        "find prod cluster by URL",
			serverURL:   "https://api.prod.example.com:6443",
			wantContext: "prod-context",
			wantErr:     false,
		},
		{
			name:        "find staging cluster by URL",
			serverURL:   "https://api.staging.example.com:6443",
			wantContext: "staging-context",
			wantErr:     false,
		},
		{
			name:        "normalize trailing slash",
			serverURL:   "https://api.prod.example.com:6443/",
			wantContext: "prod-context",
			wantErr:     false,
		},
		{
			name:        "in-cluster returns current context",
			serverURL:   "https://kubernetes.default.svc",
			wantContext: "prod-context", // current-context
			wantErr:     false,
		},
		{
			name:        "unknown server returns error",
			serverURL:   "https://unknown.example.com:6443",
			wantContext: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := FindContextByServerURL(tt.serverURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindContextByServerURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ctx != tt.wantContext {
				t.Errorf("FindContextByServerURL() = %v, want %v", ctx, tt.wantContext)
			}
		})
	}
}

func TestKubeconfig_FindContextByName(t *testing.T) {
	kc := &Kubeconfig{
		CurrentContext: "default",
		Contexts: []struct {
			Name    string `yaml:"name"`
			Context struct {
				Cluster string `yaml:"cluster"`
			} `yaml:"context"`
		}{
			{Name: "prod", Context: struct {
				Cluster string `yaml:"cluster"`
			}{Cluster: "prod-cluster"}},
			{Name: "staging", Context: struct {
				Cluster string `yaml:"cluster"`
			}{Cluster: "staging-cluster"}},
		},
	}

	// Test found
	ctx, found := kc.FindContextByName("prod")
	if !found || ctx != "prod" {
		t.Errorf("FindContextByName('prod') = %v, %v, want prod, true", ctx, found)
	}

	// Test not found
	ctx, found = kc.FindContextByName("nonexistent")
	if found {
		t.Errorf("FindContextByName('nonexistent') = %v, %v, want '', false", ctx, found)
	}
}

func TestNormalizeServerURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://api.example.com", "https://api.example.com"},
		{"https://api.example.com/", "https://api.example.com"},
		{"https://api.example.com///", "https://api.example.com"},
	}

	for _, tt := range tests {
		got := normalizeServerURL(tt.input)
		if got != tt.want {
			t.Errorf("normalizeServerURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
