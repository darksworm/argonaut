package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func goldenPath(name string) string {
	return filepath.Join("testdata", "snapshots", name+".golden")
}

func writeFile(path, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

func compareWithGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if err := writeFile(path, got); err != nil {
				t.Fatalf("failed to write golden %s: %v", path, err)
			}
			return
		}
		t.Fatalf("failed to read golden %s: %v (set UPDATE_GOLDEN=1 to create)", path, err)
	}
	want := string(wantBytes)
	if want != got {
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if err := writeFile(path, got); err != nil {
				t.Fatalf("failed to update golden %s: %v", path, err)
			}
			return
		}
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func TestGolden_ListView_Apps(t *testing.T) {
	m := buildTestModelWithApps(100, 30)
	content := m.renderListView(10)
	plain := stripANSI(content)
	compareWithGolden(t, "list_view_apps", plain)
}

func TestGolden_StatusLine(t *testing.T) {
	m := buildTestModelWithApps(80, 24)
	line := stripANSI(m.renderStatusLine())
	compareWithGolden(t, "status_line", line)
}

func TestLogHighlighting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty line",
			input:    "",
			expected: "",
		},
		{
			name:     "simple log line",
			input:    "2024/01/15 10:30:45 INFO component=app message=\"hello world\"",
			expected: "2024/01/15 10:30:45 INFO component=app message=\"hello world\"", // will be highlighted
		},
		{
			name:     "error log line",
			input:    "2024/01/15 10:30:45 ERROR component=app message=\"something failed\" error=\"connection refused\"",
			expected: "2024/01/15 10:30:45 ERROR component=app message=\"something failed\" error=\"connection refused\"", // will be highlighted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightLogLine(tt.input)
			// Since we can't easily test the ANSI color codes in a simple way,
			// we'll just verify the function doesn't crash and returns a non-empty string for non-empty input
			if tt.input != "" && result == "" {
				t.Errorf("HighlightLogLine(%q) returned empty string", tt.input)
			}
			if tt.input == "" && result != "" {
				t.Errorf("HighlightLogLine(%q) should return empty string for empty input", tt.input)
			}
			// The result should be non-empty for non-empty input and contain key parts
			stripped := stripANSI(result)
			if tt.input != "" {
				if stripped == "" {
					t.Errorf("HighlightLogLine(%q) returned empty string", tt.input)
				}
				// Check that key components are present
				if strings.Contains(tt.input, "INFO") && !strings.Contains(stripped, "INFO") {
					t.Errorf("HighlightLogLine(%q) should contain 'INFO'", tt.input)
				}
				if strings.Contains(tt.input, "ERROR") && !strings.Contains(stripped, "ERROR") {
					t.Errorf("HighlightLogLine(%q) should contain 'ERROR'", tt.input)
				}
			}
		})
	}
}
