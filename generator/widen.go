package generator

import (
	"math"
	"sort"

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
// a run travels along the container here). Each lane clears the boxes by a full
// margin (the channel's base) so the line sits in its own channel — a wall, not
// inside the node's margin — with lanes half a margin apart; a channel carrying N
// arrows therefore reserves (N+1)·base/2 of extra width.

// channelRef identifies a widening channel. A channel is a track boundary of a
// container: a *vertical* arrow run travels along a column boundary (and needs
// horizontal clearance there), a *horizontal* run along a row boundary. This one
// scheme covers a 1-D stack (one axis a single track) and a 2-D grid uniformly.
type channelRef struct {
	path     string  // container dotted path ("" = document root)
	vertical bool    // true = column boundary (a vertical run); false = row boundary
	index    int     // boundary index along that axis (0 = low perimeter … count = high)
	base     float64 // base margin to widen from
}

// channelKey identifies a channel without its (constant) base, for grouping the
// arrows that share it.
type channelKey struct {
	path     string
	vertical bool
	index    int
}

func (r channelRef) key() channelKey { return channelKey{r.path, r.vertical, r.index} }

// laneMap records, per channel, the lane each arrow occupies (0-based) and how
// many lanes the channel carries. Several arrows running along one channel are
// spread across distinct lanes so they do not overlap.
type laneMap struct {
	index map[channelKey]map[int]int // channel -> arrow index -> lane
	count map[channelKey]int         // channel -> number of lanes
}

// lane returns the lane an arrow occupies in a channel and the channel's lane
// count, or ok = false when the arrow does not run along it.
func (m laneMap) lane(k channelKey, arrowIdx int) (idx, count int, ok bool) {
	byArrow, ok := m.index[k]
	if !ok {
		return 0, 0, false
	}
	idx, ok = byArrow[arrowIdx]
	if !ok {
		return 0, 0, false
	}
	return idx, m.count[k], true
}

// channelDemand routes every arrow on the (un-widened) layout and records, per
// channel, which arrows run longitudinally along it. It returns the extra width
// each channel reserves (lanes × margin/2) and a laneMap assigning each arrow a
// distinct lane in every channel it shares. Returns (nil, empty) when nothing
// needs widening. Lanes are ordered by arrow index, so the result is deterministic.
func channelDemand(arrows []ast.Arrow, layout layoutResult, mode ast.RouteMode, fontSize float64) (*widenDemand, laneMap) {
	uses := map[channelKey]map[int]bool{} // channel -> set of arrow indices along it
	base := map[channelKey]float64{}
	labelExtra := map[channelKey]float64{} // channel -> extra width its label's text needs
	for i, a := range arrows {
		pts, ok := arrowPath(a, layout, mode)
		if !ok {
			continue
		}
		for j := 0; j+1 < len(pts); j++ {
			if samePoint(pts[j], pts[j+1]) {
				continue
			}
			ref, ok := findChannel(layout.roots, pts[j], pts[j+1])
			if !ok {
				continue
			}
			k := ref.key()
			if uses[k] == nil {
				uses[k] = map[int]bool{}
			}
			uses[k][i] = true
			base[k] = ref.base
		}
		// A label flows in the channel its longest segment follows, so that
		// channel must be wide enough to hold the text (plus a margin of clearance
		// each side) — exactly as the line itself reserves a lane. A vertical run
		// (column boundary) takes the label's width; a horizontal run its height.
		if a.Label != nil {
			if p0, p1, ok := longestSegment(pts); ok {
				if ref, ok := findChannel(layout.roots, p0, p1); ok {
					k := ref.key()
					extent := textHeight(*a.Label, fontSize)
					if k.vertical {
						extent = textWidth(*a.Label, fontSize)
					}
					if d := extent + ref.base; d > labelExtra[k] {
						labelExtra[k] = d
					}
				}
			}
		}
	}
	if len(uses) == 0 {
		return nil, laneMap{}
	}

	d := &widenDemand{cols: map[string][]float64{}, rows: map[string][]float64{}}
	lm := laneMap{index: map[channelKey]map[int]int{}, count: map[channelKey]int{}}
	for k, set := range uses {
		arrowIdxs := make([]int, 0, len(set))
		for ai := range set {
			arrowIdxs = append(arrowIdxs, ai)
		}
		sort.Ints(arrowIdxs)
		lm.count[k] = len(arrowIdxs)
		lm.index[k] = make(map[int]int, len(arrowIdxs))
		for lane, ai := range arrowIdxs {
			lm.index[k][ai] = lane
		}

		// Each lane clears the boxes by a full margin (the channel's base) and the
		// lanes are half a margin apart, so a line sits in its own channel beyond the
		// node margin rather than inside it. With the base gap already one margin,
		// N lanes need (N+1)·base/2 of extra width (giving 2·base + (N−1)·base/2).
		extra := float64(len(arrowIdxs)+1) * base[k] / 2
		// A label flowing in this channel may need it wider than the lanes alone do.
		if le := labelExtra[k]; le > extra {
			extra = le
		}
		switch {
		case k.vertical && k.path == "":
			d.root = setAt(d.root, k.index, extra) // column boundary between top-level nodes
		case k.vertical:
			d.cols[k.path] = setAt(d.cols[k.path], k.index, extra)
		default:
			d.rows[k.path] = setAt(d.rows[k.path], k.index, extra)
		}
	}
	return d, lm
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
	segHorizontal := math.Abs(p0.y-p1.y) < epsilon // the segment runs along X
	mid := point{(p0.x + p1.x) / 2, (p0.y + p1.y) / 2}

	var container *layoutNode // nil = the document root
	children := roots
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
			container = c
			children = c.children
			path = c.path
			continue
		}
		// The midpoint is in free space of this container. A vertical run travels
		// along a column boundary (it needs horizontal clearance); a horizontal run
		// along a row boundary. The segment's own orientation picks the axis.
		if segHorizontal {
			idx, b, ok := rowAxis(container, children).boundaryAt(mid.y)
			return channelRef{path: path, vertical: false, index: idx, base: b}, ok
		}
		idx, b, ok := colAxis(container, children).boundaryAt(mid.x)
		return channelRef{path: path, vertical: true, index: idx, base: b}, ok
	}
}

// snapToLanes places each interior longitudinal run of arrow arrowIdx's polyline
// in its lane within the (widened) channel it belongs to, so the line sits in a
// reserved lane rather than at a fixed inset from one box. When a channel carries
// several arrows they take distinct lanes around the centre (lane k of N is offset
// (k − (N−1)/2) × margin/2), so co-routed lines do not overlap; a single-lane
// channel keeps the centred run (offset 0). The first and last segments are left
// alone — they are the tip's exit/entry tails, anchored to the box. Snapping a run
// only moves its perpendicular coordinate, so connecting segments stay axis-aligned.
func snapToLanes(pts []point, layout layoutResult, arrowIdx int, lanes laneMap) []point {
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
		if k, n, ok := lanes.lane(ref.key(), arrowIdx); ok {
			center += (float64(k) - float64(n-1)/2) * ref.base / 2
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
// along it snaps to. A column boundary's centre is on X, a row boundary's on Y.
func channelCenterAt(roots []*layoutNode, ref channelRef) (float64, bool) {
	var container *layoutNode
	children := roots
	if ref.path != "" {
		container = nodeByPath(roots, ref.path)
		if container == nil {
			return 0, false
		}
		children = container.children
	}
	if ref.vertical {
		return colAxis(container, children).center(ref.index)
	}
	return rowAxis(container, children).center(ref.index)
}

// axisInfo describes one axis (columns or rows) of a container for widening: the
// number of tracks, each track's near/far edge (1-based; index 0 unused), the
// container's content lo/hi edges on that axis, the perimeter base margins, and the
// per-between-boundary base (the collapsed channel margin). It unifies a 1-D stack
// (one axis a single track) and a 2-D grid. ok is false when the axis is absent
// (e.g. the document root has no row axis).
type axisInfo struct {
	count             int
	near, far         []float64
	lo, hi            float64
	baseLow, baseHigh float64
	gapBase           []float64 // 1-based; gapBase[k] = base of the boundary between track k and k+1
	ok                bool
}

// boundaryAt returns the track boundary (0 = low perimeter … count = high) the
// coordinate falls in along this axis, plus that boundary's base margin. ok is
// false when the axis is absent or the coordinate sits inside a track (not a lane).
func (a axisInfo) boundaryAt(coord float64) (int, float64, bool) {
	if !a.ok || a.count == 0 {
		return 0, 0, false
	}
	if coord < a.near[1]-epsilon {
		return 0, a.baseLow, true
	}
	for k := 1; k < a.count; k++ {
		if coord > a.far[k]+epsilon && coord < a.near[k+1]-epsilon {
			return k, a.gapBase[k], true
		}
	}
	if coord > a.far[a.count]+epsilon {
		return a.count, a.baseHigh, true
	}
	return 0, 0, false
}

// center returns the centre coordinate of a track boundary — the lane a run along
// it snaps to.
func (a axisInfo) center(index int) (float64, bool) {
	if !a.ok || a.count == 0 {
		return 0, false
	}
	switch {
	case index <= 0:
		return (a.lo + a.near[1]) / 2, true
	case index >= a.count:
		return (a.far[a.count] + a.hi) / 2, true
	default:
		return (a.far[index] + a.near[index+1]) / 2, true
	}
}

// colAxis builds the column axis (the channels vertical runs travel along) of a
// container; container == nil is the document root, a single horizontal row of
// top-level nodes with no border perimeter.
func colAxis(container *layoutNode, children []*layoutNode) axisInfo {
	if container == nil {
		n := len(children)
		if n == 0 {
			return axisInfo{}
		}
		a := axisInfo{count: n, near: make([]float64, n+1), far: make([]float64, n+1), gapBase: make([]float64, n+1), ok: true}
		for k := 1; k <= n; k++ {
			a.near[k], a.far[k] = children[k-1].dim.x, children[k-1].dim.x+children[k-1].dim.width
		}
		a.lo, a.hi = a.near[1], a.far[n]
		a.baseLow, a.baseHigh = children[0].margin, children[n-1].margin
		for k := 1; k < n; k++ {
			a.gapBase[k] = math.Max(children[k-1].margin, children[k].margin)
		}
		return a
	}
	g := container.grid
	if g == nil {
		return axisInfo{}
	}
	a := trackExtents(g.cols, g.placed, container.children, true)
	b := 0.0
	if nodeBordered(container) {
		b = container.borderW
	}
	a.lo, a.hi = container.dim.x+b, container.dim.x+container.dim.width-b
	a.baseLow, a.baseHigh = g.leftPerim, g.rightPerim
	a.gapBase = g.colGap
	return a
}

// rowAxis builds the row axis (the channels horizontal runs travel along) of a
// container; the document root has no row axis. The label band sits on this axis,
// so it shifts the content lo (top band) or hi (bottom band).
func rowAxis(container *layoutNode, children []*layoutNode) axisInfo {
	if container == nil || container.grid == nil {
		return axisInfo{}
	}
	g := container.grid
	a := trackExtents(g.rows, g.placed, container.children, false)
	b := 0.0
	if nodeBordered(container) {
		b = container.borderW
	}
	a.lo, a.hi = container.dim.y+b, container.dim.y+container.dim.height-b
	if container.labelBand > 0 {
		if labelPositionOf(container.decls) == ast.LabelBottom {
			a.hi -= container.labelBand
		} else {
			a.lo += container.labelBand
		}
	}
	a.baseLow, a.baseHigh = g.topPerim, g.botPerim
	a.gapBase = g.rowGap
	return a
}

// trackExtents computes each grid track's near/far edge on one axis (vertical =
// columns/X, else rows/Y) from the single-span children placed in it.
func trackExtents(count int, placed []ast.PlacedCell, children []*layoutNode, vertical bool) axisInfo {
	a := axisInfo{count: count, near: make([]float64, count+1), far: make([]float64, count+1), ok: count > 0}
	for k := 1; k <= count; k++ {
		a.near[k], a.far[k] = math.Inf(1), math.Inf(-1)
	}
	for i, pc := range placed {
		if i >= len(children) {
			break
		}
		ch := children[i]
		track, span := pc.Row, pc.RowSpan
		near, far := ch.dim.y, ch.dim.y+ch.dim.height
		if vertical {
			track, span = pc.Col, pc.ColSpan
			near, far = ch.dim.x, ch.dim.x+ch.dim.width
		}
		if span == 1 && track >= 1 && track <= count {
			a.near[track] = math.Min(a.near[track], near)
			a.far[track] = math.Max(a.far[track], far)
		}
	}
	return a
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
