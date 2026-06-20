package generator

import (
	"math"
	"strings"

	"github.com/kurrik/arkitecture/ast"
)

// route.go turns each arrow into the sequence of points its line passes through.
// In the default straight mode that is just the two resolved endpoints (the M2
// auto-cardinal line). In orthogonal mode (`route: orthogonal` at the @layout
// root) the endpoints are joined by an axis-aligned path — an elbow or Z that
// respects each end's exit side — provided that path is clear of the boxes
// between the endpoints; otherwise it falls back to the straight line so
// orthogonal mode never renders a worse result than straight mode. A positioned
// anchor is handled the same way: the path meets the box border on the facing
// side and a tail segment runs in to the anchor, so an interior anchor is reached
// by entering the node.
//
// This is an early slice of the sized-channel router (docs/design.md): it
// establishes the polyline emission and the per-arrow obstacle set, reusing M2's
// cardinal endpoint choice. Routing *around* obstacles (the channel graph + A*)
// and channel widening build on these primitives.

// routeMode reports the document's resolved routing mode (default straight).
func routeMode(doc *ast.Document) ast.RouteMode {
	if doc != nil && doc.Route != nil && *doc.Route != ast.RouteUnset {
		return *doc.Route
	}
	return ast.RouteStraight
}

// side is a box's cardinal edge (the M2 N/E/S/W attachment).
type side int

const (
	north side = iota
	east
	south
	west
)

// horizontal reports whether the side's outward normal runs along the x axis
// (east/west), so a line exiting it starts horizontal.
func (s side) horizontal() bool { return s == east || s == west }

// endpoint is one resolved arrow end. tip is the actual attachment point — a
// positioned anchor, or the cardinal edge midpoint for a bare reference. edge is
// where the orthogonal path meets the box border on its exit side: equal to tip
// for a bare reference (or an anchor already on that side), otherwise the border
// point aligned with the anchor — the tip then sits inside the box and a final
// tail segment runs between them, so the line enters the node to reach an
// interior anchor. side is the exit normal.
type endpoint struct {
	tip  point
	edge point
	side side
}

// cardinalEndpoint returns the midpoint of box's edge facing aim and that edge's
// side, chosen by the dominant axis of the centre-to-aim vector. Exact diagonals
// (|dx| == |dy|) favour the horizontal (E/W) side. It is the shared core of M2's
// straight routing and the orthogonal elbow.
func cardinalEndpoint(box dimensions, aim point) (point, side) {
	c := centerOf(box)
	dx, dy := aim.x-c.x, aim.y-c.y
	if math.Abs(dx) >= math.Abs(dy) {
		if dx >= 0 {
			return point{box.x + box.width, c.y}, east
		}
		return point{box.x, c.y}, west
	}
	if dy >= 0 {
		return point{c.x, box.y + box.height}, south
	}
	return point{c.x, box.y}, north
}

// resolveOrthoEndpoint resolves self's attachment. The tip is the actual point —
// an explicit anchor, or the leaf box's cardinal edge facing the other leaf
// (unchanged M2 selection, so straight mode is identical). The orthogonal path,
// though, leaves through the **breakout box**: the outermost container holding
// self but not the other node. The exit side and border point are taken on that
// container (facing the other's breakout box), so the gap-crossing runs in the
// channel between containers rather than along a container's border; a tail then
// connects the tip out to that border. For two nodes in the same parent the
// breakout box is the leaf itself, so nothing changes. ok is false when a
// referenced node or anchor is missing (the validator reports those).
func resolveOrthoEndpoint(self, other string, layout layoutResult) (endpoint, bool) {
	path, anchor, explicit := splitRef(self)
	selfBox, ok := layout.nodeBoxes[path]
	if !ok {
		return endpoint{}, false
	}
	otherPath := nodePathOf(other)
	otherBox, ok := layout.nodeBoxes[otherPath]
	if !ok {
		return endpoint{}, false
	}

	exitBox := breakoutBox(path, otherPath, layout.nodeBoxes)
	entryBox := breakoutBox(otherPath, path, layout.nodeBoxes)
	_, s := cardinalEndpoint(exitBox, centerOf(entryBox))

	var tip point
	if explicit {
		ap := findAnchor(layout.anchorPositions, path, anchor)
		if ap == nil {
			return endpoint{}, false
		}
		tip = point{ap.x, ap.y}
	} else {
		tip, _ = cardinalEndpoint(selfBox, centerOf(otherBox))
	}
	return endpoint{tip: tip, edge: edgePointOnSide(exitBox, s, tip), side: s}, true
}

// breakoutBox returns the outermost box that contains self but not other — the
// container an arrow must leave to reach the other node. It is the self-side child
// of the two paths' lowest common container: for siblings under one parent it is
// self's own leaf box, and for nodes in different branches it is self's ancestor
// just inside the common container. Crossing the gap between these boxes (instead
// of the leaf boxes) keeps the orthogonal run in the channel between containers
// rather than along a container border.
func breakoutBox(self, other string, boxes map[string]dimensions) dimensions {
	ss := strings.Split(self, ".")
	os := strings.Split(other, ".")
	k := 0
	for k < len(ss) && k < len(os) && ss[k] == os[k] {
		k++
	}
	n := k + 1
	if n > len(ss) {
		n = len(ss)
	}
	if b, ok := boxes[strings.Join(ss[:n], ".")]; ok {
		return b
	}
	return boxes[self]
}

// edgePointOnSide returns the point on box's given side that shares tip's
// coordinate on the axis parallel to that side — where an orthogonal path crosses
// the border to reach (or leave) the anchor at tip.
func edgePointOnSide(box dimensions, s side, tip point) point {
	switch s {
	case east:
		return point{box.x + box.width, tip.y}
	case west:
		return point{box.x, tip.y}
	case south:
		return point{tip.x, box.y + box.height}
	default: // north
		return point{tip.x, box.y}
	}
}

// arrowPath returns the ordered points an arrow's line passes through, or ok =
// false when an endpoint is unresolved. In straight mode it is the two attachment
// points. In orthogonal mode it is tip → border → elbow across the gap → border →
// tip: each end leaves (or enters) its box on the side facing the other, joined by
// an axis-aligned elbow. Collinear tails merge into the adjacent run, and a tip
// already on its border (a bare reference or an edge anchor) collapses its tail to
// nothing, so a bare-to-bare arrow is exactly the M2 elbow. The orthogonal path is
// used only when clear of the arrow's obstacles; otherwise it falls back to the
// straight tip-to-tip line.
func arrowPath(a ast.Arrow, layout layoutResult, mode ast.RouteMode) ([]point, bool) {
	src, ok := resolveOrthoEndpoint(a.Source, a.Target, layout)
	if !ok {
		return nil, false
	}
	tgt, ok := resolveOrthoEndpoint(a.Target, a.Source, layout)
	if !ok {
		return nil, false
	}
	if mode != ast.RouteOrthogonal {
		return []point{src.tip, tgt.tip}, true
	}
	pts := append([]point{src.tip}, elbow(src.edge, src.side, tgt.edge, tgt.side)...)
	pts = simplify(append(pts, tgt.tip))
	obstacles := arrowObstacles(nodePathOf(a.Source), nodePathOf(a.Target), layout.nodeBoxes)
	if pathClear(pts, obstacles) {
		return pts, true
	}
	return []point{src.tip, tgt.tip}, true
}

// elbow joins two cardinal endpoints with an axis-aligned path that leaves a
// along side sa and arrives at b along side sb. Perpendicular sides give a single
// corner; parallel sides give a Z through the midpoint of the gap. Collinear
// runs collapse to a straight segment. The result is deterministic.
func elbow(a point, sa side, b point, sb side) []point {
	var pts []point
	switch {
	case sa.horizontal() != sb.horizontal():
		// One end exits horizontally, the other vertically: the corner takes its x
		// from the vertical-exit end and its y from the horizontal-exit end, so both
		// exit directions are respected.
		corner := point{b.x, a.y}
		if !sa.horizontal() {
			corner = point{a.x, b.y}
		}
		pts = []point{a, corner, b}
	case sa.horizontal():
		// Both exit horizontally: a Z with the cross segment at the mid-x.
		mx := (a.x + b.x) / 2
		pts = []point{a, {mx, a.y}, {mx, b.y}, b}
	default:
		// Both exit vertically: a Z with the cross segment at the mid-y.
		my := (a.y + b.y) / 2
		pts = []point{a, {a.x, my}, {b.x, my}, b}
	}
	return simplify(pts)
}

// simplify drops points that coincide with their predecessor or lie on the
// straight run between their neighbours, so an unnecessary elbow collapses to a
// straight line (and a degenerate Z to two points).
func simplify(pts []point) []point {
	out := pts[:0:0]
	for _, p := range pts {
		if n := len(out); n > 0 && samePoint(out[n-1], p) {
			continue
		}
		out = append(out, p)
	}
	// Remove collinear middles.
	i := 1
	for i < len(out)-1 {
		if collinear(out[i-1], out[i], out[i+1]) {
			out = append(out[:i], out[i+1:]...)
			continue
		}
		i++
	}
	return out
}

func samePoint(a, b point) bool {
	return math.Abs(a.x-b.x) < epsilon && math.Abs(a.y-b.y) < epsilon
}

// collinear reports whether b lies on the axis-aligned run from a to c (the only
// kind of collinearity an orthogonal path produces).
func collinear(a, b, c point) bool {
	if math.Abs(a.x-b.x) < epsilon && math.Abs(b.x-c.x) < epsilon {
		return true // shared vertical
	}
	return math.Abs(a.y-b.y) < epsilon && math.Abs(b.y-c.y) < epsilon // shared horizontal
}

// arrowObstacles is the set of box rectangles an arrow must avoid: every box
// except those on the source's or target's own root-to-node lineage (their
// ancestors, descendants, and the endpoints themselves). An arrow is never
// blocked by a box it legitimately starts inside, ends inside, or must leave —
// the arrow-relative obstacle rule from the design.
func arrowObstacles(srcPath, tgtPath string, boxes map[string]dimensions) []dimensions {
	var out []dimensions
	for p, d := range boxes {
		if related(p, srcPath) || related(p, tgtPath) {
			continue
		}
		out = append(out, d)
	}
	return out
}

// related reports whether dotted paths a and b are on the same lineage: equal, or
// one a prefix-ancestor of the other.
func related(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+".") || strings.HasPrefix(b, a+".")
}

// pathClear reports whether none of the polyline's axis-aligned segments passes
// through the interior of any obstacle rectangle.
func pathClear(pts []point, obstacles []dimensions) bool {
	for i := 0; i+1 < len(pts); i++ {
		for _, r := range obstacles {
			if segIntersectsRect(pts[i], pts[i+1], r) {
				return false
			}
		}
	}
	return true
}

// segIntersectsRect reports whether the axis-aligned segment p1-p2 crosses the
// open interior of rect r. Running exactly along an edge (a shared border) does
// not count, so a route may graze a box without being blocked by it.
func segIntersectsRect(p1, p2 point, r dimensions) bool {
	rx1, ry1, rx2, ry2 := r.x, r.y, r.x+r.width, r.y+r.height
	if math.Abs(p1.y-p2.y) < epsilon { // horizontal
		y := p1.y
		if y <= ry1+epsilon || y >= ry2-epsilon {
			return false
		}
		lo, hi := math.Min(p1.x, p2.x), math.Max(p1.x, p2.x)
		return math.Max(lo, rx1) < math.Min(hi, rx2)-epsilon
	}
	// vertical
	x := p1.x
	if x <= rx1+epsilon || x >= rx2-epsilon {
		return false
	}
	lo, hi := math.Min(p1.y, p2.y), math.Max(p1.y, p2.y)
	return math.Max(lo, ry1) < math.Min(hi, ry2)-epsilon
}

const epsilon = 1e-9
