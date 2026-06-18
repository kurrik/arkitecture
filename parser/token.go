package parser

// TokenType enumerates the lexical tokens of the DSL. The string values match
// the names used in parser error messages (e.g. "got NEWLINE").
type TokenType string

const (
	TokenIdentifier TokenType = "IDENTIFIER"
	TokenString     TokenType = "STRING"
	TokenNumber     TokenType = "NUMBER"
	TokenLBrace     TokenType = "LBRACE"   // {
	TokenRBrace     TokenType = "RBRACE"   // }
	TokenLBracket   TokenType = "LBRACKET" // [
	TokenRBracket   TokenType = "RBRACKET" // ]
	TokenColon      TokenType = "COLON"    // :
	TokenComma      TokenType = "COMMA"    // ,
	TokenArrow      TokenType = "ARROW"    // -->
	TokenDot        TokenType = "DOT"      // .
	TokenHash       TokenType = "HASH"     // #
	TokenAt         TokenType = "AT"       // @ (introduces a directive, e.g. @layout)
	TokenEOF        TokenType = "EOF"
	TokenNewline    TokenType = "NEWLINE"
)

// Token is a single lexical token with its 1-based source position.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}
