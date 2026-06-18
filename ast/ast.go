// Package ast defines the Arkitecture syntax tree and the shared diagnostic
// type produced by every stage of the pipeline.
//
// It deliberately has no dependencies so that the parser, validator, resolver,
// and generator can all build on it without creating an import cycle through
// the top-level arkitecture package.
//
// The tree is split into two layers (see docs/design.md):
//
//   - a semantic tree of [ContainerNode]s carrying id, label, kind, anchor
//     *names*, and children;
//   - a layout layer of [Declarations] — direction, size, margin, box, and
//     anchor *positions* — authored in `@layout` blocks, either inline on a
//     node or as standalone sheet rules ([Document.Layout]). Layout can be
//     bundled into reusable named [Block]s and imported with [Use]; a node's
//     `kind` imports the block of the same name as a baseline.
package ast

import "fmt"

// Direction is the layout direction of a node. The empty value means "unset";
// consumers default it to Vertical.
type Direction string

const (
	// DirectionUnset is the zero value, meaning the author did not specify a
	// direction. Layout treats it as Vertical.
	DirectionUnset Direction = ""
	// Vertical stacks children top to bottom.
	Vertical Direction = "vertical"
	// Horizontal places children left to right.
	Horizontal Direction = "horizontal"
)

// Box controls whether a node draws its border. The empty value means
// "default" — a bordered box. BoxNone makes the node invisible: it draws no
// rectangle and, as a parent, collapses its children's perimeter margins. It
// replaces the old layout-only group on a node that still has an ID, label,
// and anchors.
type Box string

const (
	// BoxDefault is the zero value: the node draws its 1px border.
	BoxDefault Box = ""
	// BoxNone makes the node borderless and invisible.
	BoxNone Box = "none"
)

// ContainerNode is the single node type: a component identified by ID, with an
// optional label, an optional semantic kind, the named anchors it exposes, and
// nested children. Layout (direction, size, margin, box, anchor positions) is
// not stored on the node — it lives in the document's layout rules, which the
// resolve stage merges onto the node by exact path. An inline `@layout {…}` in
// a node body is desugared by the parser into a [LayoutRule] whose selector is
// the node's full dotted path, so inline and standalone layout are uniform.
type ContainerNode struct {
	ID       string
	Label    *string          // optional display text
	Kind     string           // optional semantic classification; "" = none
	Anchors  []string         // declared anchor names (positions live in layout)
	Children []*ContainerNode // nested child nodes
}

// Declarations is a set of layout properties — the body of an `@layout` block.
// Each scalar is a pointer so "unset" stays distinguishable from a real value
// (which matters for the resolve stage's duplicate-property conflict check).
// Anchors maps an anchor name to its relative [x, y] position. Arrangement, when
// non-empty, is the node's ordered child layout (with optional `@group`
// wrappers); empty means "lay children out in semantic order".
type Declarations struct {
	Direction   *Direction
	Size        *float64
	Margin      *float64
	Box         *Box
	Anchors     map[string][2]float64
	Arrangement []ArrangementItem
}

// ArrangementItem is one entry in a node's child arrangement: either a reference
// to a direct child (by id) or an anonymous `@group` wrapper. Exactly one of
// ChildID / Group is set. A group is itself a [Declarations] whose own
// Arrangement holds its nested items — an invisible (`box: none`) layout
// sub-container with its own direction/size/margin. Line/Column point at the
// entry for arrangement diagnostics.
type ArrangementItem struct {
	ChildID string
	Group   *Declarations
	Line    int
	Column  int
}

// Use is an `@use <block>` directive: a request to import a named layout block's
// declarations as a baseline. Uses compose (a block may itself `@use` another;
// cycles are a validation error) and sit in the imported precedence tier, below
// direct declarations. Line/Column point at the block name for undefined-block
// diagnostics.
type Use struct {
	Block  string
	Line   int
	Column int
}

// LayoutRule is one `@layout` selector block: a node targeted by exact dotted
// path, the direct declarations applied to it, and any `@use` directives it
// imports. Line/Column point at the selector for dangling-selector diagnostics.
// An inline `@layout {…}` is desugared into a rule whose selector is the node's
// full path, so inline and standalone layout are uniform.
type LayoutRule struct {
	Selector string
	Decls    *Declarations
	Uses     []Use // @use directives in this block, in source order
	Line     int
	Column   int
}

// Block is a named, reusable bundle of layout declarations defined by
// `@block <name> { decls }` inside an `@layout` sheet. A block may itself `@use`
// other blocks (composition); a node pulls one in explicitly via `@use` or
// implicitly via a matching `kind`. A user block overrides a built-in of the
// same name. Line/Column point at the block name.
type Block struct {
	Name   string
	Decls  *Declarations
	Uses   []Use
	Line   int
	Column int
}

// Arrow is a directed connection between two node paths. Source and Target are
// the raw textual references (e.g. "c1.n2" or "c1.n3#a1"); the validator
// resolves them.
type Arrow struct {
	Source string
	Target string
}

// Document is a whole parsed .ark file: the top-level semantic nodes, the
// layout sheet rules, the named layout blocks (`@block`), and the arrows between
// nodes. Layout, blocks, and arrows are parsed in phases, so they live in flat
// lists rather than on the nodes.
type Document struct {
	Nodes  []*ContainerNode
	Layout []LayoutRule
	Blocks []Block
	Arrows []Arrow
}

// BuiltinBlocks are the layout blocks every document gets for free, keyed by
// name. A `kind` or `@use` naming one applies its declarations as a baseline; a
// user `@block` of the same name overrides it. v1 ships a single kind,
// `invisible` (box: none) — the layout layer is structural, so built-ins can
// only set structural properties for now. A fresh map is returned per call so
// callers can treat it as their own.
func BuiltinBlocks() map[string]*Declarations {
	none := BoxNone
	return map[string]*Declarations{
		"invisible": {Box: &none},
	}
}

// ErrorType categorises a diagnostic.
type ErrorType string

const (
	// ErrorSyntax is a lexing or grammar error.
	ErrorSyntax ErrorType = "syntax"
	// ErrorReference is an unresolved node, anchor, or selector reference.
	ErrorReference ErrorType = "reference"
	// ErrorConstraint is an out-of-range value (size, margin, or coordinate).
	ErrorConstraint ErrorType = "constraint"
)

// Error is a single diagnostic with source position. Failures are collected as
// data rather than thrown, so every stage returns a slice of these. It also
// satisfies the standard error interface for convenience.
type Error struct {
	Line    int
	Column  int
	Message string
	Type    ErrorType
}

func (e Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s (line %d, column %d): %s", e.Type, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ParseResult is the output of the parser: a (possibly partial) document plus
// any collected errors.
type ParseResult struct {
	Success  bool
	Document *Document
	Errors   []Error
}
