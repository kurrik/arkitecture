package validator

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func dirp(d ast.Direction) *ast.Direction { return &d }

func TestValidateValidDocument(t *testing.T) {
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{
			{ID: "a"},
			{ID: "b", Anchors: []string{"top"}},
		},
		Layout: []ast.LayoutRule{
			{Selector: "b", Decls: &ast.Declarations{Anchors: map[string][2]float64{"top": {0.5, 0.0}}}},
		},
		Arrows: []ast.Arrow{
			{Source: "a", Target: "b#top"},
			{Source: "a", Target: "b#center"}, // implicit center anchor
			{Source: "a", Target: "b"},        // no anchor
		},
	}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("expected no errors, got %+v", errs)
	}
}

func TestValidateArrowReferences(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a"}},
		Arrows: []ast.Arrow{{Source: "a", Target: "nonexistent"}},
	}
	errs := Validate(doc)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %+v", len(errs), errs)
	}
	if errs[0].Type != ast.ErrorReference || errs[0].Line != 1 || errs[0].Column != 1 {
		t.Errorf("error = %+v, want reference at 1,1", errs[0])
	}
	if errs[0].Message != "Arrow target node 'nonexistent' does not exist" {
		t.Errorf("message = %q", errs[0].Message)
	}
}

func TestValidateMissingSourceAndAnchor(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "b", Anchors: []string{"top"}}},
		Arrows: []ast.Arrow{{Source: "ghost", Target: "b#missing"}},
	}
	errs := Validate(doc)
	if len(errs) != 2 {
		t.Fatalf("got %d errors, want 2: %+v", len(errs), errs)
	}
	if !containsMsg(errs, "Arrow source node 'ghost' does not exist") {
		t.Errorf("missing source-node error: %+v", errs)
	}
	if !containsMsg(errs, "Arrow target anchor 'missing' does not exist on node 'b'") {
		t.Errorf("missing target-anchor error: %+v", errs)
	}
}

func TestValidateNestedPaths(t *testing.T) {
	// c1 contains grp (box:none), which contains n3, so n3's path is "c1.grp.n3".
	c1 := &ast.ContainerNode{
		ID: "c1",
		Children: []*ast.ContainerNode{
			{ID: "grp", Children: []*ast.ContainerNode{
				{ID: "n3", Anchors: []string{"a1"}},
			}},
		},
	}
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{c1},
		Arrows: []ast.Arrow{{Source: "c1", Target: "c1.grp.n3#a1"}},
	}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("expected nested path to resolve, got %+v", errs)
	}
}

func TestValidateDuplicateIDs(t *testing.T) {
	t.Run("root scope", func(t *testing.T) {
		doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a"}, {ID: "a"}}}
		errs := Validate(doc)
		if !containsMsg(errs, "Duplicate node ID 'a' within root scope") {
			t.Errorf("expected root-scope duplicate error, got %+v", errs)
		}
	})

	t.Run("nested scope", func(t *testing.T) {
		parent := &ast.ContainerNode{ID: "p", Children: []*ast.ContainerNode{
			{ID: "x"},
			{ID: "x"},
		}}
		doc := &ast.Document{Nodes: []*ast.ContainerNode{parent}}
		errs := Validate(doc)
		if !containsMsg(errs, "Duplicate node ID 'x' within p scope") {
			t.Errorf("expected p-scope duplicate error, got %+v", errs)
		}
	})
}

func TestValidateLayoutConstraints(t *testing.T) {
	bigSize := 1.5
	negMargin := -2.0
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{ID: "a", Anchors: []string{"bad"}}},
		Layout: []ast.LayoutRule{{
			Selector: "a",
			Decls: &ast.Declarations{
				Size:    &bigSize,
				Margin:  &negMargin,
				Anchors: map[string][2]float64{"bad": {2.0, -1.0}},
			},
		}},
	}
	errs := Validate(doc)
	if !containsMsg(errs, "Node 'a' size 1.5 is out of range") {
		t.Errorf("expected size constraint error, got %+v", errs)
	}
	if !containsMsg(errs, "Node 'a' margin -2 is out of range") {
		t.Errorf("expected margin constraint error, got %+v", errs)
	}
	if !containsMsg(errs, "anchor 'bad' X coordinate 2 is out of range") {
		t.Errorf("expected X coordinate error, got %+v", errs)
	}
	if !containsMsg(errs, "anchor 'bad' Y coordinate -1 is out of range") {
		t.Errorf("expected Y coordinate error, got %+v", errs)
	}
	for _, e := range errs {
		if e.Type != ast.ErrorConstraint {
			t.Errorf("expected constraint type, got %+v", e)
		}
	}
}

func TestValidateDanglingSelector(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a"}},
		Layout: []ast.LayoutRule{{Selector: "ghost", Decls: &ast.Declarations{Direction: dirp(ast.Vertical)}, Line: 3, Column: 5}},
	}
	errs := Validate(doc)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %+v", len(errs), errs)
	}
	if errs[0].Type != ast.ErrorReference || errs[0].Line != 3 || errs[0].Column != 5 {
		t.Errorf("error = %+v, want reference at 3,5", errs[0])
	}
	if !strings.Contains(errs[0].Message, "Layout selector 'ghost' matches no node") {
		t.Errorf("message = %q", errs[0].Message)
	}
}

func TestValidateConflictingProperty(t *testing.T) {
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{{ID: "a"}},
		Layout: []ast.LayoutRule{
			{Selector: "a", Decls: &ast.Declarations{Direction: dirp(ast.Vertical)}},
			{Selector: "a", Decls: &ast.Declarations{Direction: dirp(ast.Horizontal)}},
		},
	}
	errs := Validate(doc)
	if !containsMsg(errs, "Conflicting layout property 'direction' on node 'a'") {
		t.Errorf("expected conflict error, got %+v", errs)
	}
}

func TestValidateAnchorPositionUndeclared(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a"}}, // no declared anchors
		Layout: []ast.LayoutRule{{Selector: "a", Decls: &ast.Declarations{Anchors: map[string][2]float64{"x": {0.5, 0.5}}}}},
	}
	errs := Validate(doc)
	if !containsMsg(errs, "Layout positions anchor 'x' not declared on node 'a'") {
		t.Errorf("expected undeclared-anchor error, got %+v", errs)
	}
}

func TestValidateUndefinedBlock(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a"}},
		Layout: []ast.LayoutRule{{Selector: "a", Uses: []ast.Use{{Block: "ghost", Line: 4, Column: 9}}}},
	}
	errs := Validate(doc)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %+v", len(errs), errs)
	}
	e := errs[0]
	if e.Type != ast.ErrorReference || e.Line != 4 || e.Column != 9 {
		t.Errorf("error = %+v, want reference at 4,9", e)
	}
	if !strings.Contains(e.Message, "Layout block 'ghost' is not defined") {
		t.Errorf("message = %q", e.Message)
	}
}

func TestValidateUndefinedBlockInComposition(t *testing.T) {
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "wide", Uses: []ast.Use{{Block: "missing"}}, Decls: &ast.Declarations{}}},
	}
	if !containsMsg(Validate(doc), "Layout block 'missing' is not defined") {
		t.Error("expected undefined-block error from block composition")
	}
}

func TestValidateUseOfBuiltinIsValid(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a"}},
		Layout: []ast.LayoutRule{{Selector: "a", Uses: []ast.Use{{Block: "invisible"}}}},
	}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("@use of a built-in should be valid, got %+v", errs)
	}
}

func TestValidateUserDefinedBlockIsValid(t *testing.T) {
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{{ID: "a"}},
		Blocks: []ast.Block{{Name: "wide", Decls: &ast.Declarations{Size: ptr(0.75)}}},
		Layout: []ast.LayoutRule{{Selector: "a", Uses: []ast.Use{{Block: "wide"}}}},
	}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("@use of a defined block should be valid, got %+v", errs)
	}
}

func TestValidateUnknownKindIsNotAnError(t *testing.T) {
	// kind is a semantic tag; an unknown one contributes no layout but is not
	// itself an error (unlike an explicit @use).
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{ID: "a", Kind: "database"}}}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("unknown kind should not error, got %+v", errs)
	}
}

func TestValidateBlockCycle(t *testing.T) {
	doc := &ast.Document{
		Blocks: []ast.Block{
			{Name: "a", Uses: []ast.Use{{Block: "b"}}, Line: 2, Column: 3},
			{Name: "b", Uses: []ast.Use{{Block: "a"}}, Line: 3, Column: 3},
		},
	}
	errs := Validate(doc)
	if !containsMsg(errs, "Layout block cycle detected") {
		t.Fatalf("expected a cycle error, got %+v", errs)
	}
	// Exactly one cycle is reported despite two back-edges being reachable.
	n := 0
	for _, e := range errs {
		if strings.Contains(e.Message, "cycle detected") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("got %d cycle errors, want 1: %+v", n, errs)
	}
}

func TestValidateBlockSelfCycle(t *testing.T) {
	doc := &ast.Document{
		Blocks: []ast.Block{{Name: "a", Uses: []ast.Use{{Block: "a"}}, Line: 5, Column: 3}},
	}
	errs := Validate(doc)
	if !containsMsg(errs, "Layout block cycle detected: a -> a") {
		t.Fatalf("expected self-cycle error, got %+v", errs)
	}
}

func TestValidateBlockCompositionNoCycle(t *testing.T) {
	// A diamond (wide and tall both use base) is not a cycle.
	doc := &ast.Document{
		Blocks: []ast.Block{
			{Name: "base", Decls: &ast.Declarations{Margin: ptr(4)}},
			{Name: "wide", Uses: []ast.Use{{Block: "base"}}},
			{Name: "tall", Uses: []ast.Use{{Block: "base"}}},
			{Name: "both", Uses: []ast.Use{{Block: "wide"}, {Block: "tall"}}},
		},
	}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("diamond composition should be valid, got %+v", errs)
	}
}

func ptr(f float64) *float64 { return &f }

func containsMsg(errs []ast.Error, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e.Message, sub) {
			return true
		}
	}
	return false
}
