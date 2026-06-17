//go:build js && wasm

// Command wasm exposes the arkitecture library to JavaScript when built with
// GOOS=js GOARCH=wasm. It registers a global function
// arkitectureToSVG(dsl, opts?) that returns { success, svg, errors }, mirroring
// the library's Result.
package main

import (
	"syscall/js"

	"github.com/kurrik/arkitecture"
)

func main() {
	js.Global().Set("arkitectureToSVG", js.FuncOf(toSVG))
	select {} // keep the Go runtime alive to serve callbacks
}

func toSVG(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return resultObject(false, "", []arkitecture.Error{{
			Type:    arkitecture.ErrorSyntax,
			Message: "arkitectureToSVG expects a DSL string as its first argument",
		}})
	}

	opts := &arkitecture.Options{}
	if len(args) >= 2 && args[1].Type() == js.TypeObject {
		o := args[1]
		if v := o.Get("validateOnly"); v.Type() == js.TypeBoolean {
			opts.ValidateOnly = v.Bool()
		}
		if v := o.Get("fontSize"); v.Type() == js.TypeNumber {
			opts.FontSize = v.Int()
		}
		if v := o.Get("fontFamily"); v.Type() == js.TypeString {
			opts.FontFamily = v.String()
		}
	}

	res := arkitecture.ToSVG(args[0].String(), opts)
	return resultObject(res.Success, res.SVG, res.Errors)
}

func resultObject(success bool, svg string, errs []arkitecture.Error) map[string]any {
	jsErrs := make([]any, 0, len(errs))
	for _, e := range errs {
		jsErrs = append(jsErrs, map[string]any{
			"line":    e.Line,
			"column":  e.Column,
			"message": e.Message,
			"type":    string(e.Type),
		})
	}
	return map[string]any{"success": success, "svg": svg, "errors": jsErrs}
}
