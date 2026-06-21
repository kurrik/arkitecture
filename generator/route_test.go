package generator

import (
	"math"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func eqPts(a, b []point) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !samePoint(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestCardinalEndpoint(t *testing.T) {
	box := dimensions{x: 10, y: 10, width: 20, height: 20} // centre (20,20)
	for _, tt := range []struct {
		name     string
		aim      point
		wantP    point
		wantSide side
	}{
		{"east", point{100, 22}, point{30, 20}, east},
		{"west", point{-100, 18}, point{10, 20}, west},
		{"south", point{21, 100}, point{20, 30}, south},
		{"north", point{19, -100}, point{20, 10}, north},
		{"exact diagonal favours horizontal", point{40, 40}, point{30, 20}, east},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, s := cardinalEndpoint(box, tt.aim)
			if !samePoint(p, tt.wantP) || s != tt.wantSide {
				t.Errorf("got (%v, %d), want (%v, %d)", p, s, tt.wantP, tt.wantSide)
			}
		})
	}
}

func TestEdgePointOnSide(t *testing.T) {
	box := dimensions{x: 10, y: 10, width: 20, height: 20} // 10,10 .. 30,30
	tip := point{17, 23}                                   // an interior point
	for _, tt := range []struct {
		s    side
		want point
	}{
		{east, point{30, 23}},  // right border, anchor's y
		{west, point{10, 23}},  // left border, anchor's y
		{south, point{17, 30}}, // bottom border, anchor's x
		{north, point{17, 10}}, // top border, anchor's x
	} {
		if got := edgePointOnSide(box, tt.s, tip); !samePoint(got, tt.want) {
			t.Errorf("edgePointOnSide(side %d) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// TestArrowPathEntersInteriorAnchor checks that orthogonal routing to an interior
// anchor produces an axis-aligned path that ends at the anchor inside the box —
// the line enters the node, rather than stopping at the border.
func TestArrowPathEntersInteriorAnchor(t *testing.T) {
	layout := layoutResult{
		nodeBoxes: map[string]dimensions{
			"a": {x: 0, y: 0, width: 20, height: 20},
			"b": {x: 60, y: 40, width: 40, height: 40},
		},
		anchorPositions: []anchorPosition{
			{x: 80, y: 60, nodeID: "b", anchorID: "core"}, // centre of b (interior)
		},
	}
	pts, ok := arrowPath(ast.Arrow{Source: "a", Target: "b#core"}, layout, ast.RouteOrthogonal)
	if !ok {
		t.Fatal("arrowPath returned ok=false")
	}
	last := pts[len(pts)-1]
	if !samePoint(last, point{80, 60}) {
		t.Errorf("path end = %v, want the interior anchor (80,60)", last)
	}
	for i := 0; i+1 < len(pts); i++ {
		if math.Abs(pts[i].x-pts[i+1].x) > epsilon && math.Abs(pts[i].y-pts[i+1].y) > epsilon {
			t.Errorf("segment %v -> %v is not axis-aligned", pts[i], pts[i+1])
		}
	}
	b := layout.nodeBoxes["b"]
	if last.x <= b.x || last.x >= b.x+b.width || last.y <= b.y || last.y >= b.y+b.height {
		t.Errorf("path end %v is not interior to b %v — the line did not enter the node", last, b)
	}
}

func TestBreakoutBox(t *testing.T) {
	boxes := map[string]dimensions{
		"ordering":            {x: 0, width: 10},
		"ordering.orders":     {x: 1, width: 5},
		"ordering.row":        {x: 2, width: 5},
		"ordering.row.cart":   {x: 3, width: 2},
		"ordering.row.basket": {x: 4, width: 2},
		"inventory":           {x: 20, width: 10},
	}
	for _, tt := range []struct {
		name        string
		self, other string
		wantX       float64 // identifies the expected box by its x
	}{
		{"different top-level branches", "ordering.orders", "inventory", 0},                   // -> ordering
		{"target nested in other branch", "inventory", "ordering.orders", 20},                 // -> inventory
		{"siblings share a parent", "ordering.row.cart", "ordering.row.basket", 3},            // -> cart (leaf)
		{"deeper self breaks out to common child", "ordering.orders", "ordering.row.cart", 1}, // -> orders (leaf, child of ordering)
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := breakoutBox(tt.self, tt.other, boxes); got.x != tt.wantX {
				t.Errorf("breakoutBox(%q,%q).x = %v, want %v", tt.self, tt.other, got.x, tt.wantX)
			}
		})
	}
}

func TestAnchorEdge(t *testing.T) {
	box := dimensions{x: 10, y: 10, width: 20, height: 20} // 10,10 .. 30,30
	for _, tt := range []struct {
		name string
		tip  point
		want side
		ok   bool
	}{
		{"south", point{20, 30}, south, true},
		{"north", point{20, 10}, north, true},
		{"west", point{10, 20}, west, true},
		{"east", point{30, 20}, east, true},
		{"interior is not an edge", point{20, 20}, 0, false},
		{"corner is two edges, not one", point{30, 30}, 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := anchorEdge(box, tt.tip)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("anchorEdge(%v) = (%d,%v), want (%d,%v)", tt.tip, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// TestArrowPathEdgeAnchorExitsPerpendicular checks that an anchor on a box edge
// that is not the side facing the target leaves *perpendicular to its own edge*
// (here straight down from a bottom anchor) and detours around its own box rather
// than cutting back through it.
func TestArrowPathEdgeAnchorExitsPerpendicular(t *testing.T) {
	layout := layoutResult{
		nodeBoxes: map[string]dimensions{
			"a": {x: 0, y: 0, width: 24, height: 24},
			"b": {x: 40, y: 0, width: 24, height: 24}, // to the east
		},
		anchorPositions: []anchorPosition{
			{x: 12, y: 24, nodeID: "a", anchorID: "out"}, // bottom-centre: a south edge
		},
		defMargin: 16,
	}
	pts, ok := arrowPath(ast.Arrow{Source: "a#out", Target: "b"}, layout, ast.RouteOrthogonal)
	if !ok {
		t.Fatal("arrowPath ok=false")
	}
	if !samePoint(pts[0], point{12, 24}) {
		t.Fatalf("path starts at %v, want the anchor (12,24)", pts[0])
	}
	if pts[1].x != pts[0].x || pts[1].y <= pts[0].y {
		t.Errorf("first segment %v->%v is not a downward (south) exit perpendicular to the edge", pts[0], pts[1])
	}
	if !pathClear(pts, []dimensions{layout.nodeBoxes["a"]}) {
		t.Errorf("path %v cuts back through its own source box a", pts)
	}
}

func TestElbow(t *testing.T) {
	for _, tt := range []struct {
		name string
		a    point
		sa   side
		b    point
		sb   side
		want []point
	}{
		{
			"both horizontal, offset -> Z through mid-x",
			point{10, 0}, east, point{30, 8}, west,
			[]point{{10, 0}, {20, 0}, {20, 8}, {30, 8}},
		},
		{
			"both vertical, offset -> Z through mid-y",
			point{0, 10}, south, point{8, 30}, north,
			[]point{{0, 10}, {0, 20}, {8, 20}, {8, 30}},
		},
		{
			"perpendicular, horizontal exit first -> single corner",
			point{0, 0}, east, point{10, 20}, north,
			[]point{{0, 0}, {10, 0}, {10, 20}},
		},
		{
			"perpendicular, vertical exit first -> single corner",
			point{0, 0}, south, point{20, 10}, west,
			[]point{{0, 0}, {0, 10}, {20, 10}},
		},
		{
			"aligned horizontal collapses to a straight segment",
			point{10, 5}, east, point{30, 5}, west,
			[]point{{10, 5}, {30, 5}},
		},
		{
			"aligned vertical collapses to a straight segment",
			point{5, 10}, south, point{5, 30}, north,
			[]point{{5, 10}, {5, 30}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := elbow(tt.a, tt.sa, tt.b, tt.sb)
			if !eqPts(got, tt.want) {
				t.Errorf("elbow = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimplifyDropsDuplicatesAndCollinear(t *testing.T) {
	in := []point{{0, 0}, {0, 0}, {5, 0}, {10, 0}, {10, 10}}
	want := []point{{0, 0}, {10, 0}, {10, 10}}
	if got := simplify(in); !eqPts(got, want) {
		t.Errorf("simplify = %v, want %v", got, want)
	}
}

func TestSegIntersectsRect(t *testing.T) {
	r := dimensions{x: 10, y: 10, width: 20, height: 20} // interior (10,10)-(30,30)
	for _, tt := range []struct {
		name string
		p1   point
		p2   point
		want bool
	}{
		{"horizontal through interior", point{0, 20}, point{40, 20}, true},
		{"vertical through interior", point{20, 0}, point{20, 40}, true},
		{"runs along top edge -> grazes, not blocked", point{0, 10}, point{40, 10}, false},
		{"runs along left edge -> grazes, not blocked", point{10, 0}, point{10, 40}, false},
		{"clear above", point{0, 5}, point{40, 5}, false},
		{"ends at the edge, no interior crossing", point{0, 20}, point{10, 20}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := segIntersectsRect(tt.p1, tt.p2, r); got != tt.want {
				t.Errorf("segIntersectsRect = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelated(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want bool
	}{
		{"a", "a", true},
		{"a", "a.b", true},     // ancestor
		{"a.b.c", "a", true},   // descendant
		{"a.b", "a.c", false},  // siblings
		{"a", "ab", false},     // not a path-prefix
		{"a.b", "a.bc", false}, // not a path-prefix
	} {
		if got := related(tt.a, tt.b); got != tt.want {
			t.Errorf("related(%q,%q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestArrowObstaclesExcludesEndpointLineages(t *testing.T) {
	boxes := map[string]dimensions{
		"col":         {},
		"col.a":       {x: 1},
		"col.filler":  {x: 2},
		"other":       {x: 3},
		"other.b":     {x: 4},
		"other.inner": {x: 5},
	}
	obs := arrowObstacles("col.a", "other.b", boxes)
	// Excluded: col, col.a (source lineage); other, other.b (target lineage).
	// Obstacles: col.filler and other.inner (other.inner is a descendant of the
	// target's *container* but not of the target itself, so it is an obstacle).
	gotX := map[float64]bool{}
	for _, d := range obs {
		gotX[d.x] = true
	}
	if len(obs) != 2 || !gotX[2] || !gotX[5] {
		t.Errorf("obstacles = %+v, want exactly col.filler(x=2) and other.inner(x=5)", obs)
	}
}
