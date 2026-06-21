package generator

import (
	"math"
	"strings"

	"github.com/kurrik/arkitecture/ast"
)

// route.go turns each arrow into the sequence of points its line passes through.
// In the default straight mode that is just the two resolved endpoints (the M2
// auto-cardinal line). In orthogonal mode (`route: orthogonal` at the @layout
// root) the endpoints are joined by an axis-aligned path: first the direct elbow
// or Z respecting each end's exit side, and when that would cut through a box, a
// detour routed *around* the obstacles on the channel graph (channel.go);
// failing both it falls back to the straight line, so orthogonal mode never
// renders a worse result than straight mode. A positioned anchor pinned to a box
// edge leaves/enters perpendicular to *that* edge; an interior anchor is met at
// the facing border and a tail runs in, so the line enters the node.
//
// This realises the sized-channel router (docs/design.md) up to channel
// widening: the polyline emission, the per-arrow obstacle set, the cardinal
// endpoint choice, and routing around obstacles are all here; lane counts pushing
// boxes apart (widening) is the remaining slice.

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
// where the orthogonal path begins outside the tip, and side is the exit normal.
// Three cases set them:
//   - bare reference / anchor on the facing side: edge is the breakout box's
//     border on the facing side (equal to tip for a bare reference);
//   - interior anchor: edge is that border too, and the tip sits inside, so a
//     tail segment runs in — the line enters the node;
//   - anchor pinned to one of its box's own edges (not the facing side): edge is
//     a stub one inset outside that edge, and side is the edge's normal, so the
//     line departs perpendicular to the edge into the box's channel. edgeExit is
//     set, so arrowPath walls off that box and the route detours around it rather
//     than turning back through it.
type endpoint struct {
	tip      point
	edge     point
	side     side
	edgeExit bool
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

	inset := layout.defMargin / 2
	if !explicit {
		// A bare reference leaves/enters perpendicular to its facing edge, met one
		// inset past the breakout-box border — out in the channel — so when the
		// route must detour it runs up the gap *centre* rather than hugging the box
		// edge. In the clear case this point is collinear with the elbow's exit and
		// is absorbed by simplify, so a clear arrow is byte-identical.
		tip, _ := cardinalEndpoint(selfBox, centerOf(otherBox))
		return endpoint{tip: tip, edge: offsetPoint(edgePointOnSide(exitBox, s, tip), s, inset), side: s}, true
	}

	ap := findAnchor(layout.anchorPositions, path, anchor)
	if ap == nil {
		return endpoint{}, false
	}
	tip := point{ap.x, ap.y}
	es, isEdge := anchorEdge(selfBox, tip)
	switch {
	case isEdge && es != s:
		// An anchor pinned to one of its box's edges that is **not** the facing side
		// leaves **perpendicular to that edge**, into the leaf's adjacent channel —
		// honouring where the author placed it rather than running the line along the
		// edge toward the facing side. The exit point is a stub one inset outside the
		// leaf edge; the channel router follows corridors (and breaks out of
		// containers) from there. edgeExit walls off the leaf so the route detours
		// around it instead of turning straight back through it.
		return endpoint{tip: tip, edge: offsetPoint(tip, es, inset), side: es, edgeExit: true}, true
	case isEdge:
		// An anchor already on the facing side is met one inset past the breakout-box
		// border, out in the channel, so the router approaches it **along the facing
		// normal** instead of sliding up the border when the rest of the route arrives
		// askew. (At the breakout-box border this still crosses in the channel between
		// containers, so the break-out case is unchanged.)
		return endpoint{tip: tip, edge: offsetPoint(edgePointOnSide(exitBox, s, tip), s, inset), side: s}, true
	default:
		// A bare reference or an interior/corner anchor meets the breakout-box border
		// on the facing side, with the tail entering the node for an interior anchor.
		return endpoint{tip: tip, edge: edgePointOnSide(exitBox, s, tip), side: s}, true
	}
}

// anchorEdge reports the single box edge an anchor sits on (and true), or false
// when the anchor is interior or on a corner (two edges at once).
func anchorEdge(box dimensions, tip point) (side, bool) {
	on := func(a, b float64) bool { return math.Abs(a-b) < epsilon }
	count := 0
	var s side
	if on(tip.x, box.x) {
		count, s = count+1, west
	}
	if on(tip.x, box.x+box.width) {
		count, s = count+1, east
	}
	if on(tip.y, box.y) {
		count, s = count+1, north
	}
	if on(tip.y, box.y+box.height) {
		count, s = count+1, south
	}
	if count == 1 {
		return s, true
	}
	return 0, false
}

// offsetPoint moves p by d along side s's outward normal.
func offsetPoint(p point, s side, d float64) point {
	switch s {
	case east:
		return point{p.x + d, p.y}
	case west:
		return point{p.x - d, p.y}
	case south:
		return point{p.x, p.y + d}
	default: // north
		return point{p.x, p.y - d}
	}
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
// points. In orthogonal mode it is tip → border → path across the gap → border →
// tip: each end leaves (or enters) its box on the side facing the other. The
// gap-crossing path is, in order of preference: the clear-case elbow (one corner
// or a Z); failing that, a channel-graph detour routed *around* the obstacles;
// and failing that, the straight tip-to-tip line — so orthogonal mode never
// renders worse than straight mode. Collinear tails merge into the adjacent run,
// and a tip already on its border (a bare reference or an edge anchor) collapses
// its tail to nothing, so a clear bare-to-bare arrow is exactly the M2 elbow.
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
	obstacles := arrowObstacles(nodePathOf(a.Source), nodePathOf(a.Target), layout.nodeBoxes)
	// An endpoint that exits perpendicular to a non-facing edge walls off its own
	// box, so the route detours around it instead of cutting back through it.
	if src.edgeExit {
		obstacles = append(obstacles, layout.nodeBoxes[nodePathOf(a.Source)])
	}
	if tgt.edgeExit {
		obstacles = append(obstacles, layout.nodeBoxes[nodePathOf(a.Target)])
	}

	// Fast path: the direct elbow between the breakout-box border points. When it
	// is clear this is byte-identical to the pre-router output.
	if pts := assembleRoute(src, elbow(src.edge, src.side, tgt.edge, tgt.side), tgt); pathClear(pts, obstacles) {
		return pts, true
	}
	// The elbow cuts through a box: route around the obstacles on the channel grid.
	if mid := routeAround(src.edge, tgt.edge, obstacles, layout.defMargin/2); mid != nil {
		if pts := assembleRoute(src, mid, tgt); pathClear(pts, obstacles) {
			return pts, true
		}
	}
	// No clear orthogonal path exists: fall back to the straight line.
	return []point{src.tip, tgt.tip}, true
}

// assembleRoute frames a gap-crossing path (from src.edge to tgt.edge) with each
// end's tip→edge tail and simplifies: tip → edge → … → edge → tip, with collinear
// points and zero-length tails (a bare reference, or an edge anchor) collapsed.
func assembleRoute(src endpoint, mid []point, tgt endpoint) []point {
	pts := append([]point{src.tip}, mid...)
	return simplify(append(pts, tgt.tip))
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
