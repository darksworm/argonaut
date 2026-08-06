//go:build e2e && unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var binPath = "a9s_e2e"

// versionedBinPath is a build of the app with releasedTestVersion injected
// (same ldflags as goreleaser), for tests exercising version-gated startup
// behaviour that a "dev" build skips. Built once here so `go build` doesn't
// compete for CPU with running tests.
var versionedBinPath string

const releasedTestVersion = "2.18.0"

func TestMain(m *testing.M) {
	// e2e dir
	e2eDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	binPath = e2eDir + "/a9s_e2e"

	// Build the TUI binary from cmd/app
	fmt.Println("Building a9s test binary…")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/app")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		fmt.Printf("failed to build test binary: %v\n", err)
		os.Exit(1)
	}

	versionedBinPath = e2eDir + "/a9s_e2e_v" + releasedTestVersion
	cmd = exec.Command("go", "build",
		"-ldflags", "-X main.appVersion="+releasedTestVersion,
		"-o", versionedBinPath, "./cmd/app")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		fmt.Printf("failed to build versioned test binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	_ = os.Remove(binPath)
	_ = os.Remove(versionedBinPath)
	os.Exit(code)
}
