package generator

import (
	"math"

	"github.com/kurrik/arkitecture/ast"
)

// defaultBorderWidth is the 1px border a bordered node draws when its resolved
// layout (and the document default) set no borderWidth.
const defaultBorderWidth = 1.0

// defaultMargin is the uniform space a node reserves around its border box when
// its resolved layout sets no explicit margin. It is non-zero so flush-packed
// siblings get a visible gap (and auto-routed arrows room to travel); author
// `margin: 0` in @layout to restore the old flush look.
const defaultMargin = 8.0

type dimensions struct {
	width, height, x, y float64
}

type anchorPosition struct {
	x, y     float64
	nodeID   string // full dotted path
	anchorID string
}

type layoutResult struct {
	roots           []*layoutNode
	nodeBoxes       map[string]dimensions // border boxes keyed by full dotted path
	anchorPositions []anchorPosition
	canvasWidth     float64
	canvasHeight    float64
	viewMinX        float64 // viewBox origin: content shifted so perimeter strokes aren't clipped
	viewMinY        float64
	defMargin       float64 // the document's effective default margin (channel width)
}

type layoutNode struct {
	node      *ast.ContainerNode // nil for a synthetic @group wrapper
	path      string             // full dotted path ("" for a group)
	decls     *ast.Declarations  // resolved layout for this node (may be nil)
	dim       dimensions         // the border box: the visible rectangle (content + border)
	margin    float64            // uniform margin around the border box (the margin box)
	labelBand float64            // reserved label-strip height (0 = no reserved strip)
	borderW   float64            // effective border width (geometry inset + stroke width)
	isGroup   bool               // true for an anonymous @group: invisible, unaddressable
	children  []*layoutNode
	grid      *gridInfo // non-nil when this node lays its children out as a grid

	// Channel widening (route: orthogonal). gapExtra[i] is extra main-axis space
	// reserved at gap i (0 = leading perimeter … len(children) = trailing perimeter,
	// i = before child i) for arrows routing along that gap; railExtra is the same
	// for the two cross-axis perimeter rails (0 = low/left/top side, 1 = high). Both
	// default to zero, so a document without widened channels lays out unchanged.
	gapExtra  []float64
	railExtra [2]float64
}

// widenDemand is the extra width each channel reserves for the arrows routing
// along it, keyed by container dotted path ("" = the document root). It is the
// output of the routing pass and the input to the second (widened) layout pass.
type widenDemand struct {
	gaps  map[string][]float64  // container path -> per-gap extra (len = children+1)
	rails map[string][2]float64 // container path -> [low, high] cross-axis perimeter extra
	root  []float64             // extra at each gap between top-level nodes (len = roots+1)
}

// gapExtraAt returns the widening reserved at gap i of l (0 when none).
func gapExtraAt(l *layoutNode, i int) float64 {
	if i >= 0 && i < len(l.gapExtra) {
		return l.gapExtra[i]
	}
	return 0
}

// annotateWidening copies a container's channel widening from the demand onto the
// layout tree (by path), so calcDimensions/positionNodes can reserve it.
func annotateWidening(l *layoutNode, d *widenDemand) {
	if d != nil && !l.isGroup {
		l.gapExtra = d.gaps[l.path]
		l.railExtra = d.rails[l.path]
	}
	for _, c := range l.children {
		annotateWidening(c, d)
	}
}

// computeLayout sizes and positions every node bottom-up then top-down, sizes
// the canvas to fit, and resolves anchor coordinates. It is deterministic. demand
// is the channel widening from the routing pass (nil on the first/un-widened pass).
func computeLayout(doc *ast.Document, layout map[string]*ast.Declarations, fontSize float64, demand *widenDemand) layoutResult {
	// The document may override the built-in default margin with a bare `margin:`
	// at an @layout sheet root; it is the fallback for any node that sets none.
	defMargin := defaultMargin
	if doc.DefaultMargin != nil {
		defMargin = *doc.DefaultMargin
	}
	// The document may also default the border width (a bare `borderWidth:` at a
	// sheet root); it is the fallback for any node that sets none.
	defBW := defaultBorderWidth
	if doc.Defaults != nil && doc.Defaults.BorderWidth != nil {
		defBW = *doc.Defaults.BorderWidth
	}

	roots := make([]*layoutNode, 0, len(doc.Nodes))
	for _, n := range doc.Nodes {
		l := buildLayoutTree(n, "", layout)
		annotateWidening(l, demand)
		calcDimensions(l, fontSize, defMargin, defBW)
		roots = append(roots, l)
	}

	// The document root is invisible: top-level nodes pack left-to-right with
	// only the collapsed gap between siblings (the larger of their facing
	// margins) — there is no perimeter, so the canvas gains no phantom padding.
	// A widened top-level gap (demand.root) spreads the siblings further.
	currentX := 0.0
	for i, l := range roots {
		if i > 0 {
			currentX += math.Max(roots[i-1].margin, l.margin)
			if demand != nil && i < len(demand.root) {
				currentX += demand.root[i]
			}
		}
		positionNodes(l, currentX, 0)
		currentX += l.dim.width
	}

	// The canvas fits the content, but a border stroke is *centred* on its box
	// edge — half of it sits outside the box. On the outermost edges that half
	// would fall outside the SVG viewport and be clipped (rendering perimeter
	// borders at half width), so grow the view bounds to include every stroke's
	// outer half. Emitted as a viewBox offset, leaving element coordinates intact.
	cw, ch := canvasSize(roots)
	minX, minY, maxX, maxY := 0.0, 0.0, cw, ch
	if sMinX, sMinY, sMaxX, sMaxY, ok := strokeBounds(roots); ok {
		minX, minY = math.Min(minX, sMinX), math.Min(minY, sMinY)
		maxX, maxY = math.Max(maxX, sMaxX), math.Max(maxY, sMaxY)
	}

	boxes := make(map[string]dimensions)
	var anchors []anchorPosition
	for _, l := range roots {
		collectBoxes(l, boxes)
		anchors = append(anchors, collectAnchors(l)...)
	}

	return layoutResult{roots: roots, nodeBoxes: boxes, anchorPositions: anchors, canvasWidth: maxX - minX, canvasHeight: maxY - minY, viewMinX: minX, viewMinY: minY, defMargin: defMargin}
}

func buildLayoutTree(node *ast.ContainerNode, parentPath string, layout map[string]*ast.Declarations) *layoutNode {
	path := node.ID
	if parentPath != "" {
		path = parentPath + "." + node.ID
	}
	l := &layoutNode{node: node, path: path, decls: layout[path]}
	if l.decls != nil && len(l.decls.Arrangement) > 0 {
		l.children = buildArrangement(l.decls.Arrangement, node, path, layout)
	} else {
		for _, c := range node.Children {
			l.children = append(l.children, buildLayoutTree(c, path, layout))
		}
	}
	return l
}

// buildArrangement turns a node's resolved arrangement into layout children:
// direct child nodes and synthetic invisible @group wrappers. A child inside a
// group keeps the *node's* path (parentPath) — a group adds no path segment,
// since it is purely presentational. Foreign/duplicate/missing child references
// are the validator's concern; here an unknown id is simply skipped.
func buildArrangement(items []ast.ArrangementItem, parent *ast.ContainerNode, parentPath string, layout map[string]*ast.Declarations) []*layoutNode {
	byID := make(map[string]*ast.ContainerNode, len(parent.Children))
	for _, c := range parent.Children {
		byID[c.ID] = c
	}
	var out []*layoutNode
	for _, it := range items {
		if it.Group != nil {
			g := &layoutNode{decls: it.Group, isGroup: true}
			g.children = buildArrangement(it.Group.Arrangement, parent, parentPath, layout)
			out = append(out, g)
			continue
		}
		if child, ok := byID[it.ChildID]; ok {
			out = append(out, buildLayoutTree(child, parentPath, layout))
		}
	}
	return out
}

// calcDimensions sizes l and its subtree bottom-up and sets l.margin to l's
// *effective* margin — the space a parent reserves around it. A bordered node's
// border is the boundary, so its effective margin is just its own. A `box: none`
// node is transparent: it adds no perimeter of its own, and its children's
// margins show through it, so its effective margin is the larger of its own and
// its children's. That single effective margin then collapses against neighbours
// like any other, so a box:none group never doubles a channel.
func calcDimensions(l *layoutNode, fontSize, defMargin, defBW float64) {
	for _, c := range l.children {
		calcDimensions(c, fontSize, defMargin, defBW)
	}

	ownMargin := marginOf(l.decls, defMargin)
	direction := directionOf(l.decls)
	minDim := fontSize * 2
	bordered := nodeBordered(l)
	bw := borderWidthOf(l.decls, defBW)
	l.borderW = bw

	if len(l.children) == 0 {
		l.margin = ownMargin
		if l.node == nil {
			return // an empty @group occupies no space
		}
		label, _ := nodeLabel(l)
		l.dim.width = math.Max(textWidth(label, fontSize)+2*bw, minDim)
		l.dim.height = math.Max(textHeight(label, fontSize)+2*bw, minDim)
		return
	}

	// A grid node sizes its children's tracks jointly on both axes — the one
	// thing 1-D packing can't do — instead of stacking along a single direction.
	if l.decls != nil && l.decls.Grid != nil {
		calcGrid(l, fontSize, ownMargin, bordered, bw)
		return
	}

	// Effective margin: a transparent box:none group carries its children's
	// margins outward (the larger of its own and theirs).
	l.margin = ownMargin
	if !bordered {
		for _, c := range l.children {
			l.margin = math.Max(l.margin, c.margin)
		}
	}

	// A labelled parent reserves a strip for its label — a top (default) or
	// bottom band, sized like a leaf box holding that label — so the label is
	// never obscured by the children, which lay out in the remaining area. In a
	// bordered parent the band's inner edge is a wall the children's facing margin
	// lands against; a box:none parent packs its children flush below the band,
	// just as it packs them flush everywhere (it adds no perimeter of its own).
	// labelW keeps the box at least as wide as its label.
	var band, labelW float64
	if label, ok := nodeLabel(l); ok {
		band = labelBandHeight(label, fontSize, bw)
		labelW = textWidth(label, fontSize) + 2*bw
	}
	l.labelBand = band

	// Channels between boxes collapse rather than stack: the gap between two
	// adjacent siblings is the larger of their facing margins (not the sum), so
	// every channel is one uniform margin wide. A bordered parent additionally
	// reserves a perimeter (the edge children's margins become padding inside the
	// border) and stretches children to fill the cross axis; a transparent
	// box:none parent (and the root) reserves no perimeter and does not stretch,
	// bounding the children's border boxes plus the collapsed gaps only.
	// Channel widening reserves extra space at the gaps an arrow routes along:
	// gapExtra grows the main axis (at each between-children gap and the
	// leading/trailing perimeter), railExtra grows the cross axis (the two
	// perimeter rails). Both are zero unless the routing pass widened a channel,
	// so an un-widened document is byte-identical.
	rail := l.railExtra[0] + l.railExtra[1]
	n := len(l.children)
	if direction == ast.Horizontal {
		main, cross := 0.0, 0.0
		for i, c := range l.children {
			if i > 0 {
				main += math.Max(l.children[i-1].margin, c.margin) + gapExtraAt(l, i)
			}
			main += c.dim.width
			ch := c.dim.height
			if bordered {
				ch += 2 * c.margin
			}
			cross = math.Max(cross, ch)
		}
		if bordered && n > 0 {
			main += l.children[0].margin + l.children[n-1].margin
		}
		main += gapExtraAt(l, 0) + gapExtraAt(l, n)
		// The band is a full-width strip stacked above/below the children area
		// (cross); the label must also fit across the main (width) axis.
		l.dim.width = math.Max(main, labelW)
		l.dim.height = cross + band + rail
		if bordered {
			for _, c := range l.children {
				c.dim.height = cross - 2*c.margin
			}
		}
	} else {
		main, cross := 0.0, 0.0
		for i, c := range l.children {
			if i > 0 {
				main += math.Max(l.children[i-1].margin, c.margin) + gapExtraAt(l, i)
			}
			main += c.dim.height
			cw := c.dim.width
			if bordered {
				cw += 2 * c.margin
			}
			cross = math.Max(cross, cw)
		}
		if bordered && n > 0 {
			main += l.children[0].margin + l.children[n-1].margin
		}
		main += gapExtraAt(l, 0) + gapExtraAt(l, n)
		// The band is a full-width strip stacked above/below the children stack
		// (main); the label must also fit across the cross (width) axis.
		contentW := math.Max(cross, labelW)
		l.dim.height = main + band
		l.dim.width = contentW + rail
		if bordered {
			for _, c := range l.children {
				c.dim.width = contentW - 2*c.margin
			}
		}
	}

	// Apply the size override to the orthogonal dimension, after children have
	// been stretched to the pre-override parent size.
	if size := sizeOf(l.decls); size != nil {
		if direction == ast.Horizontal {
			l.dim.height *= *size
		} else {
			l.dim.width *= *size
		}
	}
}

// positionNodes places l's border box at (x, y) and lays out its children with
// collapsed channels (mirroring calcDimensions). A bordered parent insets the
// first child by its margin (perimeter) and insets each child on the cross axis
// by its margin; a transparent box:none parent (and the root) places children at
// its own origin. Between two siblings the gap is the larger of their facing
// margins — margins collapse, they do not stack.
func positionNodes(l *layoutNode, x, y float64) {
	l.dim.x, l.dim.y = x, y
	direction := directionOf(l.decls)
	bordered := nodeBordered(l)

	// A top label band shifts the child-layout origin down past the reserved
	// strip; a bottom band leaves children at the top and sits below them.
	childY := y
	if l.labelBand > 0 && labelPositionOf(l.decls) != ast.LabelBottom {
		childY = y + l.labelBand
	}

	if l.grid != nil {
		positionGrid(l, x, childY)
		return
	}

	// Mirror calcDimensions' widening: the leading gap and each between-children
	// gap advance the main-axis cursor by gapExtra; the low-side rail offsets every
	// child on the cross axis by railExtra[0].
	if direction == ast.Horizontal {
		cursor := x + gapExtraAt(l, 0)
		if bordered && len(l.children) > 0 {
			cursor += l.children[0].margin
		}
		for i, c := range l.children {
			if i > 0 {
				cursor += math.Max(l.children[i-1].margin, c.margin) + gapExtraAt(l, i)
			}
			cy := childY + l.railExtra[0]
			if bordered {
				cy += c.margin
			}
			positionNodes(c, cursor, cy)
			cursor += c.dim.width
		}
	} else {
		cursor := childY + gapExtraAt(l, 0)
		if bordered && len(l.children) > 0 {
			cursor += l.children[0].margin
		}
		for i, c := range l.children {
			if i > 0 {
				cursor += math.Max(l.children[i-1].margin, c.margin) + gapExtraAt(l, i)
			}
			cx := x + l.railExtra[0]
			if bordered {
				cx += c.margin
			}
			positionNodes(c, cx, cursor)
			cursor += c.dim.height
		}
	}
}

func canvasSize(roots []*layoutNode) (w, h float64) {
	for _, l := range roots {
		w = math.Max(w, l.dim.x+l.dim.width)
		h = math.Max(h, l.dim.y+l.dim.height)
	}
	return w, h
}

// strokeBounds returns the bounding box of every bordered node's drawn stroke —
// each border box grown by half its border width on all sides, since the stroke
// is centred on the edge. ok is false when nothing is bordered (a box:none-only
// diagram), in which case the caller keeps the plain content bounds.
func strokeBounds(roots []*layoutNode) (minX, minY, maxX, maxY float64, ok bool) {
	var visit func(l *layoutNode)
	visit = func(l *layoutNode) {
		if nodeBordered(l) {
			h := l.borderW / 2
			x0, y0 := l.dim.x-h, l.dim.y-h
			x1, y1 := l.dim.x+l.dim.width+h, l.dim.y+l.dim.height+h
			if !ok {
				minX, minY, maxX, maxY, ok = x0, y0, x1, y1, true
			} else {
				minX, minY = math.Min(minX, x0), math.Min(minY, y0)
				maxX, maxY = math.Max(maxX, x1), math.Max(maxY, y1)
			}
		}
		for _, c := range l.children {
			visit(c)
		}
	}
	for _, l := range roots {
		visit(l)
	}
	return minX, minY, maxX, maxY, ok
}

// collectAnchors yields the resolved anchor positions for l and its descendants:
// the implicit centre plus each declared anchor name at its layout position
// (defaulting to centre when unpositioned). Groups are anonymous and contribute
// no anchors.
func collectAnchors(l *layoutNode) []anchorPosition {
	var out []anchorPosition
	if !l.isGroup {
		out = resolveNodeAnchors(l)
	}
	for _, c := range l.children {
		out = append(out, collectAnchors(c)...)
	}
	return out
}

// collectBoxes records each node's border box keyed by its full dotted path.
// Arrow routing uses it to find cardinal edges by path. Groups are anonymous
// (no path) and are skipped.
func collectBoxes(l *layoutNode, out map[string]dimensions) {
	if !l.isGroup {
		out[l.path] = l.dim
	}
	for _, c := range l.children {
		collectBoxes(c, out)
	}
}

func resolveNodeAnchors(l *layoutNode) []anchorPosition {
	d := l.dim
	out := []anchorPosition{{
		x: d.x + d.width*0.5, y: d.y + d.height*0.5, nodeID: l.path, anchorID: "center",
	}}
	for _, name := range l.node.Anchors {
		rx, ry := 0.5, 0.5 // an unpositioned named anchor defaults to centre
		if l.decls != nil {
			if pos, ok := l.decls.Anchors[name]; ok {
				rx, ry = pos[0], pos[1]
			}
		}
		if rx < 0 || rx > 1 || ry < 0 || ry > 1 {
			continue // out-of-range anchors are reported by the validator
		}
		out = append(out, anchorPosition{
			x: d.x + d.width*rx, y: d.y + d.height*ry, nodeID: l.path, anchorID: name,
		})
	}
	return out
}

// --- resolved-layout accessors (apply defaults for unset properties) ---

// nodeBordered reports whether a layout node draws a border (and so insets its
// children like padding). An anonymous @group is always invisible; otherwise it
// follows the resolved box property.
func nodeBordered(l *layoutNode) bool {
	return !l.isGroup && isBordered(l.decls)
}

// isBordered reports whether a declaration set draws a border. A box:none node
// and the document root are invisible: they collapse their children's perimeter
// margins and do not stretch children to fill the cross axis.
func isBordered(d *ast.Declarations) bool {
	return !(d != nil && d.Box != nil && *d.Box == ast.BoxNone)
}

// marginOf returns a node's own margin, falling back to def (the document
// default, itself defaulting to the built-in defaultMargin) when it sets none.
func marginOf(d *ast.Declarations, def float64) float64 {
	if d != nil && d.Margin != nil {
		return *d.Margin
	}
	return def
}

// borderWidthOf returns a node's effective border width, falling back to def
// (the document default, itself defaulting to defaultBorderWidth) when it sets
// none. It governs both the box geometry (content fits inside the border) and the
// rendered stroke width.
func borderWidthOf(d *ast.Declarations, def float64) float64 {
	if d != nil && d.BorderWidth != nil {
		return *d.BorderWidth
	}
	return def
}

func directionOf(d *ast.Declarations) ast.Direction {
	if d != nil && d.Direction != nil && *d.Direction != ast.DirectionUnset {
		return *d.Direction
	}
	return ast.Vertical
}

func sizeOf(d *ast.Declarations) *float64 {
	if d != nil {
		return d.Size
	}
	return nil
}

// nodeLabel returns a layout node's non-empty label text, if it has one. A
// synthetic @group (node == nil) never does.
func nodeLabel(l *layoutNode) (string, bool) {
	if l.node != nil && l.node.Label != nil && *l.node.Label != "" {
		return *l.node.Label, true
	}
	return "", false
}

// labelBandHeight is the strip a bordered, labelled parent reserves for its
// label. It is sized exactly like a leaf box holding that label, so a group's
// title reads as a consistent row. bw is the parent's effective border width.
func labelBandHeight(label string, fontSize, bw float64) float64 {
	return math.Max(textHeight(label, fontSize)+2*bw, fontSize*2)
}

// labelPositionOf returns the resolved label-strip position (default top).
func labelPositionOf(d *ast.Declarations) ast.LabelPosition {
	if d != nil && d.LabelPos != nil && *d.LabelPos != ast.LabelPositionUnset {
		return *d.LabelPos
	}
	return ast.LabelTop
}
