package generator

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func ptr(s string) *string { return &s }

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
