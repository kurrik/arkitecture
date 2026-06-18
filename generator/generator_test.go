package generator

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func ptr(s string) *string    { return &s }
func fptr(f float64) *float64 { return &f }

func TestGenerateMarginSpacing(t *testing.T) {
	// A bordered vertical parent insets each child by the child's margin and
	// separates two siblings by the sum of their facing margins; the parent
	// grows to contain the children's margin boxes.
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{
		ID:        "p",
		Direction: ast.Vertical,
		Children: []ast.Node{
			&ast.ContainerNode{ID: "a", Label: ptr("A"), Margin: fptr(10)},
			&ast.ContainerNode{ID: "b", Label: ptr("B"), Margin: fptr(10)},
		},
	}}}
	svg, _ := GenerateSVG(doc, Options{}) // 12px => leaf min 24x24
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
	zero := func(id, label string) *ast.ContainerNode {
		return &ast.ContainerNode{ID: id, Label: ptr(label), Margin: fptr(0)}
	}
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{
		ID: "p", Direction: ast.Vertical, Margin: fptr(0),
		Children: []ast.Node{zero("a", "A"), zero("b", "B")},
	}}}
	svg, _ := GenerateSVG(doc, Options{})
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
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{
		ID:  "p",
		Box: ast.BoxNone,
		Children: []ast.Node{
			&ast.ContainerNode{ID: "c", Label: ptr("child"), Margin: fptr(8)},
		},
	}}}
	svg, _ := GenerateSVG(doc, Options{})
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

func TestGenerateSingleNode(t *testing.T) {
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr("Hi")}}}
	svg, errs := GenerateSVG(doc, Options{}) // defaults: 12px Arial
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
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
	svg, _ := GenerateSVG(doc, Options{FontSize: 10})
	if got := strings.Count(svg, "<tspan"); got != 2 {
		t.Errorf("got %d tspans, want 2:\n%s", got, svg)
	}
}

func TestGenerateEscapesXML(t *testing.T) {
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Label: ptr(`x & <y>`)}}}
	svg, _ := GenerateSVG(doc, Options{})
	if !strings.Contains(svg, "x &amp; &lt;y&gt;") {
		t.Errorf("XML not escaped:\n%s", svg)
	}
}

func TestGenerateGroupsAreInvisible(t *testing.T) {
	// A group renders nothing itself, but its child container still does.
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{
		ID: "p",
		Children: []ast.Node{
			&ast.GroupNode{Children: []ast.Node{
				&ast.ContainerNode{ID: "c", Label: ptr("child")},
			}},
		},
	}}}
	svg, _ := GenerateSVG(doc, Options{})
	if got := strings.Count(svg, "<rect"); got != 2 {
		t.Errorf("got %d rects, want 2 (p and c):\n%s", got, svg)
	}
	if !strings.Contains(svg, ">child</text>") {
		t.Errorf("child label missing:\n%s", svg)
	}
}
