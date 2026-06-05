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

// TestREPLRunsJavaScriptInteractivelyFromStdinWithStdAndExitsCleanly
// verifies that the REPL binary starts, reads JavaScript from stdin,
// evaluates it, captures the console.log output in stdout, and exits
// cleanly on EOF. The --std flag loads std/os/bjson globals.
//
// Acceptance criteria:
//   - The REPL binary can be built via `go build ./repl/`.
//   - Writing "console.log('repl test')" to stdin produces "repl test\n" in stdout.
//   - The process exits cleanly (exit code 0) on EOF.
//   - Init with nil args works (defaults to ["qjs", "--std"]).
//   - The REPL uses wasihost.NewModuleConfig with WithStdin/WithStdout/WithStderr
//     and fs.FS from os.OpenRoot.
func TestREPLRunsJavaScriptInteractivelyFromStdinWithStdAndExitsCleanly(t *testing.T) {
	// Build the REPL binary.
	projRoot := findProjectRoot(t)
	replDir := filepath.Join(projRoot, "wasm2go-quickjs", "repl")
	binPath := filepath.Join(t.TempDir(), "repl-test-binary")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = replDir
	buildCmd.Env = os.Environ()
	var buildStderr bytes.Buffer
	buildCmd.Stderr = &buildStderr

	t.Logf("building REPL binary: go build -o %s .", binPath)
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build REPL binary: %v\nstderr:\n%s", err, buildStderr.String())
	}

	// Run the REPL binary with stdin input.
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(binPath)
	cmd.Stdin = strings.NewReader("console.log('repl test')\n")
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
			t.Fatalf("REPL process exited with error: %v\nstderr:\n%s", err, stderr.String())
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("REPL process timed out after 10 seconds")
	}

	// Assert stdout contains the expected console.log output.
	got := stdout.String()
	if !strings.Contains(got, "repl test\n") {
		t.Fatalf("expected stdout to contain 'repl test\\n', got: %q\nstderr: %q", got, stderr.String())
	}
}


