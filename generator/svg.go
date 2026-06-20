package generator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kurrik/arkitecture/ast"
)

// defsBlock is emitted verbatim (including the trailing space after
// markerHeight) so output stays byte-for-byte stable with the golden fixtures.
const defsBlock = "  <defs>\n" +
	"    <marker id=\"arrowhead\" markerWidth=\"10\" markerHeight=\"7\" \n" +
	"            refX=\"9\" refY=\"3.5\" orient=\"auto\" markerUnits=\"strokeWidth\">\n" +
	"      <polygon points=\"0 0, 10 3.5, 0 7\" fill=\"black\" />\n" +
	"    </marker>\n" +
	"  </defs>"

func renderSVG(doc *ast.Document, layout layoutResult, fontSize float64, fontFamily string) string {
	parts := []string{
		fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s">`, num(layout.canvasWidth), num(layout.canvasHeight)),
		defsBlock,
	}

	if nodes := renderNodes(doc, layout, fontSize, fontFamily); nodes != "" {
		parts = append(parts, "", "  <!-- Node rectangles and labels -->", nodes)
	}
	if arrows := renderArrows(doc.Arrows, layout, routeMode(doc)); arrows != "" {
		parts = append(parts, "", "  <!-- Arrows -->", arrows)
	}
	parts = append(parts, "</svg>")

	return strings.Join(parts, "\n")
}

func renderNodes(doc *ast.Document, layout layoutResult, fontSize float64, fontFamily string) string {
	var els []string
	for _, l := range layout.roots {
		collectNodeElements(l, fontSize, fontFamily, &els)
	}
	return strings.Join(els, "\n")
}

func collectNodeElements(l *layoutNode, fontSize float64, fontFamily string, els *[]string) {
	if nodeBordered(l) {
		*els = append(*els, nodeRect(l.dim))
	}
	if label, ok := nodeLabel(l); ok {
		*els = append(*els, nodeText(label, labelDim(l), fontSize, fontFamily))
	}
	for _, c := range l.children {
		collectNodeElements(c, fontSize, fontFamily, els)
	}
}

// labelDim is the rectangle a node's label is centred in: the reserved band (at
// the top or bottom) for a labelled parent, or the whole border box otherwise (a
// leaf, or a node with no reserved band).
func labelDim(l *layoutNode) dimensions {
	if l.labelBand <= 0 {
		return l.dim
	}
	d := dimensions{x: l.dim.x, y: l.dim.y, width: l.dim.width, height: l.labelBand}
	if labelPositionOf(l.decls) == ast.LabelBottom {
		d.y = l.dim.y + l.dim.height - l.labelBand
	}
	return d
}

func nodeRect(d dimensions) string {
	return fmt.Sprintf(`  <rect x="%s" y="%s" width="%s" height="%s" fill="white" stroke="black" stroke-width="1" />`,
		num(d.x), num(d.y), num(d.width), num(d.height))
}

func nodeText(label string, d dimensions, fontSize float64, fontFamily string) string {
	cx := d.x + d.width/2
	cy := d.y + d.height/2
	lines := strings.Split(label, "\n")

	if len(lines) == 1 {
		return fmt.Sprintf(`  <text x="%s" y="%s" text-anchor="middle" dominant-baseline="middle" font-family="%s" font-size="%s">%s</text>`,
			num(cx), num(cy), fontFamily, num(fontSize), escapeXML(label))
	}

	lineHeight := fontSize * lineHeightRatio
	totalHeight := float64(len(lines)-1) * lineHeight
	startY := cy - totalHeight/2

	var b strings.Builder
	fmt.Fprintf(&b, `  <text x="%s" y="%s" text-anchor="middle" dominant-baseline="middle" font-family="%s" font-size="%s">`,
		num(cx), num(startY), fontFamily, num(fontSize))
	for i, line := range lines {
		dy := 0.0
		if i != 0 {
			dy = lineHeight
		}
		fmt.Fprintf(&b, "\n    <tspan x=\"%s\" dy=\"%s\">%s</tspan>", num(cx), num(dy), escapeXML(line))
	}
	b.WriteString("\n  </text>")
	return b.String()
}

// point is a resolved coordinate where an arrow attaches.
type point struct{ x, y float64 }

func renderArrows(arrows []ast.Arrow, layout layoutResult, mode ast.RouteMode) string {
	var els []string
	for _, a := range arrows {
		pts, ok := arrowPath(a, layout, mode)
		if !ok {
			continue // missing nodes/anchors are reported by the validator
		}
		els = append(els, arrowElement(pts))
	}
	return strings.Join(els, "\n")
}

// arrowElement renders an arrow's resolved polyline. A two-point path emits the
// same <line> straight mode always has (keeping that output byte-stable); a path
// with a bend emits a <polyline>. The arrowhead marker sits at the final point.
func arrowElement(pts []point) string {
	if len(pts) == 2 {
		return fmt.Sprintf(`  <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="black" stroke-width="1" marker-end="url(#arrowhead)" />`,
			num(pts[0].x), num(pts[0].y), num(pts[1].x), num(pts[1].y))
	}
	coords := make([]string, len(pts))
	for i, p := range pts {
		coords[i] = num(p.x) + "," + num(p.y)
	}
	return fmt.Sprintf(`  <polyline points="%s" fill="none" stroke="black" stroke-width="1" marker-end="url(#arrowhead)" />`,
		strings.Join(coords, " "))
}

func centerOf(d dimensions) point {
	return point{d.x + d.width/2, d.y + d.height/2}
}

// splitRef splits an arrow reference into its node path and optional anchor.
// explicit reports whether an #anchor was present, distinguishing a bare "a"
// (which auto-routes) from "a#center" (which forces the centre).
func splitRef(ref string) (path, anchor string, explicit bool) {
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		return ref[:i], ref[i+1:], true
	}
	return ref, "", false
}

func nodePathOf(ref string) string {
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		return ref[:i]
	}
	return ref
}

func findAnchor(positions []anchorPosition, path, anchorID string) *anchorPosition {
	for i := range positions {
		if positions[i].nodeID == path && positions[i].anchorID == anchorID {
			return &positions[i]
		}
	}
	return nil
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// num formats a layout number the way JavaScript's Number-to-string does:
// shortest round-trip, integers without a decimal point.
func num(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
