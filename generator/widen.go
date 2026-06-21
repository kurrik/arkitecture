package generator

import (
	"math"

	"github.com/kurrik/arkitecture/ast"
)

// widen.go is channel widening (docs/design.md "channel vs margin"): an arrow
// routing *along* a channel reserves a lane there, so the channel widens and the
// boxes spread rather than the line sitting inside a node's margin. It runs as a
// second layout pass — route once on the un-widened layout to learn each
// channel's lane demand (channelDemand), widen those gaps, then lay out again.
//
// laneSpacing is half the channel's base margin (the author's "half a margin per
// lane"): a gap carrying N parallel arrows widens by N * margin/2.

// channelDemand routes every arrow on the (un-widened) layout and tallies, per
// container gap, how many arrows run longitudinally along it, turning that into
// the extra width each gap reserves. Returns nil when no channel needs widening.
func channelDemand(arrows []ast.Arrow, layout layoutResult, mode ast.RouteMode) *widenDemand {
	lanes := map[string]map[int]int{}    // container path -> gap index -> arrows along it
	base := map[string]map[int]float64{} // container path -> gap index -> base margin
	for _, a := range arrows {
		pts, ok := arrowPath(a, layout, mode)
		if !ok {
			continue
		}
		for i := 0; i+1 < len(pts); i++ {
			if samePoint(pts[i], pts[i+1]) {
				continue
			}
			path, gi, b, ok := findChannel(layout.roots, pts[i], pts[i+1])
			if !ok {
				continue
			}
			if lanes[path] == nil {
				lanes[path] = map[int]int{}
				base[path] = map[int]float64{}
			}
			lanes[path][gi]++
			base[path][gi] = b
		}
	}
	if len(lanes) == 0 {
		return nil
	}
	d := &widenDemand{gaps: map[string][]float64{}}
	for path, m := range lanes {
		maxGap := 0
		for gi := range m {
			if gi > maxGap {
				maxGap = gi
			}
		}
		extra := make([]float64, maxGap+1)
		for gi, n := range m {
			extra[gi] = float64(n) * base[path][gi] / 2
		}
		if path == "" {
			d.root = extra
		} else {
			d.gaps[path] = extra
		}
	}
	return d
}

// findChannel attributes a longitudinal segment to the container gap it runs
// along. It descends by the segment's midpoint into the deepest box that contains
// it; the gap is in the container holding that midpoint in free space (not inside
// a child). The segment must run perpendicular to that container's main axis (so
// it lies along a between-children gap or the leading/trailing perimeter, not a
// cross-axis rail — rails are a later slice). Returns the container path, the gap
// index (0 = leading … len(children) = trailing), the gap's base margin, and ok.
func findChannel(roots []*layoutNode, p0, p1 point) (string, int, float64, bool) {
	horizontal := math.Abs(p0.y-p1.y) < epsilon // the segment runs along X
	mid := point{(p0.x + p1.x) / 2, (p0.y + p1.y) / 2}

	children := roots
	mainHorizontal := true // the document root lays top-level nodes out left-to-right
	path := ""
	for {
		inside := -1
		for i, c := range children {
			if c.isGroup {
				continue // group regions are deferred
			}
			d := c.dim
			if mid.x > d.x+epsilon && mid.x < d.x+d.width-epsilon &&
				mid.y > d.y+epsilon && mid.y < d.y+d.height-epsilon {
				inside = i
				break
			}
		}
		if inside >= 0 {
			c := children[inside]
			if len(c.children) == 0 {
				return "", 0, 0, false // midpoint inside a leaf box — not a channel
			}
			children = c.children
			mainHorizontal = directionOf(c.decls) == ast.Horizontal
			path = c.path
			continue
		}
		// The midpoint is in a gap of this container. Only a segment running
		// perpendicular to the container's main axis lies *along* a main-axis gap.
		if horizontal == mainHorizontal {
			return "", 0, 0, false // along the main axis: a cross-axis rail (deferred)
		}
		gi, b, ok := gapIndexAt(children, mainHorizontal, mid)
		return path, gi, b, ok
	}
}

// snapToLanes centres each interior longitudinal run of a routed polyline in the
// (widened) channel it belongs to, so the line sits in its lane rather than at a
// fixed inset from one box. The first and last segments are left alone — they are
// the tip's exit/entry tails, anchored to the box. Snapping a run only moves its
// perpendicular coordinate, so the connecting segments stay axis-aligned.
func snapToLanes(pts []point, layout layoutResult) []point {
	if len(pts) < 4 {
		return pts // no interior segment to snap
	}
	out := append([]point(nil), pts...)
	for i := 1; i < len(out)-2; i++ {
		p0, p1 := out[i], out[i+1]
		if samePoint(p0, p1) {
			continue
		}
		path, gi, _, ok := findChannel(layout.roots, p0, p1)
		if !ok {
			continue
		}
		center, ok := gapCenterAt(layout.roots, path, gi)
		if !ok {
			continue
		}
		if math.Abs(p0.y-p1.y) < epsilon { // horizontal run -> snap its y
			out[i].y, out[i+1].y = center, center
		} else { // vertical run -> snap its x
			out[i].x, out[i+1].x = center, center
		}
	}
	return out
}

// gapCenterAt returns the centre of gap gi in the container at path, on that
// container's main axis — the lane coordinate a run along the gap snaps to.
func gapCenterAt(roots []*layoutNode, path string, gi int) (float64, bool) {
	var children []*layoutNode
	var mainHorizontal bool
	var loEdge, hiEdge float64 // container content edges on the main axis (for perimeter gaps)
	if path == "" {
		children = roots
		mainHorizontal = true
		if len(children) == 0 {
			return 0, false
		}
		loEdge = children[0].dim.x
		last := children[len(children)-1].dim
		hiEdge = last.x + last.width
	} else {
		c := nodeByPath(roots, path)
		if c == nil || len(c.children) == 0 {
			return 0, false
		}
		children = c.children
		mainHorizontal = directionOf(c.decls) == ast.Horizontal
		b := 0.0
		if nodeBordered(c) {
			b = borderWidth
		}
		if mainHorizontal {
			loEdge, hiEdge = c.dim.x+b, c.dim.x+c.dim.width-b
		} else {
			loEdge, hiEdge = c.dim.y+b+c.labelBand, c.dim.y+c.dim.height-b
		}
	}
	mn := func(l *layoutNode) float64 {
		if mainHorizontal {
			return l.dim.x
		}
		return l.dim.y
	}
	mf := func(l *layoutNode) float64 {
		if mainHorizontal {
			return l.dim.x + l.dim.width
		}
		return l.dim.y + l.dim.height
	}
	n := len(children)
	var lo, hi float64
	switch {
	case gi <= 0:
		lo, hi = loEdge, mn(children[0])
	case gi >= n:
		lo, hi = mf(children[n-1]), hiEdge
	default:
		lo, hi = mf(children[gi-1]), mn(children[gi])
	}
	return (lo + hi) / 2, true
}

// nodeByPath returns the layout node with the given dotted path, or nil.
func nodeByPath(roots []*layoutNode, path string) *layoutNode {
	for _, r := range roots {
		if !r.isGroup && r.path == path {
			return r
		}
		if n := nodeByPath(r.children, path); n != nil {
			return n
		}
	}
	return nil
}

// gapIndexAt returns which main-axis gap the point sits in among children laid
// out along the main axis, and that gap's base margin. ok is false when the point
// is beside a child on the cross axis (a rail, not a main-axis gap).
func gapIndexAt(children []*layoutNode, mainHorizontal bool, p point) (int, float64, bool) {
	coord := p.y
	if mainHorizontal {
		coord = p.x
	}
	for i, c := range children {
		near, far := c.dim.y, c.dim.y+c.dim.height
		if mainHorizontal {
			near, far = c.dim.x, c.dim.x+c.dim.width
		}
		if coord < near-epsilon {
			b := c.margin
			if i > 0 {
				b = math.Max(children[i-1].margin, c.margin)
			}
			return i, b, true // leading or between-children gap before child i
		}
		if coord <= far+epsilon {
			return 0, 0, false // within child i's main span but beside it — a rail
		}
	}
	n := len(children)
	return n, children[n-1].margin, true // trailing perimeter
}
