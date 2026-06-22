// Package resolve merges the layout layer onto the semantic tree. It takes the
// parsed document's layout rules (inline `@layout` blocks, desugared by the
// parser into path selectors, plus standalone sheet selectors), the named
// `@block` definitions, and each node's `kind`, and produces a per-node resolved
// layout keyed by full dotted path.
//
// It is a pure function of its input and assumes the document already passed
// validation, so merging is a deterministic overlay. Resolution has two
// precedence tiers (see docs/design.md):
//
//   - Imported (lower): the `kind` baseline first, then each `@use <block>` in
//     source order. A block expands recursively — its own `@use`s, then its own
//     declarations.
//   - Direct (higher): declarations naming the node itself (inline or sheet),
//     which override the imported tier.
//
// Within a tier the merge is last-write-wins. An unknown `kind` or `@use` block
// resolves to nothing here (the validator reports a dangling `@use`); cycles in
// block composition are stopped defensively (the validator reports them).
package resolve

import "github.com/kurrik/arkitecture/ast"

// Layout maps a node's full dotted path to its merged layout declarations.
// A path missing from the map has no explicit layout; the generator applies
// defaults (vertical, margin 8, bordered).
type Layout map[string]*ast.Declarations

// blockDef is a resolved block entry: its declarations plus the blocks it
// imports. Built-ins have no imports.
type blockDef struct {
	decls *ast.Declarations
	uses  []ast.Use
}

// Resolve computes the merged layout for every node that carries a `kind` or is
// named by a layout rule.
func Resolve(doc *ast.Document) Layout {
	out := Layout{}
	if doc == nil {
		return out
	}

	blocks := buildBlockTable(doc)
	kinds := map[string]string{}
	for _, n := range doc.Nodes {
		collectKinds(n, "", kinds)
	}

	paths := map[string]bool{}
	for path := range kinds {
		paths[path] = true
	}
	for _, r := range doc.Layout {
		paths[r.Selector] = true
	}

	for path := range paths {
		merged := &ast.Declarations{}

		// Imported tier (lowest precedence first): the kind baseline, then each
		// @use in source order — later writes win within the tier.
		if kind := kinds[path]; kind != "" {
			expandInto(merged, kind, blocks, map[string]bool{})
		}
		for _, r := range doc.Layout {
			if r.Selector != path {
				continue
			}
			for _, u := range r.Uses {
				expandInto(merged, u.Block, blocks, map[string]bool{})
			}
		}

		// Direct tier: declarations naming the node override the imported tier.
		// The child arrangement is direct-only — it is never imported from a
		// block or kind (mergeDecls leaves it untouched), so a later direct rule
		// is the only thing that sets it (last one wins).
		for _, r := range doc.Layout {
			if r.Selector != path {
				continue
			}
			mergeDecls(merged, r.Decls)
			if r.Decls != nil && r.Decls.Arrangement != nil {
				merged.Arrangement = r.Decls.Arrangement
			}
			// The grid arrangement is direct-only too (mergeDecls leaves it
			// untouched), so a later direct rule is the only thing that sets it.
			if r.Decls != nil && r.Decls.Grid != nil {
				merged.Grid = r.Decls.Grid
			}
		}

		out[path] = merged
	}
	return out
}

// buildBlockTable seeds the built-in kinds, then layers user `@block`s on top so
// a user block of the same name overrides a built-in (last definition wins).
func buildBlockTable(doc *ast.Document) map[string]*blockDef {
	table := map[string]*blockDef{}
	for name, d := range ast.BuiltinBlocks() {
		table[name] = &blockDef{decls: d}
	}
	for _, b := range doc.Blocks {
		table[b.Name] = &blockDef{decls: b.Decls, uses: b.Uses}
	}
	return table
}

func collectKinds(n *ast.ContainerNode, parentPath string, out map[string]string) {
	path := n.ID
	if parentPath != "" {
		path = parentPath + "." + n.ID
	}
	if n.Kind != "" {
		out[path] = n.Kind
	}
	for _, c := range n.Children {
		collectKinds(c, path, out)
	}
}

// expandInto overlays the named block onto dst: first the blocks it imports (in
// source order), then its own declarations. stack tracks the current expansion
// path so a composition cycle stops instead of recursing forever (a valid
// document has none — the validator rejects cycles first).
func expandInto(dst *ast.Declarations, name string, table map[string]*blockDef, stack map[string]bool) {
	bd := table[name]
	if bd == nil || stack[name] {
		return // unknown block (validator reports a dangling @use) or a cycle
	}
	stack[name] = true
	for _, u := range bd.uses {
		expandInto(dst, u.Block, table, stack)
	}
	mergeDecls(dst, bd.decls)
	delete(stack, name)
}

// mergeDecls overlays src's set properties onto dst (last-write-wins); anchor
// positions union.
func mergeDecls(dst, src *ast.Declarations) {
	if src == nil {
		return
	}
	if src.Direction != nil {
		dst.Direction = src.Direction
	}
	if src.Size != nil {
		dst.Size = src.Size
	}
	if src.Margin != nil {
		dst.Margin = src.Margin
	}
	if src.Box != nil {
		dst.Box = src.Box
	}
	if src.LabelPos != nil {
		dst.LabelPos = src.LabelPos
	}
	if src.BorderWidth != nil {
		dst.BorderWidth = src.BorderWidth
	}
	if src.BorderColor != nil {
		dst.BorderColor = src.BorderColor
	}
	if src.BackgroundColor != nil {
		dst.BackgroundColor = src.BackgroundColor
	}
	if src.PathWidth != nil {
		dst.PathWidth = src.PathWidth
	}
	if src.PathColor != nil {
		dst.PathColor = src.PathColor
	}
	if src.Col != nil {
		dst.Col = src.Col
	}
	if src.Row != nil {
		dst.Row = src.Row
	}
	if src.ColSpan != nil {
		dst.ColSpan = src.ColSpan
	}
	if src.RowSpan != nil {
		dst.RowSpan = src.RowSpan
	}
	if src.Justify != nil {
		dst.Justify = src.Justify
	}
	if src.Align != nil {
		dst.Align = src.Align
	}
	for name, pos := range src.Anchors {
		if dst.Anchors == nil {
			dst.Anchors = map[string][2]float64{}
		}
		dst.Anchors[name] = pos
	}
}
