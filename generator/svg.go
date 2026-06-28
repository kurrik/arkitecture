package generator

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/kurrik/arkitecture/ast"
)

// The built-in plain look, used when neither a node nor the document default sets
// a style. The colour sentinels are the literal strings the original output used
// ("white"/"black"), so an unstyled diagram renders byte-for-byte as before.
const (
	defaultBorderColor     = "black"
	defaultBackgroundColor = "white"
	defaultPathColor       = "black"
	defaultPathWidth       = 1.0
)

func renderSVG(doc *ast.Document, layout layoutResult, fontSize float64, fontFamily string, lanes laneMap, resolved map[string]*ast.Declarations) string {
	arrows, markerColors, labelBox := renderArrows(doc.Arrows, layout, routeMode(doc), lanes, resolved, doc.Defaults, fontSize, fontFamily)

	// Arrow labels can extend past the box bounds (a straight-mode label offset
	// into open canvas); grow the viewport to include them so they are not
	// clipped. (In orthogonal mode the widened channel already made room.)
	minX, minY := layout.viewMinX, layout.viewMinY
	w, h := layout.canvasWidth, layout.canvasHeight
	if labelBox != nil {
		maxX, maxY := math.Max(minX+w, labelBox.maxX), math.Max(minY+h, labelBox.maxY)
		minX, minY = math.Min(minX, labelBox.minX), math.Min(minY, labelBox.minY)
		w, h = maxX-minX, maxY-minY
	}

	parts := []string{
		fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="%s %s %s %s">`,
			num(w), num(h), num(minX), num(minY), num(w), num(h)),
		buildDefs(markerColors),
	}

	if nodes := renderNodes(doc, layout, fontSize, fontFamily); nodes != "" {
		parts = append(parts, "", "  <!-- Node rectangles and labels -->", nodes)
	}
	if arrows != "" {
		parts = append(parts, "", "  <!-- Arrows -->", arrows)
	}
	parts = append(parts, "</svg>")

	return strings.Join(parts, "\n")
}

// buildDefs emits the <defs> with one arrowhead marker per path colour: the black
// default (id "arrowhead", kept byte-identical to the original) plus one coloured
// marker per distinct non-default path colour, so a coloured arrow gets a matching
// arrowhead. colors is sorted and free of the default.
func buildDefs(colors []string) string {
	var b strings.Builder
	b.WriteString("  <defs>\n")
	b.WriteString(markerDef("arrowhead", defaultPathColor))
	for _, c := range colors {
		b.WriteString(markerDef(markerID(c), c))
	}
	b.WriteString("  </defs>")
	return b.String()
}

// markerDef renders one arrowhead <marker> with the given id and fill. The layout
// (including the trailing space after markerHeight) matches the original verbatim
// marker so the unstyled defs stay byte-for-byte stable.
func markerDef(id, fill string) string {
	return fmt.Sprintf("    <marker id=\"%s\" markerWidth=\"10\" markerHeight=\"7\" \n"+
		"            refX=\"9\" refY=\"3.5\" orient=\"auto\" markerUnits=\"strokeWidth\">\n"+
		"      <polygon points=\"0 0, 10 3.5, 0 7\" fill=\"%s\" />\n"+
		"    </marker>\n", id, fill)
}

// markerID is the arrowhead marker id for a path colour: the shared "arrowhead"
// for the default black, else "arrowhead-<hex>" (the '#' stripped for a valid id).
func markerID(color string) string {
	if color == defaultPathColor {
		return "arrowhead"
	}
	return "arrowhead-" + strings.TrimPrefix(color, "#")
}

func renderNodes(doc *ast.Document, layout layoutResult, fontSize float64, fontFamily string) string {
	var els []string
	for _, l := range layout.roots {
		collectNodeElements(l, fontSize, fontFamily, doc.Defaults, &els)
	}
	return strings.Join(els, "\n")
}

func collectNodeElements(l *layoutNode, fontSize float64, fontFamily string, defStyle *ast.Declarations, els *[]string) {
	if nodeBordered(l) {
		*els = append(*els, nodeRect(l, defStyle))
	}
	if label, ok := nodeLabel(l); ok {
		*els = append(*els, nodeText(label, labelDim(l), fontSize, fontFamily))
	}
	for _, c := range l.children {
		collectNodeElements(c, fontSize, fontFamily, defStyle, els)
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

// nodeRect renders a bordered node's rectangle with its resolved fill, stroke,
// and stroke width. shape-rendering="crispEdges" keeps the 1px (or wider) border
// a consistent width regardless of the box's sub-pixel position — without it, a
// border whose coordinate lands off the half-pixel grid anti-aliases into a
// fainter, wider smear, so otherwise-identical boxes look inconsistent.
func nodeRect(l *layoutNode, defStyle *ast.Declarations) string {
	d := l.dim
	return fmt.Sprintf(`  <rect x="%s" y="%s" width="%s" height="%s" fill="%s" stroke="%s" stroke-width="%s" shape-rendering="crispEdges" />`,
		num(d.x), num(d.y), num(d.width), num(d.height),
		backgroundColorOf(l.decls, defStyle), borderColorOf(l.decls, defStyle), num(l.borderW))
}

// --- resolved style accessors (node value, else document default, else built-in) ---

func borderColorOf(d, def *ast.Declarations) string {
	if d != nil && d.BorderColor != nil {
		return *d.BorderColor
	}
	if def != nil && def.BorderColor != nil {
		return *def.BorderColor
	}
	return defaultBorderColor
}

func backgroundColorOf(d, def *ast.Declarations) string {
	if d != nil && d.BackgroundColor != nil {
		return *d.BackgroundColor
	}
	if def != nil && def.BackgroundColor != nil {
		return *def.BackgroundColor
	}
	return defaultBackgroundColor
}

func pathColorOf(d, def *ast.Declarations) string {
	if d != nil && d.PathColor != nil {
		return *d.PathColor
	}
	if def != nil && def.PathColor != nil {
		return *def.PathColor
	}
	return defaultPathColor
}

func pathWidthOf(d, def *ast.Declarations) float64 {
	if d != nil && d.PathWidth != nil {
		return *d.PathWidth
	}
	if def != nil && def.PathWidth != nil {
		return *def.PathWidth
	}
	return defaultPathWidth
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

// renderArrows draws every arrow and reports the distinct non-default path colours
// it used, so the caller can emit a matching arrowhead marker for each. An arrow's
// width and colour come from its **source** node's resolved pathWidth/pathColor
// (falling back to the document default, then the plain 1px black), so styling a
// node restyles the arrows that start at it.
func renderArrows(arrows []ast.Arrow, layout layoutResult, mode ast.RouteMode, lanes laneMap, resolved map[string]*ast.Declarations, defStyle *ast.Declarations, fontSize float64, fontFamily string) (string, []string, *bbox) {
	var els, labels []string
	colorSet := map[string]bool{}
	var lbox *bbox
	for i, a := range arrows {
		pts, ok := arrowPath(a, layout, mode)
		if !ok {
			continue // missing nodes/anchors are reported by the validator
		}
		if mode == ast.RouteOrthogonal {
			pts = snapToLanes(pts, layout, i, lanes) // place each run in its lane
		}
		src := resolved[nodePathOf(a.Source)]
		color := pathColorOf(src, defStyle)
		width := pathWidthOf(src, defStyle)
		if color != defaultPathColor {
			colorSet[color] = true
		}
		els = append(els, arrowElement(pts, color, width))
		if a.Label != nil {
			if p0, p1, ok := longestSegment(pts); ok {
				el, b := arrowLabel(*a.Label, p0, p1, fontSize, fontFamily)
				labels = append(labels, el)
				lbox = lbox.union(b)
			}
		}
	}
	// Labels render after every line so a later arrow never draws over a label.
	els = append(els, labels...)
	return strings.Join(els, "\n"), sortedColorKeys(colorSet), lbox
}

// bbox is an axis-aligned bounding box used to grow the viewport around labels.
type bbox struct{ minX, minY, maxX, maxY float64 }

// union returns the smallest box covering both b and o (b may be nil).
func (b *bbox) union(o bbox) *bbox {
	if b == nil {
		return &o
	}
	return &bbox{math.Min(b.minX, o.minX), math.Min(b.minY, o.minY), math.Max(b.maxX, o.maxX), math.Max(b.maxY, o.maxY)}
}

// longestSegment returns the endpoints of an arrow path's longest segment (the
// single segment of a straight arrow; the dominant run of a bent one). ok is
// false for a degenerate path with no segment.
func longestSegment(pts []point) (p0, p1 point, ok bool) {
	best := -1.0
	for i := 0; i+1 < len(pts); i++ {
		dx, dy := pts[i+1].x-pts[i].x, pts[i+1].y-pts[i].y
		if l := dx*dx + dy*dy; l > best {
			best = l
			p0, p1, ok = pts[i], pts[i+1], true
		}
	}
	return p0, p1, ok
}

// arrowLabel draws an arrow's label centred at the midpoint of its longest
// segment (p0→p1), on an opaque plate so the line beneath stays legible. In
// orthogonal mode that midpoint sits in the channel the run follows, which the
// widening pass has grown to hold the text (a margin of clearance each side); in
// straight mode the viewport grows around the plate so it is not clipped. The
// segment endpoints are unused for positioning today but pin the label to the
// dominant run.
func arrowLabel(label string, p0, p1 point, fontSize float64, fontFamily string) (string, bbox) {
	const pad = 2.0
	w := textWidth(label, fontSize)
	h := textHeight(label, fontSize)
	at := point{(p0.x + p1.x) / 2, (p0.y + p1.y) / 2}

	rx, ry, rw, rh := at.x-w/2-pad, at.y-h/2-pad, w+2*pad, h+2*pad
	rect := fmt.Sprintf(`  <rect x="%s" y="%s" width="%s" height="%s" fill="%s" />`,
		num(rx), num(ry), num(rw), num(rh), defaultBackgroundColor)
	el := rect + "\n" + nodeText(label, dimensions{x: at.x - w/2, y: at.y - h/2, width: w, height: h}, fontSize, fontFamily)
	return el, bbox{rx, ry, rx + rw, ry + rh}
}

// arrowElement renders an arrow's resolved polyline with the given stroke colour
// and width. A two-point path emits a <line>, a bent path a <polyline>. An
// axis-aligned path gets shape-rendering="crispEdges" so horizontal/vertical runs
// stay a consistent width (a diagonal straight line keeps its anti-aliasing). The
// arrowhead marker — colour-matched to the stroke — sits at the final point.
func arrowElement(pts []point, color string, width float64) string {
	crisp := ""
	if axisAligned(pts) {
		crisp = ` shape-rendering="crispEdges"`
	}
	marker := markerID(color)
	if len(pts) == 2 {
		return fmt.Sprintf(`  <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s"%s marker-end="url(#%s)" />`,
			num(pts[0].x), num(pts[0].y), num(pts[1].x), num(pts[1].y), color, num(width), crisp, marker)
	}
	coords := make([]string, len(pts))
	for i, p := range pts {
		coords[i] = num(p.x) + "," + num(p.y)
	}
	return fmt.Sprintf(`  <polyline points="%s" fill="none" stroke="%s" stroke-width="%s"%s marker-end="url(#%s)" />`,
		strings.Join(coords, " "), color, num(width), crisp, marker)
}

// axisAligned reports whether every segment of pts runs horizontally or
// vertically (so crispEdges is safe). A diagonal segment makes it false.
func axisAligned(pts []point) bool {
	if len(pts) < 2 {
		return false
	}
	for i := 0; i+1 < len(pts); i++ {
		if math.Abs(pts[i].x-pts[i+1].x) > epsilon && math.Abs(pts[i].y-pts[i+1].y) > epsilon {
			return false
		}
	}
	return true
}

func sortedColorKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
