/**
 * Tokenizer for Arkitecture DSL
 */

export enum TokenType {
  IDENTIFIER = 'IDENTIFIER',
  STRING = 'STRING',
  NUMBER = 'NUMBER',
  LBRACE = 'LBRACE', // {
  RBRACE = 'RBRACE', // }
  LBRACKET = 'LBRACKET', // [
  RBRACKET = 'RBRACKET', // ]
  COLON = 'COLON', // :
  COMMA = 'COMMA', // ,
  ARROW = 'ARROW', // -->
  DOT = 'DOT', // .
  HASH = 'HASH', // #
  GROUP = 'GROUP', // group keyword
  EOF = 'EOF',
  NEWLINE = 'NEWLINE',
}

export interface Token {
  type: TokenType;
  value: string;
  line: number;
  column: number;
}

export class TokenizerError extends Error {
  constructor(
    message: string,
    public line: number,
    public column: number
  ) {
    super(`${message} at line ${line}, column ${column}`);
    this.name = 'TokenizerError';
  }
}

export class Tokenizer {
  private input: string;
  private position: number;
  private line: number;
  private column: number;

  constructor(input: string) {
    this.input = input;
    this.position = 0;
    this.line = 1;
    this.column = 1;
  }

  tokenize(): Token[] {
    const tokens: Token[] = [];
    
    while (!this.isAtEnd()) {
      const token = this.nextToken();
      if (token) {
        tokens.push(token);
      }
    }
    
    tokens.push({
      type: TokenType.EOF,
      value: '',
      line: this.line,
      column: this.column,
    });
    
    return tokens;
  }

  private nextToken(): Token | null {
    this.skipWhitespace();
    
    if (this.isAtEnd()) {
      return null;
    }
    
    const startLine = this.line;
    const startColumn = this.column;
    const char = this.peek();
    
    // Comments vs Hash character
    if (char === '#') {
      // If the # is preceded by whitespace or start of line, it's a comment
      // Otherwise, it's a hash token for anchor references
      if (this.column === 1 || this.isPrecedingWhitespace()) {
        this.skipComment();
        return this.nextToken();
      } else {
        this.advance();
        return {
          type: TokenType.HASH,
          value: '#',
          line: startLine,
          column: startColumn,
        };
      }
    }
    
    // Newlines
    if (char === '\n') {
      this.advance();
      return {
        type: TokenType.NEWLINE,
        value: '\n',
        line: startLine,
        column: startColumn,
      };
    }
    
    // Arrow operator
    if (char === '-' && this.peek(1) === '-' && this.peek(2) === '>') {
      this.advance();
      this.advance();
      this.advance();
      return {
        type: TokenType.ARROW,
        value: '-->',
        line: startLine,
        column: startColumn,
      };
    }
    
    // Single character tokens
    switch (char) {
      case '{':
        this.advance();
        return {
          type: TokenType.LBRACE,
          value: '{',
          line: startLine,
          column: startColumn,
        };
      case '}':
        this.advance();
        return {
          type: TokenType.RBRACE,
          value: '}',
          line: startLine,
          column: startColumn,
        };
      case '[':
        this.advance();
        return {
          type: TokenType.LBRACKET,
          value: '[',
          line: startLine,
          column: startColumn,
        };
      case ']':
        this.advance();
        return {
          type: TokenType.RBRACKET,
          value: ']',
          line: startLine,
          column: startColumn,
        };
      case ':':
        this.advance();
        return {
          type: TokenType.COLON,
          value: ':',
          line: startLine,
          column: startColumn,
        };
      case ',':
        this.advance();
        return {
          type: TokenType.COMMA,
          value: ',',
          line: startLine,
          column: startColumn,
        };
      case '.':
        this.advance();
        return {
          type: TokenType.DOT,
          value: '.',
          line: startLine,
          column: startColumn,
        };
    }
    
    // String literals
    if (char === '"') {
      return this.scanString(startLine, startColumn);
    }
    
    // Numbers
    if (this.isDigit(char)) {
      return this.scanNumber(startLine, startColumn);
    }
    
    // Identifiers and keywords
    if (this.isAlpha(char)) {
      return this.scanIdentifier(startLine, startColumn);
    }
    
    throw new TokenizerError(
      `Unexpected character '${char}'`,
      startLine,
      startColumn
    );
  }

  private scanString(startLine: number, startColumn: number): Token {
    this.advance(); // Skip opening quote
    const value: string[] = [];
    
    while (!this.isAtEnd() && this.peek() !== '"') {
      if (this.peek() === '\n') {
        this.advance();
        this.line++;
        this.column = 1;
        value.push('\n');
      } else if (this.peek() === '\\') {
        this.advance(); // Skip backslash
        if (this.isAtEnd()) {
          throw new TokenizerError(
            'Unterminated string escape',
            startLine,
            startColumn
          );
        }
        const escaped = this.advance();
        switch (escaped) {
          case 'n':
            value.push('\n');
            break;
          case 't':
            value.push('\t');
            break;
          case 'r':
            value.push('\r');
            break;
          case '\\':
            value.push('\\');
            break;
          case '"':
            value.push('"');
            break;
          default:
            value.push(escaped);
        }
      } else {
        value.push(this.advance());
      }
    }
    
    if (this.isAtEnd()) {
      throw new TokenizerError(
        'Unterminated string',
        startLine,
        startColumn
      );
    }
    
    this.advance(); // Skip closing quote
    
    return {
      type: TokenType.STRING,
      value: value.join(''),
      line: startLine,
      column: startColumn,
    };
  }

  private scanNumber(startLine: number, startColumn: number): Token {
    const value: string[] = [];
    
    while (this.isDigit(this.peek())) {
      value.push(this.advance());
    }
    
    // Handle decimal numbers
    if (this.peek() === '.' && this.isDigit(this.peek(1))) {
      value.push(this.advance()); // Add the '.'
      while (this.isDigit(this.peek())) {
        value.push(this.advance());
      }
    }
    
    return {
      type: TokenType.NUMBER,
      value: value.join(''),
      line: startLine,
      column: startColumn,
    };
  }

  private scanIdentifier(startLine: number, startColumn: number): Token {
    const value: string[] = [];
    
    while (this.isAlphaNumeric(this.peek()) || this.peek() === '_') {
      value.push(this.advance());
    }
    
    const text = value.join('');
    const type = text === 'group' ? TokenType.GROUP : TokenType.IDENTIFIER;
    
    return {
      type,
      value: text,
      line: startLine,
      column: startColumn,
    };
  }

  private skipWhitespace(): void {
    while (!this.isAtEnd()) {
      const char = this.peek();
      if (char === ' ' || char === '\r' || char === '\t') {
        this.advance();
      } else {
        break;
      }
    }
  }

  private skipComment(): void {
    while (!this.isAtEnd() && this.peek() !== '\n') {
      this.advance();
    }
  }

  private peek(offset: number = 0): string {
    const pos = this.position + offset;
    if (pos >= this.input.length) {
      return '\\0';
    }
    return this.input[pos];
  }

  private advance(): string {
    if (this.isAtEnd()) {
      return '\\0';
    }
    
    const char = this.input[this.position];
    this.position++;
    
    if (char === '\n') {
      this.line++;
      this.column = 1;
    } else {
      this.column++;
    }
    
    return char;
  }

  private isAtEnd(): boolean {
    return this.position >= this.input.length;
  }

  private isDigit(char: string): boolean {
    return char >= '0' && char <= '9';
  }

  private isAlpha(char: string): boolean {
    return (char >= 'a' && char <= 'z') ||
           (char >= 'A' && char <= 'Z') ||
           char === '_';
  }

  private isAlphaNumeric(char: string): boolean {
    return this.isAlpha(char) || this.isDigit(char);
  }

  private isPrecedingWhitespace(): boolean {
    if (this.position === 0) {
      return true;
    }
    const prevChar = this.input[this.position - 1];
    return prevChar === ' ' || prevChar === '\t' || prevChar === '\r' || prevChar === '\n';
  }
}