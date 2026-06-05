package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExampleProgramRunsEndToEnd verifies that the example binary built from
// wasm2go-quickjs/example initializes QuickJS with --std, evaluates
// 'console.log("hello")', runs the event loop, and prints "Init OK",
// "Eval OK", "RunLoop OK" to stdout in order.
//
// Acceptance criteria:
//   - The example binary can be built via `go build ./wasm2go-quickjs/example/`.
//   - The binary runs to completion (exit code 0).
//   - Stdout contains "Init OK\nEval OK\nRunLoop OK\n" in that exact order.
//   - The example uses wasihost.NewModuleConfig with WithStdout and WithStderr.
func TestExampleProgramRunsEndToEnd(t *testing.T) {
	// Build the example binary.
	projRoot := findProjectRoot(t)
	exampleDir := filepath.Join(projRoot, "wasm2go-quickjs", "example")
	binPath := filepath.Join(t.TempDir(), "example-test-binary")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = exampleDir
	buildCmd.Env = os.Environ()
	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr

	t.Logf("building example binary: go build -o %s .", binPath)
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build example binary: %v\nstderr:\n%s", err, buildStderr.String())
	}

	// Run the example binary and capture stdout.
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(binPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Ensure the process doesn't hang forever.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("example process exited with error: %v\nstderr:\n%s", err, stderr.String())
		}
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("example process timed out after 30 seconds")
	}

	// Assert stdout contains the expected output in order.
	got := stdout.String()
	expected := "Init OK\nEval OK\nRunLoop OK\n"
	if !strings.Contains(got, expected) {
		t.Fatalf("expected stdout to contain %q, got: %q\nstderr: %q", expected, got, stderr.String())
	}
}

// findProjectRoot walks up from the current working directory to find
// the directory containing go.work (the project root).
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (no go.work found)")
		}
		dir = parent
	}
}


