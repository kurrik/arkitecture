// Package arkitecture compiles the Arkitecture DSL (.ark) into SVG architecture
// diagrams with manual, deterministic layout.
//
// This package is the library: the CLI (cmd/arkitecture) and the WASM shim
// (wasm/) are thin wrappers over ToSVG. The pipeline is a one-way, side-effect
// free sequence of pure stages — parse, validate, generate — and failures are
// collected as []Error rather than thrown across stage boundaries.
package arkitecture

import (
	"fmt"

	"github.com/kurrik/arkitecture/ast"
	"github.com/kurrik/arkitecture/generator"
	"github.com/kurrik/arkitecture/parser"
	"github.com/kurrik/arkitecture/resolve"
	"github.com/kurrik/arkitecture/validator"
)

// The AST and diagnostic types are re-exported as aliases so callers can use
// arkitecture.Document, arkitecture.Error, etc. without importing ast directly.
type (
	Document      = ast.Document
	ContainerNode = ast.ContainerNode
	Declarations  = ast.Declarations
	LayoutRule    = ast.LayoutRule
	Block         = ast.Block
	Use           = ast.Use
	Arrow         = ast.Arrow
	Direction     = ast.Direction
	Box           = ast.Box
	Error         = ast.Error
	ErrorType     = ast.ErrorType
	ParseResult   = ast.ParseResult
)

// Re-exported enum values for convenience.
const (
	Vertical   = ast.Vertical
	Horizontal = ast.Horizontal

	BoxDefault = ast.BoxDefault
	BoxNone    = ast.BoxNone

	ErrorSyntax     = ast.ErrorSyntax
	ErrorReference  = ast.ErrorReference
	ErrorConstraint = ast.ErrorConstraint
)

// Options configures a ToSVG run. The zero value is valid and uses defaults.
type Options struct {
	// ValidateOnly parses and validates but skips SVG generation.
	ValidateOnly bool
	// FontSize overrides the default font size (12). Zero means default.
	FontSize int
	// FontFamily overrides the default font family (Arial). Empty means default.
	FontFamily string
}

// Result is the output of ToSVG: the SVG on success, or the collected errors.
type Result struct {
	Success bool
	SVG     string
	Errors  []Error
}

// ToSVG runs the full pipeline: parse -> validate -> generate. It never panics
// across stages; any unexpected panic is recovered and returned as an internal
// error. A nil opts is treated as the zero Options.
func ToSVG(dsl string, opts *Options) (result Result) {
	if opts == nil {
		opts = &Options{}
	}
	defer func() {
		if r := recover(); r != nil {
			result = Result{Success: false, Errors: []Error{{
				Type:    ErrorSyntax,
				Message: internalMessage(r),
			}}}
		}
	}()

	parsed := parser.Parse(dsl)
	if !parsed.Success || parsed.Document == nil {
		return Result{Success: false, Errors: parsed.Errors}
	}

	if errs := validator.Validate(parsed.Document); len(errs) > 0 {
		return Result{Success: false, Errors: errs}
	}

	if opts.ValidateOnly {
		return Result{Success: true}
	}

	layout := resolve.Resolve(parsed.Document)
	svg, errs := generator.GenerateSVG(parsed.Document, layout, generator.Options{
		FontSize:   opts.FontSize,
		FontFamily: opts.FontFamily,
	})
	if len(errs) > 0 {
		return Result{Success: false, Errors: errs}
	}
	return Result{Success: true, SVG: svg}
}

// Parse tokenizes and parses DSL content into a Document AST, returning all
// syntax errors collected along the way.
func Parse(dsl string) ParseResult {
	return parser.Parse(dsl)
}

// Validate runs semantic checks on an already-parsed document.
func Validate(doc *Document) []Error {
	return validator.Validate(doc)
}

// GenerateSVG lays out and renders an already-parsed document: it resolves the
// layout layer onto the semantic tree, then generates SVG. A nil opts is
// treated as the zero Options.
func GenerateSVG(doc *Document, opts *Options) (string, []Error) {
	if opts == nil {
		opts = &Options{}
	}
	layout := resolve.Resolve(doc)
	return generator.GenerateSVG(doc, layout, generator.Options{
		FontSize:   opts.FontSize,
		FontFamily: opts.FontFamily,
	})
}

func internalMessage(r any) string {
	if err, ok := r.(error); ok {
		return "Internal error: " + err.Error()
	}
	return fmt.Sprintf("Internal error: %v", r)
}
