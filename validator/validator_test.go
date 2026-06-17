package validator

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture/ast"
)

func TestValidateValidDocument(t *testing.T) {
	doc := &ast.Document{
		Nodes: []*ast.ContainerNode{
			{ID: "a"},
			{ID: "b", Anchors: map[string][2]float64{"top": {0.5, 0.0}}},
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
		Nodes:  []*ast.ContainerNode{{ID: "b", Anchors: map[string][2]float64{"top": {0.5, 0}}}},
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

func TestValidateGroupFlatteningAndNestedPaths(t *testing.T) {
	// c1 contains a group, which contains n3. The group is layout-only, so n3's
	// path is "c1.n3".
	c1 := &ast.ContainerNode{
		ID: "c1",
		Children: []ast.Node{
			&ast.GroupNode{Children: []ast.Node{
				&ast.ContainerNode{ID: "n3", Anchors: map[string][2]float64{"a1": {1.0, 0.0}}},
			}},
		},
	}
	doc := &ast.Document{
		Nodes:  []*ast.ContainerNode{c1},
		Arrows: []ast.Arrow{{Source: "c1", Target: "c1.n3#a1"}},
	}
	if errs := Validate(doc); len(errs) != 0 {
		t.Errorf("expected group-flattened path to resolve, got %+v", errs)
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
		parent := &ast.ContainerNode{ID: "p", Children: []ast.Node{
			&ast.ContainerNode{ID: "x"},
			&ast.ContainerNode{ID: "x"},
		}}
		doc := &ast.Document{Nodes: []*ast.ContainerNode{parent}}
		errs := Validate(doc)
		if !containsMsg(errs, "Duplicate node ID 'x' within p scope") {
			t.Errorf("expected p-scope duplicate error, got %+v", errs)
		}
	})
}

func TestValidateConstraints(t *testing.T) {
	bigSize := 1.5
	doc := &ast.Document{Nodes: []*ast.ContainerNode{{
		ID:      "a",
		Size:    &bigSize,
		Anchors: map[string][2]float64{"bad": {2.0, -1.0}},
	}}}
	errs := Validate(doc)
	if !containsMsg(errs, "Node 'a' size 1.5 is out of range") {
		t.Errorf("expected size constraint error, got %+v", errs)
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

func containsMsg(errs []ast.Error, sub string) bool {
	for _, e := range errs {
		if strings.Contains(e.Message, sub) {
			return true
		}
	}
	return false
}
