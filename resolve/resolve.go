// Package resolve merges the layout layer onto the semantic tree. It takes the
// parsed document's layout rules (inline `@layout` blocks, desugared by the
// parser into path selectors, plus standalone sheet selectors) and produces a
// per-node resolved layout keyed by full dotted path.
//
// It is a pure function of its input and assumes the document already passed
// validation: dangling selectors and duplicate-property conflicts are the
// validator's job, so here merging is a simple overlay. The two precedence
// tiers from the design (imported `kind`/`@use` below direct declarations)
// collapse to a single direct tier in M3 — `kind`/`@use` arrive in M4 and
// extend this stage with a lower-precedence pass.
package resolve

import "github.com/kurrik/arkitecture/ast"

// Layout maps a node's full dotted path to its merged layout declarations.
// A path missing from the map has no explicit layout; the generator applies
// defaults (vertical, margin 8, bordered).
type Layout map[string]*ast.Declarations

// Resolve overlays every layout rule onto its target path, in source order.
// Within the direct tier scalars are last-write-wins and anchor positions union
// — but the validator rejects duplicate direct properties first, so a valid
// document never actually overwrites.
func Resolve(doc *ast.Document) Layout {
	out := Layout{}
	if doc == nil {
		return out
	}
	for _, rule := range doc.Layout {
		if rule.Decls == nil {
			continue
		}
		mergeInto(out, rule.Selector, rule.Decls)
	}
	return out
}

func mergeInto(out Layout, path string, d *ast.Declarations) {
	dst := out[path]
	if dst == nil {
		dst = &ast.Declarations{}
		out[path] = dst
	}
	if d.Direction != nil {
		dst.Direction = d.Direction
	}
	if d.Size != nil {
		dst.Size = d.Size
	}
	if d.Margin != nil {
		dst.Margin = d.Margin
	}
	if d.Box != nil {
		dst.Box = d.Box
	}
	for name, pos := range d.Anchors {
		if dst.Anchors == nil {
			dst.Anchors = map[string][2]float64{}
		}
		dst.Anchors[name] = pos
	}
}
