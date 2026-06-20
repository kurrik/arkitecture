// Package parser turns Arkitecture DSL text into an ast.Document. It is a
// hand-written tokenizer plus a recursive-descent parser that collects all
// syntax errors as data rather than failing fast.
//
// The grammar has three kinds of top-level item: semantic nodes, standalone
// `@layout { … }` sheets, and arrows. A node body holds only semantics (label,
// kind, anchor names, child nodes) plus an inline `@layout {…}`; layout
// properties (direction, size, margin, box, anchor positions) live exclusively
// inside `@layout` blocks. An inline block is desugared into a LayoutRule
// selecting the enclosing node's full path, so the resolver and validator treat
// inline and standalone layout identically. Inside an `@layout` sheet a
// `@block <name> { … }` defines a reusable bundle, and a selector or inline
// block may `@use <name>` to import one.
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kurrik/arkitecture/ast"
)

// Parse tokenizes and parses DSL content into a Document, collecting every
// error it finds. A tokenizer error short-circuits to a single syntax error;
// otherwise the (possibly partial) document is returned alongside any errors.
func Parse(dslContent string) ast.ParseResult {
	tokens, terr := Tokenize(dslContent)
	if terr != nil {
		return ast.ParseResult{
			Success: false,
			Errors: []ast.Error{{
				Type:    ast.ErrorSyntax,
				Message: terr.Error(),
				Line:    terr.Line,
				Column:  terr.Column,
			}},
		}
	}
	p := &parser{tokens: tokens}
	return p.parseDocument()
}

type parser struct {
	tokens        []Token
	current       int
	errors        []ast.Error
	rules         []ast.LayoutRule // collected inline + standalone layout rules
	blocks        []ast.Block      // collected @block definitions
	defaultMargin *float64         // document-wide default margin (bare `margin:` at a sheet root)
	route         *ast.RouteMode   // document-wide routing mode (bare `route:` at a sheet root)
}

func (p *parser) parseDocument() ast.ParseResult {
	nodes, arrows := p.parseTopLevel()
	doc := &ast.Document{Nodes: nodes, Layout: p.rules, Blocks: p.blocks, Arrows: arrows, DefaultMargin: p.defaultMargin, Route: p.route}
	if len(p.errors) > 0 {
		return ast.ParseResult{Success: false, Document: doc, Errors: p.errors}
	}
	return ast.ParseResult{Success: true, Document: doc}
}

// parseTopLevel reads the document's top level — node definitions, standalone
// @layout sheets, and arrow statements — in any order, so an arrow can be
// colocated with the nodes it connects rather than forced into a trailing block.
// Each statement is dispatched by lookahead: an identifier that reaches a `-->`
// (after an optional dotted path and #anchor) is an arrow; `@` begins a sheet;
// otherwise it is a node. Arrow endpoints are resolved later by the validator,
// so forward references to not-yet-defined nodes are fine.
func (p *parser) parseTopLevel() ([]*ast.ContainerNode, []ast.Arrow) {
	var nodes []*ast.ContainerNode
	var arrows []ast.Arrow
	for !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		if p.isArrowStatement() {
			if arrow, ok := p.parseArrow(); ok {
				arrows = append(arrows, arrow)
			}
			continue
		}
		if p.check(TokenAt) {
			p.parseLayoutSheet()
			continue
		}
		if node := p.parseNode(""); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes, arrows
}

// parseNode parses `id { … }`. parentPath is the dotted path of the enclosing
// node ("" at the top level), used to desugar an inline @layout into a rule.
func (p *parser) parseNode(parentPath string) *ast.ContainerNode {
	if !p.check(TokenIdentifier) {
		if !p.isAtEnd() {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected node identifier, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
		}
		return nil
	}

	idTok := p.advance()
	id := idTok.Value
	fullPath := id
	if parentPath != "" {
		fullPath = parentPath + "." + id
	}

	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after node id '%s', got %s", id, tok.Type), tok.Line, tok.Column)
		p.skipUntilNodeOrEOF()
		return nil
	}
	p.advance() // consume '{'

	node := &ast.ContainerNode{ID: id}
	p.parseNodeContent(node, fullPath)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close node '%s', got %s", id, tok.Type), tok.Line, tok.Column)
		return node
	}
	p.advance() // consume '}'
	return node
}

func (p *parser) parseNodeContent(node *ast.ContainerNode, fullPath string) {
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}

		if p.check(TokenAt) {
			// Inline @layout: desugar to a rule selecting this node's path.
			if rule, ok := p.parseInlineLayout(fullPath); ok {
				p.rules = append(p.rules, rule)
			}
			continue
		}

		if p.check(TokenIdentifier) {
			// IDENTIFIER followed by '{' is a nested node; otherwise a property.
			if nx := p.peekNext(); nx != nil && nx.Type == TokenLBrace {
				if child := p.parseNode(fullPath); child != nil {
					node.Children = append(node.Children, child)
				}
				continue
			}
			p.parseSemanticProperty(node)
		} else {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected property name, nested node, or @layout, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
		}
	}
}

// parseSemanticProperty handles a node body's semantic properties: label, kind,
// and anchor names. Layout properties are rejected here with a hint to move
// them into @layout.
func (p *parser) parseSemanticProperty(node *ast.ContainerNode) {
	propTok := p.advance()
	name := propTok.Value

	if !p.check(TokenColon) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ':' after property '%s', got %s", name, tok.Type), tok.Line, tok.Column)
		p.skipUntilRecovery()
		return
	}
	p.advance() // consume ':'

	switch name {
	case "label":
		p.parseLabel(node)
	case "kind":
		p.parseKind(node)
	case "anchors":
		p.parseAnchorNames(node)
	case "direction", "size", "margin", "box":
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Layout property '%s' must be set inside an @layout block, not on the node", name), propTok.Line, propTok.Column)
		if !p.check(TokenRBrace) && !p.isAtEnd() {
			p.advance()
		}
	default:
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown property '%s'", name), propTok.Line, propTok.Column)
		if !p.check(TokenRBrace) && !p.isAtEnd() {
			p.advance()
		}
	}
}

func (p *parser) parseLabel(node *ast.ContainerNode) {
	if !p.check(TokenString) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected string value for label, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	v := p.advance().Value
	node.Label = &v
}

func (p *parser) parseKind(node *ast.ContainerNode) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected identifier value for kind, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	node.Kind = p.advance().Value
}

// parseAnchorNames reads `anchors: [name, name, …]` — the declared anchor name
// set. Positions are layout and live in @layout.
func (p *parser) parseAnchorNames(node *ast.ContainerNode) {
	if !p.check(TokenLBracket) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '[' to start anchor name list, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	p.advance() // consume '['

	seen := map[string]bool{}
	for !p.check(TokenRBracket) && !p.isAtEnd() {
		if p.check(TokenNewline) || p.check(TokenComma) {
			p.advance()
			continue
		}
		if !p.check(TokenIdentifier) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected anchor name, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
			continue
		}
		nameTok := p.advance()
		if seen[nameTok.Value] {
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Duplicate anchor name '%s'", nameTok.Value), nameTok.Line, nameTok.Column)
			continue
		}
		seen[nameTok.Value] = true
		node.Anchors = append(node.Anchors, nameTok.Value)
	}

	if !p.check(TokenRBracket) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ']' to close anchor name list, got %s", tok.Type), tok.Line, tok.Column)
		return
	}
	p.advance() // consume ']'
}

// --- @layout blocks ---

// parseDirective consumes `@ <name>` and returns the directive name token. It
// reports an error if `@` is not followed by an identifier.
func (p *parser) parseDirective() (Token, bool) {
	atTok := p.advance() // consume '@'
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected directive name after '@', got %s", tok.Type), atTok.Line, atTok.Column)
		return Token{}, false
	}
	return p.advance(), true
}

// parseInlineLayout parses an inline `@layout { decls }` and returns a rule
// targeting selector (the enclosing node's full path).
func (p *parser) parseInlineLayout(selector string) (ast.LayoutRule, bool) {
	dirTok, ok := p.parseDirective()
	if !ok {
		return ast.LayoutRule{}, false
	}
	if dirTok.Value != "layout" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown directive '@%s'", dirTok.Value), dirTok.Line, dirTok.Column)
		p.skipBalancedBlock()
		return ast.LayoutRule{}, false
	}
	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after @layout, got %s", tok.Type), tok.Line, tok.Column)
		return ast.LayoutRule{}, false
	}
	p.advance() // consume '{'

	decls := &ast.Declarations{}
	uses := p.parseDeclarations(decls)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close @layout, got %s", tok.Type), tok.Line, tok.Column)
		return ast.LayoutRule{}, false
	}
	p.advance() // consume '}'
	return ast.LayoutRule{Selector: selector, Decls: decls, Uses: uses, Line: dirTok.Line, Column: dirTok.Column}, true
}

// parseLayoutSheet parses a standalone `@layout { selector { decls } … }`,
// appending one rule per selector block.
func (p *parser) parseLayoutSheet() {
	dirTok, ok := p.parseDirective()
	if !ok {
		return
	}
	if dirTok.Value != "layout" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown directive '@%s'", dirTok.Value), dirTok.Line, dirTok.Column)
		p.skipBalancedBlock()
		return
	}
	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after @layout, got %s", tok.Type), tok.Line, tok.Column)
		return
	}
	p.advance() // consume '{'

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		if p.check(TokenAt) {
			p.parseBlockDef()
			continue
		}
		// A bare `property: value` at the sheet root is a document-wide default,
		// distinguished from a selector (`path { … }`) by the ':' after the name.
		if p.check(TokenIdentifier) {
			if nx := p.peekNext(); nx != nil && nx.Type == TokenColon {
				p.parseDocumentDefault()
				continue
			}
		}
		if rule, ok := p.parseSelectorBlock(); ok {
			p.rules = append(p.rules, rule)
		}
	}

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close @layout, got %s", tok.Type), tok.Line, tok.Column)
		return
	}
	p.advance() // consume '}'
}

// parseDocumentDefault parses a bare `property: value` at the root of an @layout
// sheet — a document-wide setting that is not a per-node layout property. v1
// supports `margin` (the fallback spacing) and `route` (the arrow routing mode);
// the cursor is on the property identifier.
func (p *parser) parseDocumentDefault() {
	propTok := p.advance() // the property identifier
	name := propTok.Value
	p.advance() // consume ':' (the caller confirmed it via lookahead)

	switch name {
	case "margin":
		p.parseNumberDecl(&p.defaultMargin, "margin", propTok)
	case "route":
		p.parseRouteDecl(propTok)
	default:
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown document default '%s'; only 'margin' and 'route' may be set at the @layout root", name), propTok.Line, propTok.Column)
		if !p.check(TokenRBrace) && !p.isAtEnd() {
			p.advance()
		}
	}
}

// parseRouteDecl reads `route: straight | orthogonal`, the document-wide arrow
// routing mode. The cursor is past the ':'.
func (p *parser) parseRouteDecl(propTok Token) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected 'straight' or 'orthogonal' for route, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	tok := p.advance()
	var mode ast.RouteMode
	switch tok.Value {
	case "straight":
		mode = ast.RouteStraight
	case "orthogonal":
		mode = ast.RouteOrthogonal
	default:
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid route '%s', expected 'straight' or 'orthogonal'", tok.Value), tok.Line, tok.Column)
		return
	}
	if p.route != nil {
		p.addError(ast.ErrorSyntax, "Duplicate document property 'route'", propTok.Line, propTok.Column)
		return
	}
	p.route = &mode
}

// parseBlockDef parses `@block <name> { decls }` inside an @layout sheet,
// appending the definition to p.blocks. A block body holds the same
// declarations as a selector block and may itself `@use` other blocks.
func (p *parser) parseBlockDef() {
	dirTok, ok := p.parseDirective()
	if !ok {
		return
	}
	if dirTok.Value != "block" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown directive '@%s' inside @layout, expected @block or a selector", dirTok.Value), dirTok.Line, dirTok.Column)
		p.skipBalancedBlock()
		return
	}
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected block name after @block, got %s", tok.Type), tok.Line, tok.Column)
		p.skipBalancedBlock()
		return
	}
	nameTok := p.advance()
	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after @block '%s', got %s", nameTok.Value, tok.Type), tok.Line, tok.Column)
		return
	}
	p.advance() // consume '{'

	decls := &ast.Declarations{}
	uses := p.parseDeclarations(decls)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close @block '%s', got %s", nameTok.Value, tok.Type), tok.Line, tok.Column)
		return
	}
	p.advance() // consume '}'
	p.blocks = append(p.blocks, ast.Block{Name: nameTok.Value, Decls: decls, Uses: uses, Line: nameTok.Line, Column: nameTok.Column})
}

func (p *parser) parseSelectorBlock() (ast.LayoutRule, bool) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected a selector path, got %s", tok.Type), tok.Line, tok.Column)
		p.advance()
		return ast.LayoutRule{}, false
	}
	selTok := p.peek()
	selector, _ := p.parseDottedPath()

	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after selector '%s', got %s", selector, tok.Type), tok.Line, tok.Column)
		p.skipUntilNodeOrEOF()
		return ast.LayoutRule{}, false
	}
	p.advance() // consume '{'

	decls := &ast.Declarations{}
	uses := p.parseDeclarations(decls)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close selector '%s', got %s", selector, tok.Type), tok.Line, tok.Column)
		return ast.LayoutRule{}, false
	}
	p.advance() // consume '}'
	return ast.LayoutRule{Selector: selector, Decls: decls, Uses: uses, Line: selTok.Line, Column: selTok.Column}, true
}

// parseDeclarations reads the body of an @layout block: `@use` imports plus the
// direct properties direction, size, margin, box, and anchor positions. It
// returns the `@use` directives in source order. A property set twice in the
// same block is a syntax error (across-rule duplicates are the validator's job).
func (p *parser) parseDeclarations(d *ast.Declarations) []ast.Use {
	var uses []ast.Use
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		if p.check(TokenAt) {
			// @group is an arrangement entry; any other @ directive (@use) imports.
			if nx := p.peekNext(); nx != nil && nx.Type == TokenIdentifier && nx.Value == "group" {
				if grp, gtok, ok := p.parseGroup(); ok {
					d.Arrangement = append(d.Arrangement, ast.ArrangementItem{Group: grp, Line: gtok.Line, Column: gtok.Column})
				}
				continue
			}
			if u, ok := p.parseUse(); ok {
				uses = append(uses, u)
			}
			continue
		}
		if !p.check(TokenIdentifier) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected a layout property or @use, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
			continue
		}

		propTok := p.peek()
		name := propTok.Value
		if name == "anchor" {
			p.parseAnchorPosition(d)
			continue
		}
		p.advance() // consume the identifier

		if !p.check(TokenColon) {
			// A bare identifier (no ':') is an arrangement child reference.
			d.Arrangement = append(d.Arrangement, ast.ArrangementItem{ChildID: name, Line: propTok.Line, Column: propTok.Column})
			continue
		}
		p.advance() // consume ':'

		switch name {
		case "direction":
			p.parseDirectionDecl(d, propTok)
		case "size":
			p.parseNumberDecl(&d.Size, "size", propTok)
		case "margin":
			p.parseNumberDecl(&d.Margin, "margin", propTok)
		case "box":
			p.parseBoxDecl(d, propTok)
		case "label":
			p.parseLabelPosDecl(d, propTok)
		default:
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown layout property '%s'", name), propTok.Line, propTok.Column)
			if !p.check(TokenRBrace) && !p.isAtEnd() {
				p.advance()
			}
		}
	}
	return uses
}

// parseUse parses `@use <block>` inside a declarations block. It assumes the
// cursor is on the `@`.
func (p *parser) parseUse() (ast.Use, bool) {
	dirTok, ok := p.parseDirective()
	if !ok {
		return ast.Use{}, false
	}
	if dirTok.Value != "use" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Unknown directive '@%s' in layout block, expected @use", dirTok.Value), dirTok.Line, dirTok.Column)
		return ast.Use{}, false
	}
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected block name after @use, got %s", tok.Type), tok.Line, tok.Column)
		return ast.Use{}, false
	}
	nameTok := p.advance()
	return ast.Use{Block: nameTok.Value, Line: nameTok.Line, Column: nameTok.Column}, true
}

// parseGroup parses an anonymous `@group { … }` arrangement entry. Its body is
// the same grammar as a node's @layout block (declarations + nested arrangement),
// except `@use` is not allowed inside a group. It returns the group's
// declarations (a nested arrangement-bearing Declarations) and the directive
// token for positioning. The caller has already confirmed the directive is
// `@group` via lookahead.
func (p *parser) parseGroup() (*ast.Declarations, Token, bool) {
	dirTok, ok := p.parseDirective() // consume @group
	if !ok {
		return nil, Token{}, false
	}
	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after @group, got %s", tok.Type), tok.Line, tok.Column)
		return nil, Token{}, false
	}
	p.advance() // consume '{'

	g := &ast.Declarations{}
	uses := p.parseDeclarations(g)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close @group, got %s", tok.Type), tok.Line, tok.Column)
		return nil, Token{}, false
	}
	p.advance() // consume '}'

	// Keep the group (so its children still count for completeness) but flag the
	// unsupported import.
	if len(uses) > 0 {
		p.addError(ast.ErrorSyntax, "@use is not allowed inside @group", dirTok.Line, dirTok.Column)
	}
	return g, dirTok, true
}

func (p *parser) parseDirectionDecl(d *ast.Declarations, propTok Token) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected 'vertical' or 'horizontal' for direction, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	tok := p.advance()
	if tok.Value != "vertical" && tok.Value != "horizontal" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid direction '%s', expected 'vertical' or 'horizontal'", tok.Value), tok.Line, tok.Column)
		return
	}
	if d.Direction != nil {
		p.addError(ast.ErrorSyntax, "Duplicate layout property 'direction'", propTok.Line, propTok.Column)
		return
	}
	dir := ast.Direction(tok.Value)
	d.Direction = &dir
}

func (p *parser) parseNumberDecl(dst **float64, name string, propTok Token) {
	if !p.check(TokenNumber) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected number value for %s, got %s", name, tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	tok := p.advance()
	v, err := strconv.ParseFloat(tok.Value, 64)
	if err != nil {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid %s value '%s', expected a number", name, tok.Value), tok.Line, tok.Column)
		return
	}
	if *dst != nil {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Duplicate layout property '%s'", name), propTok.Line, propTok.Column)
		return
	}
	*dst = &v
}

func (p *parser) parseBoxDecl(d *ast.Declarations, propTok Token) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected 'none' or 'default' for box, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	tok := p.advance()
	var box ast.Box
	switch tok.Value {
	case "none":
		box = ast.BoxNone
	case "default":
		box = ast.BoxDefault
	default:
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid box '%s', expected 'none' or 'default'", tok.Value), tok.Line, tok.Column)
		return
	}
	if d.Box != nil {
		p.addError(ast.ErrorSyntax, "Duplicate layout property 'box'", propTok.Line, propTok.Column)
		return
	}
	d.Box = &box
}

// parseLabelPosDecl reads `label: top | bottom`, the position of a parent's
// reserved label strip. (In a node *body* `label:` is the text; here in @layout
// it is the strip's placement.)
func (p *parser) parseLabelPosDecl(d *ast.Declarations, propTok Token) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected 'top' or 'bottom' for label, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	tok := p.advance()
	var pos ast.LabelPosition
	switch tok.Value {
	case "top":
		pos = ast.LabelTop
	case "bottom":
		pos = ast.LabelBottom
	default:
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid label position '%s', expected 'top' or 'bottom'", tok.Value), tok.Line, tok.Column)
		return
	}
	if d.LabelPos != nil {
		p.addError(ast.ErrorSyntax, "Duplicate layout property 'label'", propTok.Line, propTok.Column)
		return
	}
	d.LabelPos = &pos
}

// parseAnchorPosition reads `anchor name: [x, y]`.
func (p *parser) parseAnchorPosition(d *ast.Declarations) {
	p.advance() // consume 'anchor'
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected anchor name after 'anchor', got %s", tok.Type), tok.Line, tok.Column)
		return
	}
	nameTok := p.advance()
	name := nameTok.Value

	if !p.check(TokenColon) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ':' after anchor '%s', got %s", name, tok.Type), tok.Line, tok.Column)
		p.skipUntilRecovery()
		return
	}
	p.advance() // consume ':'

	coord, ok := p.parseCoordinate()
	if !ok {
		return
	}
	if d.Anchors == nil {
		d.Anchors = map[string][2]float64{}
	}
	if _, dup := d.Anchors[name]; dup {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Duplicate anchor position '%s'", name), nameTok.Line, nameTok.Column)
		return
	}
	d.Anchors[name] = coord
}

func (p *parser) parseCoordinate() ([2]float64, bool) {
	var zero [2]float64

	if !p.check(TokenLBracket) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '[' to start coordinate array, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) && !p.check(TokenComma) && !p.isAtEnd() {
			p.advance()
		}
		return zero, false
	}
	p.advance() // consume '['

	x, ok := p.parseCoordinateValue("X")
	if !ok {
		return zero, false
	}

	if !p.check(TokenComma) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ',' between coordinates, got %s", tok.Type), tok.Line, tok.Column)
		p.skipUntilBracketOrComma()
		return zero, false
	}
	p.advance() // consume ','

	y, ok := p.parseCoordinateValue("Y")
	if !ok {
		return zero, false
	}

	if !p.check(TokenRBracket) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ']' to close coordinate array, got %s", tok.Type), tok.Line, tok.Column)
		p.skipUntilBracketOrComma()
		return zero, false
	}
	p.advance() // consume ']'

	return [2]float64{x, y}, true
}

// parseCoordinateValue reads one numeric coordinate. Range is checked by the
// validator, not here.
func (p *parser) parseCoordinateValue(axis string) (float64, bool) {
	if !p.check(TokenNumber) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected number for %s coordinate, got %s", axis, tok.Type), tok.Line, tok.Column)
		p.skipUntilBracketOrComma()
		return 0, false
	}
	tok := p.advance()
	v, err := strconv.ParseFloat(tok.Value, 64)
	if err != nil {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid %s coordinate '%s', expected a number", axis, tok.Value), tok.Line, tok.Column)
		p.skipUntilBracketOrComma()
		return 0, false
	}
	return v, true
}

// --- arrows ---

// isArrowStatement looks ahead for the pattern
// IDENTIFIER (DOT IDENTIFIER)* (HASH IDENTIFIER)? ARROW without consuming any
// tokens.
func (p *parser) isArrowStatement() bool {
	if !p.check(TokenIdentifier) {
		return false
	}
	pos := p.current
	for pos < len(p.tokens) && p.tokens[pos].Type == TokenIdentifier {
		pos++
		if pos < len(p.tokens) && p.tokens[pos].Type == TokenArrow {
			return true
		}
		if pos < len(p.tokens) && p.tokens[pos].Type == TokenDot {
			pos++
			continue
		}
		if pos < len(p.tokens) && p.tokens[pos].Type == TokenHash {
			pos++ // skip hash
			if pos < len(p.tokens) && p.tokens[pos].Type == TokenIdentifier {
				pos++ // skip anchor id
				if pos < len(p.tokens) && p.tokens[pos].Type == TokenArrow {
					return true
				}
			}
			return false
		}
		if pos < len(p.tokens) && p.tokens[pos].Type == TokenLBrace {
			return false
		}
		break
	}
	return false
}

func (p *parser) parseArrow() (ast.Arrow, bool) {
	if !p.check(TokenIdentifier) {
		if !p.isAtEnd() {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected arrow source identifier, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
		}
		return ast.Arrow{}, false
	}

	source, ok := p.parseRefWithAnchor()
	if !ok {
		return ast.Arrow{}, false
	}

	if !p.check(TokenArrow) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '-->' arrow operator after source '%s', got %s", source, tok.Type), tok.Line, tok.Column)
		p.skipUntilNodeOrEOF()
		return ast.Arrow{}, false
	}
	p.advance() // consume '-->'

	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected arrow target identifier after '-->', got %s", tok.Type), tok.Line, tok.Column)
		p.skipUntilNodeOrEOF()
		return ast.Arrow{}, false
	}

	target, ok := p.parseRefWithAnchor()
	if !ok {
		return ast.Arrow{}, false
	}

	return ast.Arrow{Source: source, Target: target}, true
}

// parseDottedPath reads IDENTIFIER (DOT IDENTIFIER)* and joins it with dots.
func (p *parser) parseDottedPath() (string, bool) {
	if !p.check(TokenIdentifier) {
		return "", false
	}
	parts := []string{p.advance().Value}
	for p.check(TokenDot) {
		p.advance() // consume '.'
		if !p.check(TokenIdentifier) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected identifier after '.', got %s", tok.Type), tok.Line, tok.Column)
			return strings.Join(parts, "."), true
		}
		parts = append(parts, p.advance().Value)
	}
	return strings.Join(parts, "."), true
}

func (p *parser) parseRefWithAnchor() (string, bool) {
	nodePath, ok := p.parseDottedPath()
	if !ok {
		return "", false
	}
	if p.check(TokenHash) {
		p.advance() // consume '#'
		if !p.check(TokenIdentifier) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected anchor identifier after '#', got %s", tok.Type), tok.Line, tok.Column)
			return nodePath, true
		}
		return nodePath + "#" + p.advance().Value, true
	}
	return nodePath, true
}

// --- token cursor helpers ---

func (p *parser) check(tt TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tt
}

func (p *parser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *parser) peek() Token { return p.tokens[p.current] }

func (p *parser) peekNext() *Token {
	if p.current+1 >= len(p.tokens) {
		return nil
	}
	return &p.tokens[p.current+1]
}

func (p *parser) previous() Token { return p.tokens[p.current-1] }

func (p *parser) isAtEnd() bool {
	return p.current >= len(p.tokens) || p.peek().Type == TokenEOF
}

func (p *parser) addError(t ast.ErrorType, msg string, line, col int) {
	p.errors = append(p.errors, ast.Error{Type: t, Message: msg, Line: line, Column: col})
}

func (p *parser) skipUntilNodeOrEOF() {
	for !p.isAtEnd() && !p.check(TokenIdentifier) {
		p.advance()
	}
}

func (p *parser) skipUntilRecovery() {
	for !p.isAtEnd() && !p.check(TokenColon) && !p.check(TokenIdentifier) && !p.check(TokenRBrace) {
		p.advance()
	}
}

func (p *parser) skipUntilBracketOrComma() {
	for !p.isAtEnd() && !p.check(TokenRBracket) && !p.check(TokenComma) && !p.check(TokenRBrace) {
		p.advance()
	}
}

// skipBalancedBlock skips a `{ … }` block (with nesting) after an unrecognised
// directive, so parsing can resume cleanly.
func (p *parser) skipBalancedBlock() {
	if !p.check(TokenLBrace) {
		return
	}
	depth := 0
	for !p.isAtEnd() {
		switch p.peek().Type {
		case TokenLBrace:
			depth++
		case TokenRBrace:
			depth--
			if depth == 0 {
				p.advance()
				return
			}
		}
		p.advance()
	}
}
