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
	doc := parseOK(t, `a { label: "x" @layout { direction: horizontal; margin: 12; box: none } }`)
	rule := ruleFor(doc, "a")
	if rule == nil {
		t.Fatalf("no inline layout rule for 'a': %+v", doc.Layout)
	}
	d := rule.Decls
	if d.Direction == nil || *d.Direction != ast.Horizontal {
		t.Errorf("direction = %v, want horizontal", d.Direction)
	}
	if d.Margin == nil || *d.Margin != 12 {
		t.Errorf("margin = %v, want 12", d.Margin)
	}
	if d.Box == nil || *d.Box != ast.BoxNone {
		t.Errorf("box = %v, want none", d.Box)
	}
}

func TestParseLabelPosition(t *testing.T) {
	doc := parseOK(t, "a { c {} }\n@layout { a { label: top }\n  a { label: bottom } }")
	// Two rules on 'a'; the parser keeps both (the validator rejects the conflict).
	var got []ast.LabelPosition
	for i := range doc.Layout {
		if doc.Layout[i].Selector == "a" && doc.Layout[i].Decls.LabelPos != nil {
			got = append(got, *doc.Layout[i].Decls.LabelPos)
		}
	}
	if len(got) != 2 || got[0] != ast.LabelTop || got[1] != ast.LabelBottom {
		t.Errorf("label positions = %v, want [top bottom]", got)
	}
}

func TestParseDocumentDefaultMargin(t *testing.T) {
	doc := parseOK(t, "a {}\n@layout {\n  margin: 20\n  a { margin: 4 }\n}")
	if doc.DefaultMargin == nil || *doc.DefaultMargin != 20 {
		t.Errorf("DefaultMargin = %v, want 20", doc.DefaultMargin)
	}
	// A selector with a `margin:` is still a per-node margin, not a default.
	if r := ruleFor(doc, "a"); r == nil || r.Decls.Margin == nil || *r.Decls.Margin != 4 {
		t.Errorf("a rule = %+v, want margin 4", r)
	}
}

func TestParseStyleProperties(t *testing.T) {
	doc := parseOK(t, `a {}`+"\n@layout { a { borderWidth: 2; borderColor: #ff0000; backgroundColor: #00ff00; pathWidth: 3; pathColor: #0000ff } }")
	d := ruleFor(doc, "a").Decls
	if d.BorderWidth == nil || *d.BorderWidth != 2 {
		t.Errorf("borderWidth = %v, want 2", d.BorderWidth)
	}
	if d.BorderColor == nil || *d.BorderColor != "#ff0000" {
		t.Errorf("borderColor = %v, want #ff0000", d.BorderColor)
	}
	if d.BackgroundColor == nil || *d.BackgroundColor != "#00ff00" {
		t.Errorf("backgroundColor = %v, want #00ff00", d.BackgroundColor)
	}
	if d.PathWidth == nil || *d.PathWidth != 3 {
		t.Errorf("pathWidth = %v, want 3", d.PathWidth)
	}
	if d.PathColor == nil || *d.PathColor != "#0000ff" {
		t.Errorf("pathColor = %v, want #0000ff", d.PathColor)
	}
}

func TestParseStyleDocumentDefaults(t *testing.T) {
	doc := parseOK(t, `a {}`+"\n@layout {\n  borderColor: #111111\n  backgroundColor: #eeeeee\n  pathWidth: 2\n}")
	if doc.Defaults == nil {
		t.Fatal("Defaults is nil, want style document defaults")
	}
	if doc.Defaults.BorderColor == nil || *doc.Defaults.BorderColor != "#111111" {
		t.Errorf("default borderColor = %v, want #111111", doc.Defaults.BorderColor)
	}
	if doc.Defaults.BackgroundColor == nil || *doc.Defaults.BackgroundColor != "#eeeeee" {
		t.Errorf("default backgroundColor = %v, want #eeeeee", doc.Defaults.BackgroundColor)
	}
	if doc.Defaults.PathWidth == nil || *doc.Defaults.PathWidth != 2 {
		t.Errorf("default pathWidth = %v, want 2", doc.Defaults.PathWidth)
	}
}

func TestParseStyleInBlock(t *testing.T) {
	doc := parseOK(t, `a {}`+"\n@layout { @block warn { borderColor: #cc0000; backgroundColor: #fff0f0 }\n  a { @use warn } }")
	if len(doc.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(doc.Blocks))
	}
	b := doc.Blocks[0]
	if b.Name != "warn" || b.Decls.BorderColor == nil || *b.Decls.BorderColor != "#cc0000" {
		t.Errorf("block = %+v, want warn with borderColor #cc0000", b)
	}
}

func TestParseDocumentRoute(t *testing.T) {
	for _, tt := range []struct {
		src  string
		want ast.RouteMode
	}{
		{"a {}\n@layout {\n  route: orthogonal\n}", ast.RouteOrthogonal},
		{"a {}\n@layout {\n  route: straight\n}", ast.RouteStraight},
	} {
		doc := parseOK(t, tt.src)
		if doc.Route == nil || *doc.Route != tt.want {
			t.Errorf("Route = %v, want %v", doc.Route, tt.want)
		}
	}

	// Unset by default.
	if doc := parseOK(t, "a {}"); doc.Route != nil {
		t.Errorf("Route = %v, want nil when unset", doc.Route)
	}

	// A node literally named `route` is still a selector, not the document setting.
	doc := parseOK(t, "route {}\n@layout {\n  route { direction: horizontal }\n}")
	if doc.Route != nil {
		t.Errorf("Route = %v, want nil for a `route` selector", doc.Route)
	}
	if r := ruleFor(doc, "route"); r == nil || r.Decls.Direction == nil || *r.Decls.Direction != ast.Horizontal {
		t.Errorf("route rule = %+v, want direction horizontal", r)
	}
}

func TestParseLayoutSheet(t *testing.T) {
	doc := parseOK(t, "p { c {} }\n@layout {\n  p { direction: vertical }\n  p.c { margin: 0.5 }\n}")
	if len(doc.Layout) != 2 {
		t.Fatalf("got %d rules, want 2: %+v", len(doc.Layout), doc.Layout)
	}
	if r := ruleFor(doc, "p"); r == nil || r.Decls.Direction == nil || *r.Decls.Direction != ast.Vertical {
		t.Errorf("p rule = %+v, want direction vertical", r)
	}
	if r := ruleFor(doc, "p.c"); r == nil || r.Decls.Margin == nil || *r.Decls.Margin != 0.5 {
		t.Errorf("p.c rule = %+v, want margin 0.5", r)
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

func TestParseArrowsInterleaved(t *testing.T) {
	// Arrows may appear anywhere at the top level — before an @layout sheet,
	// between node definitions, and may forward-reference a not-yet-defined node.
	doc := parseOK(t, `a { anchors: [out] }
a#out --> b#in
@layout { a { anchor out: [1.0, 0.5] } }
b { anchors: [in] }
b --> a`)
	if len(doc.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(doc.Nodes))
	}
	if len(doc.Layout) != 1 {
		t.Errorf("got %d layout rules, want 1", len(doc.Layout))
	}
	if len(doc.Arrows) != 2 {
		t.Fatalf("got %d arrows, want 2: %+v", len(doc.Arrows), doc.Arrows)
	}
	if doc.Arrows[0] != (ast.Arrow{Source: "a#out", Target: "b#in"}) {
		t.Errorf("arrow[0] = %+v, want a#out --> b#in", doc.Arrows[0])
	}
	if doc.Arrows[1] != (ast.Arrow{Source: "b", Target: "a"}) {
		t.Errorf("arrow[1] = %+v, want b --> a", doc.Arrows[1])
	}
}

func TestParseFullExample(t *testing.T) {
	const example = `c1 {
  label: "Container 1"

  n2 {
    label: "Node 1"
    @layout { margin: 0.5 }
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

func TestParseBlockAndUse(t *testing.T) {
	doc := parseOK(t, `s { web {} api {} }
@layout {
  @block half { margin: 0.5 }
  s.web { @use half }
  s.api { @use half; margin: 0.75 }
}`)

	if len(doc.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1: %+v", len(doc.Blocks), doc.Blocks)
	}
	b := doc.Blocks[0]
	if b.Name != "half" {
		t.Errorf("block name = %q, want half", b.Name)
	}
	if b.Decls == nil || b.Decls.Margin == nil || *b.Decls.Margin != 0.5 {
		t.Errorf("block decls = %+v, want margin 0.5", b.Decls)
	}

	web := ruleFor(doc, "s.web")
	if web == nil || len(web.Uses) != 1 || web.Uses[0].Block != "half" {
		t.Fatalf("s.web rule = %+v, want one @use half", web)
	}

	api := ruleFor(doc, "s.api")
	if api == nil || len(api.Uses) != 1 || api.Uses[0].Block != "half" {
		t.Fatalf("s.api rule = %+v, want one @use half", api)
	}
	if api.Decls == nil || api.Decls.Margin == nil || *api.Decls.Margin != 0.75 {
		t.Errorf("s.api direct margin = %+v, want 0.75", api.Decls)
	}
}

func TestParseInlineUse(t *testing.T) {
	doc := parseOK(t, `a { @layout { @use service; margin: 0.5 } }`)
	r := ruleFor(doc, "a")
	if r == nil || len(r.Uses) != 1 || r.Uses[0].Block != "service" {
		t.Fatalf("inline rule = %+v, want one @use service", r)
	}
	if r.Decls == nil || r.Decls.Margin == nil || *r.Decls.Margin != 0.5 {
		t.Errorf("inline direct margin = %+v, want 0.5", r.Decls)
	}
}

func TestParseBlockComposition(t *testing.T) {
	doc := parseOK(t, `@layout {
  @block base { margin: 4 }
  @block wide { @use base; borderWidth: 0.9 }
}`)
	if len(doc.Blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(doc.Blocks))
	}
	wide := doc.Blocks[1]
	if wide.Name != "wide" || len(wide.Uses) != 1 || wide.Uses[0].Block != "base" {
		t.Errorf("wide = %+v, want @use base", wide)
	}
}

func TestParseUseRecordsPosition(t *testing.T) {
	doc := parseOK(t, "a {}\n@layout {\n  a { @use svc }\n}")
	r := ruleFor(doc, "a")
	if r == nil || len(r.Uses) != 1 {
		t.Fatalf("rule = %+v, want one use", r)
	}
	if u := r.Uses[0]; u.Line != 3 || u.Column != 12 {
		t.Errorf("use position = %d,%d, want 3,12 (the block name token)", u.Line, u.Column)
	}
}

func TestParseArrangement(t *testing.T) {
	doc := parseOK(t, `services {
  @layout {
    direction: horizontal
    @group { direction: vertical; userService; orderService }
    payments
  }
}`)
	r := ruleFor(doc, "services")
	if r == nil {
		t.Fatal("no rule for services")
	}
	if r.Decls.Direction == nil || *r.Decls.Direction != ast.Horizontal {
		t.Errorf("direction = %v, want horizontal", r.Decls.Direction)
	}
	arr := r.Decls.Arrangement
	if len(arr) != 2 {
		t.Fatalf("arrangement len = %d, want 2: %+v", len(arr), arr)
	}
	g := arr[0].Group
	if g == nil {
		t.Fatalf("arr[0] should be a group: %+v", arr[0])
	}
	if g.Direction == nil || *g.Direction != ast.Vertical {
		t.Errorf("group direction = %v, want vertical", g.Direction)
	}
	if len(g.Arrangement) != 2 || g.Arrangement[0].ChildID != "userService" || g.Arrangement[1].ChildID != "orderService" {
		t.Errorf("group items = %+v, want [userService orderService]", g.Arrangement)
	}
	if arr[1].Group != nil || arr[1].ChildID != "payments" {
		t.Errorf("arr[1] = %+v, want child payments", arr[1])
	}
}

func TestParseNestedGroup(t *testing.T) {
	doc := parseOK(t, `n { @layout { @group { a; @group { b; c } } } }`)
	r := ruleFor(doc, "n")
	if r == nil || len(r.Decls.Arrangement) != 1 || r.Decls.Arrangement[0].Group == nil {
		t.Fatalf("expected one top-level group: %+v", r)
	}
	outer := r.Decls.Arrangement[0].Group
	if len(outer.Arrangement) != 2 || outer.Arrangement[0].ChildID != "a" {
		t.Fatalf("outer group items = %+v, want [a, <group>]", outer.Arrangement)
	}
	inner := outer.Arrangement[1].Group
	if inner == nil || len(inner.Arrangement) != 2 || inner.Arrangement[0].ChildID != "b" || inner.Arrangement[1].ChildID != "c" {
		t.Errorf("inner group = %+v, want [b c]", inner)
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
		{"non-number margin", `a { @layout { margin: "x" } }`, ast.ErrorSyntax, "Expected number value for margin"},
		{"bad box", `a { @layout { box: solid } }`, ast.ErrorSyntax, "Invalid box 'solid'"},
		{"bad label position", `a { @layout { label: middle } }`, ast.ErrorSyntax, "Invalid label position 'middle'"},
		{"duplicate label position", `a { @layout { label: top; label: bottom } }`, ast.ErrorSyntax, "Duplicate layout property 'label'"},
		{"unknown document default", `@layout { direction: vertical }`, ast.ErrorSyntax, "Unknown document default 'direction'"},
		{"non-number document default", `@layout { margin: wide }`, ast.ErrorSyntax, "Expected number value for margin"},
		{"duplicate document default", `@layout { margin: 8; margin: 9 }`, ast.ErrorSyntax, "Duplicate layout property 'margin'"},
		{"bad route", `@layout { route: curved }`, ast.ErrorSyntax, "Invalid route 'curved'"},
		{"non-identifier route", `@layout { route: 3 }`, ast.ErrorSyntax, "Expected 'straight' or 'orthogonal' for route"},
		{"duplicate route", `@layout { route: orthogonal; route: straight }`, ast.ErrorSyntax, "Duplicate document property 'route'"},
		{"unknown layout property", `a { @layout { foo: 1 } }`, ast.ErrorSyntax, "Unknown layout property 'foo'"},
		{"duplicate layout property", `a { @layout { margin: 0.5; margin: 0.6 } }`, ast.ErrorSyntax, "Duplicate layout property 'margin'"},
		{"style property on node", `a { borderColor: #ff0000 }`, ast.ErrorSyntax, "Layout property 'borderColor' must be set inside an @layout block"},
		{"non-colour border colour", `a { @layout { borderColor: 5 } }`, ast.ErrorSyntax, "Expected a hex colour (e.g. #ff0000) for borderColor"},
		{"non-number border width", `a { @layout { borderWidth: thick } }`, ast.ErrorSyntax, "Expected number value for borderWidth"},
		{"duplicate path colour", `a { @layout { pathColor: #fff; pathColor: #000 } }`, ast.ErrorSyntax, "Duplicate layout property 'pathColor'"},
		{"duplicate document default colour", `@layout { borderColor: #fff; borderColor: #000 }`, ast.ErrorSyntax, "Duplicate layout property 'borderColor'"},
		{"unknown top-level directive", `@block { }`, ast.ErrorSyntax, "Unknown directive '@block'"},
		{"block without name", `@layout { @block { margin: 0.5 } }`, ast.ErrorSyntax, "Expected block name after @block"},
		{"use without name", `a { @layout { @use } }`, ast.ErrorSyntax, "Expected block name after @use"},
		{"use at sheet top level", `@layout { @use svc }`, ast.ErrorSyntax, "Unknown directive '@use' inside @layout, expected @block or a selector"},
		{"unknown directive in block", `a { @layout { @nope x } }`, ast.ErrorSyntax, "Unknown directive '@nope' in layout block, expected @use"},
		{"use inside group", `a { @layout { @group { @use svc } } }`, ast.ErrorSyntax, "@use is not allowed inside @group"},
		{"group without brace", `a { @layout { @group x } }`, ast.ErrorSyntax, "Expected '{' after @group"},
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
		`a { @layout { margin: 1.5 } }`,
		`a { anchors: [c] }` + "\n@layout { a { anchor c: [2, 0] } }",
	} {
		if r := Parse(input); !r.Success {
			t.Errorf("Parse(%q) failed; range checks belong to the validator: %+v", input, r.Errors)
		}
	}
}
