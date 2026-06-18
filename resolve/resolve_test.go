package resolve

import (
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func dirp(d ast.Direction) *ast.Direction { return &d }
func fptr(f float64) *float64             { return &f }

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
			{Selector: "a", Decls: &ast.Declarations{Size: fptr(0.5), Anchors: map[string][2]float64{"n": {0.5, 0}}}},
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
	if a.Size == nil || *a.Size != 0.5 {
		t.Errorf("a.Size = %v, want 0.5", a.Size)
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
			{Selector: "a", Decls: &ast.Declarations{Size: fptr(0.5)}},
			{Selector: "a", Decls: &ast.Declarations{Size: fptr(0.9)}},
		},
	}
	if a := Resolve(doc)["a"]; a.Size == nil || *a.Size != 0.9 {
		t.Errorf("a.Size = %v, want 0.9 (last write)", a.Size)
	}
}
