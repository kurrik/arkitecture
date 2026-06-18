// Package parser turns Arkitecture DSL text into an ast.Document. It is a
// hand-written tokenizer plus a recursive-descent parser that collects all
// syntax and range errors as data rather than failing fast.
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
	tokens  []Token
	current int
	errors  []ast.Error
}

func (p *parser) parseDocument() ast.ParseResult {
	nodes := p.parseNodes()   // phase 1: nodes
	arrows := p.parseArrows() // phase 2: arrows
	doc := &ast.Document{Nodes: nodes, Arrows: arrows}
	if len(p.errors) > 0 {
		return ast.ParseResult{Success: false, Document: doc, Errors: p.errors}
	}
	return ast.ParseResult{Success: true, Document: doc}
}

func (p *parser) parseNodes() []*ast.ContainerNode {
	var nodes []*ast.ContainerNode
	for !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		// Arrow statements end the node phase.
		if p.isArrowStatement() {
			break
		}
		if node := p.parseNode(); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (p *parser) parseNode() *ast.ContainerNode {
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

	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after node id '%s', got %s", id, tok.Type), tok.Line, tok.Column)
		p.skipUntilNodeOrEOF()
		return nil
	}
	p.advance() // consume '{'

	node := &ast.ContainerNode{ID: id}
	p.parseNodeContent(node)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close node '%s', got %s", id, tok.Type), tok.Line, tok.Column)
		return node
	}
	p.advance() // consume '}'
	return node
}

func (p *parser) parseNodeContent(node *ast.ContainerNode) {
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}

		if p.check(TokenIdentifier) {
			// IDENTIFIER followed by '{' is a nested node; otherwise a property.
			if nx := p.peekNext(); nx != nil && nx.Type == TokenLBrace {
				if child := p.parseNode(); child != nil {
					node.Children = append(node.Children, child)
				}
				continue
			}
			p.parseProperty(node)
		} else if p.check(TokenGroup) {
			if g := p.parseGroup(); g != nil {
				node.Children = append(node.Children, g)
			}
		} else {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected property name, nested node, or group, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
		}
	}
}

func (p *parser) parseProperty(node *ast.ContainerNode) {
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
	case "direction":
		p.parseDirection(node)
	case "size":
		p.parseSize(node)
	case "margin":
		p.parseMargin(node)
	case "box":
		p.parseBox(node)
	case "anchors":
		p.parseAnchors(node)
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

func (p *parser) parseDirection(node *ast.ContainerNode) {
	dir, ok := p.parseDirectionValue()
	if ok {
		node.Direction = dir
	}
}

func (p *parser) parseSize(node *ast.ContainerNode) {
	if !p.check(TokenNumber) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected number value for size, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	sizeTok := p.advance()
	v, err := strconv.ParseFloat(sizeTok.Value, 64)
	if err != nil {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid size value '%s', expected a number", sizeTok.Value), sizeTok.Line, sizeTok.Column)
		return
	}
	if v < 0.0 || v > 1.0 {
		p.addError(ast.ErrorConstraint, fmt.Sprintf("Size value %s is out of range, expected 0.0-1.0", formatNum(v)), sizeTok.Line, sizeTok.Column)
		return
	}
	node.Size = &v
}

func (p *parser) parseMargin(node *ast.ContainerNode) {
	if !p.check(TokenNumber) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected number value for margin, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	marginTok := p.advance()
	v, err := strconv.ParseFloat(marginTok.Value, 64)
	if err != nil {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid margin value '%s', expected a number", marginTok.Value), marginTok.Line, marginTok.Column)
		return
	}
	if v < 0.0 {
		p.addError(ast.ErrorConstraint, fmt.Sprintf("Margin value %s is out of range, expected >= 0.0", formatNum(v)), marginTok.Line, marginTok.Column)
		return
	}
	node.Margin = &v
}

func (p *parser) parseBox(node *ast.ContainerNode) {
	if !p.check(TokenIdentifier) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected 'none' or 'default' for box, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	tok := p.advance()
	switch tok.Value {
	case "none":
		node.Box = ast.BoxNone
	case "default":
		node.Box = ast.BoxDefault
	default:
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid box '%s', expected 'none' or 'default'", tok.Value), tok.Line, tok.Column)
	}
}

func (p *parser) parseAnchors(node *ast.ContainerNode) {
	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' to start anchors object, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return
	}
	p.advance() // consume '{'

	anchors := map[string][2]float64{}
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		if !p.check(TokenIdentifier) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected anchor identifier, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
			continue
		}

		idTok := p.advance()
		anchorID := idTok.Value
		if _, dup := anchors[anchorID]; dup {
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Duplicate anchor ID '%s'", anchorID), idTok.Line, idTok.Column)
		}

		if !p.check(TokenColon) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ':' after anchor ID '%s', got %s", anchorID, tok.Type), tok.Line, tok.Column)
			p.skipUntilRecovery()
			continue
		}
		p.advance() // consume ':'

		if coord, ok := p.parseCoordinate(); ok {
			anchors[anchorID] = coord
		}

		if p.check(TokenComma) {
			p.advance()
		}
	}

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close anchors object, got %s", tok.Type), tok.Line, tok.Column)
		return
	}
	p.advance() // consume '}'

	if len(anchors) > 0 {
		node.Anchors = anchors
	}
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

// parseCoordinateValue reads one numeric coordinate. An out-of-range value is
// reported as a constraint error but still returned (ok stays true), matching
// the original behaviour.
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
	if v < 0.0 || v > 1.0 {
		p.addError(ast.ErrorConstraint, fmt.Sprintf("%s coordinate %s is out of range, expected 0.0-1.0", axis, formatNum(v)), tok.Line, tok.Column)
	}
	return v, true
}

func (p *parser) parseGroup() *ast.GroupNode {
	if !p.check(TokenGroup) {
		return nil
	}
	p.advance() // consume 'group'

	if !p.check(TokenLBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '{' after 'group', got %s", tok.Type), tok.Line, tok.Column)
		p.skipUntilNodeOrEOF()
		return nil
	}
	p.advance() // consume '{'

	group := &ast.GroupNode{}
	p.parseGroupContent(group)

	if !p.check(TokenRBrace) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected '}' to close group, got %s", tok.Type), tok.Line, tok.Column)
		return group
	}
	p.advance() // consume '}'
	return group
}

func (p *parser) parseGroupContent(group *ast.GroupNode) {
	for !p.check(TokenRBrace) && !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}

		if p.check(TokenIdentifier) {
			ahead := p.peek()
			if nx := p.peekNext(); nx != nil && nx.Type == TokenLBrace {
				if child := p.parseNode(); child != nil {
					group.Children = append(group.Children, child)
				}
				continue
			}
			if ahead.Value == "direction" {
				p.parseGroupProperty(group)
			} else {
				tok := p.peek()
				p.addError(ast.ErrorSyntax, fmt.Sprintf("Groups can only have 'direction' property, got '%s'", ahead.Value), tok.Line, tok.Column)
				p.advance() // skip property name
				if p.check(TokenColon) {
					p.advance()
					if !p.check(TokenRBrace) && !p.isAtEnd() {
						p.advance() // skip value
					}
				}
			}
		} else if p.check(TokenGroup) {
			if ng := p.parseGroup(); ng != nil {
				group.Children = append(group.Children, ng)
			}
		} else {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected nested node or group in group, got %s", tok.Type), tok.Line, tok.Column)
			p.advance()
		}
	}
}

func (p *parser) parseGroupProperty(group *ast.GroupNode) {
	propTok := p.advance()
	if propTok.Value != "direction" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Groups can only have 'direction' property, got '%s'", propTok.Value), propTok.Line, propTok.Column)
		return
	}
	if !p.check(TokenColon) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected ':' after 'direction', got %s", tok.Type), tok.Line, tok.Column)
		p.skipUntilRecovery()
		return
	}
	p.advance() // consume ':'

	if dir, ok := p.parseDirectionValue(); ok {
		group.Direction = dir
	}
}

// parseDirectionValue reads a "vertical"|"horizontal" string value and reports
// the appropriate error otherwise. It is shared by node and group direction.
func (p *parser) parseDirectionValue() (ast.Direction, bool) {
	if !p.check(TokenString) {
		tok := p.peek()
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected string value for direction, got %s", tok.Type), tok.Line, tok.Column)
		if !p.check(TokenRBrace) {
			p.advance()
		}
		return ast.DirectionUnset, false
	}
	tok := p.advance()
	if tok.Value != "vertical" && tok.Value != "horizontal" {
		p.addError(ast.ErrorSyntax, fmt.Sprintf("Invalid direction '%s', expected 'vertical' or 'horizontal'", tok.Value), tok.Line, tok.Column)
		return ast.DirectionUnset, false
	}
	return ast.Direction(tok.Value), true
}

// isArrowStatement looks ahead for the pattern
// IDENTIFIER (DOT (IDENTIFIER|GROUP))* (HASH IDENTIFIER)? ARROW without
// consuming any tokens.
func (p *parser) isArrowStatement() bool {
	if !p.check(TokenIdentifier) {
		return false
	}
	pos := p.current
	for pos < len(p.tokens) && (p.tokens[pos].Type == TokenIdentifier || p.tokens[pos].Type == TokenGroup) {
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

func (p *parser) parseArrows() []ast.Arrow {
	var arrows []ast.Arrow
	for !p.isAtEnd() {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		if arrow, ok := p.parseArrow(); ok {
			arrows = append(arrows, arrow)
		}
	}
	return arrows
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

	source, ok := p.parseTargetWithAnchor()
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

	target, ok := p.parseTargetWithAnchor()
	if !ok {
		return ast.Arrow{}, false
	}

	return ast.Arrow{Source: source, Target: target}, true
}

func (p *parser) parseNodePath() (string, bool) {
	if !p.check(TokenIdentifier) {
		return "", false
	}
	parts := []string{p.advance().Value}
	for p.check(TokenDot) {
		p.advance() // consume '.'
		if !p.check(TokenIdentifier) && !p.check(TokenGroup) {
			tok := p.peek()
			p.addError(ast.ErrorSyntax, fmt.Sprintf("Expected identifier after '.', got %s", tok.Type), tok.Line, tok.Column)
			return strings.Join(parts, "."), true
		}
		parts = append(parts, p.advance().Value)
	}
	return strings.Join(parts, "."), true
}

func (p *parser) parseTargetWithAnchor() (string, bool) {
	nodePath, ok := p.parseNodePath()
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

func formatNum(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
