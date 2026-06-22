package generator

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
	"github.com/kurrik/arkitecture/resolve"
)

func ptr(s string) *string                        { return &s }
func iptr(n int) *int                             { return &n }
func fptr(f float64) *float64                     { return &f }
func dirp(d ast.Direction) *ast.Direction         { return &d }
func boxp(b ast.Box) *ast.Box                     { return &b }
func lblp(p ast.LabelPosition) *ast.LabelPosition { return &p }

func rule(selector string, d *ast.Declarations) ast.LayoutRule {
	return ast.LayoutRule{Selector: selector, Decls: d}
}

// render resolves the document's layout and generates SVG, failing on any error.
func render(t *testing.T, doc *ast.Document, opts Options) string {
	t.Helper()
	svg, errs := GenerateSVG(doc, resolve.Resolve(doc), opts)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	return svg
}

func TestGridSingleTrackMatchesStack(t *testing.T) {
	// `direction` is exactly sugar for a single-track grid: `vertical` ≡ `cols: 1`,
	// `horizontal` ≡ `rows: 1`. A direction parent runs through 1-D packing; the
	// equivalent grid runs through the grid engine (the rows:1 case via the
	// transpose). With the shared margin-collapse box model — including box:none not
	// stretching — the two must produce byte-identical SVG. Children carry a uniform
	// margin and varied sizes, and the parent has a label, so band, perimeter,
	// collapsed channels, and the cross-axis default are all exercised.
	children := func() []*ast.ContainerNode {
		return []*ast.ContainerNode{
			{ID: "a", Label: ptr("Alpha")},
			{ID: "b", Label: ptr("Beta is wider")},
			{ID: "c", Label: ptr("C")},
		}
	}
	mk := func(parent *ast.Declarations) *ast.Document {
		return &ast.Document{
			Nodes:  []*ast.ContainerNode{{ID: "p", Label: ptr("Parent"), Children: children()}},
			Layout: []ast.LayoutRule{rule("p", parent), rule("p.a", &ast.Declarations{Margin: fptr(10)}), rule("p.b", &ast.Declarations{Margin: fptr(10)}), rule("p.c", &ast.Declarations{Margin: fptr(10)})},
		}
	}
	none := func(d *ast.Declarations) *ast.Declarations { d.Box = boxp(ast.BoxNone); return d }
	cases := []struct {
		name        string
		stack, grid *ast.Declarations
	}{
		{"vertical ≡ cols:1, bordered", &ast.Declarations{Direction: dirp(ast.Vertical)}, &ast.Declarations{Cols: iptr(1)}},
		{"vertical ≡ cols:1, box:none", none(&ast.Declarations{Direction: dirp(ast.Vertical)}), none(&ast.Declarations{Cols: iptr(1)})},
		{"horizontal ≡ rows:1, bordered", &ast.Declarations{Direction: dirp(ast.Horizontal)}, &ast.Declarations{Rows: iptr(1)}},
		{"horizontal ≡ rows:1, box:none", none(&ast.Declarations{Direction: dirp(ast.Horizontal)}), none(&ast.Declarations{Rows: iptr(1)})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack := render(t, mk(tc.stack), Options{})
			grid := render(t, mk(tc.grid), Options{})
			if stack != grid {
				t.Errorf("%s: grid should match the stack byte-for-byte.\n--- stack ---\n%s\n--- grid ---\n%s", tc.name, stack, grid)
			}
		})
	}
}

func TestDirectionStackRoutesPlacementThroughGrid(t *testing.T) {
	// A child opting into grid placement on a `direction` parent routes the parent
	// through the grid engine, so explicit col/row are honored (sparse placement
	// "for free") — and the result is identical to spelling the arrangement with
	// explicit tracks. Here b (col 1) sits left of a (col 2), reversing source order.
	mk := func(parent *ast.Declarations) *ast.Document {
		return &ast.Document{
			Nodes: []*ast.ContainerNode{{ID: "p", Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("Alpha")}, {ID: "b", Label: ptr("Beta")},
			}}},
			Layout: []ast.LayoutRule{
				rule("p", parent),
				rule("p.a", &ast.Declarations{Col: iptr(2), Row: iptr(1)}),
				rule("p.b", &ast.Declarations{Col: iptr(1), Row: iptr(1)}),
			},
		}
	}
	viaDirection := render(t, mk(&ast.Declarations{Direction: dirp(ast.Horizontal)}), Options{})
	viaGrid := render(t, mk(&ast.Declarations{Rows: iptr(1)}), Options{})
	if viaDirection != viaGrid {
		t.Errorf("direction:horizontal + placement should equal rows:1 + placement.\n--- direction ---\n%s\n--- grid ---\n%s", viaDirection, viaGrid)
	}
}

func TestGridEmptyTrackReservesSpacer(t *testing.T) {
	// A child skipping a row leaves the empty row reserved at a minimum size, so a
	// sparse placement shows a visible gap instead of collapsing. a is in row 1, b in
	// row 3; the empty row 2 (min height fontSize*2 = 24) pushes b down to y=72
	// instead of sitting flush below a (which would be ~y=40).
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{ID: "col", Children: []*ast.ContainerNode{
			{ID: "a", Label: ptr("A")}, {ID: "b", Label: ptr("B")},
		}}},
		Layout: []ast.LayoutRule{
			rule("col", &ast.Declarations{Cols: iptr(1)}),
			rule("col.a", &ast.Declarations{Col: iptr(1), Row: iptr(1)}),
			rule("col.b", &ast.Declarations{Col: iptr(1), Row: iptr(3)}),
		},
	}
	svg := render(t, doc, Options{})
	for _, want := range []string{
		`<rect x="8" y="8" width="24" height="24"`,  // a in row 1
		`<rect x="8" y="72" width="24" height="24"`, // b in row 3, past the empty row-2 spacer
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q (an empty row should reserve a spacer):\n%s", want, svg)
		}
	}
}

func TestGenerateMarginSpacing(t *testing.T) {
	// A bordered vertical parent insets each child by the child's margin; two
	// siblings are separated by the larger of their facing margins (collapsed,
	// not summed), so every channel is one uniform margin wide.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p",
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
				{ID: "b", Label: ptr("B")},
			},
		}},
		Layout: []ast.LayoutRule{
			rule("p", &ast.Declarations{Direction: dirp(ast.Vertical)}),
			rule("p.a", &ast.Declarations{Margin: fptr(10)}),
			rule("p.b", &ast.Declarations{Margin: fptr(10)}),
		},
	}
	svg := render(t, doc, Options{}) // 12px => leaf min 24x24
	for _, want := range []string{
		`width="45" height="79" viewBox="-0.5 -0.5 45 79">`, // 44x78 content (10+24+10+24+10) + 1px border overflow
		`<rect x="10" y="10" width="24" height="24"`,        // a inset by its margin
		`<rect x="10" y="44" width="24" height="24"`,        // b sits a 10px (collapsed) gap below a
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateChannelsCollapseToMax(t *testing.T) {
	// Along the stacking (main) axis, adjacent margins collapse to the larger of the
	// two, not their sum, and an edge child sets its own perimeter. p (vertical,
	// default margin) with a (margin 4) over b (margin 12): the gap is max(4,12)=12,
	// the top perimeter is a's 4 (a bottom=28 → b at y=40). Across the stack (cross
	// axis) the two share one column, so the unified grid model insets both by the
	// collapsed max margin (12) at the shared track perimeter — a grid keeps its
	// tracks aligned rather than insetting each child by its own cross margin.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p",
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
				{ID: "b", Label: ptr("B")},
			},
		}},
		Layout: []ast.LayoutRule{
			rule("p", &ast.Declarations{Direction: dirp(ast.Vertical)}),
			rule("p.a", &ast.Declarations{Margin: fptr(4)}),
			rule("p.b", &ast.Declarations{Margin: fptr(12)}),
		},
	}
	svg := render(t, doc, Options{})
	for _, want := range []string{
		`<rect x="12" y="4" width="24" height="24"`,  // a: main gap/perimeter per-child (y=4)
		`<rect x="12" y="40" width="24" height="24"`, // b: collapsed gap 12 → y=40
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateBoxNoneGroupPushesChildMarginsToWall(t *testing.T) {
	// A box:none group nested in a bordered parent is transparent: its children's
	// perimeter margins push out to the bordered wall, so they are inset just like
	// a normal child would be — even though the group itself has margin 0.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "outer",
			Children: []*ast.ContainerNode{{
				ID:       "grp",
				Children: []*ast.ContainerNode{{ID: "c", Label: ptr("C")}},
			}},
		}},
		Layout: []ast.LayoutRule{
			rule("outer", &ast.Declarations{Direction: dirp(ast.Vertical)}),
			rule("outer.grp", &ast.Declarations{Box: boxp(ast.BoxNone), Margin: fptr(0)}),
			rule("outer.grp.c", &ast.Declarations{Margin: fptr(8)}),
		},
	}
	svg := render(t, doc, Options{})
	// outer is bordered → a wall. grp is box:none (margin 0) but transparent, so
	// c's margin 8 insets it from outer's border: c at (8,8).
	if want := `<rect x="8" y="8" width="24" height="24"`; !strings.Contains(svg, want) {
		t.Errorf("expected c inset by its margin to the bordered wall %q:\n%s", want, svg)
	}
	if got := strings.Count(svg, "<rect"); got != 2 { // outer + c; grp draws none
		t.Errorf("got %d rects, want 2 (outer + c):\n%s", got, svg)
	}
}

func TestGenerateBoxNoneGroupDoesNotDoubleChannel(t *testing.T) {
	// A box:none row stacked with a normal sibling: the gap between the row's
	// children and the sibling is a single collapsed margin (8), not doubled. The
	// transparent group adds no perimeter that would stack with the sibling gap.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "outer",
			Children: []*ast.ContainerNode{
				{ID: "row", Children: []*ast.ContainerNode{{ID: "a", Label: ptr("A")}, {ID: "b", Label: ptr("B")}}},
				{ID: "sib", Label: ptr("S")},
			},
		}},
		Layout: []ast.LayoutRule{
			rule("outer", &ast.Declarations{Direction: dirp(ast.Vertical)}),
			rule("outer.row", &ast.Declarations{Box: boxp(ast.BoxNone), Margin: fptr(0), Direction: dirp(ast.Horizontal)}),
		},
	}
	svg := render(t, doc, Options{})
	// a (8,8) is 24 tall → bottom 32; sib sits one collapsed margin (8) below at
	// y=40, not 48 (no doubled channel from the transparent row's perimeter).
	for _, want := range []string{
		`<rect x="8" y="8" width="24" height="24"`, // a, inset to the wall
		`<rect x="8" y="40"`,                       // sib, 8 below the row
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateMarginZeroIsFlush(t *testing.T) {
	// margin:0 everywhere restores the old flush packing: children touch and
	// the parent is exactly their bounding box.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p",
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
				{ID: "b", Label: ptr("B")},
			},
		}},
		Layout: []ast.LayoutRule{
			rule("p", &ast.Declarations{Direction: dirp(ast.Vertical), Margin: fptr(0)}),
			rule("p.a", &ast.Declarations{Margin: fptr(0)}),
			rule("p.b", &ast.Declarations{Margin: fptr(0)}),
		},
	}
	svg := render(t, doc, Options{})
	for _, want := range []string{
		`<rect x="0" y="0" width="24" height="48"`,  // p is the exact bounding box
		`<rect x="0" y="0" width="24" height="24"`,  // a flush at the origin
		`<rect x="0" y="24" width="24" height="24"`, // b flush below a, no gap
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateBoxNoneDrawsNoRect(t *testing.T) {
	// A box:none container draws no rectangle of its own; its child still does,
	// and as an invisible parent it collapses the child's perimeter margin so
	// it adds no padding (canvas == the child's border box).
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p",
			Children: []*ast.ContainerNode{
				{ID: "c", Label: ptr("child")},
			},
		}},
		Layout: []ast.LayoutRule{
			rule("p", &ast.Declarations{Box: boxp(ast.BoxNone)}),
			rule("p.c", &ast.Declarations{Margin: fptr(8)}),
		},
	}
	svg := render(t, doc, Options{})
	if got := strings.Count(svg, "<rect"); got != 1 {
		t.Errorf("got %d rects, want 1 (only c; p is box:none):\n%s", got, svg)
	}
	if !strings.Contains(svg, `<rect x="0" y="0"`) {
		t.Errorf("child should sit flush at the origin (perimeter margin collapsed):\n%s", svg)
	}
	if !strings.Contains(svg, ">child</text>") {
		t.Errorf("child label missing:\n%s", svg)
	}
}

func TestGenerateGroupLabelReservesTopBand(t *testing.T) {
	// A bordered parent with a label and children reserves a top strip for the
	// label; children lay out below it (not under the label), and the label is
	// centred in the band rather than in the whole box.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p", Label: ptr("Group"),
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
				{ID: "b", Label: ptr("B")},
			},
		}},
		Layout: []ast.LayoutRule{rule("p", &ast.Declarations{Direction: dirp(ast.Vertical)})},
	}
	svg := render(t, doc, Options{}) // 12px: leaf 24, band max(round(14.4)+2,24)=24
	for _, want := range []string{
		`width="41" height="97" viewBox="-0.5 -0.5 41 97">`, // 40x96 content (24 band + 8 + 24 + 8 + 24 + 8) + 1px border overflow
		`<text x="20" y="12"`,                               // label centred in the top band
		`<rect x="8" y="32" width="24" height="24"`,         // a, an 8px margin below the band wall
		`<rect x="8" y="64" width="24" height="24"`,         // b below a
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateGroupLabelBottomBand(t *testing.T) {
	// label: bottom reserves the strip under the children: children sit at the
	// top, the label is centred in the bottom band.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p", Label: ptr("Group"),
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
				{ID: "b", Label: ptr("B")},
			},
		}},
		Layout: []ast.LayoutRule{rule("p", &ast.Declarations{Direction: dirp(ast.Vertical), LabelPos: lblp(ast.LabelBottom)})},
	}
	svg := render(t, doc, Options{})
	for _, want := range []string{
		`width="41" height="97" viewBox="-0.5 -0.5 41 97">`,
		`<rect x="8" y="8" width="24" height="24"`,  // a at the top (no top band)
		`<rect x="8" y="40" width="24" height="24"`, // b below a
		`<text x="20" y="84"`,                       // label centred in the bottom band (y in [72,96])
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateGroupLabelWidensBoxToFit(t *testing.T) {
	// When the label is wider than the children, the box widens to fit it and the
	// children stretch to the wider cross axis.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p", Label: ptr("A Very Wide Group Title"),
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
			},
		}},
		Layout: []ast.LayoutRule{rule("p", &ast.Declarations{Direction: dirp(ast.Vertical)})},
	}
	svg := render(t, doc, Options{})
	// label width = round(23*12*0.6)+2 = 168; child stretches to 168-2*8 = 152.
	for _, want := range []string{
		`width="168"`,
		`<rect x="8" y="32" width="152" height="24"`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateBoxNoneLabelReservesBand(t *testing.T) {
	// A box:none group with a label also reserves a band — centring the label
	// over the children would just let them obscure it. The group draws no
	// border and packs its child flush below the band (no perimeter of its own),
	// but the band space is still reserved and the label sits in it.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p", Label: ptr("G"),
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
			},
		}},
		Layout: []ast.LayoutRule{rule("p", &ast.Declarations{Box: boxp(ast.BoxNone)})},
	}
	svg := render(t, doc, Options{})                  // 12px: band 24, child 24
	if got := strings.Count(svg, "<rect"); got != 1 { // only a; p is box:none
		t.Errorf("got %d rects, want 1 (only a; p is box:none):\n%s", got, svg)
	}
	for _, want := range []string{
		`width="25" height="48.5" viewBox="-0.5 0 25 48.5">`, // band 24 + child 24; the border overflow grows the sides + bottom, not the band-only top
		`<text x="12" y="12"`,                                // label "G" in the top band
		`<rect x="0" y="24" width="24" height="24"`,          // child flush below the band
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateDocumentDefaultMargin(t *testing.T) {
	// A document default margin replaces the built-in 8 as the fallback channel,
	// so top-level siblings spread out by it instead of by 8.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{
			{ID: "a", Label: ptr("A")},
			{ID: "b", Label: ptr("B")},
		},
		DefaultMargin: fptr(20),
	}
	svg := render(t, doc, Options{}) // leaves 24x24; top-level horizontal pack
	for _, want := range []string{
		`width="69" height="25" viewBox="-0.5 -0.5 69 25">`, // 68x24 content (24 + 20 channel + 24) + 1px border overflow
		`<rect x="0" y="0" width="24" height="24"`,          // a
		`<rect x="44" y="0" width="24" height="24"`,         // b, a 20px channel after a
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateDocumentDefaultMarginNodeOverride(t *testing.T) {
	// A node's own margin still overrides the document default: inside a bordered
	// parent (default 20), a child that sets margin: 4 is inset by 4, not 20.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID:       "p",
			Children: []*ast.ContainerNode{{ID: "c", Label: ptr("C")}},
		}},
		Layout:        []ast.LayoutRule{rule("p.c", &ast.Declarations{Margin: fptr(4)})},
		DefaultMargin: fptr(20),
	}
	svg := render(t, doc, Options{})
	if want := `<rect x="4" y="4" width="24" height="24"`; !strings.Contains(svg, want) {
		t.Errorf("child should be inset by its own margin 4, not the default 20 %q:\n%s", want, svg)
	}
}

func TestArrowCardinalVertical(t *testing.T) {
	// Two leaves stacked vertically (default margin 8): the arrow leaves a's
	// south edge and enters b's north edge, not centre-to-centre.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{
			ID: "p",
			Children: []*ast.ContainerNode{
				{ID: "a", Label: ptr("A")},
				{ID: "b", Label: ptr("B")},
			},
		}},
		Layout: []ast.LayoutRule{rule("p", &ast.Declarations{Direction: dirp(ast.Vertical)})},
		Arrows: []ast.Arrow{{Source: "p.a", Target: "p.b"}},
	}
	svg := render(t, doc, Options{})
	// a box (8,8,24,24); b box (8,40,24,24) — collapsed 8px gap. south of a =
	// (20,32); north of b = (20,40).
	if want := `<line x1="20" y1="32" x2="20" y2="40"`; !strings.Contains(svg, want) {
		t.Errorf("expected N/S cardinal arrow %q:\n%s", want, svg)
	}
}

func TestArrowCardinalHorizontal(t *testing.T) {
	// Two top-level leaves pack left-to-right (default margin 8): a's east edge
	// to b's west edge.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{
			{ID: "a", Label: ptr("A")},
			{ID: "b", Label: ptr("B")},
		},
		Arrows: []ast.Arrow{{Source: "a", Target: "b"}},
	}
	svg := render(t, doc, Options{})
	// a box (0,0,24,24); b box (32,0,24,24) — collapsed 8px gap. east of a =
	// (24,12); west of b = (32,12).
	if want := `<line x1="24" y1="12" x2="32" y2="12"`; !strings.Contains(svg, want) {
		t.Errorf("expected E/W cardinal arrow %q:\n%s", want, svg)
	}
}

func TestArrowExplicitAnchorOverridesCardinal(t *testing.T) {
	// #center forces a centre-to-centre line; a named anchor uses its position.
	// A bare end still auto-routes, even when the other end names an anchor.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{
			{ID: "a", Label: ptr("A")},
			{ID: "b", Label: ptr("B"), Anchors: []string{"tag"}},
		},
		Layout: []ast.LayoutRule{rule("b", &ast.Declarations{Anchors: map[string][2]float64{"tag": {1.0, 0.0}}})},
		Arrows: []ast.Arrow{
			{Source: "a#center", Target: "b#center"},
			{Source: "a", Target: "b#tag"},
		},
	}
	svg := render(t, doc, Options{})
	// a (0,0,24,24) centre (12,12); b (32,0,24,24) centre (44,12); b#tag=[1,0]=(56,0).
	if want := `<line x1="12" y1="12" x2="44" y2="12"`; !strings.Contains(svg, want) {
		t.Errorf("expected centre-to-centre for #center %q:\n%s", want, svg)
	}
	if want := `<line x1="24" y1="12" x2="56" y2="0"`; !strings.Contains(svg, want) {
		t.Errorf("expected cardinal source to named-anchor target %q:\n%s", want, svg)
	}
}

func TestUnpositionedNamedAnchorDefaultsToCenter(t *testing.T) {
	// A declared anchor name with no layout position resolves to the centre.
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{
			{ID: "a", Label: ptr("A")},
			{ID: "b", Label: ptr("B"), Anchors: []string{"side"}},
		},
		Arrows: []ast.Arrow{{Source: "a#center", Target: "b#side"}},
	}
	svg := render(t, doc, Options{})
	// b (32,0,24,24) centre (44,12); a#center (12,12).
	if want := `<line x1="12" y1="12" x2="44" y2="12"`; !strings.Contains(svg, want) {
		t.Errorf("expected unpositioned anchor at centre %q:\n%s", want, svg)
	}
}

func TestGenerateSingleNode(t *testing.T) {
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr("Hi")}}}
	svg := render(t, doc, Options{}) // defaults: 12px Arial
	// "Hi" at 12px: width = max(round(2*12*0.6)+2, 24) = max(16, 24) = 24; square.
	for _, want := range []string{
		`width="25" height="25" viewBox="-0.5 -0.5 25 25">`, // 24x24 content + 1px border overflow
		`<rect x="0" y="0" width="24" height="24" fill="white" stroke="black" stroke-width="1" shape-rendering="crispEdges" />`,
		`<text x="12" y="12" text-anchor="middle" dominant-baseline="middle" font-family="Arial" font-size="12">Hi</text>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
	}
}

func TestGenerateViewBoxFitsBorderStroke(t *testing.T) {
	// A border stroke is centred on the box edge, so half its width sits outside
	// the border box. The viewBox grows by that half-width on each side so a
	// perimeter border renders full width instead of being clipped to half by the
	// SVG viewport. A 4px border ⇒ a 2px overflow ⇒ a -2 viewBox origin.
	bw := 4.0
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a", Label: ptr("A")}},
		Layout: []ast.LayoutRule{rule("a", &ast.Declarations{BorderWidth: &bw})},
	}
	svg := render(t, doc, Options{})
	if want := `viewBox="-2 -2 `; !strings.Contains(svg, want) {
		t.Errorf("a 4px border should offset the viewBox by -2 on each axis, got:\n%s", svg)
	}
}

func TestGenerateStyledNode(t *testing.T) {
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr("A")}},
		Layout: []ast.LayoutRule{rule("a", &ast.Declarations{
			BorderColor: ptr("#ff0000"), BackgroundColor: ptr("#eeeeee"), BorderWidth: fptr(3),
		})},
	}
	svg := render(t, doc, Options{})
	if want := `fill="#eeeeee" stroke="#ff0000" stroke-width="3" shape-rendering="crispEdges"`; !strings.Contains(svg, want) {
		t.Errorf("SVG missing styled rect %q:\n%s", want, svg)
	}
}

func TestGeneratePathStyling(t *testing.T) {
	// pathColor/pathWidth come from the arrow's SOURCE node (a), so styling a
	// node restyles the arrows that start there — and the arrowhead colour matches.
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a", Label: ptr("A")}, {ID: "b", Label: ptr("B")}},
		Arrows: []ast.Arrow{{Source: "a", Target: "b"}},
		Layout: []ast.LayoutRule{rule("a", &ast.Declarations{PathColor: ptr("#0066ff"), PathWidth: fptr(2)})},
	}
	svg := render(t, doc, Options{})
	if !strings.Contains(svg, `stroke="#0066ff" stroke-width="2"`) {
		t.Errorf("arrow not styled from its source node:\n%s", svg)
	}
	if !strings.Contains(svg, `marker-end="url(#arrowhead-0066ff)"`) {
		t.Errorf("arrow missing colour-matched marker reference:\n%s", svg)
	}
	if !strings.Contains(svg, `<marker id="arrowhead-0066ff"`) || !strings.Contains(svg, `<polygon points="0 0, 10 3.5, 0 7" fill="#0066ff" />`) {
		t.Errorf("defs missing the coloured arrowhead marker:\n%s", svg)
	}
}

func TestGenerateDocumentDefaultStyle(t *testing.T) {
	// The document default styles every node that sets none; a node's own style
	// overrides it.
	doc := &ast.Document{
		Nodes:    []*ast.ContainerNode{{ID: "a", Label: ptr("A")}, {ID: "b", Label: ptr("B")}},
		Defaults: &ast.Declarations{BorderColor: ptr("#333333"), BackgroundColor: ptr("#fafafa")},
		Layout:   []ast.LayoutRule{rule("b", &ast.Declarations{BorderColor: ptr("#cc0000")})},
	}
	svg := render(t, doc, Options{})
	if !strings.Contains(svg, `fill="#fafafa" stroke="#333333"`) {
		t.Errorf("document default style not applied to a:\n%s", svg)
	}
	if !strings.Contains(svg, `fill="#fafafa" stroke="#cc0000"`) {
		t.Errorf("node b should override only borderColor, keeping the default fill:\n%s", svg)
	}
}

func TestGenerateMultilineLabel(t *testing.T) {
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr("a\nb")}}}
	svg := render(t, doc, Options{FontSize: 10})
	if got := strings.Count(svg, "<tspan"); got != 2 {
		t.Errorf("got %d tspans, want 2:\n%s", got, svg)
	}
}

func TestGenerateEscapesXML(t *testing.T) {
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr(`x & <y>`)}}}
	svg := render(t, doc, Options{})
	if !strings.Contains(svg, "x &amp; &lt;y&gt;") {
		t.Errorf("XML not escaped:\n%s", svg)
	}
}
