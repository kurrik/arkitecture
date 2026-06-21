package generator

import (
	"math"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func TestAxisLines(t *testing.T) {
	// One obstacle x in [40,60]; inset 5 puts lanes at 35 and 65, plus the two
	// endpoints — sorted and de-duplicated.
	got := axisLines(0, 100, []dimensions{{x: 40, y: 0, width: 20, height: 20}}, true, 5)
	want := []float64{0, 35, 65, 100}
	if len(got) != len(want) {
		t.Fatalf("axisLines = %v, want %v", got, want)
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > epsilon {
			t.Fatalf("axisLines = %v, want %v", got, want)
		}
	}
}

func TestRouteAroundDetoursACenteredBlocker(t *testing.T) {
	// A blocker straddles the straight line from (0,50) to (100,50); the router
	// must detour over or under it. Lanes sit one inset (5) outside the blocker,
	// so the only horizontal crossing lanes are y=25 (top-inset) and y=75
	// (bottom+inset); ties resolve north, so the route goes over the top.
	blocker := dimensions{x: 40, y: 30, width: 20, height: 40} // interior x(40,60) y(30,70)
	got := routeAround(point{0, 50}, point{100, 50}, []dimensions{blocker}, 5)
	want := []point{{0, 50}, {0, 25}, {100, 25}, {100, 50}}
	if !eqPts(got, want) {
		t.Fatalf("routeAround = %v, want %v", got, want)
	}
	if !segClear(got[0], got[1], []dimensions{blocker}) || !pathClear(got, []dimensions{blocker}) {
		t.Errorf("route %v is not clear of the blocker", got)
	}
}

func TestRouteAroundReturnsNilWhenEnclosed(t *testing.T) {
	// Four walls ring the target with no gap, so no orthogonal lane reaches it.
	walls := []dimensions{
		{x: 40, y: 40, width: 5, height: 20}, // left
		{x: 55, y: 40, width: 5, height: 20}, // right
		{x: 40, y: 40, width: 20, height: 5}, // top
		{x: 40, y: 55, width: 20, height: 5}, // bottom
	}
	if got := routeAround(point{0, 0}, point{50, 50}, walls, 4); got != nil {
		t.Errorf("routeAround into an enclosed target = %v, want nil", got)
	}
}

func TestRouteAroundIsDeterministic(t *testing.T) {
	obstacles := []dimensions{
		{x: 30, y: 10, width: 20, height: 20},
		{x: 30, y: 50, width: 40, height: 15},
		{x: 70, y: 20, width: 15, height: 40},
	}
	a := routeAround(point{0, 40}, point{120, 40}, obstacles, 6)
	b := routeAround(point{0, 40}, point{120, 40}, obstacles, 6)
	if !eqPts(a, b) {
		t.Errorf("routeAround not deterministic:\n  %v\n  %v", a, b)
	}
	if a == nil {
		t.Error("expected a route around the obstacles, got nil")
	}
	if !axisAligned(a) || !pathClear(a, obstacles) {
		t.Errorf("route %v is not a clear axis-aligned path", a)
	}
}

func TestRouteAroundPrefersFewerBends(t *testing.T) {
	// The blocker sits below the straight line, leaving y=50 clear all the way
	// across: the router should take that 0-bend straight shot, not dip around.
	blocker := dimensions{x: 40, y: 60, width: 20, height: 20}
	got := routeAround(point{0, 50}, point{100, 50}, []dimensions{blocker}, 5)
	want := []point{{0, 50}, {100, 50}}
	if !eqPts(got, want) {
		t.Errorf("routeAround = %v, want the straight %v (no needless bend)", got, want)
	}
}

// TestArrowPathRoutesAroundObstacle is the end-to-end check: three boxes in a
// row, an arrow from the first to the last whose direct elbow is blocked by the
// middle box, so arrowPath must emit a detour rather than the straight fallback.
func TestArrowPathRoutesAroundObstacle(t *testing.T) {
	layout := layoutResult{
		nodeBoxes: map[string]dimensions{
			"a":   {x: 0, y: 0, width: 20, height: 20},
			"blk": {x: 40, y: 0, width: 20, height: 20},
			"c":   {x: 80, y: 0, width: 20, height: 20},
		},
		defMargin: 8,
	}
	pts, ok := arrowPath(ast.Arrow{Source: "a", Target: "c"}, layout, ast.RouteOrthogonal)
	if !ok {
		t.Fatal("arrowPath returned ok=false")
	}
	if len(pts) <= 2 {
		t.Fatalf("path %v has no bend — it fell back to a straight line through the blocker", pts)
	}
	if !axisAligned(pts) {
		t.Errorf("path %v is not axis-aligned", pts)
	}
	if !pathClear(pts, []dimensions{layout.nodeBoxes["blk"]}) {
		t.Errorf("path %v cuts through the blocker", pts)
	}
	if !samePoint(pts[0], point{20, 10}) || !samePoint(pts[len(pts)-1], point{80, 10}) {
		t.Errorf("path %v should run between the boxes' facing edges (20,10)->(80,10)", pts)
	}
}
