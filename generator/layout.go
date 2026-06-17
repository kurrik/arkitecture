package generator

import (
	"math"
	"sort"

	"github.com/kurrik/arkitecture/ast"
)

// borderWidth is the 1px border every container node draws.
const borderWidth = 1.0

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
	dim      dimensions
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

	currentX := 0.0
	for _, l := range roots {
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

	if direction == ast.Horizontal {
		sum, maxH := 0.0, 0.0
		for _, c := range l.children {
			sum += c.dim.width
			maxH = math.Max(maxH, c.dim.height)
		}
		l.dim.width, l.dim.height = sum, maxH
		if isContainer {
			for _, c := range l.children {
				c.dim.height = l.dim.height
			}
		}
	} else {
		sum, maxW := 0.0, 0.0
		for _, c := range l.children {
			sum += c.dim.height
			maxW = math.Max(maxW, c.dim.width)
		}
		l.dim.height, l.dim.width = sum, maxW
		if isContainer {
			for _, c := range l.children {
				c.dim.width = l.dim.width
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

func positionNodes(l *layoutNode, x, y float64) {
	l.dim.x, l.dim.y = x, y
	direction := directionOf(l.node)
	cx, cy := x, y
	for _, c := range l.children {
		positionNodes(c, cx, cy)
		if direction == ast.Horizontal {
			cx += c.dim.width
		} else {
			cy += c.dim.height
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
