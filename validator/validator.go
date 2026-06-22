// Package validator performs semantic checks over a parsed Document: duplicate
// IDs within a scope, layout selector/conflict/range checks, anchor-position
// declarations, undefined `@use` blocks, `@use` composition cycles, and
// arrow/anchor reference resolution. It is non-fail-fast — every error is
// collected.
//
// Most diagnostics carry line/column 1,1 because the semantic AST does not
// record node positions; dangling-selector, `@use`, and block-cycle errors are
// the exception, reporting at the position the parser preserved.
//
// An unknown `kind` is deliberately not an error: `kind` is a semantic tag, so a
// missing layout block of that name simply contributes no baseline. An explicit
// `@use` of an undefined block is an error — that is a layout import the author
// asked for that cannot be satisfied.
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
	v.validateLayoutBlocks(doc)
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
// anchor the node never declared. It also range-checks the document default
// margin.
func (v *validator) validateLayout(doc *ast.Document) {
	if doc.DefaultMargin != nil && *doc.DefaultMargin < 0.0 {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Document default margin %s is out of range, expected >= 0.0", formatNum(*doc.DefaultMargin)))
	}
	if d := doc.Defaults; d != nil {
		if d.BorderWidth != nil && *d.BorderWidth < 0.0 {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Document default borderWidth %s is out of range, expected >= 0.0", formatNum(*d.BorderWidth)))
		}
		if d.PathWidth != nil && *d.PathWidth < 0.0 {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Document default pathWidth %s is out of range, expected >= 0.0", formatNum(*d.PathWidth)))
		}
		v.validateColor("Document default borderColor", d.BorderColor)
		v.validateColor("Document default backgroundColor", d.BackgroundColor)
		v.validateColor("Document default pathColor", d.PathColor)
	}

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
		v.validateArrangement(rule.Selector, rule.Decls)
	}

	for _, sel := range order {
		v.validateNoConflicts(sel, bySelector[sel])
	}

	// Grid placement: out-of-bounds cells (including span extents) and overlaps.
	for _, sel := range order {
		v.validateGrid(sel, bySelector)
	}
}

// validateGrid checks a grid node's child placements: a cell (or its colSpan/
// rowSpan extent) outside the declared bounds, or two cells overlapping, is a
// constraint error. It resolves the grid spec and each child's direct placement
// from the by-selector direct declarations.
func (v *validator) validateGrid(selector string, bySelector map[string][]*ast.Declarations) {
	spec := mergedGrid(bySelector[selector])
	if spec == nil {
		return
	}
	node := v.nodeMap[selector]
	if node == nil {
		return
	}
	cells := make([]ast.GridCell, 0, len(node.Children))
	for _, c := range node.Children {
		col, row, colSpan, rowSpan := mergedPlacement(bySelector[selector+"."+c.ID])
		cells = append(cells, ast.GridCell{ChildID: c.ID, Col: col, Row: row, ColSpan: colSpan, RowSpan: rowSpan})
	}
	_, _, problems := ast.PlaceGrid(*spec, cells)
	for _, p := range problems {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s': %s", selector, p.Message))
	}
}

// mergedGrid returns the grid spec for a node built from its direct `cols`/`rows`
// declarations, or nil when the node is not a grid (no `cols` set).
func mergedGrid(decls []*ast.Declarations) *ast.GridSpec {
	var spec *ast.GridSpec
	for _, d := range decls {
		if d == nil {
			continue
		}
		if d.Cols != nil && spec == nil {
			spec = &ast.GridSpec{Cols: *d.Cols}
		}
	}
	if spec == nil {
		return nil
	}
	for _, d := range decls {
		if d != nil && d.Rows != nil {
			spec.Rows = d.Rows
			break
		}
	}
	return spec
}

// mergedPlacement gathers a node's direct grid-placement properties across its rules.
func mergedPlacement(decls []*ast.Declarations) (col, row, colSpan, rowSpan int) {
	for _, d := range decls {
		if d == nil {
			continue
		}
		if d.Col != nil {
			col = *d.Col
		}
		if d.Row != nil {
			row = *d.Row
		}
		if d.ColSpan != nil {
			colSpan = *d.ColSpan
		}
		if d.RowSpan != nil {
			rowSpan = *d.RowSpan
		}
	}
	return
}

// validateNoConflicts reports a property set by more than one direct rule on the
// same node. (A property repeated inside a single block is caught by the parser.)
func (v *validator) validateNoConflicts(selector string, decls []*ast.Declarations) {
	var direction, margin, box, labelPos, arrangement int
	var borderWidth, borderColor, backgroundColor, pathWidth, pathColor int
	var cols, rows, col, row, colSpan, rowSpan, justify, align int
	anchors := map[string]int{}
	for _, d := range decls {
		if d == nil {
			continue
		}
		if d.Direction != nil {
			direction++
		}
		if d.Margin != nil {
			margin++
		}
		if d.Box != nil {
			box++
		}
		if d.LabelPos != nil {
			labelPos++
		}
		if len(d.Arrangement) > 0 {
			arrangement++
		}
		if d.BorderWidth != nil {
			borderWidth++
		}
		if d.BorderColor != nil {
			borderColor++
		}
		if d.BackgroundColor != nil {
			backgroundColor++
		}
		if d.PathWidth != nil {
			pathWidth++
		}
		if d.PathColor != nil {
			pathColor++
		}
		if d.Cols != nil {
			cols++
		}
		if d.Rows != nil {
			rows++
		}
		if d.Col != nil {
			col++
		}
		if d.Row != nil {
			row++
		}
		if d.ColSpan != nil {
			colSpan++
		}
		if d.RowSpan != nil {
			rowSpan++
		}
		if d.Justify != nil {
			justify++
		}
		if d.Align != nil {
			align++
		}
		for name := range d.Anchors {
			anchors[name]++
		}
	}
	for _, c := range []struct {
		name  string
		count int
	}{{"direction", direction}, {"margin", margin}, {"box", box}, {"label", labelPos}, {"arrangement", arrangement},
		{"borderWidth", borderWidth}, {"borderColor", borderColor}, {"backgroundColor", backgroundColor}, {"pathWidth", pathWidth}, {"pathColor", pathColor},
		{"cols", cols}, {"rows", rows}, {"col", col}, {"row", row}, {"colSpan", colSpan}, {"rowSpan", rowSpan}, {"justify", justify}, {"align", align}} {
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
	if d.Margin != nil && *d.Margin < 0.0 {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' margin %s is out of range, expected >= 0.0", selector, formatNum(*d.Margin)))
	}
	if d.BorderWidth != nil && *d.BorderWidth < 0.0 {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' borderWidth %s is out of range, expected >= 0.0", selector, formatNum(*d.BorderWidth)))
	}
	if d.PathWidth != nil && *d.PathWidth < 0.0 {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' pathWidth %s is out of range, expected >= 0.0", selector, formatNum(*d.PathWidth)))
	}
	for _, f := range []struct {
		name string
		p    *int
		min  int
	}{{"cols", d.Cols, 1}, {"rows", d.Rows, 1}, {"col", d.Col, 1}, {"row", d.Row, 1}, {"colSpan", d.ColSpan, 1}, {"rowSpan", d.RowSpan, 1}} {
		if f.p != nil && *f.p < f.min {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' %s %d is out of range, expected >= %d", selector, f.name, *f.p, f.min))
		}
	}
	v.validateColor(fmt.Sprintf("Node '%s' borderColor", selector), d.BorderColor)
	v.validateColor(fmt.Sprintf("Node '%s' backgroundColor", selector), d.BackgroundColor)
	v.validateColor(fmt.Sprintf("Node '%s' pathColor", selector), d.PathColor)
	for _, name := range sortedCoordKeys(d.Anchors) {
		c := d.Anchors[name]
		if x := c[0]; x < 0.0 || x > 1.0 {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' anchor '%s' X coordinate %s is out of range, expected 0.0-1.0", selector, name, formatNum(x)))
		}
		if y := c[1]; y < 0.0 || y > 1.0 {
			v.addError(ast.ErrorConstraint, fmt.Sprintf("Node '%s' anchor '%s' Y coordinate %s is out of range, expected 0.0-1.0", selector, name, formatNum(y)))
		}
	}
	// Group declarations carry their own margin; range-check them too.
	for _, it := range d.Arrangement {
		if it.Group != nil {
			v.validateDeclRanges(selector, it.Group)
		}
	}
}

// validateColor reports a malformed hex colour. nil (unset) is fine; otherwise
// the value must be #rgb, #rgba, #rrggbb, or #rrggbbaa. label names the offending
// property in the message (e.g. "Node 'a' borderColor"). The tokenizer only emits
// a colour token for a `#` + hex run, so the usual failure here is a wrong digit
// count.
func (v *validator) validateColor(label string, c *string) {
	if c == nil {
		return
	}
	if !isValidHexColor(*c) {
		v.addError(ast.ErrorConstraint, fmt.Sprintf("%s '%s' is not a valid hex colour, expected #rgb, #rgba, #rrggbb, or #rrggbbaa", label, *c))
	}
}

// isValidHexColor reports whether s is a `#` followed by 3, 4, 6, or 8 hex digits.
func isValidHexColor(s string) bool {
	if len(s) == 0 || s[0] != '#' {
		return false
	}
	h := s[1:]
	switch len(h) {
	case 3, 4, 6, 8:
	default:
		return false
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
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

// validateArrangement checks a node's child arrangement: every referenced id is
// a direct child of the node (same-parent — including ids nested in @group
// wrappers), and once a node is arranged, each direct child is referenced
// exactly once (completeness — no foreigners, duplicates, or omissions). This
// keeps the layout tree a refinement of the semantic tree.
func (v *validator) validateArrangement(selector string, d *ast.Declarations) {
	if d == nil || len(d.Arrangement) == 0 {
		return
	}
	node := v.nodeMap[selector]
	if node == nil {
		return
	}
	childOf := make(map[string]bool, len(node.Children))
	for _, c := range node.Children {
		childOf[c.ID] = true
	}

	var refs []ast.ArrangementItem
	collectChildRefs(d.Arrangement, &refs)

	seen := map[string]int{}
	for _, r := range refs {
		if !childOf[r.ChildID] {
			v.errors = append(v.errors, ast.Error{
				Type:    ast.ErrorReference,
				Message: fmt.Sprintf("Arrangement of node '%s' references '%s', which is not one of its children", selector, r.ChildID),
				Line:    r.Line,
				Column:  r.Column,
			})
			continue
		}
		seen[r.ChildID]++
	}
	for _, name := range sortedIntKeys(seen) {
		if seen[name] > 1 {
			v.addError(ast.ErrorReference, fmt.Sprintf("Arrangement of node '%s' references child '%s' more than once", selector, name))
		}
	}
	var missing []string
	for _, c := range node.Children {
		if seen[c.ID] == 0 {
			missing = append(missing, c.ID)
		}
	}
	if len(missing) > 0 {
		v.addError(ast.ErrorReference, fmt.Sprintf("Arrangement of node '%s' omits child(ren): %s", selector, strings.Join(missing, ", ")))
	}
}

// collectChildRefs flattens an arrangement to its child-id references, descending
// into @group wrappers (which may only contain the same node's children).
func collectChildRefs(items []ast.ArrangementItem, out *[]ast.ArrangementItem) {
	for _, it := range items {
		if it.Group != nil {
			collectChildRefs(it.Group.Arrangement, out)
			continue
		}
		*out = append(*out, it)
	}
}

// validateLayoutBlocks checks the reuse layer: every `@use` (in a selector,
// inline block, or block composition) names a defined block, and block
// composition has no cycles. An unknown `kind` is intentionally not checked —
// it is a semantic tag, not a layout import.
func (v *validator) validateLayoutBlocks(doc *ast.Document) {
	defined := map[string]bool{}
	for name := range ast.BuiltinBlocks() {
		defined[name] = true
	}
	for _, b := range doc.Blocks {
		defined[b.Name] = true
	}

	for _, r := range doc.Layout {
		v.checkUsesDefined(r.Uses, defined)
	}
	for _, b := range doc.Blocks {
		v.checkUsesDefined(b.Uses, defined)
		if b.Decls != nil && len(b.Decls.Arrangement) > 0 {
			v.errors = append(v.errors, ast.Error{
				Type:    ast.ErrorReference,
				Message: fmt.Sprintf("Layout block '%s' may not contain a child arrangement", b.Name),
				Line:    b.Line,
				Column:  b.Column,
			})
		}
	}

	v.validateBlockCycles(doc.Blocks)
}

func (v *validator) checkUsesDefined(uses []ast.Use, defined map[string]bool) {
	for _, u := range uses {
		if !defined[u.Block] {
			v.errors = append(v.errors, ast.Error{
				Type:    ast.ErrorReference,
				Message: fmt.Sprintf("Layout block '%s' is not defined", u.Block),
				Line:    u.Line,
				Column:  u.Column,
			})
		}
	}
}

// validateBlockCycles reports `@use` composition cycles among `@block`
// definitions, via a coloured DFS over the block graph. Built-in blocks have no
// imports, so they cannot take part in a cycle. Each distinct cycle is reported
// once, at the position of the block the back-edge points to.
func (v *validator) validateBlockCycles(blocks []ast.Block) {
	const (
		white = iota
		gray
		black
	)
	graph := map[string][]ast.Use{}
	line := map[string]int{}
	col := map[string]int{}
	var names []string
	for _, b := range blocks {
		if _, seen := graph[b.Name]; !seen {
			names = append(names, b.Name)
		}
		graph[b.Name] = b.Uses // a redefined block keeps its last body (last wins)
		line[b.Name], col[b.Name] = b.Line, b.Column
	}
	sort.Strings(names)

	color := map[string]int{}
	reported := map[string]bool{}
	var stack []string
	var visit func(name string)
	visit = func(name string) {
		color[name] = gray
		stack = append(stack, name)
		for _, u := range graph[name] {
			if _, ok := graph[u.Block]; !ok {
				continue // built-in or undefined target: not part of a user-block cycle
			}
			switch color[u.Block] {
			case white:
				visit(u.Block)
			case gray:
				cycle := cycleFrom(stack, u.Block)
				if sig := cycleSig(cycle); !reported[sig] {
					reported[sig] = true
					v.errors = append(v.errors, ast.Error{
						Type:    ast.ErrorReference,
						Message: fmt.Sprintf("Layout block cycle detected: %s", strings.Join(append(cycle, u.Block), " -> ")),
						Line:    line[u.Block],
						Column:  col[u.Block],
					})
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[name] = black
	}
	for _, name := range names {
		if color[name] == white {
			visit(name)
		}
	}
}

// cycleFrom returns the suffix of stack starting at the first occurrence of
// start — the members of the cycle in traversal order.
func cycleFrom(stack []string, start string) []string {
	for i, n := range stack {
		if n == start {
			return append([]string(nil), stack[i:]...)
		}
	}
	return append([]string(nil), stack...)
}

// cycleSig is an order-independent signature for a cycle's member set, used to
// report each distinct cycle only once.
func cycleSig(cycle []string) string {
	s := append([]string(nil), cycle...)
	sort.Strings(s)
	return strings.Join(s, ",")
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
