package generator

import (
	"math"

	"github.com/kurrik/arkitecture/ast"
)

// borderWidth is the 1px border every bordered node draws.
const borderWidth = 1.0

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
}

type layoutNode struct {
	node     *ast.ContainerNode // nil for a synthetic @group wrapper
	path     string             // full dotted path ("" for a group)
	decls    *ast.Declarations  // resolved layout for this node (may be nil)
	dim      dimensions         // the border box: the visible rectangle (content + border)
	margin   float64            // uniform margin around the border box (the margin box)
	isGroup  bool               // true for an anonymous @group: invisible, unaddressable
	children []*layoutNode
}

// computeLayout sizes and positions every node bottom-up then top-down, sizes
// the canvas to fit, and resolves anchor coordinates. It is deterministic.
func computeLayout(doc *ast.Document, layout map[string]*ast.Declarations, fontSize float64) layoutResult {
	// wall=false at the top level: the document root is invisible and provides
	// no wall, so a top-level node's (and any all-invisible chain's) perimeter
	// margins collapse rather than padding the canvas.
	roots := make([]*layoutNode, 0, len(doc.Nodes))
	for _, n := range doc.Nodes {
		l := buildLayoutTree(n, "", layout)
		calcDimensions(l, fontSize, false)
		roots = append(roots, l)
	}

	// The document root is invisible: top-level nodes pack left-to-right with
	// only the collapsed gap between siblings (the larger of their facing
	// margins) — their outer perimeter margins collapse, so the canvas gains no
	// phantom padding.
	currentX := 0.0
	for i, l := range roots {
		if i > 0 {
			currentX += math.Max(roots[i-1].margin, l.margin)
		}
		positionNodes(l, currentX, 0, false)
		currentX += l.dim.width
	}

	cw, ch := canvasSize(roots)
	boxes := make(map[string]dimensions)
	var anchors []anchorPosition
	for _, l := range roots {
		collectBoxes(l, boxes)
		anchors = append(anchors, collectAnchors(l)...)
	}

	return layoutResult{roots: roots, nodeBoxes: boxes, anchorPositions: anchors, canvasWidth: cw, canvasHeight: ch}
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

// calcDimensions sizes l and its subtree bottom-up. wall reports whether a
// bordered ancestor encloses l: a bordered node is always a wall for its
// subtree, and an invisible (box:none) node passes the flag through unchanged.
// Perimeter margins are reserved (children's margin boxes count, like padding)
// whenever a wall encloses the node — so a box:none group nested in a bordered
// parent pushes its children's margins out to that wall, while an invisible
// chain with no wall (e.g. a top-level group) collapses them.
func calcDimensions(l *layoutNode, fontSize float64, wall bool) {
	wall = wall || nodeBordered(l)
	for _, c := range l.children {
		calcDimensions(c, fontSize, wall)
	}

	l.margin = marginOf(l.decls)
	direction := directionOf(l.decls)
	minDim := fontSize * 2

	if len(l.children) == 0 {
		if l.node == nil {
			return // an empty @group occupies no space
		}
		label := ""
		if l.node.Label != nil {
			label = *l.node.Label
		}
		l.dim.width = math.Max(textWidth(label, fontSize)+2*borderWidth, minDim)
		l.dim.height = math.Max(textHeight(label, fontSize)+2*borderWidth, minDim)
		return
	}

	// Channels between boxes collapse rather than stack: the gap between two
	// adjacent siblings is the larger of their facing margins (not the sum), so
	// it stays one uniform margin wide — matching the gap to the wall. A walled
	// parent reserves that perimeter (the edge children's margins become padding
	// inside the wall) and stretches children to fill the cross axis; a wall-less
	// invisible parent (a top-level box:none group or the root) drops both,
	// bounding the children's border boxes plus the collapsed gaps only.
	n := len(l.children)
	if direction == ast.Horizontal {
		main, cross := 0.0, 0.0
		for i, c := range l.children {
			if i > 0 {
				main += math.Max(l.children[i-1].margin, c.margin)
			}
			main += c.dim.width
			ch := c.dim.height
			if wall {
				ch += 2 * c.margin
			}
			cross = math.Max(cross, ch)
		}
		if wall && n > 0 {
			main += l.children[0].margin + l.children[n-1].margin
		}
		l.dim.width, l.dim.height = main, cross
		if wall {
			for _, c := range l.children {
				c.dim.height = l.dim.height - 2*c.margin
			}
		}
	} else {
		main, cross := 0.0, 0.0
		for i, c := range l.children {
			if i > 0 {
				main += math.Max(l.children[i-1].margin, c.margin)
			}
			main += c.dim.height
			cw := c.dim.width
			if wall {
				cw += 2 * c.margin
			}
			cross = math.Max(cross, cw)
		}
		if wall && n > 0 {
			main += l.children[0].margin + l.children[n-1].margin
		}
		l.dim.height, l.dim.width = main, cross
		if wall {
			for _, c := range l.children {
				c.dim.width = l.dim.width - 2*c.margin
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
// collapsed channels (mirroring calcDimensions). A walled parent insets the
// first child by its margin (perimeter) and insets each child on the cross axis
// by its margin; a wall-less invisible parent drops those insets. Between two
// siblings the gap is the larger of their facing margins — margins collapse,
// they do not stack. wall is threaded as in calcDimensions.
func positionNodes(l *layoutNode, x, y float64, wall bool) {
	wall = wall || nodeBordered(l)
	l.dim.x, l.dim.y = x, y
	direction := directionOf(l.decls)

	if direction == ast.Horizontal {
		cursor := x
		if wall && len(l.children) > 0 {
			cursor += l.children[0].margin
		}
		for i, c := range l.children {
			if i > 0 {
				cursor += math.Max(l.children[i-1].margin, c.margin)
			}
			cy := y
			if wall {
				cy = y + c.margin
			}
			positionNodes(c, cursor, cy, wall)
			cursor += c.dim.width
		}
	} else {
		cursor := y
		if wall && len(l.children) > 0 {
			cursor += l.children[0].margin
		}
		for i, c := range l.children {
			if i > 0 {
				cursor += math.Max(l.children[i-1].margin, c.margin)
			}
			cx := x
			if wall {
				cx = x + c.margin
			}
			positionNodes(c, cx, cursor, wall)
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

func marginOf(d *ast.Declarations) float64 {
	if d != nil && d.Margin != nil {
		return *d.Margin
	}
	return defaultMargin
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
