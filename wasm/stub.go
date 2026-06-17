//go:build !js || !wasm

// This stub lets the wasm package build (and `go vet ./...` pass) on a normal
// host toolchain. The real entry point in main.go is compiled only for
// GOOS=js GOARCH=wasm.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "arkitecture wasm shim: build with GOOS=js GOARCH=wasm")
	os.Exit(1)
}
