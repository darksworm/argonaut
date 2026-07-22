package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteDeltaHeader_ReplacesTempFilePathsWithResourceName(t *testing.T) {
	left := filepath.Join(os.TempDir(), "argonaut-live-123.yaml")
	right := filepath.Join(os.TempDir(), "argonaut-desired-456.yaml")
	output := strings.Join([]string{
		fmt.Sprintf("Δ %s ⟶ %s", left, right),
		"─────────────",
		"some diff content",
	}, "\n")

	got := rewriteDeltaHeader(output, "Deployment/my-app")

	want := strings.Join([]string{
		"Δ Deployment/my-app: Live ⟶ Desired",
		"─────────────",
		"some diff content",
	}, "\n")
	if got != want {
		t.Errorf("header not rewritten:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRewriteDeltaHeader_PreservesANSICodesAroundHeader(t *testing.T) {
	path := filepath.Join(os.TempDir(), "argonaut-live-123.yaml")
	output := fmt.Sprintf("\x1b[34mΔ %s\x1b[0m", path)

	got := rewriteDeltaHeader(output, "Service/api")

	want := "\x1b[34mΔ Service/api: Live ⟶ Desired\x1b[0m"
	if got != want {
		t.Errorf("ANSI codes not preserved:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRewriteDeltaHeader_LeavesOutputWithoutHeaderUntouched(t *testing.T) {
	output := "plain diff line\nanother line"

	got := rewriteDeltaHeader(output, "Deployment/my-app")

	if got != output {
		t.Errorf("output changed:\ngot:  %q\nwant: %q", got, output)
	}
}
