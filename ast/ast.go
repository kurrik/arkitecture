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
//   - a layout layer of [Declarations] — direction, margin, box, and
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

// RouteMode selects how arrows are drawn across the whole document. It is a
// document-level setting (a bare `route:` at an `@layout` sheet root), not a
// per-node layout property. The empty value means "unset"; consumers default it
// to RouteStraight.
type RouteMode string

const (
	// RouteUnset is the zero value, meaning the author did not specify a route
	// mode. Rendering treats it as RouteStraight.
	RouteUnset RouteMode = ""
	// RouteStraight draws each arrow as a single straight line between its
	// resolved endpoints — the default, M2 auto-cardinal routing.
	RouteStraight RouteMode = "straight"
	// RouteOrthogonal routes each arrow as an orthogonal path around the boxes
	// that lie between its endpoints, reserving sized channels for the lines.
	RouteOrthogonal RouteMode = "orthogonal"
)

// LabelPosition controls where a bordered parent's label sits. A parent reserves
// a strip for its label so it does not overlap the children; this says whether
// that strip is at the top (default) or the bottom of the box. The empty value
// means "unset"; consumers default it to LabelTop.
type LabelPosition string

const (
	// LabelPositionUnset is the zero value, meaning the author did not specify a
	// label position. Layout treats it as LabelTop.
	LabelPositionUnset LabelPosition = ""
	// LabelTop reserves the label strip at the top of the box.
	LabelTop LabelPosition = "top"
	// LabelBottom reserves the label strip at the bottom of the box.
	LabelBottom LabelPosition = "bottom"
)

// ContainerNode is the single node type: a component identified by ID, with an
// optional label, an optional semantic kind, the named anchors it exposes, and
// nested children. Layout (direction, margin, box, anchor positions) is
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
// LabelPos places a bordered parent's reserved label strip (top/bottom). Anchors
// maps an anchor name to its relative [x, y] position. Arrangement, when
// non-empty, is the node's ordered child layout (with optional `@group`
// wrappers); empty means "lay children out in semantic order".
//
// The style fields are presentation overrides (hex `#rrggbb` colours and stroke
// widths). BorderWidth/BorderColor/BackgroundColor style a node's own box;
// PathWidth/PathColor style the arrows that *start* at the node. All default to
// the plain look (white fill, 1px black border, 1px black arrows) when unset.
type Declarations struct {
	Direction   *Direction
	Margin      *float64
	Box         *Box
	LabelPos    *LabelPosition
	Anchors     map[string][2]float64
	Arrangement []ArrangementItem

	BorderWidth     *float64 // box border stroke width (default 1)
	BorderColor     *string  // box border colour, hex #rrggbb (default black)
	BackgroundColor *string  // box fill colour, hex #rrggbb (default white)
	PathWidth       *float64 // width of arrows starting at this node (default 1)
	PathColor       *string  // colour of arrows starting at this node, hex (default black)

	// Grid, when set, makes this node arrange its children as a 2-D grid (the
	// 2-D generalisation of Direction's 1-D packing). Like Arrangement it is
	// direct-only — never imported via @use/kind.
	Grid *GridSpec

	// Grid placement of *this* node within its parent's grid (meaningful only
	// when the parent is a grid). Col/Row are 1-based grid lines; nil means
	// "auto-place" (fill the next free slot, left→right, top→bottom). ColSpan/
	// RowSpan default to 1. Justify/Align position the node within its (possibly
	// larger) cell on the horizontal/vertical axis.
	Col     *int
	Row     *int
	ColSpan *int
	RowSpan *int
	Justify *GridAlign // horizontal placement within the cell
	Align   *GridAlign // vertical placement within the cell
}

// GridSpec is a node's grid track definition. Cols is the fixed number of
// columns (required); Rows, when set, fixes the row count (a placement past it
// is an error), otherwise rows grow implicitly to fit the placed children. Gap,
// when set, overrides the default inter-track spacing.
type GridSpec struct {
	Cols int
	Rows *int
}

// GridAlign is how a node sits within its grid cell on one axis.
type GridAlign string

const (
	GridStart   GridAlign = "start"
	GridEnd     GridAlign = "end"
	GridStretch GridAlign = "stretch"
)

// GridCell is the placement request for one child of a grid: its id, its
// explicit 1-based Col/Row (0 = auto), and its spans (0 = default 1). It is the
// input to PlaceGrid.
type GridCell struct {
	ChildID string
	Col     int
	Row     int
	ColSpan int
	RowSpan int
}

// PlacedCell is a resolved grid placement: 1-based Col/Row and the resolved
// spans. The covered tracks are [Col, Col+ColSpan) × [Row, Row+RowSpan).
type PlacedCell struct {
	ChildID string
	Col     int
	Row     int
	ColSpan int
	RowSpan int
}

// PlaceGrid resolves a grid's child placements deterministically. Cells are
// processed in source order: an explicitly positioned cell claims its slot; an
// auto cell (Col or Row unset) takes the next free slot scanning left→right,
// top→bottom (sparse — the cursor only moves forward). It returns the resolved
// placements, the total row count (max of the spec's fixed Rows and the rows
// actually used), and one message per problem (a cell whose span leaves the
// fixed bounds, or an overlap). It never panics: an out-of-range or overlapping
// cell is still placed (clamped to ≥1) so the geometry stage has something to
// draw, the error being the validator's to report.
func PlaceGrid(spec GridSpec, cells []GridCell) (placed []PlacedCell, rows int, problems []GridProblem) {
	cols := spec.Cols
	if cols < 1 {
		cols = 1
	}
	occupied := map[[2]int]string{} // [row,col] -> childID
	cursorRow, cursorCol := 1, 1
	maxRow := 0
	if spec.Rows != nil {
		maxRow = *spec.Rows
	}

	fits := func(r, c, cs, rs int) bool {
		for dr := 0; dr < rs; dr++ {
			for dc := 0; dc < cs; dc++ {
				if _, taken := occupied[[2]int{r + dr, c + dc}]; taken {
					return false
				}
			}
		}
		return true
	}
	mark := func(id string, r, c, cs, rs int) {
		for dr := 0; dr < rs; dr++ {
			for dc := 0; dc < cs; dc++ {
				occupied[[2]int{r + dr, c + dc}] = id
			}
			if r+dr > maxRow {
				maxRow = r + dr
			}
		}
	}

	for _, cell := range cells {
		cs, rs := cell.ColSpan, cell.RowSpan
		if cs < 1 {
			cs = 1
		}
		if rs < 1 {
			rs = 1
		}

		var r, c int
		if cell.Col > 0 && cell.Row > 0 {
			c, r = cell.Col, cell.Row
			if c+cs-1 > cols {
				problems = append(problems, GridProblem{ChildID: cell.ChildID,
					Message: fmt.Sprintf("grid cell '%s' at column %d spans %d past the %d-column bound", cell.ChildID, c, cs, cols)})
			}
			if spec.Rows != nil && r+rs-1 > *spec.Rows {
				problems = append(problems, GridProblem{ChildID: cell.ChildID,
					Message: fmt.Sprintf("grid cell '%s' at row %d spans %d past the %d-row bound", cell.ChildID, r, rs, *spec.Rows)})
			}
			if prev, taken := firstTaken(occupied, r, c, cs, rs); taken {
				problems = append(problems, GridProblem{ChildID: cell.ChildID,
					Message: fmt.Sprintf("grid cell '%s' overlaps '%s' at row %d, column %d", cell.ChildID, prev, r, c)})
			}
		} else {
			// Auto-place: scan forward from the cursor for the next free run.
			r, c = cursorRow, cursorCol
			for {
				if c+cs-1 > cols {
					r++
					c = 1
					continue
				}
				if fits(r, c, cs, rs) {
					break
				}
				c++
			}
			cursorRow, cursorCol = r, c+cs
			if cursorCol > cols {
				cursorRow++
				cursorCol = 1
			}
		}
		mark(cell.ChildID, r, c, cs, rs)
		placed = append(placed, PlacedCell{ChildID: cell.ChildID, Col: c, Row: r, ColSpan: cs, RowSpan: rs})
	}
	return placed, maxRow, problems
}

// firstTaken reports the first occupant of the [r,c]×span rectangle, if any.
func firstTaken(occupied map[[2]int]string, r, c, cs, rs int) (string, bool) {
	for dr := 0; dr < rs; dr++ {
		for dc := 0; dc < cs; dc++ {
			if id, taken := occupied[[2]int{r + dr, c + dc}]; taken {
				return id, true
			}
		}
	}
	return "", false
}

// GridProblem is a placement error from PlaceGrid, naming the offending child.
type GridProblem struct {
	ChildID string
	Message string
}

// ArrangementItem is one entry in a node's child arrangement: either a reference
// to a direct child (by id) or an anonymous `@group` wrapper. Exactly one of
// ChildID / Group is set. A group is itself a [Declarations] whose own
// Arrangement holds its nested items — an invisible (`box: none`) layout
// sub-container with its own direction/margin. Line/Column point at the
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
//
// DefaultMargin, when set, is a document-wide default margin authored as a bare
// `margin:` at the root of an `@layout` sheet. It replaces the built-in default
// (8) as the fallback for any node that sets no margin of its own; it is a single
// global baseline, not a cascade (no per-node selector wins it, and nodes still
// override it directly). nil means "use the built-in default".
//
// Route, when set, is the document-wide arrow routing mode authored as a bare
// `route:` at the root of an `@layout` sheet (mirroring DefaultMargin). nil or
// RouteStraight draws straight cardinal lines; RouteOrthogonal routes arrows as
// orthogonal paths around intervening boxes.
//
// Defaults, when set, holds document-wide style defaults authored as bare style
// properties (`borderColor:`, `pathWidth:`, …) at the root of an `@layout` sheet
// — the same fallback model as DefaultMargin, but for the presentation fields of
// [Declarations]. Only those style fields are ever populated; a node's own
// resolved style overrides the document default, which overrides the built-in
// plain look. nil means "use the built-in defaults".
type Document struct {
	Nodes         []*ContainerNode
	Layout        []LayoutRule
	Blocks        []Block
	Arrows        []Arrow
	DefaultMargin *float64
	Route         *RouteMode
	Defaults      *Declarations
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
	// ErrorConstraint is an out-of-range value (margin or coordinate).
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
