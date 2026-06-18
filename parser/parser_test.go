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

// ruleFor returns the merged-by-index rule for selector, or nil. Tests that
// expect a single rule per selector use it directly.
func ruleFor(doc *ast.Document, selector string) *ast.LayoutRule {
	for i := range doc.Layout {
		if doc.Layout[i].Selector == selector {
			return &doc.Layout[i]
		}
	}
	return nil
}

func TestParseEmptyAndComment(t *testing.T) {
	for _, input := range []string{"", "   \n  ", "# only a comment"} {
		doc := parseOK(t, input)
		if len(doc.Nodes) != 0 || len(doc.Arrows) != 0 || len(doc.Layout) != 0 {
			t.Errorf("Parse(%q) = %d nodes, %d arrows, %d rules; want empty", input, len(doc.Nodes), len(doc.Arrows), len(doc.Layout))
		}
	}
}

func TestParseSimpleNode(t *testing.T) {
	doc := parseOK(t, "a {}")
	if len(doc.Nodes) != 1 || doc.Nodes[0].ID != "a" {
		t.Fatalf("got %+v, want one node with id 'a'", doc.Nodes)
	}
	if doc.Nodes[0].Label != nil || doc.Nodes[0].Kind != "" || doc.Nodes[0].Anchors != nil {
		t.Errorf("expected unset label/kind/anchors, got %+v", doc.Nodes[0])
	}
}

func TestParseSemanticProperties(t *testing.T) {
	doc := parseOK(t, `a { label: "Hello" kind: service anchors: [db, north] }`)
	n := doc.Nodes[0]
	if n.Label == nil || *n.Label != "Hello" {
		t.Errorf("label = %v, want \"Hello\"", n.Label)
	}
	if n.Kind != "service" {
		t.Errorf("kind = %q, want service", n.Kind)
	}
	if len(n.Anchors) != 2 || n.Anchors[0] != "db" || n.Anchors[1] != "north" {
		t.Errorf("anchors = %v, want [db north]", n.Anchors)
	}
}

func TestParseInlineLayout(t *testing.T) {
	doc := parseOK(t, `a { label: "x" @layout { direction: horizontal; size: 0.5; margin: 12; box: none } }`)
	rule := ruleFor(doc, "a")
	if rule == nil {
		t.Fatalf("no inline layout rule for 'a': %+v", doc.Layout)
	}
	d := rule.Decls
	if d.Direction == nil || *d.Direction != ast.Horizontal {
		t.Errorf("direction = %v, want horizontal", d.Direction)
	}
	if d.Size == nil || *d.Size != 0.5 {
		t.Errorf("size = %v, want 0.5", d.Size)
	}
	if d.Margin == nil || *d.Margin != 12 {
		t.Errorf("margin = %v, want 12", d.Margin)
	}
	if d.Box == nil || *d.Box != ast.BoxNone {
		t.Errorf("box = %v, want none", d.Box)
	}
}

func TestParseLayoutSheet(t *testing.T) {
	doc := parseOK(t, "p { c {} }\n@layout {\n  p { direction: vertical }\n  p.c { size: 0.5 }\n}")
	if len(doc.Layout) != 2 {
		t.Fatalf("got %d rules, want 2: %+v", len(doc.Layout), doc.Layout)
	}
	if r := ruleFor(doc, "p"); r == nil || r.Decls.Direction == nil || *r.Decls.Direction != ast.Vertical {
		t.Errorf("p rule = %+v, want direction vertical", r)
	}
	if r := ruleFor(doc, "p.c"); r == nil || r.Decls.Size == nil || *r.Decls.Size != 0.5 {
		t.Errorf("p.c rule = %+v, want size 0.5", r)
	}
}

func TestParseAnchorPosition(t *testing.T) {
	doc := parseOK(t, "a { anchors: [t, c] }\n@layout {\n  a { anchor t: [1.0, 0.0]; anchor c: [0.5, 0.5] }\n}")
	r := ruleFor(doc, "a")
	if r == nil {
		t.Fatal("no rule for 'a'")
	}
	if got := r.Decls.Anchors["t"]; got != [2]float64{1.0, 0.0} {
		t.Errorf("anchor t = %v, want [1 0]", got)
	}
	if got := r.Decls.Anchors["c"]; got != [2]float64{0.5, 0.5} {
		t.Errorf("anchor c = %v, want [0.5 0.5]", got)
	}
}

func TestParseNestedNodes(t *testing.T) {
	doc := parseOK(t, `p { label: "Parent" c { label: "Child" } }`)
	p := doc.Nodes[0]
	if len(p.Children) != 1 {
		t.Fatalf("parent has %d children, want 1", len(p.Children))
	}
	if c := p.Children[0]; c.ID != "c" {
		t.Fatalf("child = %+v, want container node 'c'", c)
	}
}

func TestParseArrows(t *testing.T) {
	doc := parseOK(t, "a {}\nb { anchors: [top] }\na --> b#top")
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
	doc := parseOK(t, "p { c {} }\nq { anchors: [x] }\np.c --> q#x")
	if want := (ast.Arrow{Source: "p.c", Target: "q#x"}); doc.Arrows[0] != want {
		t.Errorf("arrow = %+v, want %+v", doc.Arrows[0], want)
	}
}

func TestParseFullExample(t *testing.T) {
	const example = `c1 {
  label: "Container 1"

  n2 {
    label: "Node 1"
    @layout { size: 0.5 }
  }

  grp {
    n3 {
      label: "Node 3"
      anchors: [a1]
    }
  }
}

@layout {
  c1     { direction: vertical }
  c1.grp { box: none; direction: horizontal }
  c1.grp.n3 { anchor a1: [1.0, 0.0] }
}

c1.n2 --> c1.grp.n3#a1`

	doc := parseOK(t, example)
	if len(doc.Nodes) != 1 || len(doc.Arrows) != 1 {
		t.Fatalf("got %d nodes, %d arrows; want 1 and 1", len(doc.Nodes), len(doc.Arrows))
	}
	if len(doc.Layout) != 4 { // inline n2 + three sheet rules
		t.Errorf("got %d layout rules, want 4: %+v", len(doc.Layout), doc.Layout)
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
		{"layout property on node", `a { direction: vertical }`, ast.ErrorSyntax, "Layout property 'direction' must be set inside an @layout block"},
		{"bad direction", `a { @layout { direction: diagonal } }`, ast.ErrorSyntax, "Invalid direction 'diagonal'"},
		{"non-number size", `a { @layout { size: x } }`, ast.ErrorSyntax, "Expected number value for size"},
		{"non-number margin", `a { @layout { margin: "x" } }`, ast.ErrorSyntax, "Expected number value for margin"},
		{"bad box", `a { @layout { box: solid } }`, ast.ErrorSyntax, "Invalid box 'solid'"},
		{"unknown layout property", `a { @layout { foo: 1 } }`, ast.ErrorSyntax, "Unknown layout property 'foo'"},
		{"duplicate layout property", `a { @layout { size: 0.5; size: 0.6 } }`, ast.ErrorSyntax, "Duplicate layout property 'size'"},
		{"unknown directive", `@block { }`, ast.ErrorSyntax, "Unknown directive '@block'"},
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

// Out-of-range values now parse cleanly; the validator flags them.
func TestParseDoesNotRangeCheck(t *testing.T) {
	for _, input := range []string{
		`a { @layout { size: 1.5 } }`,
		`a { anchors: [c] }` + "\n@layout { a { anchor c: [2, 0] } }",
	} {
		if r := Parse(input); !r.Success {
			t.Errorf("Parse(%q) failed; range checks belong to the validator: %+v", input, r.Errors)
		}
	}
}
