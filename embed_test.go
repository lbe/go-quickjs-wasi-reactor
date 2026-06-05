// Package quickjswasi provides a Go wrapper around the QuickJS WASI reactor.
package quickjswasi

import (
	"testing"
)

func TestVersionInfo(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}

	t.Logf("Version: %s", Version)
	if DownloadURL != "" {
		t.Logf("DownloadURL: %s", DownloadURL)
	}
}
