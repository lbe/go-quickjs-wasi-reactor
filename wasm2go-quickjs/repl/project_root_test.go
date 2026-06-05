package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
