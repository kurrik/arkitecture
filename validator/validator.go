// Package validator performs semantic checks over a parsed Document: duplicate
// IDs within a scope, arrow and anchor reference resolution, and range
// constraints. It is non-fail-fast — every error is collected.
//
// Diagnostics carry line/column 1,1 because the AST does not record node
// positions (the parser already reports syntactic positions). Adding positions
// to the AST is a possible future improvement.
package validator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kurrik/arkitecture/ast"
)

// Validate returns every semantic error in the document.
func Validate(doc *ast.Document) []ast.Error {
	if doc == nil {
		return nil
	}
	v := &validator{nodeMap: make(map[string]*ast.ContainerNode)}

	// A flat map of every container node by its full dotted path resolves arrow
	// references. Groups are layout-only and pass their parent path through, so
	// a node inside a group is keyed as if it were a direct child of the nearest
	// enclosing container.
	for _, n := range doc.Nodes {
		v.buildNodeMap(n, "")
	}

	v.validateIDUniqueness(toNodes(doc.Nodes), "")
	v.validateArrowReferences(doc)
	v.validateAnchorReferences(doc)
	for _, n := range doc.Nodes {
		v.validateNodeConstraints(n)
	}
	return v.errors
}

type validator struct {
	nodeMap map[string]*ast.ContainerNode
	errors  []ast.Error
}

func (v *validator) buildNodeMap(node ast.Node, parentPath string) {
	switch n := node.(type) {
	case *ast.ContainerNode:
		full := n.ID
		if parentPath != "" {
			full = parentPath + "." + n.ID
		}
		v.nodeMap[full] = n
		for _, c := range n.Children {
			v.buildNodeMap(c, full)
		}
	case *ast.GroupNode:
		for _, c := range n.Children {
			v.buildNodeMap(c, parentPath)
		}
	}
}

// validateIDUniqueness checks that container IDs are unique within a scope.
// Groups do not introduce a scope, so their children count at the enclosing
// container's level.
func (v *validator) validateIDUniqueness(nodes []ast.Node, parentPath string) {
	seen := make(map[string]bool)
	v.collectIDs(nodes, seen, parentPath)

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ContainerNode:
			cur := n.ID
			if parentPath != "" {
				cur = parentPath + "." + n.ID
			}
			v.validateIDUniqueness(n.Children, cur)
		case *ast.GroupNode:
			v.validateIDUniqueness(n.Children, parentPath)
		}
	}
}

func (v *validator) collectIDs(nodes []ast.Node, seen map[string]bool, parentPath string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ContainerNode:
			if seen[n.ID] {
				scope := parentPath
				if scope == "" {
					scope = "root"
				}
				v.addError(ast.ErrorReference, fmt.Sprintf("Duplicate node ID '%s' within %s scope", n.ID, scope))
			}
			seen[n.ID] = true
		case *ast.GroupNode:
			v.collectIDs(n.Children, seen, parentPath)
		}
	}
}

func (v *validator) validateArrowReferences(doc *ast.Document) {
	for _, a := range doc.Arrows {
		src := extractNodePath(a.Source)
		if _, ok := v.nodeMap[src]; !ok {
			v.addError(ast.ErrorReference, fmt.Sprintf("Arrow source node '%s' does not exist", src))
		}
		tgt := extractNodePath(a.Target)
		if _, ok := v.nodeMap[tgt]; !ok {
			v.addError(ast.ErrorReference, fmt.Sprintf("Arrow target node '%s' does not exist", tgt))
		}
	}
}

func (v *validator) validateAnchorReferences(doc *ast.Document) {
	for _, a := range doc.Arrows {
		if strings.Contains(a.Source, "#") {
			path, anchor := splitAnchor(a.Source)
			if node, ok := v.nodeMap[path]; ok && !hasAnchor(node, anchor) {
				v.addError(ast.ErrorReference, fmt.Sprintf("Arrow source anchor '%s' does not exist on node '%s'", anchor, path))
			}
		}
		if strings.Contains(a.Target, "#") {
			path, anchor := splitAnchor(a.Target)
			if node, ok := v.nodeMap[path]; ok && !hasAnchor(node, anchor) {
				v.addError(ast.ErrorReference, fmt.Sprintf("Arrow target anchor '%s' does not exist on node '%s'", anchor, path))
			}
		}
	}
}

func (v *validator) validateNodeConstraints(node ast.Node) {
	switch n := node.(type) {
	case *ast.ContainerNode:
		if n.Size != nil && (*n.Size < 0.0 || *n.Size > 1.0) {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' size %s is out of range, expected 0.0-1.0", n.ID, formatNum(*n.Size)))
		}
		// Iterate anchors in sorted order so diagnostics are deterministic.
		for _, anchorID := range sortedKeys(n.Anchors) {
			coord := n.Anchors[anchorID]
			if x := coord[0]; x < 0.0 || x > 1.0 {
				v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' anchor '%s' X coordinate %s is out of range, expected 0.0-1.0", n.ID, anchorID, formatNum(x)))
			}
			if y := coord[1]; y < 0.0 || y > 1.0 {
				v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' anchor '%s' Y coordinate %s is out of range, expected 0.0-1.0", n.ID, anchorID, formatNum(y)))
			}
		}
		for _, c := range n.Children {
			v.validateNodeConstraints(c)
		}
	case *ast.GroupNode:
		for _, c := range n.Children {
			v.validateNodeConstraints(c)
		}
	}
}

// addError records a diagnostic. The validator does not track positions, so all
// validator errors are reported at line 1, column 1.
func (v *validator) addError(t ast.ErrorType, msg string) {
	v.errors = append(v.errors, ast.Error{Type: t, Message: msg, Line: 1, Column: 1})
}

func toNodes(cs []*ast.ContainerNode) []ast.Node {
	out := make([]ast.Node, len(cs))
	for i, c := range cs {
		out[i] = c
	}
	return out
}

func extractNodePath(s string) string {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i]
	}
	return s
}

func splitAnchor(s string) (path, anchor string) {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// hasAnchor reports whether a node exposes anchorID. Every node has an implicit
// "center" anchor in addition to any explicit ones.
func hasAnchor(node *ast.ContainerNode, anchorID string) bool {
	if anchorID == "center" {
		return true
	}
	_, ok := node.Anchors[anchorID]
	return ok
}

func sortedKeys(m map[string][2]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatNum(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
