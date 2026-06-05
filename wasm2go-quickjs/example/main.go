package main

import (
	"fmt"
	"io"
	"log"

	quickjs "github.com/lbe/go-quickjs-wasi-reactor/wasm2go-quickjs"
	wasihost "github.com/lbe/wasm2go-wasi-host"
)

func main() {
	cfg := wasihost.NewModuleConfig().
		WithStdout(io.Discard).
		WithStderr(io.Discard)

	qjs := quickjs.NewQuickJS(cfg)

	if err := qjs.Init(nil); err != nil {
		log.Fatal("Init:", err)
	}
	fmt.Println("Init OK")

	_, _ = qjs.Eval(`console.log("hello");`, "<eval>", quickjs.JSEvalTypeGlobal)
	fmt.Println("Eval OK")

	qjs.RunLoop()
	fmt.Println("RunLoop OK")
}
