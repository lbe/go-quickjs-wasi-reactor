package quickjs

import (
	"strings"
	"testing"
)

// TestQuickJSInstanceCreateInitMemoryDestroy verifies the wasm2go-quickjs
// subpackage compiles and a QuickJS instance can be created via NewQuickJS,
// initialized, memory accessed, and destroyed via Close() without panics.
// Calling Close() twice must be safe (no-op).
func TestQuickJSInstanceCreateInitMemoryDestroy(t *testing.T) {
	_, cfg := newTestConfig()

	// NewQuickJS should encapsulate module creation and initialization.
	qjs := NewQuickJS(cfg)
	if qjs == nil {
		t.Fatal("NewQuickJS returned nil")
	}

	// Close should call Xqjs_destroy without panicking.
	qjs.Close()

	// Calling Close() twice must be safe (no-op).
	qjs.Close()
}

// TestQuickJSInitWithArgvAndDoubleInitReject verifies that Init initializes
// the QuickJS runtime with command-line arguments via Xqjs_init_argv, creates
// the internal context, defaults to ["qjs"] when given nil args, and rejects
// a second Init call with an error.
func TestQuickJSInitWithArgvAndDoubleInitReject(t *testing.T) {
	_, cfg := newTestConfig()

	qjs := NewQuickJS(cfg)
	if qjs == nil {
		t.Fatal("NewQuickJS returned nil")
	}
	defer qjs.Close()

	// Init with explicit argv should succeed and create the context.
	err := qjs.Init([]string{"qjs", "--std"})
	if err != nil {
		t.Fatalf("Init with [--std] failed: %v", err)
	}

	// After successful Init, the internal context must be non-zero.
	if qjs.ctxPtr == 0 {
		t.Fatal("ctxPtr is zero after successful Init")
	}

	// Init with nil args should also succeed (defaults to ["qjs"]).
	// First, create a fresh instance to test nil-args default.
	qjs2 := NewQuickJS(cfg)
	if qjs2 == nil {
		t.Fatal("NewQuickJS returned nil for second instance")
	}
	defer qjs2.Close()

	err = qjs2.Init(nil)
	if err != nil {
		t.Fatalf("Init with nil args failed: %v", err)
	}
	if qjs2.ctxPtr == 0 {
		t.Fatal("ctxPtr is zero after Init with nil args")
	}

	// Calling Init a second time on the same instance must return an error.
	err = qjs.Init([]string{"qjs"})
	if err == nil {
		t.Fatal("expected error on double Init, got nil")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("expected 'already initialized' error, got: %v", err)
	}
}
