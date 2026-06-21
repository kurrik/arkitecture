package generator

import "testing"

// boxNode is a small helper for hand-building layout trees in widening tests.
func boxNode(path string, x, y, w, h, margin float64, children ...*layoutNode) *layoutNode {
	return &layoutNode{path: path, dim: dimensions{x: x, y: y, width: w, height: h}, margin: margin, children: children}
}

func TestFindChannelAttributesGapRun(t *testing.T) {
	// A vertical container "box" with two stacked children leaving a gap at y 40..60.
	roots := []*layoutNode{boxNode("box", 0, 0, 100, 100, 8,
		boxNode("box.a", 10, 10, 80, 30, 8),
		boxNode("box.b", 10, 60, 80, 30, 8),
	)}

	// A horizontal run through that gap is a lane in box's between-children gap (1).
	path, gi, base, ok := findChannel(roots, point{20, 50}, point{80, 50})
	if !ok || path != "box" || gi != 1 || base != 8 {
		t.Errorf("findChannel(gap run) = (%q,%d,%v,%v), want (box,1,8,true)", path, gi, base, ok)
	}
	if c, ok := gapCenterAt(roots, "box", 1); !ok || c != 50 {
		t.Errorf("gapCenterAt(box,1) = (%v,%v), want (50,true)", c, ok)
	}

	// A vertical run through the same gap is *along* box's main axis — a cross-axis
	// rail, deferred — so it is not attributed to a main-axis gap.
	if _, _, _, ok := findChannel(roots, point{50, 45}, point{50, 55}); ok {
		t.Error("findChannel(rail run) = ok, want not attributed (rails are deferred)")
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
	out := snapToLanes(in, layout)
	// The interior horizontal run snaps to the gap centre (50); the tip tails stay put.
	if !samePoint(out[0], point{20, 20}) || !samePoint(out[3], point{80, 20}) {
		t.Errorf("snap moved a tip: %v", out)
	}
	if out[1].y != 50 || out[2].y != 50 {
		t.Errorf("interior run not centred: got y=%v,%v want 50", out[1].y, out[2].y)
	}
}
