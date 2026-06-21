package parser

import (
	"reflect"
	"testing"
)

func mustTokenize(t *testing.T, input string) []Token {
	t.Helper()
	toks, err := Tokenize(input)
	if err != nil {
		t.Fatalf("Tokenize(%q) unexpected error: %v", input, err)
	}
	return toks
}

func tokenTypes(toks []Token) []TokenType {
	out := make([]TokenType, len(toks))
	for i, tok := range toks {
		out[i] = tok.Type
	}
	return out
}

func TestTokenizeSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []TokenType
	}{
		{"empty", "", []TokenType{TokenEOF}},
		{"comment only", "# just a comment", []TokenType{TokenEOF}},
		{
			"single node",
			"a {}",
			[]TokenType{TokenIdentifier, TokenLBrace, TokenRBrace, TokenEOF},
		},
		{
			"structural punctuation",
			"{ } [ ] : , .",
			[]TokenType{TokenLBrace, TokenRBrace, TokenLBracket, TokenRBracket, TokenColon, TokenComma, TokenDot, TokenEOF},
		},
		{
			"arrow",
			"a --> b",
			[]TokenType{TokenIdentifier, TokenArrow, TokenIdentifier, TokenEOF},
		},
		{
			"at directive",
			"@layout {",
			[]TokenType{TokenAt, TokenIdentifier, TokenLBrace, TokenEOF},
		},
		{
			"semicolon is a separator, not a token",
			"a; b",
			[]TokenType{TokenIdentifier, TokenIdentifier, TokenEOF},
		},
		{
			"hash is anchor when not preceded by whitespace",
			"n1#a",
			[]TokenType{TokenIdentifier, TokenHash, TokenIdentifier, TokenEOF},
		},
		{
			"hash after whitespace is a comment",
			"x #anchor",
			[]TokenType{TokenIdentifier, TokenEOF},
		},
		{
			"newlines are tokens",
			"a\nb",
			[]TokenType{TokenIdentifier, TokenNewline, TokenIdentifier, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenTypes(mustTokenize(t, tt.input))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize(%q) types = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokenizeValues(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  TokenType
		wantValue string
	}{
		{"plain string", `"Hello World"`, TokenString, "Hello World"},
		{"escaped newline", `"a\nb"`, TokenString, "a\nb"},
		{"escaped tab and quote", `"x\t\"y\""`, TokenString, "x\t\"y\""},
		{"integer", "5", TokenNumber, "5"},
		{"decimal", "0.75", TokenNumber, "0.75"},
		{"identifier with underscore and digits", "user_service2", TokenIdentifier, "user_service2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks := mustTokenize(t, tt.input)
			if len(toks) != 2 { // value token + EOF
				t.Fatalf("Tokenize(%q) = %d tokens, want 2", tt.input, len(toks))
			}
			if toks[0].Type != tt.wantType || toks[0].Value != tt.wantValue {
				t.Errorf("Tokenize(%q)[0] = {%s %q}, want {%s %q}", tt.input, toks[0].Type, toks[0].Value, tt.wantType, tt.wantValue)
			}
		})
	}
}

func TestTokenizeColors(t *testing.T) {
	seqTests := []struct {
		name  string
		input string
		want  []TokenType
	}{
		{
			"colour value after a property",
			"borderColor: #ff0000",
			[]TokenType{TokenIdentifier, TokenColon, TokenColor, TokenEOF},
		},
		{
			"short colour at line start",
			"#abc",
			[]TokenType{TokenColor, TokenEOF},
		},
		{
			"colour glued to a brace boundary",
			"{ #fff }",
			[]TokenType{TokenLBrace, TokenColor, TokenRBrace, TokenEOF},
		},
		{
			"comment with a space stays a comment",
			"# ff0000 is red",
			[]TokenType{TokenEOF},
		},
		{
			"hash run trailed by a word is a comment",
			"#deadbeefxyz note",
			[]TokenType{TokenEOF},
		},
		{
			"anchor hash is unaffected even when the name is hex",
			"n1#abc",
			[]TokenType{TokenIdentifier, TokenHash, TokenIdentifier, TokenEOF},
		},
	}
	for _, tt := range seqTests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenTypes(mustTokenize(t, tt.input))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize(%q) types = %v, want %v", tt.input, got, tt.want)
			}
		})
	}

	// The colour token carries the full "#rrggbb" value (length is the validator's
	// concern, so even an odd length tokenizes for a better error downstream).
	valueTests := []struct{ input, want string }{
		{"#ff0000", "#ff0000"},
		{"#abc", "#abc"},
		{"#11223344", "#11223344"},
		{"#ff", "#ff"},
	}
	for _, tt := range valueTests {
		toks := mustTokenize(t, tt.input)
		if len(toks) != 2 || toks[0].Type != TokenColor || toks[0].Value != tt.want {
			t.Errorf("Tokenize(%q) = %+v, want a single COLOR %q", tt.input, toks, tt.want)
		}
	}
}

func TestTokenizePositions(t *testing.T) {
	toks := mustTokenize(t, "ab cd")
	want := []Token{
		{Type: TokenIdentifier, Value: "ab", Line: 1, Column: 1},
		{Type: TokenIdentifier, Value: "cd", Line: 1, Column: 4},
		{Type: TokenEOF, Value: "", Line: 1, Column: 6},
	}
	if !reflect.DeepEqual(toks, want) {
		t.Errorf("Tokenize positions = %+v, want %+v", toks, want)
	}

	// A node body spanning lines tracks line numbers across the newline.
	multi := mustTokenize(t, "a {\n  label: \"hi\"\n}")
	var labelTok Token
	for _, tok := range multi {
		if tok.Type == TokenIdentifier && tok.Value == "label" {
			labelTok = tok
		}
	}
	if labelTok.Line != 2 || labelTok.Column != 3 {
		t.Errorf("label token at line %d col %d, want line 2 col 3", labelTok.Line, labelTok.Column)
	}
}

func TestTokenizeErrors(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMsg    string
		wantLine   int
		wantColumn int
	}{
		{"unexpected character", "a $ b", "Unexpected character '$'", 1, 3},
		{"unterminated string", `"no end`, "Unterminated string", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Tokenize(tt.input)
			if err == nil {
				t.Fatalf("Tokenize(%q) = nil error, want one", tt.input)
			}
			if err.Message != tt.wantMsg || err.Line != tt.wantLine || err.Column != tt.wantColumn {
				t.Errorf("error = {%q %d %d}, want {%q %d %d}", err.Message, err.Line, err.Column, tt.wantMsg, tt.wantLine, tt.wantColumn)
			}
		})
	}
}
