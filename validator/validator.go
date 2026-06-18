// Package validator performs semantic checks over a parsed Document: duplicate
// IDs within a scope, layout selector/conflict/range checks, anchor-position
// declarations, and arrow/anchor reference resolution. It is non-fail-fast —
// every error is collected.
//
// Most diagnostics carry line/column 1,1 because the semantic AST does not
// record node positions; dangling-selector errors are the exception, reporting
// at the selector position the parser preserved on each layout rule.
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

	// A flat map of every node by its full dotted path resolves arrow and
	// selector references.
	for _, n := range doc.Nodes {
		v.buildNodeMap(n, "")
	}

	v.validateIDUniqueness(doc.Nodes, "")
	v.validateLayout(doc)
	v.validateArrowReferences(doc)
	v.validateAnchorReferences(doc)
	return v.errors
}

type validator struct {
	nodeMap map[string]*ast.ContainerNode
	errors  []ast.Error
}

func (v *validator) buildNodeMap(node *ast.ContainerNode, parentPath string) {
	full := node.ID
	if parentPath != "" {
		full = parentPath + "." + node.ID
	}
	v.nodeMap[full] = node
	for _, c := range node.Children {
		v.buildNodeMap(c, full)
	}
}

// validateIDUniqueness checks that node IDs are unique within a scope (a
// parent's direct children, or the document root).
func (v *validator) validateIDUniqueness(nodes []*ast.ContainerNode, parentPath string) {
	seen := make(map[string]bool)
	for _, n := range nodes {
		if seen[n.ID] {
			scope := parentPath
			if scope == "" {
				scope = "root"
			}
			v.addError(ast.ErrorReference, fmt.Sprintf("Duplicate node ID '%s' within %s scope", n.ID, scope))
		}
		seen[n.ID] = true
	}
	for _, n := range nodes {
		cur := n.ID
		if parentPath != "" {
			cur = parentPath + "." + n.ID
		}
		v.validateIDUniqueness(n.Children, cur)
	}
}

// validateLayout checks the layout rules: dangling selectors, duplicate direct
// properties on a node, out-of-range values, and anchor positions that name an
// anchor the node never declared.
func (v *validator) validateLayout(doc *ast.Document) {
	// Group rules by selector to detect duplicate direct properties on a node.
	bySelector := make(map[string][]*ast.Declarations)
	var order []string

	for _, rule := range doc.Layout {
		if _, ok := v.nodeMap[rule.Selector]; !ok {
			v.errors = append(v.errors, ast.Error{
				Type:    ast.ErrorReference,
				Message: fmt.Sprintf("Layout selector '%s' matches no node", rule.Selector),
				Line:    rule.Line,
				Column:  rule.Column,
			})
			continue
		}
		if _, seen := bySelector[rule.Selector]; !seen {
			order = append(order, rule.Selector)
		}
		bySelector[rule.Selector] = append(bySelector[rule.Selector], rule.Decls)

		v.validateDeclRanges(rule.Selector, rule.Decls)
		v.validateAnchorPositions(rule.Selector, rule.Decls)
	}

	for _, sel := range order {
		v.validateNoConflicts(sel, bySelector[sel])
	}
}

// validateNoConflicts reports a property set by more than one direct rule on the
// same node. (A property repeated inside a single block is caught by the parser.)
func (v *validator) validateNoConflicts(selector string, decls []*ast.Declarations) {
	var direction, size, margin, box int
	anchors := map[string]int{}
	for _, d := range decls {
		if d == nil {
			continue
		}
		if d.Direction != nil {
			direction++
		}
		if d.Size != nil {
			size++
		}
		if d.Margin != nil {
			margin++
		}
		if d.Box != nil {
			box++
		}
		for name := range d.Anchors {
			anchors[name]++
		}
	}
	for _, c := range []struct {
		name  string
		count int
	}{{"direction", direction}, {"size", size}, {"margin", margin}, {"box", box}} {
		if c.count > 1 {
			v.addError(ast.ErrorReference, fmt.Sprintf("Conflicting layout property '%s' on node '%s'", c.name, selector))
		}
	}
	for _, name := range sortedIntKeys(anchors) {
		if anchors[name] > 1 {
			v.addError(ast.ErrorReference, fmt.Sprintf("Conflicting layout anchor '%s' on node '%s'", name, selector))
		}
	}
}

func (v *validator) validateDeclRanges(selector string, d *ast.Declarations) {
	if d == nil {
		return
	}
	if d.Size != nil && (*d.Size < 0.0 || *d.Size > 1.0) {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' size %s is out of range, expected 0.0-1.0", selector, formatNum(*d.Size)))
	}
	if d.Margin != nil && *d.Margin < 0.0 {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' margin %s is out of range, expected >= 0.0", selector, formatNum(*d.Margin)))
	}
	for _, name := range sortedCoordKeys(d.Anchors) {
		c := d.Anchors[name]
		if x := c[0]; x < 0.0 || x > 1.0 {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' anchor '%s' X coordinate %s is out of range, expected 0.0-1.0", selector, name, formatNum(x)))
		}
		if y := c[1]; y < 0.0 || y > 1.0 {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' anchor '%s' Y coordinate %s is out of range, expected 0.0-1.0", selector, name, formatNum(y)))
		}
	}
}

// validateAnchorPositions checks that every positioned anchor names a declared
// anchor on the target node — anchor *names* are semantic, so layout may only
// position names the node exposes.
func (v *validator) validateAnchorPositions(selector string, d *ast.Declarations) {
	if d == nil || len(d.Anchors) == 0 {
		return
	}
	node := v.nodeMap[selector]
	declared := make(map[string]bool, len(node.Anchors))
	for _, name := range node.Anchors {
		declared[name] = true
	}
	for _, name := range sortedCoordKeys(d.Anchors) {
		if !declared[name] {
			v.addError(ast.ErrorReference, fmt.Sprintf("Layout positions anchor '%s' not declared on node '%s'", name, selector))
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

// addError records a diagnostic at line 1, column 1 (the semantic AST tracks no
// node positions).
func (v *validator) addError(t ast.ErrorType, msg string) {
	v.errors = append(v.errors, ast.Error{Type: t, Message: msg, Line: 1, Column: 1})
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
// "center" anchor in addition to its declared names.
func hasAnchor(node *ast.ContainerNode, anchorID string) bool {
	if anchorID == "center" {
		return true
	}
	for _, name := range node.Anchors {
		if name == anchorID {
			return true
		}
	}
	return false
}

func sortedCoordKeys(m map[string][2]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedIntKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatNum(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
