package main

import (
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func paneOperationLine(t *testing.T, details *model.SyncStatusDetails) string {
	t.Helper()
	lines := renderSyncStatusBody(details, 46, time.Now(), "")
	for _, line := range lines {
		if strings.Contains(stripANSI(line), "Operation") {
			return stripANSI(line)
		}
	}
	t.Fatalf("no Operation field rendered in:\n%s", strings.Join(lines, "\n"))
	return ""
}

func TestSyncStatusPane_QualifiesADryRunSoSucceededIsNotMisread(t *testing.T) {
	line := paneOperationLine(t, &model.SyncStatusDetails{
		Operation:     "Sync",
		OperationNote: "dry run",
		Phase:         "Succeeded",
	})

	if !strings.Contains(line, "dry run") {
		t.Errorf("expected the operation line to mark the dry run, got %q", line)
	}
}

func TestSyncStatusPane_NamesTheOperationItWasGiven(t *testing.T) {
	line := paneOperationLine(t, &model.SyncStatusDetails{Operation: "Deleting", Phase: "Running"})

	if !strings.Contains(line, "Deleting") {
		t.Errorf("expected the operation line to read Deleting, got %q", line)
	}
	if strings.Contains(line, "Sync") {
		t.Errorf("expected no hardcoded Sync on a deletion, got %q", line)
	}
}

func TestSyncStatusPane_PlainSyncCarriesNoQualifier(t *testing.T) {
	line := paneOperationLine(t, &model.SyncStatusDetails{Operation: "Sync", Phase: "Succeeded"})

	if strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Operation")) != "Sync" {
		t.Errorf("expected a bare Sync, got %q", line)
	}
}
