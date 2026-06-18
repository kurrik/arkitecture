// Package ast defines the Arkitecture syntax tree and the shared diagnostic
// type produced by every stage of the pipeline.
//
// It deliberately has no dependencies so that the parser, validator, and
// generator can all build on it without creating an import cycle through the
// top-level arkitecture package.
package ast

import "fmt"

// Direction is the layout direction of a node or group. The empty value means
// "unset"; consumers default it to Vertical.
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
// is the spiritual twin of the layout-only GroupNode, but on a node that still
// has an ID, label, and anchors.
type Box string

const (
	// BoxDefault is the zero value: the node draws its 1px border.
	BoxDefault Box = ""
	// BoxNone makes the node borderless and invisible.
	BoxNone Box = "none"
)

// Node is implemented by the two things that can appear in a node tree:
// *ContainerNode and *GroupNode.
type Node interface {
	isNode()
}

// ContainerNode is a labelled, bordered box identified by ID. It is the primary
// building block and may contain child nodes and groups.
type ContainerNode struct {
	ID        string
	Label     *string               // optional display text
	Direction Direction             // unset => Vertical
	Size      *float64              // optional override in [0,1] for the orthogonal dimension
	Margin    *float64              // optional uniform margin (>=0) around the border box; nil => default
	Box       Box                   // BoxNone draws no border (invisible); unset => bordered
	Anchors   map[string][2]float64 // optional named anchor points, relative [x,y] in [0,1]
	Children  []Node
}

func (*ContainerNode) isNode() {}

// GroupNode is a layout-only container: no ID, label, or border. It exists to
// arrange its children and is invisible in the output.
type GroupNode struct {
	Direction Direction
	Children  []Node
}

func (*GroupNode) isNode() {}

// Arrow is a directed connection between two node paths. Source and Target are
// the raw textual references (e.g. "c1.n2" or "c1.n3#a1"); the validator
// resolves them.
type Arrow struct {
	Source string
	Target string
}

// Document is a whole parsed .ark file: the top-level container nodes plus the
// arrows between them. Arrows are parsed in a second phase, so they live in a
// flat list rather than on the nodes.
type Document struct {
	Nodes  []*ContainerNode
	Arrows []Arrow
}

// ErrorType categorises a diagnostic.
type ErrorType string

const (
	// ErrorSyntax is a lexing or grammar error.
	ErrorSyntax ErrorType = "syntax"
	// ErrorReference is an unresolved node or anchor reference.
	ErrorReference ErrorType = "reference"
	// ErrorConstraint is an out-of-range value (size or coordinate).
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
