package generator

import (
	"container/heap"
	"math"
	"sort"
)

// channel.go is the channel-graph router. When an arrow's clear-case elbow
// (route.go) would cut through a box between its endpoints, this finds an
// orthogonal detour *around* the obstacles instead of letting the arrow fall
// back to a straight line through them.
//
// The graph is a sparse "channel grid". Candidate travel lanes sit one `inset`
// outside every obstacle edge — so a line runs in the gap *beside* a box, never
// on its border; in the common uniform-margin layout the two lanes flanking a
// channel coincide on its centreline (inset = margin/2). Grid vertices are the
// crossings of these lanes (plus the lines through the two endpoints); a grid
// edge joins two adjacent crossings when the segment between them grazes no
// obstacle interior. Few-bend A* over that graph assigns each arrow a path.
//
// This is the geometric realisation of the design's arrangement-derived channel
// graph (docs/design.md): the gaps between boxes are the corridors, not a pixel
// grid. Because channel *widening* (lane counts pushing boxes apart) is a later
// slice, routing here runs on the already-computed layout — no box moves, so
// there is no routing↔layout feedback loop to resolve yet. Determinism comes
// from sorted gridlines and a total order on the A* frontier, so the golden SVGs
// stay byte-stable.

// bendPenalty is the cost A* adds each time a route changes direction. It is
// large enough to prefer a longer straight-ish path over a shorter zig-zag (few
// turns read as intentional) but finite, so a genuinely shorter detour still
// wins. Among equal-cost routes the frontier's total order picks the
// smaller-coordinate (north/west) one, so the choice is deterministic.
const bendPenalty = 12.0

// routeAround finds an orthogonal path from srcEdge to tgtEdge that stays clear
// of every obstacle's interior, or returns nil when none exists. The result
// starts at srcEdge and ends at tgtEdge and bends as few times as possible; it
// is fully deterministic. inset is how far outside each obstacle edge a travel
// lane sits (typically half a margin, so lanes land on channel centrelines).
func routeAround(srcEdge, tgtEdge point, obstacles []dimensions, inset float64) []point {
	xs := axisLines(srcEdge.x, tgtEdge.x, obstacles, true, inset)
	ys := axisLines(srcEdge.y, tgtEdge.y, obstacles, false, inset)
	nx, ny := len(xs), len(ys)

	si, sj := indexNear(xs, srcEdge.x), indexNear(ys, srcEdge.y)
	ti, tj := indexNear(xs, tgtEdge.x), indexNear(ys, tgtEdge.y)
	if si < 0 || sj < 0 || ti < 0 || tj < 0 {
		return nil
	}
	if si == ti && sj == tj {
		return []point{srcEdge}
	}

	// Direction codes for the incoming move; 0 means "start" (no bend yet).
	const (
		dirNone = iota
		dirE
		dirW
		dirS
		dirN
	)
	moves := [...]struct {
		dir, di, dj int
	}{{dirE, 1, 0}, {dirW, -1, 0}, {dirS, 0, 1}, {dirN, 0, -1}}

	// A search state is (vertex, incoming direction) so a bend can be charged
	// when the direction changes. Costs/back-pointers live in dense slices keyed
	// by that state id — no maps reach the result, keeping it deterministic.
	stateID := func(i, j, dir int) int { return (i*ny+j)*5 + dir }
	gcost := make([]float64, nx*ny*5)
	for i := range gcost {
		gcost[i] = math.Inf(1)
	}
	prev := make([]int, nx*ny*5)
	for i := range prev {
		prev[i] = -1
	}

	heuristic := func(i, j int) float64 {
		return math.Abs(xs[i]-xs[ti]) + math.Abs(ys[j]-ys[tj])
	}
	clearH := func(i, j int) bool { // segment (i,j)–(i+1,j)
		return segClear(point{xs[i], ys[j]}, point{xs[i+1], ys[j]}, obstacles)
	}
	clearV := func(i, j int) bool { // segment (i,j)–(i,j+1)
		return segClear(point{xs[i], ys[j]}, point{xs[i], ys[j+1]}, obstacles)
	}

	pq := &routeHeap{{f: heuristic(si, sj), g: 0, i: si, j: sj, id: stateID(si, sj, dirNone)}}
	gcost[stateID(si, sj, dirNone)] = 0

	for pq.Len() > 0 {
		cur := heap.Pop(pq).(routeItem)
		if cur.g-gcost[cur.id] > epsilon {
			continue // a cheaper route to this state was already settled
		}
		if cur.i == ti && cur.j == tj {
			// A* steps gridline to gridline, so a straight run accrues collinear
			// vertices; collapse them to a minimal corner list.
			return simplify(reconstruct(prev, cur.id, xs, ys, ny))
		}
		curDir := cur.id % 5
		for _, m := range moves {
			ni, nj := cur.i+m.di, cur.j+m.dj
			if ni < 0 || ni >= nx || nj < 0 || nj >= ny {
				continue
			}
			var ok bool
			switch m.dir {
			case dirE:
				ok = clearH(cur.i, cur.j)
			case dirW:
				ok = clearH(ni, cur.j)
			case dirS:
				ok = clearV(cur.i, cur.j)
			default: // dirN
				ok = clearV(cur.i, nj)
			}
			if !ok {
				continue
			}
			ng := cur.g + math.Abs(xs[ni]-xs[cur.i]) + math.Abs(ys[nj]-ys[cur.j])
			if curDir != dirNone && m.dir != curDir {
				ng += bendPenalty
			}
			nid := stateID(ni, nj, m.dir)
			if ng+epsilon < gcost[nid] {
				gcost[nid] = ng
				prev[nid] = cur.id
				heap.Push(pq, routeItem{f: ng + heuristic(ni, nj), g: ng, i: ni, j: nj, id: nid})
			}
		}
	}
	return nil
}

// axisLines returns the sorted, de-duplicated grid coordinates on one axis: the
// two endpoint coordinates, plus each obstacle's two edges on that axis pushed
// out by inset (a lane in the flanking channel). horizontal selects the x axis
// (obstacle left/right) over the y axis (top/bottom).
func axisLines(a, b float64, obstacles []dimensions, horizontal bool, inset float64) []float64 {
	vals := make([]float64, 0, 2+2*len(obstacles))
	vals = append(vals, a, b)
	for _, r := range obstacles {
		if horizontal {
			vals = append(vals, r.x-inset, r.x+r.width+inset)
		} else {
			vals = append(vals, r.y-inset, r.y+r.height+inset)
		}
	}
	sort.Float64s(vals)
	out := vals[:0:0]
	for _, v := range vals {
		if n := len(out); n > 0 && math.Abs(out[n-1]-v) < epsilon {
			continue
		}
		out = append(out, v)
	}
	return out
}

// indexNear returns the index of the value within epsilon of v, or -1. The
// coordinate was placed in vals by axisLines, so the match is exact in practice.
func indexNear(vals []float64, v float64) int {
	for i, x := range vals {
		if math.Abs(x-v) < epsilon {
			return i
		}
	}
	return -1
}

// segClear reports whether the axis-aligned segment a–b grazes no obstacle
// interior (running along a shared border is allowed, per segIntersectsRect).
func segClear(a, b point, obstacles []dimensions) bool {
	for _, r := range obstacles {
		if segIntersectsRect(a, b, r) {
			return false
		}
	}
	return true
}

// reconstruct walks the back-pointers from the goal state to the start and
// returns the vertex coordinates in travel order (srcEdge … tgtEdge).
func reconstruct(prev []int, goal int, xs, ys []float64, ny int) []point {
	var ids []int
	for id := goal; id != -1; id = prev[id] {
		ids = append(ids, id)
	}
	pts := make([]point, len(ids))
	for k, id := range ids {
		vi := id / 5
		pts[len(ids)-1-k] = point{xs[vi/ny], ys[vi%ny]}
	}
	return pts
}

// routeItem is one entry on the A* frontier. id encodes (vertex, incoming
// direction); f = g + heuristic.
type routeItem struct {
	f, g float64
	i, j int
	id   int
}

// routeHeap orders the frontier by f, then g, then state id — a total order, so
// ties resolve deterministically (lower id == smaller coordinate, then a fixed
// direction order), never by map iteration.
type routeHeap []routeItem

func (h routeHeap) Len() int { return len(h) }
func (h routeHeap) Less(a, b int) bool {
	x, y := h[a], h[b]
	if math.Abs(x.f-y.f) > epsilon {
		return x.f < y.f
	}
	if math.Abs(x.g-y.g) > epsilon {
		return x.g < y.g
	}
	return x.id < y.id
}
func (h routeHeap) Swap(a, b int) { h[a], h[b] = h[b], h[a] }
func (h *routeHeap) Push(x any)   { *h = append(*h, x.(routeItem)) }
func (h *routeHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}
