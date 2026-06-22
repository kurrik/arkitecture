package generator

import (
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

// boxNode is a small helper for hand-building layout trees in widening tests.
func boxNode(path string, x, y, w, h, margin float64, children ...*layoutNode) *layoutNode {
	return &layoutNode{path: path, dim: dimensions{x: x, y: y, width: w, height: h}, margin: margin, children: children}
}

// vstackGrid builds the gridInfo a vertical stack of n children (one per row,
// single column) would carry after calcGrid, with a uniform margin — enough for the
// router's axis helpers, which read track counts, gap/perimeter bases, and placement.
func vstackGrid(n int, margin float64) *gridInfo {
	g := &gridInfo{
		cols: 1, rows: n,
		colGap: make([]float64, 2), rowGap: make([]float64, n+1),
		leftPerim: margin, rightPerim: margin, topPerim: margin, botPerim: margin,
		placed: make([]ast.PlacedCell, n),
	}
	for r := 1; r < n; r++ {
		g.rowGap[r] = margin
	}
	for i := 0; i < n; i++ {
		g.placed[i] = ast.PlacedCell{Col: 1, Row: i + 1, ColSpan: 1, RowSpan: 1}
	}
	return g
}

func vstackBox() *layoutNode {
	// A vertical container "box" with two stacked children (x 10..90), leaving a
	// between-rows gap at y 40..60 and left/right column perimeters outside x 10..90.
	box := boxNode("box", 0, 0, 100, 100, 8,
		boxNode("box.a", 10, 10, 80, 30, 8),
		boxNode("box.b", 10, 60, 80, 30, 8),
	)
	box.grid = vstackGrid(2, 8)
	return box
}

func TestFindChannelAttributesColumnsAndRows(t *testing.T) {
	roots := []*layoutNode{vstackBox()}

	// A horizontal run through the gap is a lane in box's row boundary 1.
	if ref, ok := findChannel(roots, point{20, 50}, point{80, 50}); !ok || ref.path != "box" || ref.vertical || ref.index != 1 || ref.base != 8 {
		t.Errorf("findChannel(row run) = (%+v,%v), want {path:box row index:1 base:8}", ref, ok)
	}
	if c, ok := channelCenterAt(roots, channelRef{path: "box", vertical: false, index: 1}); !ok || c != 50 {
		t.Errorf("channelCenterAt(box,row 1) = (%v,%v), want (50,true)", c, ok)
	}

	// A vertical run left of the children travels along box's left column perimeter.
	if ref, ok := findChannel(roots, point{5, 30}, point{5, 70}); !ok || ref.path != "box" || !ref.vertical || ref.index != 0 || ref.base != 8 {
		t.Errorf("findChannel(column run) = (%+v,%v), want {path:box col index:0 base:8}", ref, ok)
	}

	// A vertical run *within* the children's span would pass through them — not a channel.
	if _, ok := findChannel(roots, point{50, 45}, point{50, 55}); ok {
		t.Error("findChannel(through children) = ok, want not attributed")
	}
}

func TestSnapToLanesCentresInteriorRuns(t *testing.T) {
	roots := []*layoutNode{vstackBox()}
	layout := layoutResult{roots: roots}
	// tip -> down -> horizontal run through the gap (off-centre at y=44) -> up -> tip.
	in := []point{{20, 20}, {20, 44}, {80, 44}, {80, 20}}
	out := snapToLanes(in, layout, 0, laneMap{}) // no shared lanes -> centre
	// The interior horizontal run snaps to the row-boundary centre (50); the tips stay.
	if !samePoint(out[0], point{20, 20}) || !samePoint(out[3], point{80, 20}) {
		t.Errorf("snap moved a tip: %v", out)
	}
	if out[1].y != 50 || out[2].y != 50 {
		t.Errorf("interior run not centred: got y=%v,%v want 50", out[1].y, out[2].y)
	}
}

func TestSnapToLanesDistributesSharedChannel(t *testing.T) {
	roots := []*layoutNode{vstackBox()}
	layout := layoutResult{roots: roots}
	// box's row boundary 1 (centre 50, base 8) shared by arrows 0 and 1.
	gap := channelKey{path: "box", vertical: false, index: 1}
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
