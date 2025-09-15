package main

import (
    "os"
    "path/filepath"
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

