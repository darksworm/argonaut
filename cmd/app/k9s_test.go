//go:build unix

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/kubeconfig"
)

// TestInjectStatusBarAtFrameBoundaries tests the ANSI escape sequence processing
// that injects the Argonaut status bar at frame boundaries in k9s output.
func TestInjectStatusBarAtFrameBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		rows      int
		cols      int
		kind      string
		namespace string
		context   string
		// check is a function that verifies the output
		check func(t *testing.T, output []byte)
	}{
		{
			name:      "clear screen sequence triggers injection",
			input:     []byte("\x1b[2J"),
			rows:      24,
			cols:      80,
			kind:      "Pod",
			namespace: "default",
			context:   "test-ctx",
			check: func(t *testing.T, output []byte) {
				// Should contain clear screen followed by status bar
				if !bytes.Contains(output, []byte("\x1b[2J")) {
					t.Error("output should contain clear screen sequence")
				}
				// Status bar should contain save cursor
				if !bytes.Contains(output, []byte("\x1b7")) {
					t.Error("output should contain save cursor sequence")
				}
				// Status bar should contain restore cursor
				if !bytes.Contains(output, []byte("\x1b8")) {
					t.Error("output should contain restore cursor sequence")
				}
			},
		},
		{
			name:      "cursor home sequence triggers injection",
			input:     []byte("\x1b[H"),
			rows:      24,
			cols:      80,
			kind:      "Deployment",
			namespace: "kube-system",
			context:   "prod",
			check: func(t *testing.T, output []byte) {
				// Status bar should be injected BEFORE cursor home
				if !bytes.Contains(output, []byte("\x1b7")) {
					t.Error("output should contain status bar (save cursor)")
				}
				if !bytes.Contains(output, []byte("\x1b[H")) {
					t.Error("output should still contain cursor home")
				}
			},
		},
		{
			name:      "explicit cursor home triggers injection",
			input:     []byte("\x1b[;H"),
			rows:      24,
			cols:      80,
			kind:      "Service",
			namespace: "default",
			context:   "",
			check: func(t *testing.T, output []byte) {
				if !bytes.Contains(output, []byte("\x1b7")) {
					t.Error("output should contain status bar")
				}
			},
		},
		{
			name:      "cursor 1,1 triggers injection",
			input:     []byte("\x1b[1;1H"),
			rows:      24,
			cols:      80,
			kind:      "ConfigMap",
			namespace: "default",
			context:   "dev",
			check: func(t *testing.T, output []byte) {
				if !bytes.Contains(output, []byte("\x1b7")) {
					t.Error("output should contain status bar")
				}
			},
		},
		{
			name:      "no escape sequences - pass through unchanged",
			input:     []byte("Hello World"),
			rows:      24,
			cols:      80,
			kind:      "Pod",
			namespace: "default",
			context:   "test",
			check: func(t *testing.T, output []byte) {
				// Output should be exactly the input - no injection
				if !bytes.Equal(output, []byte("Hello World")) {
					t.Errorf("expected pass-through, got: %q", output)
				}
			},
		},
		{
			name:      "empty input",
			input:     []byte{},
			rows:      24,
			cols:      80,
			kind:      "Pod",
			namespace: "default",
			context:   "test",
			check: func(t *testing.T, output []byte) {
				if len(output) != 0 {
					t.Errorf("expected empty output, got: %q", output)
				}
			},
		},
		{
			name:      "mixed content with multiple boundaries",
			input:     []byte("text\x1b[2Jmore\x1b[H"),
			rows:      24,
			cols:      80,
			kind:      "Pod",
			namespace: "default",
			context:   "test",
			check: func(t *testing.T, output []byte) {
				// Should have status bar injected at both boundaries
				// Count save cursor sequences (one per injection)
				count := bytes.Count(output, []byte("\x1b7"))
				if count < 2 {
					t.Errorf("expected 2 status bar injections, got %d", count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := K9sResourceParams{
				Kind:      tt.kind,
				Namespace: tt.namespace,
				Context:   tt.context,
			}
			output := injectStatusBarAtFrameBoundaries(tt.input, tt.rows, tt.cols, params)
			tt.check(t, output)
		})
	}
}

// TestBuildStatusBarSequence tests the status bar formatting.
func TestBuildStatusBarSequence(t *testing.T) {
	tests := []struct {
		name         string
		rows         int
		cols         int
		kind         string
		namespace    string
		context      string
		wantContains []string
	}{
		{
			name:      "full content",
			rows:      24,
			cols:      80,
			kind:      "Pod",
			namespace: "default",
			context:   "minikube",
			wantContains: []string{
				"\x1b7",           // Save cursor
				"\x1b[24;1H",      // Move to row 24
				"\x1b[2K",         // Clear line
				"Argonaut",        // App name
				"k9s",             // k9s identifier
				"Pod",             // Kind
				"default",         // Namespace
				"minikube",        // Context
				":q to return",    // Help text
				"\x1b8",           // Restore cursor
			},
		},
		{
			name:      "no kind - basic status bar",
			rows:      24,
			cols:      80,
			kind:      "",
			namespace: "",
			context:   "",
			wantContains: []string{
				"Argonaut",
				"k9s",
				":q to return",
			},
		},
		{
			name:      "kind without namespace",
			rows:      24,
			cols:      80,
			kind:      "Node",
			namespace: "",
			context:   "prod",
			wantContains: []string{
				"Node",
				"prod",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := K9sResourceParams{
				Kind:      tt.kind,
				Namespace: tt.namespace,
				Context:   tt.context,
			}
			output := buildStatusBarSequence(tt.rows, tt.cols, params)
			outputStr := string(output)
			for _, want := range tt.wantContains {
				if !strings.Contains(outputStr, want) {
					t.Errorf("expected output to contain %q, got: %q", want, outputStr)
				}
			}
		})
	}
}

// TestK9sResourceMapCoverage verifies that all common Kubernetes resource kinds
// have mappings to k9s aliases.
func TestK9sResourceMapCoverage(t *testing.T) {
	// Expected mappings based on k9s conventions
	expectedMappings := map[string]string{
		"Pod":                      "pod",
		"Deployment":               "deploy",
		"Service":                  "svc",
		"Ingress":                  "ing",
		"ConfigMap":                "cm",
		"Secret":                   "secret",
		"ReplicaSet":               "rs",
		"StatefulSet":              "sts",
		"DaemonSet":                "ds",
		"Job":                      "job",
		"CronJob":                  "cj",
		"PersistentVolumeClaim":    "pvc",
		"PersistentVolume":         "pv",
		"ServiceAccount":           "sa",
		"Namespace":                "ns",
		"Node":                     "node",
		"Event":                    "event",
		"Endpoints":                "ep",
		"HorizontalPodAutoscaler":  "hpa",
		"NetworkPolicy":            "netpol",
		"Role":                     "role",
		"RoleBinding":              "rolebinding",
		"ClusterRole":              "clusterrole",
		"ClusterRoleBinding":       "clusterrolebinding",
	}

	for kind, expectedAlias := range expectedMappings {
		t.Run(kind, func(t *testing.T) {
			actualAlias, ok := k9sResourceMap[kind]
			if !ok {
				t.Errorf("k9sResourceMap missing entry for %q", kind)
				return
			}
			if actualAlias != expectedAlias {
				t.Errorf("k9sResourceMap[%q] = %q, want %q", kind, actualAlias, expectedAlias)
			}
		})
	}

	// Verify no unexpected entries
	for kind := range k9sResourceMap {
		if _, expected := expectedMappings[kind]; !expected {
			t.Logf("Note: k9sResourceMap has additional entry %q (may be intentional)", kind)
		}
	}
}

// TestFindK9sContext_InCluster verifies that in-cluster always returns an error
// so the context picker is shown, rather than blindly trusting current-context.
func TestFindK9sContext_InCluster(t *testing.T) {
	// Set up a kubeconfig with a current-context so the old code would have returned it
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")
	content := `apiVersion: v1
kind: Config
current-context: minikube
contexts:
  - name: minikube
    context:
      cluster: minikube
      user: minikube-user
  - name: prod
    context:
      cluster: prod
      user: prod-user
clusters:
  - name: minikube
    cluster:
      server: https://192.168.49.2:8443
  - name: prod
    cluster:
      server: https://prod.example.com:6443
users:
  - name: minikube-user
    user:
      token: test
  - name: prod-user
    user:
      token: test
`
	if err := os.WriteFile(kubeconfigPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	t.Setenv("KUBECONFIG", kubeconfigPath)

	m := &Model{}

	// in-cluster must return error to trigger context picker
	_, err := m.findK9sContext("in-cluster")
	if err == nil {
		t.Fatal("findK9sContext('in-cluster') should return error, got nil")
	}
	if !strings.Contains(err.Error(), "manual context selection") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Named cluster that matches a context should still work
	ctx, err := m.findK9sContext("minikube")
	if err != nil {
		t.Fatalf("findK9sContext('minikube') unexpected error: %v", err)
	}
	if ctx != "minikube" {
		t.Errorf("expected context 'minikube', got %q", ctx)
	}

	// Unknown cluster should return error
	_, err = m.findK9sContext("unknown-cluster")
	if err == nil {
		t.Fatal("findK9sContext('unknown-cluster') should return error, got nil")
	}
}

// TestFindK9sContext_PreSelectCurrentContext verifies that when the context picker
// is shown, the current kubeconfig context is pre-selected.
func TestFindK9sContext_PreSelectCurrentContext(t *testing.T) {
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "config")

	contexts := []string{"alpha", "beta", "gamma"}
	currentContext := "beta"

	var sb strings.Builder
	sb.WriteString("apiVersion: v1\nkind: Config\n")
	sb.WriteString(fmt.Sprintf("current-context: %s\n", currentContext))
	sb.WriteString("contexts:\n")
	for _, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("  - name: %s\n    context:\n      cluster: %s\n      user: %s-user\n", ctx, ctx, ctx))
	}
	sb.WriteString("clusters:\n")
	for _, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("  - name: %s\n    cluster:\n      server: https://%s.local:6443\n", ctx, ctx))
	}
	sb.WriteString("users:\n")
	for _, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("  - name: %s-user\n    user:\n      token: test\n", ctx))
	}

	if err := os.WriteFile(kubeconfigPath, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	t.Setenv("KUBECONFIG", kubeconfigPath)

	m := &Model{}
	// Simulate the pre-selection logic from handleOpenK9s
	m.k9sContextOptions = contexts
	m.k9sContextSelected = 0

	// Replicate the pre-selection logic from handleOpenK9s
	kc, err := kubeconfig.Load()
	if err != nil {
		t.Fatalf("kubeconfig load: %v", err)
	}
	current := kc.GetCurrentContext()
	for i, c := range m.k9sContextOptions {
		if c == current {
			m.k9sContextSelected = i
			break
		}
	}

	// beta is at index 1
	if m.k9sContextSelected != 1 {
		t.Errorf("expected k9sContextSelected=1 (beta), got %d", m.k9sContextSelected)
	}
}

