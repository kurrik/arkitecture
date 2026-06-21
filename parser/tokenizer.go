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

		// '#' is overloaded three ways. Glued to an identifier (`node#anchor`) it
		// is an anchor hash. Otherwise it introduces either a hex colour value
		// (`#ff0000`) or a comment — a complete hex run with a word boundary after
		// it is a colour; anything else (a space, a non-hex word) is a comment.
		if ch == '#' {
			if t.isPrecedingIdentChar() {
				t.advance()
				return &Token{Type: TokenHash, Value: "#", Line: startLine, Column: startColumn}, nil
			}
			if tok := t.scanColor(startLine, startColumn); tok != nil {
				return tok, nil
			}
			t.skipComment()
			continue
		}

		if ch == '\n' {
			t.advance()
			return &Token{Type: TokenNewline, Value: "\n", Line: startLine, Column: startColumn}, nil
		}

		if ch == '@' {
			t.advance()
			return &Token{Type: TokenAt, Value: "@", Line: startLine, Column: startColumn}, nil
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

// scanColor reads a hex colour token starting at the '#'. It returns nil (so the
// caller falls back to comment handling) unless the '#' is followed by a run of
// hex digits ending at a word boundary — e.g. `#ff0000`, `#abc`. A run trailed by
// a non-hex word char (`#fffg`) or no run at all (`# note`) is not a clean colour,
// so it stays a comment. The exact digit count (3/4/6/8) is the validator's check;
// here any boundary-terminated hex run becomes a TokenColor so a wrong length can
// be reported with a helpful message rather than silently swallowed as a comment.
func (t *tokenizer) scanColor(startLine, startColumn int) *Token {
	n := 1
	for isHexDigit(t.peek(n)) {
		n++
	}
	if n == 1 {
		return nil // '#' not followed by any hex digit
	}
	if isAlphaNumeric(t.peek(n)) {
		return nil // a non-hex word char follows the run: not a clean colour
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(t.advance())
	}
	return &Token{Type: TokenColor, Value: b.String(), Line: startLine, Column: startColumn}
}

func (t *tokenizer) scanIdentifier(startLine, startColumn int) *Token {
	var b strings.Builder
	for isAlphaNumeric(t.peek(0)) || t.peek(0) == '_' {
		b.WriteRune(t.advance())
	}
	return &Token{Type: TokenIdentifier, Value: b.String(), Line: startLine, Column: startColumn}
}

func (t *tokenizer) skipWhitespace() {
	// ';' is a purely cosmetic statement separator (it lets `@layout {a; b}`
	// read on one line), so it is skipped like ordinary whitespace.
	for !t.isAtEnd() {
		switch t.peek(0) {
		case ' ', '\r', '\t', ';':
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

// isPrecedingIdentChar reports whether the rune just before the cursor is an
// identifier char — the test that distinguishes an anchor hash (`node#a`, glued
// to an identifier) from a colour/comment `#` (preceded by space, ':', '{', …).
func (t *tokenizer) isPrecedingIdentChar() bool {
	return t.position > 0 && isAlphaNumeric(t.input[t.position-1])
}

func isDigit(c rune) bool { return c >= '0' && c <= '9' }

func isHexDigit(c rune) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isAlphaNumeric(c rune) bool { return isAlpha(c) || isDigit(c) }
