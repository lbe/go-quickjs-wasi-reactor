package quickjs

import (
	"strings"
	"testing"
)

// TestEvalProducesStdout verifies that Eval executes JavaScript code and
// console.log output appears in captured stdout. EvalWithFilename includes
// the filename in syntax error output. Syntax errors return an error.
// JS exceptions are caught and dumped to stderr. ES module syntax works
// with isModule=true. The JSEvalTypeGlobal and JSEvalTypeModule constants
// are defined.
func TestEvalProducesStdout(t *testing.T) {
	stdout, stderr, cfg := newTestConfigWithStderr()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	// --- console.log produces stdout ---
	evalCode(t, qjs, `console.log("hello")`, "<eval>", JSEvalTypeGlobal)

	got := stdout.String()
	if !strings.Contains(got, "hello\n") {
		t.Fatalf("expected stdout to contain 'hello\\n', got: %q", got)
	}

	// --- syntax error returns an error and includes filename in stderr ---
	stderr.Reset()
	syntaxErrCode := `function broken() { console.log("test")`
	result, isException := qjs.Eval(syntaxErrCode, "test.js", JSEvalTypeGlobal)
	if !isException {
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Fatal("expected syntax error to produce an exception JSValue")
	}

	qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
	qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)

	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "test.js") {
		t.Fatalf("expected stderr to contain filename 'test.js', got: %q", stderrOutput)
	}

	// --- JS exception is caught and dumped to stderr ---
	stdout.Reset()
	stderr.Reset()
	exceptionCode := `throw new Error("boom");`
	result, isException = qjs.Eval(exceptionCode, "<eval>", JSEvalTypeGlobal)
	if !isException {
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Fatal("expected thrown exception to produce an exception JSValue")
	}

	qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
	qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)

	stderrOutput = stderr.String()
	if !strings.Contains(stderrOutput, "boom") {
		t.Fatalf("expected stderr to contain 'boom', got: %q", stderrOutput)
	}

	// --- ES module syntax with isModule=true works ---
	// Use synchronous module code (no top-level await) since event loop
	// pumping is not available in this cycle.
	stdout.Reset()
	stderr.Reset()
	moduleCode := "const result = 42;\nconsole.log(\"module result:\", result);\n"
	result, isException = qjs.Eval(moduleCode, "<eval>", JSEvalTypeModule)
	if isException {
		qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Fatalf("Eval with isModule=true raised an exception, stderr: %q", stderr.String())
	}

	qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)

	got = stdout.String()
	if !strings.Contains(got, "module result: 42") {
		t.Fatalf("expected stdout to contain 'module result: 42', got: %q", got)
	}

	// --- JSEvalTypeGlobal and JSEvalTypeModule constants are defined ---
	if JSEvalTypeGlobal != 0 {
		t.Fatalf("expected JSEvalTypeGlobal=0, got %d", JSEvalTypeGlobal)
	}
	if JSEvalTypeModule != 1 {
		t.Fatalf("expected JSEvalTypeModule=1, got %d", JSEvalTypeModule)
	}
}
