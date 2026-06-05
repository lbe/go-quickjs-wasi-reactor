// Command repl provides an interactive JavaScript REPL backed by QuickJS
// compiled as a WASI reactor via wasm2go. It supports reading JS from stdin
// line-by-line, evaluating it with std/os/bjson globals (--std), and printing
// console.log output to stdout. Pass a filename as the first argument to
// evaluate a file instead of starting the interactive loop.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	quickjs "github.com/lbe/go-quickjs-wasi-reactor/wasm2go-quickjs"
	wasihost "github.com/lbe/wasm2go-wasi-host"
)

func main() {
	// Configure the module with stdin, stdout, stderr, and filesystem
	fsRoot, err := os.OpenRoot(".")
	if err != nil {
		log.Panic(err)
	}

	cfg := wasihost.NewModuleConfig().
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithFS(wasihost.NewFSConfig().WithFSMount("/", fsRoot.FS()))

	// Create QuickJS instance
	qjs := quickjs.NewQuickJS(cfg)

	// Initialize runtime with --std (provides std, os, bjson globals)
	if err := qjs.Init(nil); err != nil {
		log.Panicf("failed to init QuickJS: %v", err)
	}

	// If a file argument is provided, read and execute it
	if len(os.Args) > 1 {
		filename := os.Args[1]
		code, err := os.ReadFile(filename)
		if err != nil {
			log.Panicf("failed to read file %s: %v", filename, err)
		}

		_, _ = qjs.Eval(string(code), filename, quickjs.JSEvalTypeGlobal)

		// Run the event loop until complete
		qjs.RunLoop()
	} else {
		// Interactive REPL mode
		runREPL(qjs)
	}
}

// runREPL reads JavaScript line-by-line from stdin, evaluates each
// non-empty line in the QuickJS runtime, and runs the event loop after
// each evaluation. The loop exits on EOF (Ctrl+D), "exit", or "quit".
func runREPL(qjs *quickjs.QuickJS) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !scanner.Scan() {
			// EOF (Ctrl+D)
			fmt.Println()
			break
		}

		line := scanner.Text()

		// Check for exit command
		if line == "exit" || line == "quit" {
			break
		}

		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			_, _ = qjs.Eval(line, "<repl>", quickjs.JSEvalTypeGlobal)
			qjs.RunLoop()
		}
	}
}
