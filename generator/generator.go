// Package generator turns a validated Document into an SVG string: text
// measurement, bottom-up layout, anchor resolution, and SVG emission.
//
// TODO(go-port): not yet ported. The TypeScript implementation
// (src/generator/* in the pre-rewrite history) is the reference, and the golden
// fixtures under generator/testdata/golden/ are the target output to reproduce.
package generator

import "github.com/kurrik/arkitecture/ast"

// Options controls SVG generation. A FontSize <= 0 and an empty FontFamily mean
// "use the defaults" (Arial 12px).
type Options struct {
	FontSize   int
	FontFamily string
}

// GenerateSVG renders a document to an SVG string. Stub pending the port: it
// returns a single error so callers surface that generation is unavailable.
func GenerateSVG(doc *ast.Document, opts Options) (string, []ast.Error) {
	_ = doc
	_ = opts
	return "", []ast.Error{{
		Type:    ast.ErrorSyntax,
		Message: "SVG generation is not yet ported to Go",
	}}
}
