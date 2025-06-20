import { Tokenizer, TokenType, TokenizerError } from '../../src/parser/tokenizer';

describe('Tokenizer', () => {
  describe('Basic token recognition', () => {
    it('should tokenize identifiers', () => {
      const tokenizer = new Tokenizer('nodeId myNode');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(3); // 2 identifiers + EOF
      expect(tokens[0]).toEqual({
        type: TokenType.IDENTIFIER,
        value: 'nodeId',
        line: 1,
        column: 1,
      });
      expect(tokens[1]).toEqual({
        type: TokenType.IDENTIFIER,
        value: 'myNode',
        line: 1,
        column: 8,
      });
      expect(tokens[2].type).toBe(TokenType.EOF);
    });

    it('should recognize group keyword', () => {
      const tokenizer = new Tokenizer('group');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(2); // group + EOF
      expect(tokens[0]).toEqual({
        type: TokenType.GROUP,
        value: 'group',
        line: 1,
        column: 1,
      });
    });

    it('should tokenize structural characters', () => {
      const tokenizer = new Tokenizer('{}[]:.,x#y');
      const tokens = tokenizer.tokenize();
      
      const expectedTypes = [
        TokenType.LBRACE,
        TokenType.RBRACE,
        TokenType.LBRACKET,
        TokenType.RBRACKET,
        TokenType.COLON,
        TokenType.DOT,
        TokenType.COMMA,
        TokenType.IDENTIFIER, // x
        TokenType.HASH,
        TokenType.IDENTIFIER, // y
        TokenType.EOF,
      ];
      
      expect(tokens).toHaveLength(expectedTypes.length);
      tokens.forEach((token, index) => {
        if (index < expectedTypes.length - 1) {
          expect(token.type).toBe(expectedTypes[index]);
        }
      });
    });

    it('should tokenize arrow operator', () => {
      const tokenizer = new Tokenizer('-->');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(2); // arrow + EOF
      expect(tokens[0]).toEqual({
        type: TokenType.ARROW,
        value: '-->',
        line: 1,
        column: 1,
      });
    });
  });

  describe('String parsing', () => {
    it('should parse simple strings', () => {
      const tokenizer = new Tokenizer('"Hello World"');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(2); // string + EOF
      expect(tokens[0]).toEqual({
        type: TokenType.STRING,
        value: 'Hello World',
        line: 1,
        column: 1,
      });
    });

    it('should parse strings with escapes', () => {
      const tokenizer = new Tokenizer('"Hello\\nWorld\\t\\"Test\\""');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(2);
      expect(tokens[0]).toEqual({
        type: TokenType.STRING,
        value: 'Hello\nWorld\t"Test"',
        line: 1,
        column: 1,
      });
    });

    it('should handle empty strings', () => {
      const tokenizer = new Tokenizer('""');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(2);
      expect(tokens[0]).toEqual({
        type: TokenType.STRING,
        value: '',
        line: 1,
        column: 1,
      });
    });

    it('should throw error for unterminated strings', () => {
      const tokenizer = new Tokenizer('"unterminated');
      
      expect(() => tokenizer.tokenize()).toThrow(TokenizerError);
    });

    it('should throw error for unterminated escape', () => {
      const tokenizer = new Tokenizer('"test\\');
      
      expect(() => tokenizer.tokenize()).toThrow(TokenizerError);
    });
  });

  describe('Number parsing', () => {
    it('should parse integers', () => {
      const tokenizer = new Tokenizer('123 0 999');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(4); // 3 numbers + EOF
      expect(tokens[0]).toEqual({
        type: TokenType.NUMBER,
        value: '123',
        line: 1,
        column: 1,
      });
      expect(tokens[1]).toEqual({
        type: TokenType.NUMBER,
        value: '0',
        line: 1,
        column: 5,
      });
      expect(tokens[2]).toEqual({
        type: TokenType.NUMBER,
        value: '999',
        line: 1,
        column: 7,
      });
    });

    it('should parse decimal numbers', () => {
      const tokenizer = new Tokenizer('0.5 1.0 0.123');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(4); // 3 numbers + EOF
      expect(tokens[0]).toEqual({
        type: TokenType.NUMBER,
        value: '0.5',
        line: 1,
        column: 1,
      });
      expect(tokens[1]).toEqual({
        type: TokenType.NUMBER,
        value: '1.0',
        line: 1,
        column: 5,
      });
      expect(tokens[2]).toEqual({
        type: TokenType.NUMBER,
        value: '0.123',
        line: 1,
        column: 9,
      });
    });
  });

  describe('Comment handling', () => {
    it('should skip line comments', () => {
      const tokenizer = new Tokenizer('nodeId # this is a comment\nother');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(4); // nodeId + newline + other + EOF
      expect(tokens[0].type).toBe(TokenType.IDENTIFIER);
      expect(tokens[0].value).toBe('nodeId');
      expect(tokens[1].type).toBe(TokenType.NEWLINE);
      expect(tokens[2].type).toBe(TokenType.IDENTIFIER);
      expect(tokens[2].value).toBe('other');
    });

    it('should handle comments at end of file', () => {
      const tokenizer = new Tokenizer('nodeId # comment at end');
      const tokens = tokenizer.tokenize();
      
      expect(tokens).toHaveLength(2); // nodeId + EOF
      expect(tokens[0].type).toBe(TokenType.IDENTIFIER);
      expect(tokens[0].value).toBe('nodeId');
    });
  });

  describe('Line and column tracking', () => {
    it('should track line numbers correctly', () => {
      const tokenizer = new Tokenizer('line1\nline2\nline3');
      const tokens = tokenizer.tokenize();
      
      expect(tokens[0].line).toBe(1); // line1
      expect(tokens[1].line).toBe(1); // newline
      expect(tokens[2].line).toBe(2); // line2
      expect(tokens[3].line).toBe(2); // newline
      expect(tokens[4].line).toBe(3); // line3
    });

    it('should track column numbers correctly', () => {
      const tokenizer = new Tokenizer('abc def ghi');
      const tokens = tokenizer.tokenize();
      
      expect(tokens[0].column).toBe(1); // abc
      expect(tokens[1].column).toBe(5); // def
      expect(tokens[2].column).toBe(9); // ghi
    });
  });

  describe('Error cases', () => {
    it('should throw error for invalid characters', () => {
      const tokenizer = new Tokenizer('valid $ invalid');
      
      expect(() => tokenizer.tokenize()).toThrow(TokenizerError);
      expect(() => tokenizer.tokenize()).toThrow('Unexpected character');
    });

    it('should include line and column in error messages', () => {
      const tokenizer = new Tokenizer('line1\ninvalid $ char');
      
      try {
        tokenizer.tokenize();
        fail('Expected TokenizerError to be thrown');
      } catch (error) {
        expect(error).toBeInstanceOf(TokenizerError);
        expect((error as TokenizerError).message).toContain('line 2');
        expect((error as TokenizerError).message).toContain('column 9');
      }
    });
  });

  describe('Real DSL examples', () => {
    it('should tokenize simple node definition', () => {
      const dsl = 'nodeId {\nlabel: "Node Label"\ndirection: "vertical"\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      
      const tokenTypes = tokens.map(t => t.type);
      expect(tokenTypes).toContain(TokenType.IDENTIFIER);
      expect(tokenTypes).toContain(TokenType.LBRACE);
      expect(tokenTypes).toContain(TokenType.RBRACE);
      expect(tokenTypes).toContain(TokenType.COLON);
      expect(tokenTypes).toContain(TokenType.STRING);
    });

    it('should tokenize arrow syntax', () => {
      const dsl = 'parent.child1 --> parent.child2#anchor';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      
      const tokenTypes = tokens.map(t => t.type);
      expect(tokenTypes).toContain(TokenType.IDENTIFIER);
      expect(tokenTypes).toContain(TokenType.DOT);
      expect(tokenTypes).toContain(TokenType.ARROW);
      expect(tokenTypes).toContain(TokenType.HASH);
    });

    it('should tokenize anchor definitions', () => {
      const dsl = 'anchors: {\ncenter: [0.5, 0.5],\ntop: [0.5, 0.0]\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      
      const tokenTypes = tokens.map(t => t.type);
      expect(tokenTypes).toContain(TokenType.IDENTIFIER);
      expect(tokenTypes).toContain(TokenType.COLON);
      expect(tokenTypes).toContain(TokenType.LBRACE);
      expect(tokenTypes).toContain(TokenType.LBRACKET);
      expect(tokenTypes).toContain(TokenType.NUMBER);
      expect(tokenTypes).toContain(TokenType.COMMA);
      expect(tokenTypes).toContain(TokenType.RBRACKET);
      expect(tokenTypes).toContain(TokenType.RBRACE);
    });
  });
});