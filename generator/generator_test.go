package generator

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
	"github.com/kurrik/arkitecture/resolve"
)

func ptr(s string) *string                { return &s }
func fptr(f float64) *float64             { return &f }
func dirp(d ast.Direction) *ast.Direction { return &d }
func boxp(b ast.Box) *ast.Box             { return &b }

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

func TestGenerateMarginSpacing(t *testing.T) {
	// A bordered vertical parent insets each child by the child's margin and
	// separates two siblings by the sum of their facing margins; the parent
	// grows to contain the children's margin boxes.
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
		`width="44" height="88">`,                    // p contains both 24x24 margin boxes
		`<rect x="10" y="10" width="24" height="24"`, // a inset by its margin
		`<rect x="10" y="54" width="24" height="24"`, // b sits a 20px (10+10) gap below a
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
	// a box (8,8,24,24); b box (8,48,24,24). south of a = (20,32); north of b = (20,48).
	if want := `<line x1="20" y1="32" x2="20" y2="48"`; !strings.Contains(svg, want) {
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
	// a box (0,0,24,24); b box (40,0,24,24). east of a = (24,12); west of b = (40,12).
	if want := `<line x1="24" y1="12" x2="40" y2="12"`; !strings.Contains(svg, want) {
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
	// a (0,0,24,24) centre (12,12); b (40,0,24,24) centre (52,12); b#tag=[1,0]=(64,0).
	if want := `<line x1="12" y1="12" x2="52" y2="12"`; !strings.Contains(svg, want) {
		t.Errorf("expected centre-to-centre for #center %q:\n%s", want, svg)
	}
	if want := `<line x1="24" y1="12" x2="64" y2="0"`; !strings.Contains(svg, want) {
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
	// b (40,0,24,24) centre (52,12); a#center (12,12).
	if want := `<line x1="12" y1="12" x2="52" y2="12"`; !strings.Contains(svg, want) {
		t.Errorf("expected unpositioned anchor at centre %q:\n%s", want, svg)
	}
}

func TestGenerateSingleNode(t *testing.T) {
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr("Hi")}}}
	svg := render(t, doc, Options{}) // defaults: 12px Arial
	// "Hi" at 12px: width = max(round(2*12*0.6)+2, 24) = max(16, 24) = 24; square.
	for _, want := range []string{
		`width="24" height="24">`,
		`<rect x="0" y="0" width="24" height="24" fill="white" stroke="black" stroke-width="1" />`,
		`<text x="12" y="12" text-anchor="middle" dominant-baseline="middle" font-family="Arial" font-size="12">Hi</text>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q:\n%s", want, svg)
		}
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
