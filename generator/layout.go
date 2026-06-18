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
	node     *ast.ContainerNode
	path     string            // full dotted path
	decls    *ast.Declarations // resolved layout for this node (may be nil)
	dim      dimensions        // the border box: the visible rectangle (content + border)
	margin   float64           // uniform margin around the border box (the margin box)
	children []*layoutNode
}

// computeLayout sizes and positions every node bottom-up then top-down, sizes
// the canvas to fit, and resolves anchor coordinates. It is deterministic.
func computeLayout(doc *ast.Document, layout map[string]*ast.Declarations, fontSize float64) layoutResult {
	roots := make([]*layoutNode, 0, len(doc.Nodes))
	for _, n := range doc.Nodes {
		l := buildLayoutTree(n, "", layout)
		calcDimensions(l, fontSize)
		roots = append(roots, l)
	}

	// The document root is invisible: top-level nodes pack left-to-right with
	// only the gaps between siblings (each side's margins summed) — their outer
	// perimeter margins collapse, so the canvas gains no phantom padding.
	currentX := 0.0
	for i, l := range roots {
		if i > 0 {
			currentX += roots[i-1].margin + l.margin
		}
		positionNodes(l, currentX, 0)
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
	for _, c := range node.Children {
		l.children = append(l.children, buildLayoutTree(c, path, layout))
	}
	return l
}

func calcDimensions(l *layoutNode, fontSize float64) {
	for _, c := range l.children {
		calcDimensions(c, fontSize)
	}

	l.margin = marginOf(l.decls)
	direction := directionOf(l.decls)
	minDim := fontSize * 2

	if len(l.children) == 0 {
		label := ""
		if l.node.Label != nil {
			label = *l.node.Label
		}
		l.dim.width = math.Max(textWidth(label, fontSize)+2*borderWidth, minDim)
		l.dim.height = math.Max(textHeight(label, fontSize)+2*borderWidth, minDim)
		return
	}

	// A bordered parent grows to contain each child's *margin* box (margins act
	// like padding inside the border). An invisible parent (box:none or the
	// root) has no wall for perimeter margins to sit against, so it bounds the
	// children's *border* boxes plus the inter-sibling gaps only.
	bordered := isBordered(l.decls)
	if direction == ast.Horizontal {
		sum, maxH := 0.0, 0.0
		for i, c := range l.children {
			if bordered {
				sum += c.dim.width + 2*c.margin
				maxH = math.Max(maxH, c.dim.height+2*c.margin)
			} else {
				if i > 0 {
					sum += l.children[i-1].margin + c.margin
				}
				sum += c.dim.width
				maxH = math.Max(maxH, c.dim.height)
			}
		}
		l.dim.width, l.dim.height = sum, maxH
		if bordered {
			for _, c := range l.children {
				c.dim.height = l.dim.height - 2*c.margin
			}
		}
	} else {
		sum, maxW := 0.0, 0.0
		for i, c := range l.children {
			if bordered {
				sum += c.dim.height + 2*c.margin
				maxW = math.Max(maxW, c.dim.width+2*c.margin)
			} else {
				if i > 0 {
					sum += l.children[i-1].margin + c.margin
				}
				sum += c.dim.height
				maxW = math.Max(maxW, c.dim.width)
			}
		}
		l.dim.height, l.dim.width = sum, maxW
		if bordered {
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

// positionNodes places l's border box at (x, y) and lays out its children. A
// bordered parent insets each child by the child's own margin (perimeter
// margins become padding inside the border); an invisible parent drops the
// perimeter inset and instead separates siblings by the sum of their facing
// margins.
func positionNodes(l *layoutNode, x, y float64) {
	l.dim.x, l.dim.y = x, y
	direction := directionOf(l.decls)
	bordered := isBordered(l.decls)

	if direction == ast.Horizontal {
		cursor := x
		for i, c := range l.children {
			if bordered {
				positionNodes(c, cursor+c.margin, y+c.margin)
				cursor += c.dim.width + 2*c.margin
			} else {
				if i > 0 {
					cursor += l.children[i-1].margin + c.margin
				}
				positionNodes(c, cursor, y)
				cursor += c.dim.width
			}
		}
	} else {
		cursor := y
		for i, c := range l.children {
			if bordered {
				positionNodes(c, x+c.margin, cursor+c.margin)
				cursor += c.dim.height + 2*c.margin
			} else {
				if i > 0 {
					cursor += l.children[i-1].margin + c.margin
				}
				positionNodes(c, x, cursor)
				cursor += c.dim.height
			}
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
// (defaulting to centre when unpositioned).
func collectAnchors(l *layoutNode) []anchorPosition {
	out := resolveNodeAnchors(l)
	for _, c := range l.children {
		out = append(out, collectAnchors(c)...)
	}
	return out
}

// collectBoxes records each node's border box keyed by its full dotted path.
// Arrow routing uses it to find cardinal edges by path.
func collectBoxes(l *layoutNode, out map[string]dimensions) {
	out[l.path] = l.dim
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

// isBordered reports whether a node draws a border and so insets its children
// like padding. A box:none node and the document root are invisible: they
// collapse their children's perimeter margins and do not stretch children to
// fill the cross axis.
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
