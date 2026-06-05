module github.com/aperturerobotics/go-quickjs-wasi-reactor/wasm2go-quickjs

go 1.26.3

require (
	github.com/aperturerobotics/go-quickjs-wasi-reactor v0.15.0
	github.com/lbe/wasm2go-wasi-host v0.1.0
)

require golang.org/x/sys v0.44.0 // indirect

replace github.com/lbe/wasm2go-wasi-host => ../../wasm2go-wasi-host
