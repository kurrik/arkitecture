package parser

import (
	"fmt"
	"strings"
)

// TokenizerError is a lexical error with its source position.
type TokenizerError struct {
	Message string
	Line    int
	Column  int
}

func (e *TokenizerError) Error() string {
	return fmt.Sprintf("%s at line %d, column %d", e.Message, e.Line, e.Column)
}

// tokenizer scans DSL source over a slice of runes, tracking 1-based line and
// column positions.
type tokenizer struct {
	input    []rune
	position int
	line     int
	column   int
}

// Tokenize scans input into a slice of tokens terminated by a TokenEOF. It
// returns a *TokenizerError on the first lexical error (an unexpected character
// or an unterminated string).
func Tokenize(input string) ([]Token, *TokenizerError) {
	t := &tokenizer{input: []rune(input), position: 0, line: 1, column: 1}
	var tokens []Token
	for !t.isAtEnd() {
		tok, err := t.nextToken()
		if err != nil {
			return nil, err
		}
		if tok != nil {
			tokens = append(tokens, *tok)
		}
	}
	tokens = append(tokens, Token{Type: TokenEOF, Value: "", Line: t.line, Column: t.column})
	return tokens, nil
}

// nextToken returns the next significant token, skipping whitespace and
// comments. It returns (nil, nil) when only trailing whitespace/comments remain.
func (t *tokenizer) nextToken() (*Token, *TokenizerError) {
	for {
		t.skipWhitespace()
		if t.isAtEnd() {
			return nil, nil
		}

		startLine := t.line
		startColumn := t.column
		ch := t.peek(0)

		// '#' is a comment when it starts a line or follows whitespace;
		// otherwise it is a hash token used by anchor references.
		if ch == '#' {
			if t.column == 1 || t.isPrecedingWhitespace() {
				t.skipComment()
				continue
			}
			t.advance()
			return &Token{Type: TokenHash, Value: "#", Line: startLine, Column: startColumn}, nil
		}

		if ch == '\n' {
			t.advance()
			return &Token{Type: TokenNewline, Value: "\n", Line: startLine, Column: startColumn}, nil
		}

		if ch == '-' && t.peek(1) == '-' && t.peek(2) == '>' {
			t.advance()
			t.advance()
			t.advance()
			return &Token{Type: TokenArrow, Value: "-->", Line: startLine, Column: startColumn}, nil
		}

		switch ch {
		case '{':
			t.advance()
			return &Token{Type: TokenLBrace, Value: "{", Line: startLine, Column: startColumn}, nil
		case '}':
			t.advance()
			return &Token{Type: TokenRBrace, Value: "}", Line: startLine, Column: startColumn}, nil
		case '[':
			t.advance()
			return &Token{Type: TokenLBracket, Value: "[", Line: startLine, Column: startColumn}, nil
		case ']':
			t.advance()
			return &Token{Type: TokenRBracket, Value: "]", Line: startLine, Column: startColumn}, nil
		case ':':
			t.advance()
			return &Token{Type: TokenColon, Value: ":", Line: startLine, Column: startColumn}, nil
		case ',':
			t.advance()
			return &Token{Type: TokenComma, Value: ",", Line: startLine, Column: startColumn}, nil
		case '.':
			t.advance()
			return &Token{Type: TokenDot, Value: ".", Line: startLine, Column: startColumn}, nil
		}

		if ch == '"' {
			return t.scanString(startLine, startColumn)
		}
		if isDigit(ch) {
			return t.scanNumber(startLine, startColumn), nil
		}
		if isAlpha(ch) {
			return t.scanIdentifier(startLine, startColumn), nil
		}

		return nil, &TokenizerError{
			Message: fmt.Sprintf("Unexpected character '%c'", ch),
			Line:    startLine,
			Column:  startColumn,
		}
	}
}

func (t *tokenizer) scanString(startLine, startColumn int) (*Token, *TokenizerError) {
	t.advance() // opening quote
	var b strings.Builder

	for !t.isAtEnd() && t.peek(0) != '"' {
		switch t.peek(0) {
		case '\n':
			t.advance() // advance() tracks the line/column change
			b.WriteRune('\n')
		case '\\':
			t.advance() // consume backslash
			if t.isAtEnd() {
				return nil, &TokenizerError{Message: "Unterminated string escape", Line: startLine, Column: startColumn}
			}
			switch esc := t.advance(); esc {
			case 'n':
				b.WriteRune('\n')
			case 't':
				b.WriteRune('\t')
			case 'r':
				b.WriteRune('\r')
			case '\\':
				b.WriteRune('\\')
			case '"':
				b.WriteRune('"')
			default:
				b.WriteRune(esc)
			}
		default:
			b.WriteRune(t.advance())
		}
	}

	if t.isAtEnd() {
		return nil, &TokenizerError{Message: "Unterminated string", Line: startLine, Column: startColumn}
	}
	t.advance() // closing quote
	return &Token{Type: TokenString, Value: b.String(), Line: startLine, Column: startColumn}, nil
}

func (t *tokenizer) scanNumber(startLine, startColumn int) *Token {
	var b strings.Builder
	for isDigit(t.peek(0)) {
		b.WriteRune(t.advance())
	}
	if t.peek(0) == '.' && isDigit(t.peek(1)) {
		b.WriteRune(t.advance()) // the '.'
		for isDigit(t.peek(0)) {
			b.WriteRune(t.advance())
		}
	}
	return &Token{Type: TokenNumber, Value: b.String(), Line: startLine, Column: startColumn}
}

func (t *tokenizer) scanIdentifier(startLine, startColumn int) *Token {
	var b strings.Builder
	for isAlphaNumeric(t.peek(0)) || t.peek(0) == '_' {
		b.WriteRune(t.advance())
	}
	text := b.String()
	typ := TokenIdentifier
	if text == "group" {
		typ = TokenGroup
	}
	return &Token{Type: typ, Value: text, Line: startLine, Column: startColumn}
}

func (t *tokenizer) skipWhitespace() {
	for !t.isAtEnd() {
		switch t.peek(0) {
		case ' ', '\r', '\t':
			t.advance()
		default:
			return
		}
	}
}

func (t *tokenizer) skipComment() {
	for !t.isAtEnd() && t.peek(0) != '\n' {
		t.advance()
	}
}

// peek returns the rune at the current position plus offset, or 0 past the end.
func (t *tokenizer) peek(offset int) rune {
	pos := t.position + offset
	if pos < 0 || pos >= len(t.input) {
		return 0
	}
	return t.input[pos]
}

func (t *tokenizer) advance() rune {
	if t.isAtEnd() {
		return 0
	}
	ch := t.input[t.position]
	t.position++
	if ch == '\n' {
		t.line++
		t.column = 1
	} else {
		t.column++
	}
	return ch
}

func (t *tokenizer) isAtEnd() bool {
	return t.position >= len(t.input)
}

func (t *tokenizer) isPrecedingWhitespace() bool {
	if t.position == 0 {
		return true
	}
	switch t.input[t.position-1] {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func isDigit(c rune) bool { return c >= '0' && c <= '9' }

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isAlphaNumeric(c rune) bool { return isAlpha(c) || isDigit(c) }
