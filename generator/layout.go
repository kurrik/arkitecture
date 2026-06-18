package generator

import (
	"math"
	"sort"

	"github.com/kurrik/arkitecture/ast"
)

// borderWidth is the 1px border every container node draws.
const borderWidth = 1.0

// defaultMargin is the uniform space a node reserves around its border box when
// it sets no explicit margin. It is non-zero so flush-packed siblings get a
// visible gap (and auto-routed arrows, later, room to travel); author
// `margin: 0` to restore the old flush look.
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
	nodeDimensions  map[string]dimensions // keyed by bare node ID
	anchorPositions []anchorPosition
	canvasWidth     float64
	canvasHeight    float64
}

type layoutNode struct {
	node     ast.Node
	dim      dimensions // the border box: the visible rectangle (content + border)
	margin   float64    // uniform margin around the border box (the margin box)
	children []*layoutNode
}

// computeLayout sizes and positions every node bottom-up then top-down, sizes
// the canvas to fit, and resolves anchor coordinates. It is deterministic.
func computeLayout(doc *ast.Document, fontSize float64) layoutResult {
	dims := make(map[string]dimensions)
	roots := make([]*layoutNode, 0, len(doc.Nodes))

	for _, n := range doc.Nodes {
		l := buildLayoutTree(n)
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
		collectDimensions(l, dims)
		currentX += l.dim.width
	}

	cw, ch := canvasSize(roots)
	anchors := collectAnchors(toNodes(doc.Nodes), dims, "")

	return layoutResult{nodeDimensions: dims, anchorPositions: anchors, canvasWidth: cw, canvasHeight: ch}
}

func buildLayoutTree(node ast.Node) *layoutNode {
	l := &layoutNode{node: node}
	for _, c := range childrenOf(node) {
		l.children = append(l.children, buildLayoutTree(c))
	}
	return l
}

func calcDimensions(l *layoutNode, fontSize float64) {
	for _, c := range l.children {
		calcDimensions(c, fontSize)
	}

	l.margin = marginOf(l.node)
	container, isContainer := l.node.(*ast.ContainerNode)
	direction := directionOf(l.node)
	minDim := fontSize * 2

	if len(l.children) == 0 {
		if isContainer {
			label := ""
			if container.Label != nil {
				label = *container.Label
			}
			l.dim.width = math.Max(textWidth(label, fontSize)+2*borderWidth, minDim)
			l.dim.height = math.Max(textHeight(label, fontSize)+2*borderWidth, minDim)
		}
		return
	}

	// A bordered parent grows to contain each child's *margin* box (margins act
	// like padding inside the border). An invisible parent (box:none, group, or
	// the root) has no wall for perimeter margins to sit against, so it bounds
	// the children's *border* boxes plus the inter-sibling gaps only.
	bordered := isBordered(l.node)
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
	if isContainer && container.Size != nil {
		if direction == ast.Horizontal {
			l.dim.height *= *container.Size
		} else {
			l.dim.width *= *container.Size
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
	direction := directionOf(l.node)
	bordered := isBordered(l.node)

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

func collectDimensions(l *layoutNode, m map[string]dimensions) {
	if c, ok := l.node.(*ast.ContainerNode); ok {
		m[c.ID] = l.dim
	}
	for _, c := range l.children {
		collectDimensions(c, m)
	}
}

func canvasSize(roots []*layoutNode) (w, h float64) {
	for _, l := range roots {
		w = math.Max(w, l.dim.x+l.dim.width)
		h = math.Max(h, l.dim.y+l.dim.height)
	}
	return w, h
}

func collectAnchors(nodes []ast.Node, dims map[string]dimensions, parentPath string) []anchorPosition {
	var out []anchorPosition
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ContainerNode:
			full := n.ID
			if parentPath != "" {
				full = parentPath + "." + n.ID
			}
			if d, ok := dims[n.ID]; ok {
				out = append(out, resolveNodeAnchors(n, d, full)...)
			}
			out = append(out, collectAnchors(n.Children, dims, full)...)
		case *ast.GroupNode:
			out = append(out, collectAnchors(n.Children, dims, parentPath)...)
		}
	}
	return out
}

func resolveNodeAnchors(n *ast.ContainerNode, d dimensions, full string) []anchorPosition {
	anchors := []anchorPosition{{
		x: d.x + d.width*0.5, y: d.y + d.height*0.5, nodeID: full, anchorID: "center",
	}}
	for _, id := range sortedKeys(n.Anchors) {
		c := n.Anchors[id]
		rx, ry := c[0], c[1]
		if rx < 0 || rx > 1 || ry < 0 || ry > 1 {
			continue // out-of-range anchors are reported by the validator
		}
		anchors = append(anchors, anchorPosition{
			x: d.x + d.width*rx, y: d.y + d.height*ry, nodeID: full, anchorID: id,
		})
	}
	return anchors
}

func childrenOf(node ast.Node) []ast.Node {
	switch n := node.(type) {
	case *ast.ContainerNode:
		return n.Children
	case *ast.GroupNode:
		return n.Children
	}
	return nil
}

// isBordered reports whether a node draws a border and so insets its children
// like padding. Groups, box:none containers, and the document root are
// invisible: they collapse their children's perimeter margins and do not
// stretch children to fill the cross axis.
func isBordered(node ast.Node) bool {
	c, ok := node.(*ast.ContainerNode)
	return ok && c.Box != ast.BoxNone
}

// marginOf returns a node's effective uniform margin. Layout-only groups carry
// no margin of their own; a container uses its explicit margin or the default.
func marginOf(node ast.Node) float64 {
	c, ok := node.(*ast.ContainerNode)
	if !ok {
		return 0
	}
	if c.Margin != nil {
		return *c.Margin
	}
	return defaultMargin
}

func directionOf(node ast.Node) ast.Direction {
	var d ast.Direction
	switch n := node.(type) {
	case *ast.ContainerNode:
		d = n.Direction
	case *ast.GroupNode:
		d = n.Direction
	}
	if d == ast.DirectionUnset {
		return ast.Vertical
	}
	return d
}

func toNodes(cs []*ast.ContainerNode) []ast.Node {
	out := make([]ast.Node, len(cs))
	for i, c := range cs {
		out[i] = c
	}
	return out
}

func sortedKeys(m map[string][2]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
