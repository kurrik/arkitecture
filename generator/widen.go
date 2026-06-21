package generator

import (
	"math"

	"github.com/kurrik/arkitecture/ast"
)

// widen.go is channel widening (docs/design.md "channel vs margin"): an arrow
// routing *along* a channel reserves a lane there, so the channel widens and the
// boxes spread rather than the line sitting inside a node's margin. It runs as a
// second layout pass — route once on the un-widened layout to learn each
// channel's lane demand (channelDemand), widen those channels, then lay out again.
//
// A channel is either a **main-axis gap** (between two children, or the
// leading/trailing perimeter — a run crosses the container's main axis here) or a
// **cross-axis rail** (one of the two perimeter sides parallel to the main axis —
// a run travels along the container here). laneSpacing is half the channel's base
// margin (the author's "half a margin per lane"): a channel carrying N parallel
// arrows widens by N * margin/2.

// channelRef identifies a widening channel so demand and snapping treat gaps and
// rails uniformly.
type channelRef struct {
	path  string  // container dotted path ("" = document root)
	rail  bool    // true = cross-axis perimeter rail; false = main-axis gap
	index int     // gap index (0..len children), or rail side (0 = low, 1 = high)
	base  float64 // base margin to widen from
}

// channelDemand routes every arrow on the (un-widened) layout and tallies, per
// channel, how many arrows run longitudinally along it, turning that into the
// extra width each channel reserves. Returns nil when nothing needs widening.
func channelDemand(arrows []ast.Arrow, layout layoutResult, mode ast.RouteMode) *widenDemand {
	type key struct {
		path  string
		rail  bool
		index int
	}
	lanes := map[key]int{}
	base := map[key]float64{}
	for _, a := range arrows {
		pts, ok := arrowPath(a, layout, mode)
		if !ok {
			continue
		}
		for i := 0; i+1 < len(pts); i++ {
			if samePoint(pts[i], pts[i+1]) {
				continue
			}
			ref, ok := findChannel(layout.roots, pts[i], pts[i+1])
			if !ok {
				continue
			}
			k := key{ref.path, ref.rail, ref.index}
			lanes[k]++
			base[k] = ref.base
		}
	}
	if len(lanes) == 0 {
		return nil
	}

	d := &widenDemand{gaps: map[string][]float64{}, rails: map[string][2]float64{}}
	for k, n := range lanes {
		extra := float64(n) * base[k] / 2
		switch {
		case k.rail:
			r := d.rails[k.path]
			r[k.index] = extra
			d.rails[k.path] = r
		case k.path == "":
			d.root = setAt(d.root, k.index, extra)
		default:
			d.gaps[k.path] = setAt(d.gaps[k.path], k.index, extra)
		}
	}
	return d
}

// setAt sets s[i] = v, growing s with zeros as needed.
func setAt(s []float64, i int, v float64) []float64 {
	for len(s) <= i {
		s = append(s, 0)
	}
	s[i] = v
	return s
}

// findChannel attributes a longitudinal segment to the channel it runs along. It
// descends by the segment's midpoint into the deepest box that contains it; the
// channel is in the container holding that midpoint in free space (not inside a
// child). A segment perpendicular to that container's main axis lies along a
// main-axis gap; one parallel to the main axis lies along a cross-axis perimeter
// rail. The document root has no perimeter, so a run along it is not a channel.
func findChannel(roots []*layoutNode, p0, p1 point) (channelRef, bool) {
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
				return channelRef{}, false // midpoint inside a leaf box — not a channel
			}
			children = c.children
			mainHorizontal = directionOf(c.decls) == ast.Horizontal
			path = c.path
			continue
		}
		// The midpoint is in free space of this container.
		if horizontal != mainHorizontal {
			// Perpendicular to the main axis: a between-children gap or perimeter.
			gi, b, ok := gapIndexAt(children, mainHorizontal, mid)
			return channelRef{path: path, index: gi, base: b}, ok
		}
		// Parallel to the main axis: a cross-axis perimeter rail. The root has no
		// perimeter to widen.
		if path == "" {
			return channelRef{}, false
		}
		side, b, ok := railSideAt(children, mainHorizontal, mid)
		return channelRef{path: path, rail: true, index: side, base: b}, ok
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
		ref, ok := findChannel(layout.roots, p0, p1)
		if !ok {
			continue
		}
		center, ok := channelCenterAt(layout.roots, ref)
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

// channelCenterAt returns the centre coordinate of a channel — the lane a run
// along it snaps to. For a gap it is on the container's main axis; for a rail, on
// the cross axis.
func channelCenterAt(roots []*layoutNode, ref channelRef) (float64, bool) {
	if ref.rail {
		return railCenterAt(roots, ref.path, ref.index)
	}
	return gapCenterAt(roots, ref.path, ref.index)
}

// gapCenterAt returns the centre of gap gi in the container at path, on that
// container's main axis.
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

// railCenterAt returns the centre of a cross-axis perimeter rail (side 0 = low,
// 1 = high) in the container at path: the midpoint between the children's cross
// edge and the container's content edge on that side.
func railCenterAt(roots []*layoutNode, path string, side int) (float64, bool) {
	c := nodeByPath(roots, path)
	if c == nil || len(c.children) == 0 {
		return 0, false
	}
	mainHorizontal := directionOf(c.decls) == ast.Horizontal
	b := 0.0
	if nodeBordered(c) {
		b = borderWidth
	}
	near, far := childrenCrossBand(c.children, mainHorizontal)
	var contentNear, contentFar float64
	if mainHorizontal {
		// Cross axis is Y; the label band sits on it (top by default, bottom if set).
		contentNear, contentFar = c.dim.y+b, c.dim.y+c.dim.height-b
		if labelPositionOf(c.decls) == ast.LabelBottom {
			contentFar -= c.labelBand
		} else {
			contentNear += c.labelBand
		}
	} else {
		// Cross axis is X; no band here.
		contentNear, contentFar = c.dim.x+b, c.dim.x+c.dim.width-b
	}
	if side == 0 {
		return (contentNear + near) / 2, true
	}
	return (far + contentFar) / 2, true
}

// childrenCrossBand returns the span the children collectively occupy on the
// container's cross axis (min near edge, max far edge).
func childrenCrossBand(children []*layoutNode, mainHorizontal bool) (near, far float64) {
	near, far = math.Inf(1), math.Inf(-1)
	for _, c := range children {
		var n, f float64
		if mainHorizontal {
			n, f = c.dim.y, c.dim.y+c.dim.height
		} else {
			n, f = c.dim.x, c.dim.x+c.dim.width
		}
		near, far = math.Min(near, n), math.Max(far, f)
	}
	return near, far
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
// is beside a child on the cross axis (a rail is handled separately).
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

// railSideAt returns which cross-axis rail (0 = low, 1 = high) the point sits in,
// and the rail's base margin. ok is false when the point is within the children's
// cross band (not in a perimeter rail).
func railSideAt(children []*layoutNode, mainHorizontal bool, p point) (int, float64, bool) {
	near, far := childrenCrossBand(children, mainHorizontal)
	cross := p.y
	if !mainHorizontal {
		cross = p.x
	}
	base := 0.0
	for _, c := range children {
		base = math.Max(base, c.margin)
	}
	switch {
	case cross < near-epsilon:
		return 0, base, true // low-side rail (top / left perimeter)
	case cross > far+epsilon:
		return 1, base, true // high-side rail (bottom / right perimeter)
	}
	return 0, 0, false
}
