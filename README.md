# go-quickjs-wasi-reactor

[![GoDoc Widget]][GoDoc] [![Go Report Card Widget]][Go Report Card]

> A Go module that embeds the QuickJS-NG WASI WebAssembly runtime (reactor model).

[GoDoc]: https://godoc.org/github.com/aperturerobotics/go-quickjs-wasi-reactor
[GoDoc Widget]: https://godoc.org/github.com/aperturerobotics/go-quickjs-wasi-reactor?status.svg
[Go Report Card Widget]: https://goreportcard.com/badge/github.com/aperturerobotics/go-quickjs-wasi-reactor
[Go Report Card]: https://goreportcard.com/report/github.com/aperturerobotics/go-quickjs-wasi-reactor

## Related Projects

- [js-quickjs-wasi-reactor](https://github.com/aperturerobotics/js-quickjs-wasi-reactor) - JavaScript/TypeScript harness for browser environments
- [go-quickjs-wasi](https://github.com/paralin/go-quickjs-wasi) - Standard WASI command model with blocking `_start()` entry point
- [QuickJS-NG](https://github.com/quickjs-ng/quickjs) - Upstream QuickJS-NG (includes WASI reactor build)

## Variants

This repository provides the **reactor model** WASM binary for re-entrant execution. If you only need simple blocking execution where QuickJS runs to completion in `_start()`, see the **command model** variant above.

## About QuickJS-NG

QuickJS is a small and embeddable JavaScript engine. It aims to support the latest ECMAScript specification.

This project uses [QuickJS-NG], a community-driven fork of the original
[QuickJS project] by Fabrice Bellard and Charlie Gordon. Both projects are
actively maintained. The WASI reactor build target is part of upstream QuickJS-NG,
and the WASM binary is pulled directly from QuickJS-NG GitHub release artifacts.

[QuickJS-NG]: https://github.com/quickjs-ng/quickjs
[QuickJS project]: https://bellard.org/quickjs/

## Purpose

This module provides easy access to the QuickJS-NG JavaScript engine compiled to
WebAssembly with WASI support using the **reactor model**. The WASM binary is
embedded directly in the Go module, making it easy to use QuickJS in Go
applications without external dependencies.

### Reactor Model

Unlike the standard WASI "command" model that blocks in `_start()`, the reactor
model exports the raw QuickJS C API functions, enabling full control over the
JavaScript runtime lifecycle from the host environment.

The reactor exports the complete QuickJS C API including:

**Core Runtime:**

- `JS_NewRuntime`, `JS_FreeRuntime` - Runtime lifecycle
- `JS_NewContext`, `JS_FreeContext` - Context lifecycle
- `JS_Eval` - Evaluate JavaScript code
- `JS_Call` - Call JavaScript functions

**Standard Library (quickjs-libc.h):**

- `js_init_module_std`, `js_init_module_os`, `js_init_module_bjson` - Module initialization
- `js_std_init_handlers`, `js_std_free_handlers` - I/O handler setup
- `js_std_add_helpers` - Add console.log, print, etc.
- `js_std_loop_once` - Run one iteration of the event loop (non-blocking)
- `js_std_poll_io` - Poll for I/O events

**Memory Management:**

- `malloc`, `free`, `realloc`, `calloc` - For host to allocate memory

## Features

- Embeds the QuickJS-NG WASI reactor WebAssembly binary
- Provides version information about the embedded QuickJS release
- High-level Go API via the `wasm2go-quickjs` subpackage
- Update script to build and copy from local QuickJS checkout

## Packages

### Root Package (`github.com/aperturerobotics/go-quickjs-wasi-reactor`)

Provides the embedded WASM binary and version information:

```go
package main

import (
    "fmt"
    quickjswasi "github.com/aperturerobotics/go-quickjs-wasi-reactor"
)

func main() {
    // Access the embedded WASM binary
    wasmBytes := quickjswasi.QuickJSWASM
    fmt.Printf("QuickJS WASM size: %d bytes\n", len(wasmBytes))

    // Get version information
    fmt.Printf("QuickJS version: %s\n", quickjswasi.Version)
    fmt.Printf("Download URL: %s\n", quickjswasi.DownloadURL)
}
```

### wasm2go QuickJS Library (`github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs`)

High-level Go API for running JavaScript using the wasm2go transpiler and wasihost:

```go
package main

import (
    "fmt"
    "os"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs"
)

func main() {
    qjs, err := quickjs.NewQuickJS(nil)
    if err != nil {
        panic(err)
    }
    defer qjs.Close()

    if err := qjs.Init(nil); err != nil {
        panic(err)
    }

    if err := qjs.Eval(`console.log("Hello from QuickJS!")`, false); err != nil {
        panic(err)
    }

    if err := qjs.RunLoop(); err != nil {
        panic(err)
    }
}
```

### Initialization Options

```go
// Basic runtime with std modules available for import
qjs.Init(nil)
qjs.Eval(`import * as std from 'qjs:std'; std.printf("Hello\n")`, true)

// With --std flag to expose std, os, bjson as globals
qjs.Init([]string{"qjs", "--std"})
qjs.Eval(`std.printf("Hello\n")`, false)  // std is already global

// With script args accessible via scriptArgs global
qjs.Init([]string{"qjs", "script.js", "--verbose"})
qjs.Eval(`console.log(scriptArgs)`, false)  // ['qjs', 'script.js', '--verbose']
```

See the [wasm2go-quickjs README](./wasm2go-quickjs/README.md) for more details.

## Command-Line REPL

A command-line JavaScript runner with interactive REPL mode is provided:

```bash
# Install
go install github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs/repl@master

# Interactive REPL
repl

# Run a JavaScript file
repl script.js

# Run as ES module
repl script.mjs --module
```

## Updating

To update to the latest QuickJS-NG reactor build from upstream releases:

```bash
# Download latest release automatically
./update-quickjs.bash

# Or specify a release tag
./update-quickjs.bash v0.12.1
```

This script will:

1. Fetch the latest release tag from `quickjs-ng/quickjs` (or use the provided tag)
2. Download the `qjs-wasi-reactor.wasm` release artifact
3. Generate `version.go` with version and download URL constants

A GitHub Actions workflow (`.github/workflows/update-quickjs.yml`) also runs daily to automatically check for new upstream releases and publish updated versions.

### Building the WASM file from source

To build the WASM file from source:

```bash
cd /path/to/quickjs

# Create build directory
mkdir -p build-wasi-reactor && cd build-wasi-reactor

# Configure with WASI SDK
cmake .. \
  -DCMAKE_TOOLCHAIN_FILE=/opt/wasi-sdk/share/cmake/wasi-sdk.cmake \
  -DQJS_WASI_REACTOR=ON \
  -DCMAKE_BUILD_TYPE=Release

# Build
make -j

# Copy output
cp qjs.wasm ../qjs-wasi-reactor.wasm
```

## Testing

```bash
go test ./...
cd wasm2go-quickjs && go test ./...
```

## License

This module is released under the same license as the embedded QuickJS-NG project.

MIT
