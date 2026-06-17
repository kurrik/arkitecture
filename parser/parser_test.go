package parser

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func parseOK(t *testing.T, input string) *ast.Document {
	t.Helper()
	r := Parse(input)
	if !r.Success {
		t.Fatalf("Parse(%q) failed unexpectedly: %+v", input, r.Errors)
	}
	if r.Document == nil {
		t.Fatalf("Parse(%q) returned nil document", input)
	}
	return r.Document
}

func TestParseEmptyAndComment(t *testing.T) {
	for _, input := range []string{"", "   \n  ", "# only a comment"} {
		doc := parseOK(t, input)
		if len(doc.Nodes) != 0 || len(doc.Arrows) != 0 {
			t.Errorf("Parse(%q) = %d nodes, %d arrows; want empty", input, len(doc.Nodes), len(doc.Arrows))
		}
	}
}

func TestParseSimpleNode(t *testing.T) {
	doc := parseOK(t, "a {}")
	if len(doc.Nodes) != 1 || doc.Nodes[0].ID != "a" {
		t.Fatalf("got %+v, want one node with id 'a'", doc.Nodes)
	}
	if doc.Nodes[0].Label != nil || doc.Nodes[0].Direction != ast.DirectionUnset {
		t.Errorf("expected unset label/direction, got %+v", doc.Nodes[0])
	}
}

func TestParseProperties(t *testing.T) {
	doc := parseOK(t, `a { label: "Hello" direction: "horizontal" size: 0.5 }`)
	n := doc.Nodes[0]
	if n.Label == nil || *n.Label != "Hello" {
		t.Errorf("label = %v, want \"Hello\"", n.Label)
	}
	if n.Direction != ast.Horizontal {
		t.Errorf("direction = %q, want horizontal", n.Direction)
	}
	if n.Size == nil || *n.Size != 0.5 {
		t.Errorf("size = %v, want 0.5", n.Size)
	}
}

func TestParseAnchors(t *testing.T) {
	doc := parseOK(t, `a { anchors: { top: [0.5, 0.0], c: [0.5, 0.5] } }`)
	n := doc.Nodes[0]
	if got := n.Anchors["top"]; got != [2]float64{0.5, 0.0} {
		t.Errorf("anchor top = %v, want [0.5 0]", got)
	}
	if got := n.Anchors["c"]; got != [2]float64{0.5, 0.5} {
		t.Errorf("anchor c = %v, want [0.5 0.5]", got)
	}
}

func TestParseNestedNodes(t *testing.T) {
	doc := parseOK(t, `p { label: "Parent" c { label: "Child" } }`)
	p := doc.Nodes[0]
	if len(p.Children) != 1 {
		t.Fatalf("parent has %d children, want 1", len(p.Children))
	}
	c, ok := p.Children[0].(*ast.ContainerNode)
	if !ok || c.ID != "c" {
		t.Fatalf("child = %+v, want container node 'c'", p.Children[0])
	}
}

func TestParseGroup(t *testing.T) {
	doc := parseOK(t, "p {\n  group {\n    direction: \"horizontal\"\n    c {}\n  }\n}")
	p := doc.Nodes[0]
	g, ok := p.Children[0].(*ast.GroupNode)
	if !ok {
		t.Fatalf("child[0] = %T, want *ast.GroupNode", p.Children[0])
	}
	if g.Direction != ast.Horizontal {
		t.Errorf("group direction = %q, want horizontal", g.Direction)
	}
	if len(g.Children) != 1 {
		t.Errorf("group has %d children, want 1", len(g.Children))
	}
}

func TestParseArrows(t *testing.T) {
	doc := parseOK(t, "a {}\nb {}\na --> b#top")
	if len(doc.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(doc.Nodes))
	}
	if len(doc.Arrows) != 1 {
		t.Fatalf("got %d arrows, want 1", len(doc.Arrows))
	}
	if want := (ast.Arrow{Source: "a", Target: "b#top"}); doc.Arrows[0] != want {
		t.Errorf("arrow = %+v, want %+v", doc.Arrows[0], want)
	}
}

func TestParseArrowWithPath(t *testing.T) {
	doc := parseOK(t, "p { c {} }\nq {}\np.c --> q#x")
	if want := (ast.Arrow{Source: "p.c", Target: "q#x"}); doc.Arrows[0] != want {
		t.Errorf("arrow = %+v, want %+v", doc.Arrows[0], want)
	}
}

func TestParseFullExample(t *testing.T) {
	const example = `c1 {
  label: "Container 1"
  direction: "vertical"
  n2 { label: "Node 1" size: 0.5 }
  group {
    direction: "horizontal"
    n3 {
      label: "Node 3"
      anchors: { a1: [1.0, 0.0], c: [0.5, 0.5] }
    }
  }
}
c1.n2 --> c1.n3#a1`

	doc := parseOK(t, example)
	if len(doc.Nodes) != 1 || len(doc.Arrows) != 1 {
		t.Fatalf("got %d nodes, %d arrows; want 1 and 1", len(doc.Nodes), len(doc.Arrows))
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantType        ast.ErrorType
		wantMsgContains string
	}{
		{"missing brace", "a", ast.ErrorSyntax, "Expected '{' after node id 'a'"},
		{"unknown property", `a { foo: 1 }`, ast.ErrorSyntax, "Unknown property 'foo'"},
		{"bad direction", `a { direction: "diagonal" }`, ast.ErrorSyntax, "Invalid direction 'diagonal'"},
		{"size out of range", `a { size: 1.5 }`, ast.ErrorConstraint, "Size value 1.5 is out of range"},
		{"coordinate out of range", `a { anchors: { c: [2, 0] } }`, ast.ErrorConstraint, "X coordinate 2 is out of range"},
		{"unterminated string", `a { label: "oops`, ast.ErrorSyntax, "Unterminated string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Parse(tt.input)
			if r.Success {
				t.Fatalf("Parse(%q) succeeded, want failure", tt.input)
			}
			if len(r.Errors) == 0 {
				t.Fatalf("Parse(%q) returned no errors", tt.input)
			}
			first := r.Errors[0]
			if first.Type != tt.wantType {
				t.Errorf("error type = %q, want %q", first.Type, tt.wantType)
			}
			if !strings.Contains(first.Message, tt.wantMsgContains) {
				t.Errorf("error message = %q, want it to contain %q", first.Message, tt.wantMsgContains)
			}
		})
	}
}
