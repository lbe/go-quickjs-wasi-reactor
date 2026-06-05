# wasm2go-quickjs

High-level Go API for running JavaScript using QuickJS-NG with the wasm2go transpiler and wasihost.

## Installation

```bash
go get github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs
```

## Usage

### Evaluate JavaScript code

```go
package main

import (
    "fmt"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs"
    wasihost "github.com/lbe/wasm2go-wasi-host"
)

func main() {
    cfg := &wasihost.ModuleConfig{}
    qjs := quickjs.NewQuickJS(cfg)
    defer qjs.Close()

    result, isException := qjs.Eval(`console.log("Hello from QuickJS!")`, "<eval>", quickjs.JSEvalTypeGlobal)
    if isException {
        panic("JS exception")
    }
    _ = result

    qjs.RunLoop()
}
```

### Initialization Options

```go
// Basic runtime with std modules available for import
qjs.Init(nil)
qjs.Eval(`import * as std from 'qjs:std'; std.printf("Hello\n")`, "<eval>", quickjs.JSEvalTypeModule)

// With --std flag to expose std, os, bjson as globals
qjs.Init([]string{"qjs", "--std"})
qjs.Eval(`std.printf("Hello\n")`, "<eval>", quickjs.JSEvalTypeGlobal)

// With script args accessible via scriptArgs global
qjs.Init([]string{"qjs", "script.js", "--verbose"})
qjs.Eval(`console.log(scriptArgs)`, "<eval>", quickjs.JSEvalTypeGlobal)
```

## API

### `NewQuickJS(config) *QuickJS`

Creates a new QuickJS instance. The returned instance must be closed with `Close()` to release resources.

### `(*QuickJS) Init(args) error`

Initializes the QuickJS runtime and context. Modules `qjs:std`, `qjs:os`, and `qjs:bjson` can be imported in evaluated code.

Pass `nil` or empty slice for default initialization. Pass `[]string{"qjs", "--std"}` to expose `std`, `os`, and `bjson` as globals.

### `(*QuickJS) Eval(code, filename, flags) (int64, bool)`

Evaluates JavaScript code. Returns the raw JSValue result and a boolean indicating whether the result is an exception. Use `JSEvalTypeGlobal` (0) for global scripts and `JSEvalTypeModule` (1) for ES modules.

### `(*QuickJS) EvalWithFilename(code, filename) (int64, bool)`

Convenience wrapper for `Eval` that uses `JSEvalTypeGlobal` as the eval flags.

### `(*QuickJS) RunLoop()`

Processes the QuickJS event loop until it becomes idle. This blocks until all JavaScript execution completes.

### `(*QuickJS) Close()`

Destroys the QuickJS runtime and releases resources. Calling `Close()` twice is safe (no-op).

## Subdirectories

### `example/`

A minimal example demonstrating library usage.

```bash
cd example && go run .
```

### `repl/`

A command-line JavaScript runner with interactive REPL mode.

```bash
# Run directly
cd repl && go run .

# Install globally
go install github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs/repl@master

# Interactive REPL (no arguments)
repl

# Run a JavaScript file
repl script.js

# Run as ES module
repl script.mjs --module
```

## Notes

- `setTimeout` and `setInterval` are on the `os` module, not global (use `os.setTimeout()`)
- The std module provides environment variable access via `std.getenv()`, `std.setenv()`, etc.
- Promises work out of the box; `RunLoop` will wait for all pending promises
- The API does not require `context.Context` — wasm2go transpiles the WASM module to native Go code
