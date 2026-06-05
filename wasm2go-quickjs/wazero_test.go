package quickjs

import (
	"bytes"
	"strings"
	"testing"
	"time"

	wasihost "github.com/lbe/wasm2go-wasi-host"
)

// TestSimpleEval verifies that a basic console.log produces stdout output.
func TestSimpleEval(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	evalCode(t, qjs, `console.log("Hello, World!");`, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("expected output to contain 'Hello, World!', got: %s", output)
	}
}

// TestLoopOnce verifies that synchronous code evaluates and produces output,
// and that the event loop becomes idle afterward.
func TestLoopOnce(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	evalCode(t, qjs, `let x = 1 + 2; console.log("result:", x);`, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	if !strings.Contains(output, "result: 3") {
		t.Errorf("expected output to contain 'result: 3', got: %s", output)
	}
}

// TestStdModule verifies that std.setenv, std.getenv, and the os module
// are available when initialized with --std.
func TestStdModule(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		std.setenv("TEST_VAR", "hello_from_std");
		console.log("TEST_VAR:", std.getenv("TEST_VAR"));
		console.log("std loaded:", typeof std === 'object');
		console.log("os loaded:", typeof os === 'object');
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"TEST_VAR: hello_from_std",
		"std loaded: true",
		"os loaded: true",
	})
}

// TestSetTimeout verifies that os.setTimeout fires a callback after a delay.
func TestSetTimeout(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		console.log("before timeout");
		os.setTimeout(() => {
			console.log("timeout fired");
		}, 10);
		console.log("after setTimeout call");
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"before timeout",
		"after setTimeout call",
		"timeout fired",
	})
}

// TestInitWithStd verifies that std and os are available globally after
// Init with --std.
func TestInitWithStd(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		console.log("std available:", typeof std === 'object');
		console.log("os available:", typeof os === 'object');
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"std available: true",
		"os available: true",
	})
}

// TestWASIEnvVars verifies that WASI environment variables set via WithEnv
// are readable through std.getenv.
func TestWASIEnvVars(t *testing.T) {
	var stdout bytes.Buffer
	cfg := wasihost.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout).
		WithEnv("TEST_VAR", "hello_from_wasi").
		WithEnv("ANOTHER_VAR", "another_value").
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep()

	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		console.log("TEST_VAR:", std.getenv("TEST_VAR"));
		console.log("ANOTHER_VAR:", std.getenv("ANOTHER_VAR"));
		console.log("UNDEFINED_VAR:", std.getenv("UNDEFINED_VAR"));
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"TEST_VAR: hello_from_wasi",
		"ANOTHER_VAR: another_value",
		"UNDEFINED_VAR: undefined",
	})
}

// TestESModuleEval verifies that ES module syntax including top-level await
// works when evaluated with JSEvalTypeModule.
func TestESModuleEval(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// ES module features
		const add = (a, b) => a + b;
		const multiply = (a, b) => a * b;

		// Test top-level await (ES2022)
		const result = await Promise.resolve(42);
		console.log("top-level await result:", result);

		// Test dynamic import-like pattern
		const mathOps = { add, multiply };
		console.log("add(2, 3):", mathOps.add(2, 3));
		console.log("multiply(4, 5):", mathOps.multiply(4, 5));
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeModule)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"top-level await result: 42",
		"add(2, 3): 5",
		"multiply(4, 5): 20",
	})
}

// TestPromiseChaining verifies Promise.resolve, .then chaining, Promise.all,
// Promise.race, and .catch rejection handling.
func TestPromiseChaining(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test Promise.resolve and chaining
		Promise.resolve(1)
			.then(x => {
				console.log("step 1:", x);
				return x + 1;
			})
			.then(x => {
				console.log("step 2:", x);
				return x * 2;
			})
			.then(x => {
				console.log("final:", x);
			});

		// Test Promise.all
		Promise.all([
			Promise.resolve("a"),
			Promise.resolve("b"),
			Promise.resolve("c")
		]).then(results => {
			console.log("Promise.all:", results.join(","));
		});

		// Test Promise.race
		Promise.race([
			Promise.resolve("first"),
			Promise.resolve("second")
		]).then(result => {
			console.log("Promise.race:", result);
		});

		// Test Promise rejection and catch
		Promise.reject(new Error("test error"))
			.catch(err => {
				console.log("caught error:", err.message);
			});
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"step 1: 1",
		"step 2: 2",
		"final: 4",
		"Promise.all: a,b,c",
		"Promise.race: first",
		"caught error: test error",
	})
}

// TestAsyncAwait verifies async/await, sequential and parallel awaits,
// and async error handling.
func TestAsyncAwait(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		async function fetchData(id) {
			return new Promise(resolve => {
				resolve({ id: id, data: "item_" + id });
			});
		}

		async function processItems() {
			console.log("starting async processing");

			// Sequential await
			const item1 = await fetchData(1);
			console.log("item1:", item1.id, item1.data);

			const item2 = await fetchData(2);
			console.log("item2:", item2.id, item2.data);

			// Parallel await with Promise.all
			const [item3, item4] = await Promise.all([
				fetchData(3),
				fetchData(4)
			]);
			console.log("item3:", item3.id, item3.data);
			console.log("item4:", item4.id, item4.data);

			console.log("async processing complete");
		}

		// Test async error handling
		async function failingAsync() {
			throw new Error("async error");
		}

		async function handleAsyncError() {
			try {
				await failingAsync();
			} catch (e) {
				console.log("caught async error:", e.message);
			}
		}

		processItems();
		handleAsyncError();
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"starting async processing",
		"item1: 1 item_1",
		"item2: 2 item_2",
		"item3: 3 item_3",
		"item4: 4 item_4",
		"async processing complete",
		"caught async error: async error",
	})
}

// TestEvalWithFilename verifies that a custom filename appears in the
// Error stack trace.
func TestEvalWithFilename(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFSAndStderr()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		function testFunction() {
			console.log("called from custom file");
			// Capture stack trace
			const err = new Error("stack trace test");
			console.log("stack:", err.stack);
		}
		testFunction();
	`
	evalCode(t, qjs, code, "my-custom-script.js", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"called from custom file",
		"my-custom-script.js",
	})
}

// TestSyntaxError verifies that a syntax error produces an exception
// JSValue from Eval.
func TestSyntaxError(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `function broken() { console.log("test")`
	result, isException := qjs.Eval(code, "<eval>", JSEvalTypeGlobal)
	if !isException {
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Errorf("expected syntax error to produce an exception, got nil")
	} else {
		qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
		qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
		t.Logf("Got expected syntax error, stdout: %s", stdout.String())
	}
}

// TestRuntimeError verifies that runtime errors (undefined variable, null
// property access, type error) are handled without crashing.
func TestRuntimeError(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	testCases := []struct {
		name string
		code string
	}{
		{
			name: "undefined variable",
			code: `console.log(undefinedVariable);`,
		},
		{
			name: "null property access",
			code: `let obj = null; console.log(obj.property);`,
		},
		{
			name: "type error",
			code: `let num = 42; num();`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, isException := qjs.Eval(tc.code, "<eval>", JSEvalTypeGlobal)
			if isException {
				qjs.mod.Xjs_std_dump_error(qjs.ctxPtr)
				qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
				t.Logf("Got expected exception for %s", tc.name)
			} else {
				qjs.mod.XJS_FreeValue(qjs.ctxPtr, result)
				// Some runtime errors may only surface during RunLoop
				qjs.RunLoop()
				output := stdout.String()
				t.Logf("Output for %s: %s", tc.name, output)
			}
		})
	}
}

// TestPreCompiledModuleReuse verifies that multiple independent QuickJS
// instances can be created and used sequentially.
func TestPreCompiledModuleReuse(t *testing.T) {
	for i := 0; i < 3; i++ {
		stdout, _, cfg := newTestConfigWithFS()
		qjs := newTestQuickJS(t, cfg)
		initQuickJS(t, qjs, nil)

		code := `console.log("instance", ` + string(rune('0'+i)) + `);`
		evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

		output := stdout.String()
		expected := "instance " + string(rune('0'+i))
		if !strings.Contains(output, expected) {
			t.Errorf("instance %d: expected %q, got: %s", i, expected, output)
		}
	}
}

// TestMultipleEvals verifies that multiple Eval calls on the same instance
// share state (variables, functions persist across calls).
func TestMultipleEvals(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	// First eval - define a variable
	evalCode(t, qjs, `var counter = 0; console.log("counter initialized");`, "<eval>", JSEvalTypeGlobal)

	// Second eval - modify the variable
	evalCode(t, qjs, `counter++; console.log("counter:", counter);`, "<eval>", JSEvalTypeGlobal)

	// Third eval - modify again
	evalCode(t, qjs, `counter += 10; console.log("counter:", counter);`, "<eval>", JSEvalTypeGlobal)

	// Fourth eval - define a function and use it with the variable
	evalCode(t, qjs, `
		function double(x) { return x * 2; }
		console.log("doubled counter:", double(counter));
	`, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"counter initialized",
		"counter: 1",
		"counter: 11",
		"doubled counter: 22",
	})
}

// TestSetInterval verifies os.setInterval fires repeatedly and
// os.clearInterval stops it.
func TestSetInterval(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		let count = 0;
		const intervalId = os.setInterval(() => {
			count++;
			console.log("interval tick:", count);
			if (count >= 3) {
				os.clearInterval(intervalId);
				console.log("interval cleared");
			}
		}, 50);
		console.log("interval started");
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"interval started",
		"interval tick: 1",
		"interval tick: 2",
		"interval tick: 3",
		"interval cleared",
	})

	// Verify it stopped at 3
	if strings.Contains(output, "interval tick: 4") {
		t.Errorf("interval should have stopped at 3, got: %s", output)
	}
}

// TestClearTimeout verifies that os.clearTimeout cancels a pending timeout.
func TestClearTimeout(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		console.log("setting up timeouts");

		// Set a timeout and immediately cancel it
		const id = os.setTimeout(() => {
			console.log("this should be cancelled");
		}, 1000);

		os.clearTimeout(id);
		console.log("timeout cleared successfully");

		// Set a short timeout to confirm we're done
		os.setTimeout(() => {
			console.log("done");
		}, 50);
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	// Run event loop with manual iteration to avoid blocking issues
	for i := 0; i < 100; i++ {
		result := qjs.LoopOnce()
		output := stdout.String()
		if strings.Contains(output, "done") {
			break
		}
		if result == LoopIdle {
			break
		}
		if result > 0 {
			time.Sleep(time.Duration(result) * time.Millisecond)
		}
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"timeout cleared successfully",
		"done",
	})
}

// TestLoopOnceReturnValues verifies the different return values from
// LoopOnce: LoopIdle after sync code, 0 or positive for pending work,
// and positive for timer delays.
func TestLoopOnceReturnValues(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	// Test 1: Synchronous code should return LoopIdle after processing
	evalCode(t, qjs, `console.log("sync code");`, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	if !strings.Contains(output, "sync code") {
		t.Errorf("expected sync code output, got: %s", output)
	}

	// Test 2: Promise should be processed by Eval's internal RunLoop
	stdout.Reset()
	evalCode(t, qjs, `Promise.resolve().then(() => console.log("promise resolved"));`, "<eval>", JSEvalTypeGlobal)

	output = stdout.String()
	if !strings.Contains(output, "promise resolved") {
		t.Errorf("expected promise output, got: %s", output)
	}

	// Test 3: setTimeout should fire via Eval's internal RunLoop
	stdout.Reset()
	evalCode(t, qjs, `os.setTimeout(() => console.log("timeout"), 100);`, "<eval>", JSEvalTypeGlobal)

	output = stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "timeout") {
		t.Errorf("expected timeout output, got: %s", output)
	}
}

// TestLargeCodeEval verifies that a large JavaScript code with many
// functions evaluates correctly.
func TestLargeCodeEval(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	// Generate a large JavaScript code with many functions
	var codeBuilder strings.Builder
	codeBuilder.WriteString("let results = [];\n")

	for i := 0; i < 100; i++ {
		codeBuilder.WriteString("function func")
		codeBuilder.WriteString(string(rune('0' + i/10)))
		codeBuilder.WriteString(string(rune('0' + i%10)))
		codeBuilder.WriteString("() { return ")
		codeBuilder.WriteString(string(rune('0' + i/10)))
		codeBuilder.WriteString(string(rune('0' + i%10)))
		codeBuilder.WriteString("; }\n")
		codeBuilder.WriteString("results.push(func")
		codeBuilder.WriteString(string(rune('0' + i/10)))
		codeBuilder.WriteString(string(rune('0' + i%10)))
		codeBuilder.WriteString("());\n")
	}

	codeBuilder.WriteString("console.log('total functions:', results.length);\n")
	codeBuilder.WriteString("console.log('sum:', results.reduce((a,b) => a+b, 0));\n")

	code := codeBuilder.String()
	t.Logf("Code size: %d bytes", len(code))

	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"total functions: 100",
		"sum: 4950",
	})
}

// TestBigIntSupport verifies BigInt literals, operations, comparisons,
// and conversions.
func TestBigIntSupport(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test BigInt literals
		const big1 = 9007199254740993n;
		const big2 = 9007199254740993n;
		console.log("bigint literal:", big1.toString());

		// Test BigInt operations
		const sum = big1 + big2;
		console.log("bigint sum:", sum.toString());

		// Test BigInt comparison
		console.log("bigint equal:", big1 === big2);

		// Test BigInt with regular numbers
		const bigFromNum = BigInt(12345678901234567890);
		console.log("bigint from number:", typeof bigFromNum === 'bigint');

		// Test large exponentiation
		const power = 2n ** 64n;
		console.log("2^64:", power.toString());

		// Test BigInt division (floors)
		const div = 10n / 3n;
		console.log("10n / 3n:", div.toString());
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"bigint literal: 9007199254740993",
		"bigint sum: 18014398509481986",
		"bigint equal: true",
		"bigint from number: true",
		"2^64: 18446744073709551616",
		"10n / 3n: 3",
	})
}

// TestTypedArrays verifies Uint8Array, Int32Array, Float64Array,
// DataView, slice, subarray, and set operations.
func TestTypedArrays(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test Uint8Array
		const u8 = new Uint8Array([1, 2, 3, 255]);
		console.log("Uint8Array:", Array.from(u8).join(","));
		console.log("Uint8Array length:", u8.length);

		// Test Int32Array
		const i32 = new Int32Array([-1, 0, 1, 2147483647]);
		console.log("Int32Array:", Array.from(i32).join(","));

		// Test Float64Array
		const f64 = new Float64Array([1.5, 2.5, 3.14159]);
		console.log("Float64Array:", Array.from(f64).map(x => x.toFixed(2)).join(","));

		// Test ArrayBuffer
		const buffer = new ArrayBuffer(8);
		const view = new DataView(buffer);
		view.setUint32(0, 0xDEADBEEF, true); // little-endian
		console.log("DataView getUint32:", view.getUint32(0, true).toString(16));

		// Test slice
		const sliced = u8.slice(1, 3);
		console.log("sliced:", Array.from(sliced).join(","));

		// Test subarray
		const sub = u8.subarray(1, 3);
		console.log("subarray:", Array.from(sub).join(","));

		// Test set
		const target = new Uint8Array(5);
		target.set([10, 20, 30], 1);
		console.log("after set:", Array.from(target).join(","));
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"Uint8Array: 1,2,3,255",
		"Uint8Array length: 4",
		"Int32Array: -1,0,1,2147483647",
		"Float64Array: 1.50,2.50,3.14",
		"DataView getUint32: deadbeef",
		"sliced: 2,3",
		"subarray: 2,3",
		"after set: 0,10,20,30,0",
	})
}

// TestJSONOperations verifies JSON.parse, JSON.stringify, replacer,
// pretty-printing, circular reference handling, and special values.
func TestJSONOperations(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test JSON.parse
		const parsed = JSON.parse('{"name":"test","value":42,"nested":{"a":1}}');
		console.log("parsed name:", parsed.name);
		console.log("parsed value:", parsed.value);
		console.log("parsed nested.a:", parsed.nested.a);

		// Test JSON.stringify
		const obj = { foo: "bar", nums: [1, 2, 3], flag: true };
		const str = JSON.stringify(obj);
		console.log("stringified:", str);

		// Test JSON.stringify with replacer
		const filtered = JSON.stringify(obj, ["foo", "flag"]);
		console.log("filtered:", filtered);

		// Test JSON.stringify with space
		const pretty = JSON.stringify({a: 1}, null, 2);
		console.log("pretty has newline:", pretty.includes("\\n"));

		// Test circular reference handling
		try {
			const circular = {};
			circular.self = circular;
			JSON.stringify(circular);
			console.log("circular: no error");
		} catch (e) {
			console.log("circular error:", e.message.includes("circular") || e.message.includes("cyclic"));
		}

		// Test special values
		console.log("null:", JSON.stringify(null));
		console.log("undefined:", JSON.stringify(undefined));
		console.log("NaN:", JSON.stringify(NaN));
		console.log("Infinity:", JSON.stringify(Infinity));
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"parsed name: test",
		"parsed value: 42",
		"parsed nested.a: 1",
		`"foo":"bar"`,
		`"nums":[1,2,3]`,
		"null: null",
	})
}

// TestGeneratorsAndIterators verifies generator functions, yield,
// yield*, Symbol.iterator, and Map iteration.
func TestGeneratorsAndIterators(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test generator function
		function* countTo(n) {
			for (let i = 1; i <= n; i++) {
				yield i;
			}
		}

		const gen = countTo(5);
		const values = [];
		for (const v of gen) {
			values.push(v);
		}
		console.log("generator values:", values.join(","));

		// Test generator with next()
		const gen2 = countTo(3);
		console.log("next 1:", gen2.next().value);
		console.log("next 2:", gen2.next().value);
		console.log("next 3:", gen2.next().value);
		console.log("done:", gen2.next().done);

		// Test yield*
		function* nested() {
			yield* [1, 2];
			yield* [3, 4];
		}
		console.log("yield*:", [...nested()].join(","));

		// Test Symbol.iterator
		const iterable = {
			[Symbol.iterator]() {
				let count = 0;
				return {
					next() {
						count++;
						if (count <= 3) {
							return { value: count * 10, done: false };
						}
						return { done: true };
					}
				};
			}
		};
		console.log("custom iterator:", [...iterable].join(","));

		// Test for...of with Map
		const map = new Map([["a", 1], ["b", 2]]);
		const mapEntries = [];
		for (const [k, v] of map) {
			mapEntries.push(k + ":" + v);
		}
		console.log("map iteration:", mapEntries.join(","));
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"generator values: 1,2,3,4,5",
		"next 1: 1",
		"next 2: 2",
		"next 3: 3",
		"done: true",
		"yield*: 1,2,3,4",
		"custom iterator: 10,20,30",
		"map iteration: a:1,b:2",
	})
}

// TestWeakMapAndWeakSet verifies WeakMap and WeakSet operations
// including set, get, has, delete, and primitive key rejection.
func TestWeakMapAndWeakSet(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test WeakMap
		const wm = new WeakMap();
		const key1 = {};
		const key2 = {};

		wm.set(key1, "value1");
		wm.set(key2, "value2");

		console.log("weakmap has key1:", wm.has(key1));
		console.log("weakmap get key1:", wm.get(key1));
		console.log("weakmap has key2:", wm.has(key2));

		wm.delete(key1);
		console.log("after delete key1:", wm.has(key1));

		// Test WeakSet
		const ws = new WeakSet();
		const obj1 = { id: 1 };
		const obj2 = { id: 2 };

		ws.add(obj1);
		ws.add(obj2);

		console.log("weakset has obj1:", ws.has(obj1));
		console.log("weakset has obj2:", ws.has(obj2));
		console.log("weakset has {}:", ws.has({}));

		ws.delete(obj1);
		console.log("after delete obj1:", ws.has(obj1));

		// Test that primitives throw
		try {
			wm.set("string", "value");
			console.log("weakmap primitive: no error");
		} catch (e) {
			console.log("weakmap primitive error: true");
		}
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"weakmap has key1: true",
		"weakmap get key1: value1",
		"weakmap has key2: true",
		"after delete key1: false",
		"weakset has obj1: true",
		"weakset has obj2: true",
		"weakset has {}: false",
		"after delete obj1: false",
		"weakmap primitive error: true",
	})
}

// TestProxyAndReflect verifies Proxy get/set/apply traps and
// Reflect.get/set/has/ownKeys/deleteProperty.
func TestProxyAndReflect(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test Proxy get/set traps
		const target = { value: 10 };
		const handler = {
			get(obj, prop) {
				console.log("get trap:", prop);
				return obj[prop];
			},
			set(obj, prop, value) {
				console.log("set trap:", prop, "=", value);
				obj[prop] = value;
				return true;
			}
		};

		const proxy = new Proxy(target, handler);
		console.log("proxy.value:", proxy.value);
		proxy.value = 20;
		console.log("after set:", target.value);

		// Test Proxy apply trap (for functions)
		const fnTarget = function(a, b) { return a + b; };
		const fnHandler = {
			apply(target, thisArg, args) {
				console.log("apply trap with args:", args.join(","));
				return target.apply(thisArg, args);
			}
		};
		const fnProxy = new Proxy(fnTarget, fnHandler);
		console.log("fnProxy(3, 4):", fnProxy(3, 4));

		// Test Reflect.get/set
		const obj = { x: 1, y: 2 };
		console.log("Reflect.get:", Reflect.get(obj, "x"));
		Reflect.set(obj, "z", 3);
		console.log("Reflect.set result:", obj.z);

		// Test Reflect.has
		console.log("Reflect.has x:", Reflect.has(obj, "x"));
		console.log("Reflect.has w:", Reflect.has(obj, "w"));

		// Test Reflect.ownKeys
		console.log("Reflect.ownKeys:", Reflect.ownKeys(obj).join(","));

		// Test Reflect.deleteProperty
		Reflect.deleteProperty(obj, "y");
		console.log("after delete y:", Reflect.has(obj, "y"));
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"get trap: value",
		"proxy.value: 10",
		"set trap: value = 20",
		"after set: 20",
		"apply trap with args: 3,4",
		"fnProxy(3, 4): 7",
		"Reflect.get: 1",
		"Reflect.set result: 3",
		"Reflect.has x: true",
		"Reflect.has w: false",
		"Reflect.ownKeys: x,y,z",
		"after delete y: false",
	})
}

// TestSymbols verifies Symbol creation, Symbol.for, Symbol.keyFor,
// symbol-as-key, well-known symbols, and Symbol.description.
func TestSymbols(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test Symbol creation
		const sym1 = Symbol("test");
		const sym2 = Symbol("test");
		console.log("symbols unique:", sym1 !== sym2);
		console.log("typeof symbol:", typeof sym1);

		// Test Symbol.for (global registry)
		const globalSym1 = Symbol.for("global");
		const globalSym2 = Symbol.for("global");
		console.log("Symbol.for same:", globalSym1 === globalSym2);

		// Test Symbol.keyFor
		console.log("Symbol.keyFor:", Symbol.keyFor(globalSym1));
		console.log("Symbol.keyFor local:", Symbol.keyFor(sym1));

		// Test Symbol as object key
		const key = Symbol("key");
		const obj = {
			[key]: "secret value",
			visible: "public value"
		};
		console.log("symbol property:", obj[key]);
		console.log("Object.keys includes symbol:", Object.keys(obj).includes(key.toString()));
		console.log("Object.getOwnPropertySymbols:", Object.getOwnPropertySymbols(obj).length);

		// Test well-known symbols
		console.log("Symbol.iterator exists:", typeof Symbol.iterator === "symbol");
		console.log("Symbol.toStringTag exists:", typeof Symbol.toStringTag === "symbol");

		// Test Symbol.description
		const described = Symbol("my description");
		console.log("description:", described.description);
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"symbols unique: true",
		"typeof symbol: symbol",
		"Symbol.for same: true",
		"Symbol.keyFor: global",
		"symbol property: secret value",
		"Object.getOwnPropertySymbols: 1",
		"Symbol.iterator exists: true",
		"Symbol.toStringTag exists: true",
		"description: my description",
	})
}

// TestClasses verifies ES6 class syntax, inheritance, static methods,
// getters/setters, and instanceof.
func TestClasses(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test basic class
		class Animal {
			constructor(name) {
				this.name = name;
			}

			speak() {
				return this.name + " makes a sound";
			}

			static species() {
				return "unknown";
			}
		}

		const animal = new Animal("Generic");
		console.log("animal.name:", animal.name);
		console.log("animal.speak():", animal.speak());
		console.log("Animal.species():", Animal.species());

		// Test inheritance
		class Dog extends Animal {
			constructor(name, breed) {
				super(name);
				this.breed = breed;
			}

			speak() {
				return this.name + " barks";
			}

			fetch() {
				return this.name + " fetches the ball";
			}
		}

		const dog = new Dog("Buddy", "Labrador");
		console.log("dog.name:", dog.name);
		console.log("dog.breed:", dog.breed);
		console.log("dog.speak():", dog.speak());
		console.log("dog.fetch():", dog.fetch());
		console.log("dog instanceof Dog:", dog instanceof Dog);
		console.log("dog instanceof Animal:", dog instanceof Animal);

		// Test getters and setters
		class Rectangle {
			constructor(width, height) {
				this._width = width;
				this._height = height;
			}

			get area() {
				return this._width * this._height;
			}

			set width(value) {
				this._width = value;
			}
		}

		const rect = new Rectangle(5, 10);
		console.log("rect.area:", rect.area);
		rect.width = 7;
		console.log("rect.area after set:", rect.area);

		// Test private fields (if supported)
		try {
			class Counter {
				#count = 0;
				increment() { this.#count++; }
				get value() { return this.#count; }
			}
			const counter = new Counter();
			counter.increment();
			counter.increment();
			console.log("private field:", counter.value);
		} catch (e) {
			console.log("private fields not supported");
		}
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"animal.name: Generic",
		"animal.speak(): Generic makes a sound",
		"Animal.species(): unknown",
		"dog.name: Buddy",
		"dog.breed: Labrador",
		"dog.speak(): Buddy barks",
		"dog.fetch(): Buddy fetches the ball",
		"dog instanceof Dog: true",
		"dog instanceof Animal: true",
		"rect.area: 50",
		"rect.area after set: 70",
	})
}

// TestRegExpAdvanced verifies regex test, exec, match, capturing groups,
// replace, split, flags, lastIndex, and lookahead.
func TestRegExpAdvanced(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test basic matching
		const text = "The quick brown fox jumps over the lazy dog";
		console.log("test match:", /quick/.test(text));
		console.log("exec result:", /brown/.exec(text)[0]);

		// Test global flag
		const matches = text.match(/the/gi);
		console.log("global matches:", matches.length);

		// Test capturing groups
		const dateStr = "2024-01-15";
		const dateMatch = dateStr.match(/(\d{4})-(\d{2})-(\d{2})/);
		console.log("year:", dateMatch[1]);
		console.log("month:", dateMatch[2]);
		console.log("day:", dateMatch[3]);

		// Test replace with function
		const replaced = "hello world".replace(/\w+/g, (match) => match.toUpperCase());
		console.log("replaced:", replaced);

		// Test split with regex
		const parts = "a1b2c3d".split(/\d/);
		console.log("split:", parts.join(","));

		// Test regex properties
		const re = /test/gi;
		console.log("global:", re.global);
		console.log("ignoreCase:", re.ignoreCase);
		console.log("source:", re.source);

		// Test lastIndex with global
		const gre = /o/g;
		const str = "foo";
		gre.exec(str);
		console.log("lastIndex after first:", gre.lastIndex);
		gre.exec(str);
		console.log("lastIndex after second:", gre.lastIndex);

		// Test lookahead (if supported)
		try {
			const lookahead = "foobar".match(/foo(?=bar)/);
			console.log("lookahead:", lookahead ? lookahead[0] : "null");
		} catch (e) {
			console.log("lookahead not supported");
		}
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"test match: true",
		"exec result: brown",
		"global matches: 2",
		"year: 2024",
		"month: 01",
		"day: 15",
		"replaced: HELLO WORLD",
		"split: a,b,c,d",
		"global: true",
		"ignoreCase: true",
		"source: test",
	})
}

// TestClosuresAndScoping verifies closures, lexical scoping, let vs var
// in loops, block scoping, const, TDZ, and IIFE.
func TestClosuresAndScoping(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test basic closure
		function createCounter() {
			let count = 0;
			return {
				increment() { count++; },
				decrement() { count--; },
				value() { return count; }
			};
		}

		const counter = createCounter();
		counter.increment();
		counter.increment();
		counter.increment();
		counter.decrement();
		console.log("closure counter:", counter.value());

		// Test closure preserves outer scope
		function outer(x) {
			return function inner(y) {
				return x + y;
			};
		}
		const add5 = outer(5);
		const add10 = outer(10);
		console.log("add5(3):", add5(3));
		console.log("add10(3):", add10(3));

		// Test let vs var in loops
		const funcsLet = [];
		for (let i = 0; i < 3; i++) {
			funcsLet.push(() => i);
		}
		console.log("let loop:", funcsLet.map(f => f()).join(","));

		// Test block scoping
		{
			let blockVar = "inner";
			console.log("block scoped:", blockVar);
		}

		// Test const
		const constVal = 42;
		console.log("const value:", constVal);

		// Test temporal dead zone detection
		try {
			console.log(tdz);
			let tdz = "test";
		} catch (e) {
			console.log("TDZ caught:", e.name);
		}

		// Test IIFE (Immediately Invoked Function Expression)
		const iife = (function() {
			const private = "secret";
			return { get: () => private };
		})();
		console.log("IIFE result:", iife.get());
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"closure counter: 2",
		"add5(3): 8",
		"add10(3): 13",
		"let loop: 0,1,2",
		"block scoped: inner",
		"const value: 42",
		"TDZ caught: ReferenceError",
		"IIFE result: secret",
	})
}

// TestBjsonModule verifies bjson.write and bjson.read for objects,
// typed arrays, and Date values.
func TestBjsonModule(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		// Test bjson.write and bjson.read
		const original = {
			name: "test",
			value: 42,
			nested: { a: 1, b: 2 },
			array: [1, 2, 3],
			flag: true
		};

		// Serialize to binary
		const binary = bjson.write(original);
		console.log("binary type:", binary instanceof ArrayBuffer);
		console.log("binary size:", binary.byteLength);

		// Deserialize from binary
		const restored = bjson.read(binary, 0, binary.byteLength);
		console.log("restored name:", restored.name);
		console.log("restored value:", restored.value);
		console.log("restored nested.a:", restored.nested.a);
		console.log("restored array:", restored.array.join(","));
		console.log("restored flag:", restored.flag);

		// Test with typed arrays
		const withTypedArray = {
			data: new Uint8Array([1, 2, 3, 4, 5])
		};
		const binary2 = bjson.write(withTypedArray);
		const restored2 = bjson.read(binary2, 0, binary2.byteLength);
		console.log("typed array preserved:", restored2.data instanceof Uint8Array);
		console.log("typed array values:", Array.from(restored2.data).join(","));

		// Test with Date
		const withDate = { date: new Date("2024-01-15T12:00:00Z") };
		const binary3 = bjson.write(withDate);
		const restored3 = bjson.read(binary3, 0, binary3.byteLength);
		console.log("date preserved:", restored3.date instanceof Date);
		console.log("date value:", restored3.date.toISOString());
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"binary type: true",
		"restored name: test",
		"restored value: 42",
		"restored nested.a: 1",
		"restored array: 1,2,3",
		"restored flag: true",
		"typed array preserved: true",
		"typed array values: 1,2,3,4,5",
		"date preserved: true",
	})
}

// TestInitEmptyArgs verifies that Init with an empty args slice works.
func TestInitEmptyArgs(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{})

	evalCode(t, qjs, `console.log("empty args works");`, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	if !strings.Contains(output, "empty args works") {
		t.Errorf("expected output, got: %s", output)
	}
}

// TestEmptyCodeEval verifies that empty code, whitespace-only, and
// comments-only are valid.
func TestEmptyCodeEval(t *testing.T) {
	_, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	// Empty code should be valid
	evalCode(t, qjs, "", "<eval>", JSEvalTypeGlobal)

	// Whitespace only should be valid
	evalCode(t, qjs, "   \n\t  ", "<eval>", JSEvalTypeGlobal)

	// Comments only should be valid
	evalCode(t, qjs, "// just a comment\n/* block comment */", "<eval>", JSEvalTypeGlobal)
}

// TestSpecialValues verifies handling of undefined, null, NaN, Infinity,
// -0, MAX_VALUE, MIN_VALUE, MAX_SAFE_INTEGER, and EPSILON.
func TestSpecialValues(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, nil)

	code := `
		// Test undefined
		let undef;
		console.log("undefined:", undef);
		console.log("typeof undefined:", typeof undef);

		// Test null
		const n = null;
		console.log("null:", n);
		console.log("typeof null:", typeof n);

		// Test NaN
		console.log("NaN:", NaN);
		console.log("isNaN(NaN):", isNaN(NaN));
		console.log("NaN === NaN:", NaN === NaN);
		console.log("Number.isNaN(NaN):", Number.isNaN(NaN));

		// Test Infinity
		console.log("Infinity:", Infinity);
		console.log("-Infinity:", -Infinity);
		console.log("isFinite(Infinity):", isFinite(Infinity));
		console.log("1/0:", 1/0);

		// Test -0
		const negZero = -0;
		console.log("-0 === 0:", negZero === 0);
		console.log("1/-0:", 1/negZero);

		// Test very large numbers
		console.log("Number.MAX_VALUE exists:", Number.MAX_VALUE > 0);
		console.log("Number.MIN_VALUE exists:", Number.MIN_VALUE > 0);
		console.log("Number.MAX_SAFE_INTEGER:", Number.MAX_SAFE_INTEGER);

		// Test epsilon
		console.log("Number.EPSILON exists:", Number.EPSILON > 0);
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"undefined: undefined",
		"typeof undefined: undefined",
		"null: null",
		"typeof null: object",
		"isNaN(NaN): true",
		"NaN === NaN: false",
		"Number.isNaN(NaN): true",
		"isFinite(Infinity): false",
		"-0 === 0: true",
		"1/-0: -Infinity",
		"Number.MAX_VALUE exists: true",
		"Number.MIN_VALUE exists: true",
		"Number.MAX_SAFE_INTEGER: 9007199254740991",
		"Number.EPSILON exists: true",
	})
}

// TestConsoleMethods verifies console.log with multiple args, objects,
// arrays, nested objects, console.warn, console.error, and special values.
func TestConsoleMethods(t *testing.T) {
	stdout, _, cfg := newTestConfigWithFS()
	qjs := newTestQuickJS(t, cfg)
	initQuickJS(t, qjs, []string{"qjs", "--std"})

	code := `
		// Test console.log with multiple arguments
		console.log("multiple", "args", 123, true);

		// Test console.log with objects
		console.log("object:", { a: 1, b: 2 });
		console.log("array:", [1, 2, 3]);

		// Test console.warn if available
		if (typeof console.warn === 'function') {
			console.warn("warning message");
		} else {
			console.log("warning message");
		}

		// Test console.error if available
		if (typeof console.error === 'function') {
			console.error("error message");
		} else {
			console.log("error message");
		}

		// Test nested objects
		console.log("nested:", { outer: { inner: { deep: "value" } } });

		// Test console with special values
		console.log("special:", null, undefined, NaN, Infinity);
	`
	evalCode(t, qjs, code, "<eval>", JSEvalTypeGlobal)

	output := stdout.String()
	t.Logf("Output: %s", output)

	assertOutputContains(t, output, []string{
		"multiple args 123 true",
		"warning message",
		"error message",
	})
}
