package resolve

import (
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func dirp(d ast.Direction) *ast.Direction { return &d }
func fptr(f float64) *float64             { return &f }
func boxp(b ast.Box) *ast.Box             { return &b }

func TestResolveNil(t *testing.T) {
	if got := Resolve(nil); len(got) != 0 {
		t.Errorf("Resolve(nil) = %v, want empty", got)
	}
}

func TestResolveMergesRulesByPath(t *testing.T) {
	// Two rules on the same path (e.g. an inline block plus a sheet selector)
	// merge into one resolved declaration set; anchor positions union.
	doc := &ast.Document{
		Layout: []ast.LayoutRule{
			{Selector: "a", Decls: &ast.Declarations{Direction: dirp(ast.Horizontal)}},
			{Selector: "a", Decls: &ast.Declarations{Margin: fptr(0.5), Anchors: map[string][2]float64{"n": {0.5, 0}}}},
			{Selector: "a", Decls: &ast.Declarations{Anchors: map[string][2]float64{"s": {0.5, 1}}}},
			{Selector: "b", Decls: &ast.Declarations{Margin: fptr(0)}},
		},
	}
	got := Resolve(doc)

	a := got["a"]
	if a == nil {
		t.Fatal("no resolved layout for 'a'")
	}
	if a.Direction == nil || *a.Direction != ast.Horizontal {
		t.Errorf("a.Direction = %v, want horizontal", a.Direction)
	}
	if a.Margin == nil || *a.Margin != 0.5 {
		t.Errorf("a.Margin = %v, want 0.5", a.Margin)
	}
	if len(a.Anchors) != 2 || a.Anchors["n"] != ([2]float64{0.5, 0}) || a.Anchors["s"] != ([2]float64{0.5, 1}) {
		t.Errorf("a.Anchors = %v, want union of n and s", a.Anchors)
	}
	if b := got["b"]; b == nil || b.Margin == nil || *b.Margin != 0 {
		t.Errorf("b = %v, want margin 0", b)
	}
}

func TestResolveLastWriteWins(t *testing.T) {
	// The validator rejects conflicting direct properties, but if reached,
	// resolution is deterministic last-write-wins in source order.
	doc := &ast.Document{
		Layout: []ast.LayoutRule{
			{Selector: "a", Decls: &ast.Declarations{Margin: fptr(0.5)}},
			{Selector: "a", Decls: &ast.Declarations{Margin: fptr(0.9)}},
		},
	}
	if a := Resolve(doc)["a"]; a.Margin == nil || *a.Margin != 0.9 {
		t.Errorf("a.Margin = %v, want 0.9 (last write)", a.Margin)
	}
}

func TestResolveLabelPosition(t *testing.T) {
	// label position merges like any other scalar: a block supplies it via @use
	// (imported tier) and a direct rule overrides the import.
	bottom, top := ast.LabelBottom, ast.LabelTop
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "titled", Decls: &ast.Declarations{LabelPos: &bottom}}},
		Layout: []ast.LayoutRule{
			{Selector: "a", Uses: []ast.Use{{Block: "titled"}}},
			{Selector: "b", Uses: []ast.Use{{Block: "titled"}}, Decls: &ast.Declarations{LabelPos: &top}},
		},
	}
	got := Resolve(doc)
	if a := got["a"]; a == nil || a.LabelPos == nil || *a.LabelPos != ast.LabelBottom {
		t.Errorf("a.LabelPos = %v, want bottom (imported via @use)", a)
	}
	if b := got["b"]; b == nil || b.LabelPos == nil || *b.LabelPos != ast.LabelTop {
		t.Errorf("b.LabelPos = %v, want top (direct overrides import)", b)
	}
}

func TestResolveKindBaseline(t *testing.T) {
	// The built-in `invisible` kind applies box:none with no @block needed.
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "g", Kind: "invisible"}}}
	g := Resolve(doc)["g"]
	if g == nil || g.Box == nil || *g.Box != ast.BoxNone {
		t.Errorf("g = %+v, want box none from kind baseline", g)
	}
}

func TestResolveUnknownKindIsNoop(t *testing.T) {
	// An unknown kind is a semantic tag with no layout block: it contributes
	// nothing rather than erroring.
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "n", Kind: "database"}}}
	n := Resolve(doc)["n"]
	if n == nil {
		t.Fatal("expected an (empty) entry for the kind-bearing node")
	}
	if n.Box != nil || n.Direction != nil || n.Margin != nil {
		t.Errorf("unknown kind should contribute nothing, got %+v", n)
	}
}

func TestResolveUseImportsBlock(t *testing.T) {
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "wide", Decls: &ast.Declarations{Margin: fptr(0.75)}}},
		Layout: []ast.LayoutRule{{Selector: "a", Uses: []ast.Use{{Block: "wide"}}}},
	}
	if a := Resolve(doc)["a"]; a == nil || a.Margin == nil || *a.Margin != 0.75 {
		t.Errorf("a = %+v, want margin 0.75 imported from @use wide", a)
	}
}

func TestResolveDirectBeatsImported(t *testing.T) {
	// `@use wide` imports margin 0.75; the direct margin 0.5 overrides it.
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "wide", Decls: &ast.Declarations{Margin: fptr(0.75)}}},
		Layout: []ast.LayoutRule{{
			Selector: "a",
			Uses:     []ast.Use{{Block: "wide"}},
			Decls:    &ast.Declarations{Margin: fptr(0.5)},
		}},
	}
	if a := Resolve(doc)["a"]; a == nil || a.Margin == nil || *a.Margin != 0.5 {
		t.Errorf("a.Margin = %v, want 0.5 (direct beats imported)", a.Margin)
	}
}

func TestResolveStyleImportAndOverride(t *testing.T) {
	// A block sets a border colour/width baseline; a direct borderColor overrides
	// it while the imported borderWidth carries through (last-write-wins per field).
	red, blue := "#cc0000", "#0000ff"
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "warn", Decls: &ast.Declarations{BorderColor: &red, BorderWidth: fptr(3)}}},
		Layout: []ast.LayoutRule{{
			Selector: "a",
			Uses:     []ast.Use{{Block: "warn"}},
			Decls:    &ast.Declarations{BorderColor: &blue},
		}},
	}
	a := Resolve(doc)["a"]
	if a == nil || a.BorderColor == nil || *a.BorderColor != blue {
		t.Errorf("a.BorderColor = %v, want #0000ff (direct beats imported)", a.BorderColor)
	}
	if a == nil || a.BorderWidth == nil || *a.BorderWidth != 3 {
		t.Errorf("a.BorderWidth = %v, want 3 (imported, not overridden)", a.BorderWidth)
	}
}

func TestResolveKindBaselineLowestPrecedence(t *testing.T) {
	// kind: invisible would set box:none, but a direct box:default wins.
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "n", Kind: "invisible"}},
		Layout: []ast.LayoutRule{{Selector: "n", Decls: &ast.Declarations{Box: boxp(ast.BoxDefault)}}},
	}
	if n := Resolve(doc)["n"]; n == nil || n.Box == nil || *n.Box != ast.BoxDefault {
		t.Errorf("n.Box = %v, want default (direct overrides kind baseline)", n.Box)
	}
}

func TestResolveUseComposition(t *testing.T) {
	// wide @use base, then overrides base's borderWidth with its own.
	doc := &ast.Document{
		Blocks: []ast.Block{
			{Name: "base", Decls: &ast.Declarations{Margin: fptr(4), BorderWidth: fptr(0.2)}},
			{Name: "wide", Uses: []ast.Use{{Block: "base"}}, Decls: &ast.Declarations{BorderWidth: fptr(0.9)}},
		},
		Layout: []ast.LayoutRule{{Selector: "a", Uses: []ast.Use{{Block: "wide"}}}},
	}
	a := Resolve(doc)["a"]
	if a == nil || a.Margin == nil || *a.Margin != 4 {
		t.Errorf("a.Margin = %v, want 4 (composed from base)", a.Margin)
	}
	if a.BorderWidth == nil || *a.BorderWidth != 0.9 {
		t.Errorf("a.BorderWidth = %v, want 0.9 (wide's own decl beats base's)", a.BorderWidth)
	}
}

func TestResolveUseLastWins(t *testing.T) {
	// Within the imported tier, a later @use beats an earlier one.
	doc := &ast.Document{
		Blocks: []ast.Block{
			{Name: "small", Decls: &ast.Declarations{Margin: fptr(0.25)}},
			{Name: "big", Decls: &ast.Declarations{Margin: fptr(0.9)}},
		},
		Layout: []ast.LayoutRule{{Selector: "a", Uses: []ast.Use{{Block: "small"}, {Block: "big"}}}},
	}
	if a := Resolve(doc)["a"]; a == nil || a.Margin == nil || *a.Margin != 0.9 {
		t.Errorf("a.Margin = %v, want 0.9 (later @use wins)", a.Margin)
	}
}

func TestResolveCycleTerminates(t *testing.T) {
	// A composition cycle must not loop forever; the validator reports it
	// separately. Resolution just stops.
	doc := &ast.Document{
		Blocks: []ast.Block{
			{Name: "a", Uses: []ast.Use{{Block: "b"}}, Decls: &ast.Declarations{Margin: fptr(1)}},
			{Name: "b", Uses: []ast.Use{{Block: "a"}}, Decls: &ast.Declarations{BorderWidth: fptr(0.5)}},
		},
		Layout: []ast.LayoutRule{{Selector: "n", Uses: []ast.Use{{Block: "a"}}}},
	}
	n := Resolve(doc)["n"] // would hang if the cycle weren't guarded
	if n == nil || n.Margin == nil || n.BorderWidth == nil {
		t.Errorf("n = %+v, want both composed props before the cycle stops", n)
	}
}

func TestResolveArrangementIsDirect(t *testing.T) {
	doc := &ast.Document{
		Layout: []ast.LayoutRule{
			{Selector: "n", Decls: &ast.Declarations{
				Direction:   dirp(ast.Horizontal),
				Arrangement: []ast.ArrangementItem{{ChildID: "a"}, {ChildID: "b"}},
			}},
		},
	}
	n := Resolve(doc)["n"]
	if n == nil || len(n.Arrangement) != 2 {
		t.Fatalf("n.Arrangement = %+v, want 2 items", n)
	}
	if n.Arrangement[0].ChildID != "a" || n.Arrangement[1].ChildID != "b" {
		t.Errorf("arrangement = %+v, want [a b]", n.Arrangement)
	}
}

func TestResolveArrangementNotImportedFromBlock(t *testing.T) {
	// A @use imports a block's scalar properties but never its arrangement.
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "b", Decls: &ast.Declarations{
			Margin:      fptr(0.5),
			Arrangement: []ast.ArrangementItem{{ChildID: "x"}},
		}}},
		Layout: []ast.LayoutRule{{Selector: "n", Uses: []ast.Use{{Block: "b"}}}},
	}
	n := Resolve(doc)["n"]
	if n == nil || n.Margin == nil || *n.Margin != 0.5 {
		t.Fatalf("n = %+v, want margin 0.5 imported", n)
	}
	if len(n.Arrangement) != 0 {
		t.Errorf("arrangement must not be imported from a block, got %+v", n.Arrangement)
	}
}

func TestResolveUserBlockOverridesBuiltin(t *testing.T) {
	// A user @block invisible redefines the built-in kind.
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "n", Kind: "invisible"}},
		Blocks: []ast.Block{{Name: "invisible", Decls: &ast.Declarations{Box: boxp(ast.BoxDefault), Margin: fptr(0.3)}}},
	}
	n := Resolve(doc)["n"]
	if n == nil || n.Box == nil || *n.Box != ast.BoxDefault {
		t.Errorf("n.Box = %v, want default (user block overrode built-in)", n.Box)
	}
	if n.Margin == nil || *n.Margin != 0.3 {
		t.Errorf("n.Margin = %v, want 0.3 from the redefined block", n.Margin)
	}
}
