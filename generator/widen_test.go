package generator

import "testing"

// boxNode is a small helper for hand-building layout trees in widening tests.
func boxNode(path string, x, y, w, h, margin float64, children ...*layoutNode) *layoutNode {
	return &layoutNode{path: path, dim: dimensions{x: x, y: y, width: w, height: h}, margin: margin, children: children}
}

func TestFindChannelAttributesGapsAndRails(t *testing.T) {
	// A vertical container "box" with two stacked children (x 10..90), leaving a
	// between-children gap at y 40..60 and left/right perimeter rails outside x 10..90.
	roots := []*layoutNode{boxNode("box", 0, 0, 100, 100, 8,
		boxNode("box.a", 10, 10, 80, 30, 8),
		boxNode("box.b", 10, 60, 80, 30, 8),
	)}

	// A horizontal run through the gap is a lane in box's between-children gap (1).
	if ref, ok := findChannel(roots, point{20, 50}, point{80, 50}); !ok || ref.path != "box" || ref.rail || ref.index != 1 || ref.base != 8 {
		t.Errorf("findChannel(gap run) = (%+v,%v), want {path:box gap index:1 base:8}", ref, ok)
	}
	if c, ok := gapCenterAt(roots, "box", 1); !ok || c != 50 {
		t.Errorf("gapCenterAt(box,1) = (%v,%v), want (50,true)", c, ok)
	}

	// A vertical run left of the children runs *along* box's main axis — its low
	// (left) cross-axis perimeter rail.
	if ref, ok := findChannel(roots, point{5, 30}, point{5, 70}); !ok || ref.path != "box" || !ref.rail || ref.index != 0 || ref.base != 8 {
		t.Errorf("findChannel(rail run) = (%+v,%v), want {path:box rail side:0 base:8}", ref, ok)
	}

	// A vertical run *within* the children's span would pass through them — not a channel.
	if _, ok := findChannel(roots, point{50, 45}, point{50, 55}); ok {
		t.Error("findChannel(through children) = ok, want not attributed")
	}
}

func TestSnapToLanesCentresInteriorRuns(t *testing.T) {
	roots := []*layoutNode{boxNode("box", 0, 0, 100, 100, 8,
		boxNode("box.a", 10, 10, 80, 30, 8),
		boxNode("box.b", 10, 60, 80, 30, 8),
	)}
	layout := layoutResult{roots: roots}
	// tip -> down -> horizontal run through the gap (off-centre at y=44) -> up -> tip.
	in := []point{{20, 20}, {20, 44}, {80, 44}, {80, 20}}
	out := snapToLanes(in, layout, 0, laneMap{}) // no shared lanes -> centre
	// The interior horizontal run snaps to the gap centre (50); the tip tails stay put.
	if !samePoint(out[0], point{20, 20}) || !samePoint(out[3], point{80, 20}) {
		t.Errorf("snap moved a tip: %v", out)
	}
	if out[1].y != 50 || out[2].y != 50 {
		t.Errorf("interior run not centred: got y=%v,%v want 50", out[1].y, out[2].y)
	}
}

func TestSnapToLanesDistributesSharedChannel(t *testing.T) {
	roots := []*layoutNode{boxNode("box", 0, 0, 100, 100, 8,
		boxNode("box.a", 10, 10, 80, 30, 8),
		boxNode("box.b", 10, 60, 80, 30, 8),
	)}
	layout := layoutResult{roots: roots}
	// box's between-children gap (centre 50, base 8) shared by arrows 0 and 1.
	gap := channelKey{path: "box", index: 1}
	lanes := laneMap{
		index: map[channelKey]map[int]int{gap: {0: 0, 1: 1}},
		count: map[channelKey]int{gap: 2},
	}
	run := []point{{20, 20}, {20, 44}, {80, 44}, {80, 20}}
	a0 := snapToLanes(run, layout, 0, lanes)
	a1 := snapToLanes(run, layout, 1, lanes)
	// Two lanes around the centre, margin/2 apart: lane 0 at 50−2, lane 1 at 50+2.
	if a0[1].y != 48 || a1[1].y != 52 {
		t.Errorf("shared-channel lanes at y=%v (lane 0) and y=%v (lane 1), want 48 and 52", a0[1].y, a1[1].y)
	}
}
