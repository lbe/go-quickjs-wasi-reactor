package quickjs

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	wasihost "github.com/lbe/wasm2go-wasi-host"
)

// TestIOPollingInvokesRegisteredReadHandlersOnStdin verifies that
// PollIO dispatches data to a registered os.setReadHandler callback
// on stdin. The test writes data to a pipe connected as stdin, then
// relies on Eval's background PollIO goroutine to invoke the handler,
// which logs the received bytes to stdout.
func TestIOPollingInvokesRegisteredReadHandlersOnStdin(t *testing.T) {
	// Create a pipe-like stdin so we can inject data after the read
	// handler is registered.
	stdinR, stdinW := io.Pipe()
	defer func() { _ = stdinR.Close() }()
	defer func() { _ = stdinW.Close() }()

	var stdout bytes.Buffer
	cfg := wasihost.NewModuleConfig().
		WithStdin(stdinR).
		WithStdout(&stdout).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep()

	qjs := NewQuickJS(cfg)
	if qjs == nil {
		t.Fatal("NewQuickJS returned nil")
	}
	defer qjs.Close()

	if err := qjs.Init(nil); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Evaluate JS that registers a read handler on stdin (fd 0).
	// The handler reads available data and logs it to stdout.
	handlerCode := `
		const stdinFd = 0;
		const readBuf = new Uint8Array(1024);
		os.setReadHandler(stdinFd, () => {
			const n = os.read(stdinFd, readBuf.buffer, 0, readBuf.length);
			if (n > 0) {
				const bytes = readBuf.slice(0, n);
				const text = String.fromCharCode.apply(null, bytes);
				console.log("stdin received:", text);
			}
		});
		console.log("read handler registered");
	`
	result, isException := qjs.Eval(handlerCode, "<eval>", JSEvalTypeGlobal)
	if isException {
		qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Fatalf("Eval of handler setup code raised an exception")
	}
	qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)

	// Verify the handler registration message appeared in stdout.
	got := stdout.String()
	if !strings.Contains(got, "read handler registered\n") {
		t.Fatalf("expected 'read handler registered' in stdout, got: %q", got)
	}

	// Write test data to stdin so the read handler has something to consume.
	go func() {
		_, _ = stdinW.Write([]byte("hello from stdin"))
	}()

	// Give the write a moment to be available.
	time.Sleep(50 * time.Millisecond)

	// After PollIO processes I/O, the read handler should have fired and
	// logged the received data to stdout.
	got = stdout.String()
	if !strings.Contains(got, "stdin received: hello from stdin") {
		t.Fatalf("expected stdout to contain 'stdin received: hello from stdin' after PollIO, got: %q", got)
	}
}
