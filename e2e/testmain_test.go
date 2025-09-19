//go:build e2e && unix

package main

import (
    "fmt"
    "os"
    "os/exec"
    "testing"
)

var binPath = "a9s_e2e"

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

    code := m.Run()

    _ = os.Remove(binPath)
    os.Exit(code)
}

