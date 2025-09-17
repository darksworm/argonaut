package main

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// Test helper function
func stringPtr(s string) *string {
	return &s
}

// buildTestModelWithApps creates a minimal model configured for a deterministic list view.
func buildTestModelWithApps(cols, rows int) *Model {
	m := NewModel()
	m.ready = true
	m.state.Terminal.Cols = cols
	m.state.Terminal.Rows = rows
	m.state.Mode = model.ModeNormal
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 1 // highlight middle row
	// Ensure no overlays
	m.state.Modals = model.ModalState{}
	m.state.Diff = nil

	ns1, pr1 := stringPtr("ns-a"), stringPtr("proj-a")
	ns2, pr2 := stringPtr("ns-b"), stringPtr("proj-b")
	ns3, pr3 := stringPtr("ns-c"), stringPtr("proj-c")
	m.state.Apps = []model.App{
		{Name: "app-a", Sync: "Synced", Health: "Healthy", Namespace: ns1, Project: pr1},
		{Name: "app-b", Sync: "OutOfSync", Health: "Degraded", Namespace: ns2, Project: pr2},
		{Name: "app-c", Sync: "Unknown", Health: "Progressing", Namespace: ns3, Project: pr3},
	}
	return m
}

func TestRender_ListHeaderAndRowsContainExpectedFields(t *testing.T) {
	m := buildTestModelWithApps(100, 30)
	// Render only the inner list area for stable checks
	content := m.renderListView(10)
	plain := stripANSI(content)

	// Header contains the expected labels
	if !strings.Contains(plain, "NAME") || !strings.Contains(plain, "SYNC") || !strings.Contains(plain, "HEALTH") {
		t.Fatalf("header missing expected labels. content=\n%s", plain)
	}

	// Rows include app names in order and abbreviated statuses present
	idxNameA := strings.Index(plain, "app-a")
	idxNameB := strings.Index(plain, "app-b")
	idxNameC := strings.Index(plain, "app-c")
	if idxNameA < 0 || idxNameB < 0 || idxNameC < 0 || !(idxNameA < idxNameB && idxNameB < idxNameC) {
		t.Fatalf("apps not rendered in expected order: a=%d b=%d c=%d\ncontent=\n%s", idxNameA, idxNameB, idxNameC, plain)
	}

	// Icons or words should appear for statuses; since we stripped ANSI, check for text fallback
	// We expect at least the long status words somewhere in the rendered rows.
	for _, want := range []string{"Synced", "OutOfSync", "Healthy", "Degraded", "Unknown", "Progressing"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected %q in list rows. content=\n%s", want, plain)
		}
	}
}

func TestRender_StatusLineFormatting(t *testing.T) {
	m := buildTestModelWithApps(80, 24)
	line := stripANSI(m.renderStatusLine())
	if !strings.Contains(line, "<apps>") {
		t.Fatalf("status line should include view tag: %q", line)
	}
	if !strings.Contains(line, "/3") { // 3 items total
		t.Fatalf("status line should include total count: %q", line)
	}
}
