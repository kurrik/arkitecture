package generator

import (
	"fmt"
	"math"
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
	if arrows := renderArrows(doc.Arrows, layout); arrows != "" {
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
	if isBordered(l.decls) {
		*els = append(*els, nodeRect(l.dim))
	}
	if l.node.Label != nil && *l.node.Label != "" {
		*els = append(*els, nodeText(*l.node.Label, l.dim, fontSize, fontFamily))
	}
	for _, c := range l.children {
		collectNodeElements(c, fontSize, fontFamily, els)
	}
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

func renderArrows(arrows []ast.Arrow, layout layoutResult) string {
	var els []string
	for _, a := range arrows {
		src := resolveEndpoint(a.Source, a.Target, layout)
		tgt := resolveEndpoint(a.Target, a.Source, layout)
		if src == nil || tgt == nil {
			continue // missing nodes/anchors are reported by the validator
		}
		els = append(els, fmt.Sprintf(`  <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="black" stroke-width="1" marker-end="url(#arrowhead)" />`,
			num(src.x), num(src.y), num(tgt.x), num(tgt.y)))
	}
	return strings.Join(els, "\n")
}

// resolveEndpoint finds where an arrow attaches to self. An explicit #anchor
// (named, or #center) uses that anchor's fixed position. A bare reference
// auto-routes: it attaches to the cardinal edge (N/E/S/W) of self's border box
// facing the other node's centre.
func resolveEndpoint(self, other string, layout layoutResult) *point {
	path, anchor, explicit := splitRef(self)
	if explicit {
		ap := findAnchor(layout.anchorPositions, path, anchor)
		if ap == nil {
			return nil
		}
		return &point{ap.x, ap.y}
	}
	selfBox, ok := layout.nodeBoxes[path]
	if !ok {
		return nil
	}
	otherBox, ok := layout.nodeBoxes[nodePathOf(other)]
	if !ok {
		return nil
	}
	return cardinalPoint(selfBox, centerOf(otherBox))
}

// cardinalPoint returns the midpoint of box's edge facing aim, chosen by the
// dominant axis of the centre-to-aim vector. Exact diagonals (|dx| == |dy|)
// favour the horizontal (E/W) side.
func cardinalPoint(box dimensions, aim point) *point {
	c := centerOf(box)
	dx, dy := aim.x-c.x, aim.y-c.y
	if math.Abs(dx) >= math.Abs(dy) {
		if dx >= 0 {
			return &point{box.x + box.width, c.y} // east
		}
		return &point{box.x, c.y} // west
	}
	if dy >= 0 {
		return &point{c.x, box.y + box.height} // south
	}
	return &point{c.x, box.y} // north
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
