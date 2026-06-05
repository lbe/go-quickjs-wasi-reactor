package quickjs

import (
	"bytes"
	"embed"
	"strings"
	"testing"

	wasihost "github.com/lbe/wasm2go-wasi-host"
)

//go:embed quickjs_test.js
var fixtureFS embed.FS

// newTestConfig builds a wasihost.ModuleConfig with stdout capture and
// auto-configured clocks for use in tests.
func newTestConfig() (*bytes.Buffer, *wasihost.ModuleConfig) {
	var stdout bytes.Buffer
	cfg := wasihost.NewModuleConfig().
		WithStdout(&stdout).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep()
	return &stdout, cfg
}

// newTestConfigWithStderr builds a wasihost.ModuleConfig with separate
// stdout and stderr capture plus auto-configured clocks for use in tests
// that need to inspect stderr output (e.g. syntax errors, exceptions).
func newTestConfigWithStderr() (*bytes.Buffer, *bytes.Buffer, *wasihost.ModuleConfig) {
	var stdout, stderr bytes.Buffer
	cfg := wasihost.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep()
	return &stdout, &stderr, cfg
}

// newTestConfigWithFS builds a wasihost.ModuleConfig with stdout (merged
// with stderr), clocks, and a read-only filesystem mounted at / using the
// embedded test fixtures.
func newTestConfigWithFS() (*bytes.Buffer, *bytes.Buffer, *wasihost.ModuleConfig) {
	var stdout bytes.Buffer
	cfg := wasihost.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithReadOnlyFS("/", fixtureFS)
	return &stdout, &stdout, cfg
}

// newTestConfigWithFSAndStderr builds a config with separate stdout and
// stderr plus clocks and a read-only filesystem mounted at /.
func newTestConfigWithFSAndStderr() (*bytes.Buffer, *bytes.Buffer, *wasihost.ModuleConfig) {
	var stdout, stderr bytes.Buffer
	cfg := wasihost.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithReadOnlyFS("/", fixtureFS)
	return &stdout, &stderr, cfg
}

// newTestQuickJS creates a QuickJS instance and asserts it is non-nil.
// The instance is registered with t.Cleanup for automatic Close.
func newTestQuickJS(t *testing.T, cfg *wasihost.ModuleConfig) *QuickJS {
	t.Helper()
	qjs := NewQuickJS(cfg)
	if qjs == nil {
		t.Fatal("NewQuickJS returned nil")
	}
	t.Cleanup(qjs.Close)
	return qjs
}

// initQuickJS initializes a QuickJS instance and asserts no error.
func initQuickJS(t *testing.T, qjs *QuickJS, args []string) {
	t.Helper()
	if err := qjs.Init(args); err != nil {
		t.Fatalf("Init(%v) failed: %v", args, err)
	}
}

// evalCode evaluates code on a QuickJS instance, failing the test if
// the result is an exception. The raw JSValue is freed before returning.
func evalCode(t *testing.T, qjs *QuickJS, code string, filename string, flags int32) {
	t.Helper()
	result, isException := qjs.Eval(code, filename, flags)
	if isException {
		qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Fatalf("Eval(%q) raised an exception", truncate(code, 60))
	}
	qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
}

// assertOutputContains checks that stdout contains all expected patterns.
func assertOutputContains(t *testing.T, stdout string, patterns []string) {
	t.Helper()
	for _, pattern := range patterns {
		if !strings.Contains(stdout, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, stdout)
		}
	}
}

// truncate returns a shortened representation of s for use in error messages.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
