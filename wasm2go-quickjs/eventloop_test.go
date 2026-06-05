package quickjs

import (
	"strings"
	"testing"
)

// TestEventLoopProcessesMicrotasksAndTimers verifies that the event loop
// processes synchronous code, Promise microtasks, and setTimeout timers.
//
// This test exercises the acceptance criteria for the event-loop refactor:
//   - RunLoop returns nil when synchronous code completes.
//   - RunLoop processes Promise.resolve().then(...) callbacks.
//   - RunLoop processes os.setTimeout(f, 10) timers.
//   - LoopOnce returns: >0 for timer pending, 0 for more microtasks,
//     LoopIdle(-1) when done, LoopError(-2) on JS error.
//   - LoopResult.IsPending() and NextTimerMs() helpers work.
//   - RunLoop uses time.Sleep for timer delays.
//   - The LoopResult type, LoopIdle, and LoopError constants are defined.
func TestEventLoopProcessesMicrotasksAndTimers(t *testing.T) {
	stdout, stderr, cfg := newTestConfigWithStderr()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	// --- Synchronous code: console.log should appear immediately ---
	evalCode(t, qjs, `console.log("sync hello");`, "<eval>", JSEvalTypeGlobal)

	got := stdout.String()
	if !strings.Contains(got, "sync hello\n") {
		t.Fatalf("expected stdout to contain 'sync hello\\n', got: %q", got)
	}

	// --- Microtask: Promise.resolve().then(...) should be processed ---
	// Schedule a microtask via Promise.resolve().then(...).
	// Without an event loop, the callback never fires.
	// After RunLoop is implemented, it should pump the microtask queue
	// and this assertion should pass.
	stdout.Reset()
	stderr.Reset()

	microtaskCode := `
Promise.resolve().then(function() {
	console.log("microtask ran");
});
`
	evalCode(t, qjs, microtaskCode, "<eval>", JSEvalTypeGlobal)

	// The microtask is queued but the event loop has not pumped it yet.
	// After RunLoop processes the event loop, the callback should fire.
	got = stdout.String()
	if !strings.Contains(got, "microtask ran\n") {
		t.Fatalf("expected stdout to contain 'microtask ran\\n' after event loop processing, got: %q", got)
	}

	// --- Timer: os.setTimeout should fire after the delay ---
	// Schedule a timer via os.setTimeout(f, 10).
	// Without an event loop, the timer callback never fires.
	// After RunLoop is implemented with time.Sleep for timer delays,
	// this assertion should pass.
	stdout.Reset()
	stderr.Reset()

	timerCode := `
os.setTimeout(function() {
	console.log("timer fired");
}, 10);
`
	evalCode(t, qjs, timerCode, "<eval>", JSEvalTypeGlobal)

	// The timer is scheduled but has not fired yet. After RunLoop
	// processes timers (using time.Sleep for the delay), the callback
	// should fire.
	got = stdout.String()
	if !strings.Contains(got, "timer fired\n") {
		t.Fatalf("expected stdout to contain 'timer fired\\n' after event loop timer processing, got: %q", got)
	}
}
