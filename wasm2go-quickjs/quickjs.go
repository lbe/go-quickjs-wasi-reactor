// Package quickjs provides a high-level Go API for running JavaScript
// using the QuickJS WASI reactor module with wasm2go.
package quickjs

import (
	"encoding/binary"
	"errors"
	"time"

	qjs_wasm "github.com/lbe/go-quickjs-wasi-reactor/qjs-wasi"
	wasihost "github.com/lbe/wasm2go-wasi-host"
)

// JSEvalTypeGlobal evaluates code as a global script (default).
const JSEvalTypeGlobal = 0

// JSEvalTypeModule evaluates code as an ES module.
const JSEvalTypeModule = 1

// LoopResult is the return value of LoopOnce. It encodes the event loop
// status after a single iteration:
//   - > 0: timers are pending; the value is the number of milliseconds
//     until the next timer fires.
//   - 0: no timers pending but more microtasks/jobs to process.
//   - LoopIdle (-1): nothing pending, the loop is idle.
//   - LoopError (-2): a JS error occurred during processing.
type LoopResult int32

// LoopIdle indicates the event loop has no pending work.
const LoopIdle LoopResult = -1

// LoopError indicates a JS error occurred during event loop processing.
const LoopError LoopResult = -2

// IsPending reports whether the event loop has more work to do.
// It returns true when the result is > 0 (timers pending) or == 0
// (more microtasks to process), and false for LoopIdle or LoopError.
func (r LoopResult) IsPending() bool {
	return r >= 0
}

// NextTimerMs returns the number of milliseconds until the next timer
// fires. This is only meaningful when LoopOnce returns a value > 0.
func (r LoopResult) NextTimerMs() int32 {
	if r > 0 {
		return int32(r)
	}
	return 0
}

// jsTagException is the tag value for JS exception objects in the QuickJS
// C API (JSTagException = 6). It is used by jsValueIsException to detect
// whether a returned JSValue represents a thrown exception.
const jsTagException = 6

// jsValueIsException checks whether a JSValue returned from XJS_Eval
// represents an exception by inspecting the tag in the upper 32 bits.
func jsValueIsException(val int64) bool {
	return val>>32 == jsTagException
}

// QuickJS wraps a QuickJS WASI reactor module instance created via wasm2go.
// The zero value is not usable; use NewQuickJS to construct an instance and
// call Close when done to release resources.
type QuickJS struct {
	mod    *qjs_wasm.Module
	state  *wasihost.State
	ctxPtr int32
}

// NewQuickJS creates a new QuickJS instance with the given module configuration.
// It instantiates the qjs_wasm module with a wasihost.State, then calls
// X_initialize. The returned instance must be closed with Close() to release
// resources.
//
// The constructor auto-configures WithSysWalltime, WithSysNanotime, and
// WithSysNanosleep on the module configuration so that timers work by
// default, matching wazero's default behavior.
func NewQuickJS(config *wasihost.ModuleConfig) *QuickJS {
	if config == nil {
		config = wasihost.NewModuleConfig()
	}
	config = config.
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep()

	// mod is declared before the memory callback so the closure can capture
	// it by reference; the actual module is assigned after wasihost.New
	// returns, at which point the callback becomes valid.
	var mod *qjs_wasm.Module
	state := wasihost.New(func() []byte {
		return *mod.Xmemory().Slice()
	}, config)
	mod = qjs_wasm.New(state)
	mod.X_initialize()
	return &QuickJS{mod: mod, state: state}
}

// Init initializes the QuickJS runtime with command-line arguments.
// If args is nil, it defaults to ["qjs", "--std"]. Calling Init a
// second time returns an error.
func (q *QuickJS) Init(args []string) error {
	if q.ctxPtr != 0 {
		return errors.New("already initialized")
	}
	if args == nil {
		args = []string{"qjs", "--std"}
	}

	// Allocate null-terminated strings in wasm memory.
	strPtrs := make([]int32, len(args))
	for i, arg := range args {
		strPtrs[i] = q.allocString(arg)
	}

	// Allocate argv array (int32 pointers).
	argvSize := int32(len(args)) * 4
	argvPtr := q.mod.Xmalloc(argvSize)

	// Write argv pointers into wasm memory.
	mem := *q.mod.Xmemory().Slice()
	for i, ptr := range strPtrs {
		binary.LittleEndian.PutUint32(mem[argvPtr+int32(i)*4:], uint32(ptr))
	}

	// Call Xqjs_init_argv(argc, argv).
	ret := q.mod.Xqjs_init_argv(int32(len(args)), argvPtr)

	// Free the argv array and individual strings.
	q.mod.Xfree(argvPtr)
	for _, ptr := range strPtrs {
		q.freePtr(ptr)
	}

	if ret != 0 {
		return errors.New("qjs_init_argv failed")
	}

	// Retrieve the internal context pointer.
	q.ctxPtr = q.mod.Xqjs_get_context()

	// Register the std and os built-in modules so that
	// os.setTimeout, std.out, etc. are available globally.
	q.mod.Xjs_std_add_helpers(q.ctxPtr, 0, 0)

	return nil
}

// allocString allocates a null-terminated string in wasm memory via Xmalloc
// and copies the string bytes into the allocated region.
func (q *QuickJS) allocString(s string) int32 {
	mem := *q.mod.Xmemory().Slice()
	ptr := q.mod.Xmalloc(int32(len(s) + 1))
	copy(mem[ptr:ptr+int32(len(s))], s)
	mem[ptr+int32(len(s))] = 0
	return ptr
}

// freePtr frees a pointer allocated via Xmalloc.
func (q *QuickJS) freePtr(ptr int32) {
	q.mod.Xfree(ptr)
}

// Eval evaluates JavaScript code in the QuickJS runtime.
// It returns the raw JSValue result and a boolean indicating whether
// the result is an exception. The caller is responsible for freeing
// the result via XJS_FreeValue and, if it is an exception, dumping
// the error via Xjs_std_dump_error.
//
// Eval pumps the event loop after evaluation to process any pending
// microtasks and timers, then spawns a background goroutine to poll I/O
// so that registered read handlers (e.g. os.setReadHandler) can fire
// without an explicit PollIO call.
func (q *QuickJS) Eval(code string, filename string, flags int32) (int64, bool) {
	codePtr := q.allocString(code)
	filenamePtr := q.allocString(filename)
	result := q.mod.XJS_Eval(q.ctxPtr, codePtr, int32(len(code)), filenamePtr, flags)
	q.freePtr(codePtr)
	q.freePtr(filenamePtr)
	q.RunLoop()
	go q.PollIO(0)
	return result, jsValueIsException(result)
}

// LoopOnce performs a single iteration of the QuickJS event loop.
// It calls the native Xjs_std_loop_once function and returns a LoopResult
// indicating what kind of pending work remains:
//   - > 0: timers pending (value is ms until next timer)
//   - 0: more microtasks/jobs to process
//   - LoopIdle (-1): nothing pending
//   - LoopError (-2): JS error occurred
func (q *QuickJS) LoopOnce() LoopResult {
	ret := q.mod.Xjs_std_loop_once(q.ctxPtr)
	return LoopResult(ret)
}

// RunLoop processes the QuickJS event loop until it becomes idle.
// It repeatedly calls LoopOnce, sleeping when timers are pending,
// and returns when the loop is idle or an error occurs.
func (q *QuickJS) RunLoop() {
	for {
		result := q.LoopOnce()
		switch {
		case result == LoopIdle || result == LoopError:
			return
		case result > 0:
			time.Sleep(time.Duration(result.NextTimerMs()) * time.Millisecond)
		default:
			// result == 0: more microtasks, continue immediately
		}
	}
}

// PollIO polls for pending I/O events and invokes any registered
// read/write handlers. It calls the native Xjs_std_poll_io with the
// given timeout in milliseconds.
//
// Returns 0 on success, or -2 if the instance is not initialized
// or has been closed.
func (q *QuickJS) PollIO(timeoutMs int) int32 {
	if q.ctxPtr == 0 || q.mod == nil {
		return -2
	}
	return q.mod.Xjs_std_poll_io(q.ctxPtr, int32(timeoutMs))
}

// EvalWithFilename is a convenience wrapper for Eval that uses
// JSEvalTypeGlobal as the eval flags.
func (q *QuickJS) EvalWithFilename(code string, filename string) (int64, bool) {
	return q.Eval(code, filename, JSEvalTypeGlobal)
}

// Close destroys the QuickJS instance and releases resources.
// Calling Close() twice is safe (no-op).
func (q *QuickJS) Close() {
	if q.mod != nil {
		q.mod.Xqjs_destroy()
		q.mod = nil
	}
}
