// Package generator turns a validated Document into an SVG string: text
// measurement, bottom-up layout with anchor resolution, and SVG emission. It is
// pure and deterministic — the same document always yields the same SVG.
package generator

import "github.com/kurrik/arkitecture/ast"

// Options controls SVG generation. A FontSize <= 0 and an empty FontFamily mean
// "use the defaults" (Arial 12px).
type Options struct {
	FontSize   int
	FontFamily string
}

const (
	defaultFontSize   = 12.0
	defaultFontFamily = "Arial"
)

// GenerateSVG lays out the document and renders it to an SVG string. resolved
// is the per-node layout produced by the resolve stage (keyed by full dotted
// path); a nil map is valid and means every node uses defaults. Generation
// itself does not produce errors (reference and constraint problems are the
// validator's job), so the error slice is always empty; the signature keeps the
// stage uniform with the rest of the pipeline.
func GenerateSVG(doc *ast.Document, resolved map[string]*ast.Declarations, opts Options) (string, []ast.Error) {
	if doc == nil {
		return "", nil
	}

	fontSize := defaultFontSize
	if opts.FontSize > 0 {
		fontSize = float64(opts.FontSize)
	}
	fontFamily := defaultFontFamily
	if opts.FontFamily != "" {
		fontFamily = opts.FontFamily
	}

	layout := computeLayout(doc, resolved, fontSize, nil)
	// Channel widening: in orthogonal mode, learn each channel's lane demand (and
	// which lane each arrow takes) from a first routing pass, then lay out again
	// with those channels widened so arrows get their own lanes instead of sitting
	// in the boxes' margins.
	var lanes laneMap
	if routeMode(doc) == ast.RouteOrthogonal {
		if demand, lm := channelDemand(doc.Arrows, layout, ast.RouteOrthogonal); demand != nil {
			layout = computeLayout(doc, resolved, fontSize, demand)
			lanes = lm
		}
	}
	return renderSVG(doc, layout, fontSize, fontFamily, lanes), nil
}
