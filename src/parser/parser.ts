/**
 * Parser for Arkitecture DSL
 */

import { Token, TokenType } from './tokenizer';
import { ContainerNode, GroupNode, ParseResult, ValidationError } from '../types';

export class ParseError extends Error {
  constructor(
    message: string,
    public line: number,
    public column: number
  ) {
    super(`${message} at line ${line}, column ${column}`);
    this.name = 'ParseError';
  }
}

export class Parser {
  private tokens: Token[];
  private current: number;
  private errors: ValidationError[];

  constructor(tokens: Token[]) {
    this.tokens = tokens;
    this.current = 0;
    this.errors = [];
  }

  parseDocument(): ParseResult {
    try {
      const nodes = this.parseNodes();
      
      if (this.errors.length > 0) {
        return {
          success: false,
          document: {
            nodes,
            arrows: [], // Step 3 doesn't handle arrows yet
          },
          errors: this.errors,
        };
      }

      return {
        success: true,
        document: {
          nodes,
          arrows: [], // Step 3 doesn't handle arrows yet
        },
        errors: [],
      };
    } catch (error) {
      if (error instanceof ParseError) {
        this.addError('syntax', error.message, error.line, error.column);
      } else {
        this.addError('syntax', (error as Error).message || 'Unknown error', 1, 1);
      }

      return {
        success: false,
        errors: this.errors,
      };
    }
  }

  private parseNodes(): ContainerNode[] {
    const nodes: ContainerNode[] = [];
    
    while (!this.isAtEnd() && !this.check(TokenType.EOF)) {
      // Skip newlines between top-level nodes
      if (this.check(TokenType.NEWLINE)) {
        this.advance();
        continue;
      }

      try {
        const node = this.parseNode();
        if (node) {
          nodes.push(node);
        }
      } catch (error) {
        // Skip to next token and continue parsing
        if (!this.isAtEnd()) {
          this.advance();
        }
      }
    }

    return nodes;
  }

  private parseNode(): ContainerNode | null {
    if (!this.check(TokenType.IDENTIFIER)) {
      if (!this.isAtEnd() && !this.check(TokenType.EOF)) {
        const token = this.peek();
        this.addError(
          'syntax',
          `Expected node identifier, got ${token.type}`,
          token.line,
          token.column
        );
        this.advance(); // Skip invalid token
      }
      return null;
    }

    const idToken = this.advance();
    const id = idToken.value;

    if (!this.check(TokenType.LBRACE)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '{' after node id '${id}', got ${token.type}`,
        token.line,
        token.column
      );
      // Skip remaining tokens until we find a valid node start or EOF
      this.skipUntilNodeOrEOF();
      return null;
    }

    this.advance(); // consume '{'

    const node: ContainerNode = {
      id,
      children: [],
    };

    // Parse node properties and children
    this.parseNodeContent(node);

    if (!this.check(TokenType.RBRACE)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '}' to close node '${id}', got ${token.type}`,
        token.line,
        token.column
      );
      return node;
    }

    this.advance(); // consume '}'
    return node;
  }

  private parseNodeContent(node: ContainerNode): void {
    while (!this.check(TokenType.RBRACE) && !this.isAtEnd()) {
      // Skip newlines within node body
      if (this.check(TokenType.NEWLINE)) {
        this.advance();
        continue;
      }

      // Check for nested nodes (identifiers) or groups
      if (this.check(TokenType.IDENTIFIER)) {
        // If identifier is followed by '{', it's a nested node
        if (this.peekNext() && this.peekNext()!.type === TokenType.LBRACE) {
          const childNode = this.parseNode();
          if (childNode) {
            node.children.push(childNode);
          }
          continue;
        }
        
        // Otherwise, it's a property
        this.parseProperty(node);
      } else if (this.check(TokenType.GROUP)) {
        const group = this.parseGroup();
        if (group) {
          node.children.push(group);
        }
      } else {
        const token = this.peek();
        this.addError(
          'syntax',
          `Expected property name, nested node, or group, got ${token.type}`,
          token.line,
          token.column
        );
        this.advance(); // Skip invalid token
      }
    }
  }

  private parseProperty(node: ContainerNode): void {
    const propertyToken = this.advance();
    const propertyName = propertyToken.value;

    if (!this.check(TokenType.COLON)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected ':' after property '${propertyName}', got ${token.type}`,
        token.line,
        token.column
      );
      // Skip tokens until we find a colon, property name, or closing brace
      this.skipUntilRecovery();
      return;
    }

    this.advance(); // consume ':'

    switch (propertyName) {
      case 'label':
        this.parseLabel(node);
        break;
      case 'direction':
        this.parseDirection(node);
        break;
      case 'size':
        this.parseSize(node);
        break;
      case 'anchors':
        this.parseAnchors(node);
        break;
      default: {
        this.addError(
          'syntax',
          `Unknown property '${propertyName}'`,
          propertyToken.line,
          propertyToken.column
        );
        // Skip the value
        if (!this.check(TokenType.RBRACE) && !this.isAtEnd()) {
          this.advance();
        }
        break;
      }
    }
  }

  private parseGroup(): GroupNode | null {
    if (!this.check(TokenType.GROUP)) {
      return null;
    }

    this.advance(); // consume 'group'

    if (!this.check(TokenType.LBRACE)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '{' after 'group', got ${token.type}`,
        token.line,
        token.column
      );
      this.skipUntilNodeOrEOF();
      return null;
    }

    this.advance(); // consume '{'

    const group: GroupNode = {
      children: [],
    };

    // Parse group content (only direction property and children)
    this.parseGroupContent(group);

    if (!this.check(TokenType.RBRACE)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '}' to close group, got ${token.type}`,
        token.line,
        token.column
      );
      return group;
    }

    this.advance(); // consume '}'
    return group;
  }

  private parseGroupContent(group: GroupNode): void {
    while (!this.check(TokenType.RBRACE) && !this.isAtEnd()) {
      // Skip newlines within group body
      if (this.check(TokenType.NEWLINE)) {
        this.advance();
        continue;
      }

      // Check for nested nodes or groups
      if (this.check(TokenType.IDENTIFIER)) {
        const ahead = this.peek();
        
        // If identifier is followed by '{', it's a nested node
        if (this.peekNext() && this.peekNext()!.type === TokenType.LBRACE) {
          const childNode = this.parseNode();
          if (childNode) {
            group.children.push(childNode);
          }
          continue;
        }
        
        // Otherwise, it might be a direction property for the group
        if (ahead.value === 'direction') {
          this.parseGroupProperty(group);
        } else {
          const token = this.peek();
          this.addError(
            'syntax',
            `Groups can only have 'direction' property, got '${ahead.value}'`,
            token.line,
            token.column
          );
          // Skip invalid property and its value
          this.advance(); // Skip property name
          if (this.check(TokenType.COLON)) {
            this.advance(); // Skip colon
            if (!this.check(TokenType.RBRACE) && !this.isAtEnd()) {
              this.advance(); // Skip value
            }
          }
        }
      } else if (this.check(TokenType.GROUP)) {
        const nestedGroup = this.parseGroup();
        if (nestedGroup) {
          group.children.push(nestedGroup);
        }
      } else {
        const token = this.peek();
        this.addError(
          'syntax',
          `Expected nested node or group in group, got ${token.type}`,
          token.line,
          token.column
        );
        this.advance(); // Skip invalid token
      }
    }
  }

  private parseGroupProperty(group: GroupNode): void {
    const propertyToken = this.advance();
    const propertyName = propertyToken.value;

    if (propertyName !== 'direction') {
      this.addError(
        'syntax',
        `Groups can only have 'direction' property, got '${propertyName}'`,
        propertyToken.line,
        propertyToken.column
      );
      return;
    }

    if (!this.check(TokenType.COLON)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected ':' after 'direction', got ${token.type}`,
        token.line,
        token.column
      );
      this.skipUntilRecovery();
      return;
    }

    this.advance(); // consume ':'

    if (!this.check(TokenType.STRING)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected string value for direction, got ${token.type}`,
        token.line,
        token.column
      );
      if (!this.check(TokenType.RBRACE)) {
        this.advance(); // Skip invalid token
      }
      return;
    }

    const directionToken = this.advance();
    const direction = directionToken.value;

    if (direction !== 'vertical' && direction !== 'horizontal') {
      this.addError(
        'syntax',
        `Invalid direction '${direction}', expected 'vertical' or 'horizontal'`,
        directionToken.line,
        directionToken.column
      );
      return;
    }

    group.direction = direction as 'vertical' | 'horizontal';
  }

  private parseLabel(node: ContainerNode): void {
    if (!this.check(TokenType.STRING)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected string value for label, got ${token.type}`,
        token.line,
        token.column
      );
      if (!this.check(TokenType.RBRACE)) {
        this.advance(); // Skip invalid token
      }
      return;
    }

    const labelToken = this.advance();
    node.label = labelToken.value;
  }

  private parseDirection(node: ContainerNode): void {
    if (!this.check(TokenType.STRING)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected string value for direction, got ${token.type}`,
        token.line,
        token.column
      );
      if (!this.check(TokenType.RBRACE)) {
        this.advance(); // Skip invalid token
      }
      return;
    }

    const directionToken = this.advance();
    const direction = directionToken.value;

    if (direction !== 'vertical' && direction !== 'horizontal') {
      this.addError(
        'syntax',
        `Invalid direction '${direction}', expected 'vertical' or 'horizontal'`,
        directionToken.line,
        directionToken.column
      );
      return;
    }

    node.direction = direction as 'vertical' | 'horizontal';
  }

  private parseSize(node: ContainerNode): void {
    if (!this.check(TokenType.NUMBER)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected number value for size, got ${token.type}`,
        token.line,
        token.column
      );
      if (!this.check(TokenType.RBRACE)) {
        this.advance(); // Skip invalid token
      }
      return;
    }

    const sizeToken = this.advance();
    const sizeValue = parseFloat(sizeToken.value);

    if (isNaN(sizeValue)) {
      this.addError(
        'syntax',
        `Invalid size value '${sizeToken.value}', expected a number`,
        sizeToken.line,
        sizeToken.column
      );
      return;
    }

    if (sizeValue < 0.0 || sizeValue > 1.0) {
      this.addError(
        'constraint',
        `Size value ${sizeValue} is out of range, expected 0.0-1.0`,
        sizeToken.line,
        sizeToken.column
      );
      return;
    }

    node.size = sizeValue;
  }

  private parseAnchors(node: ContainerNode): void {
    if (!this.check(TokenType.LBRACE)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '{' to start anchors object, got ${token.type}`,
        token.line,
        token.column
      );
      if (!this.check(TokenType.RBRACE)) {
        this.advance(); // Skip invalid token
      }
      return;
    }

    this.advance(); // consume '{'

    const anchors: Record<string, [number, number]> = {};

    while (!this.check(TokenType.RBRACE) && !this.isAtEnd()) {
      // Skip newlines within anchors object
      if (this.check(TokenType.NEWLINE)) {
        this.advance();
        continue;
      }

      if (!this.check(TokenType.IDENTIFIER)) {
        const token = this.peek();
        this.addError(
          'syntax',
          `Expected anchor identifier, got ${token.type}`,
          token.line,
          token.column
        );
        this.advance(); // Skip invalid token
        continue;
      }

      const anchorIdToken = this.advance();
      const anchorId = anchorIdToken.value;

      // Check for duplicate anchor IDs
      if (anchors[anchorId]) {
        this.addError(
          'syntax',
          `Duplicate anchor ID '${anchorId}'`,
          anchorIdToken.line,
          anchorIdToken.column
        );
      }

      if (!this.check(TokenType.COLON)) {
        const token = this.peek();
        this.addError(
          'syntax',
          `Expected ':' after anchor ID '${anchorId}', got ${token.type}`,
          token.line,
          token.column
        );
        this.skipUntilRecovery();
        continue;
      }

      this.advance(); // consume ':'

      const coordinate = this.parseCoordinate();
      if (coordinate) {
        anchors[anchorId] = coordinate;
      }

      // Optional comma
      if (this.check(TokenType.COMMA)) {
        this.advance();
      }
    }

    if (!this.check(TokenType.RBRACE)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '}' to close anchors object, got ${token.type}`,
        token.line,
        token.column
      );
      return;
    }

    this.advance(); // consume '}'

    if (Object.keys(anchors).length > 0) {
      node.anchors = anchors;
    }
  }

  private parseCoordinate(): [number, number] | null {
    if (!this.check(TokenType.LBRACKET)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected '[' to start coordinate array, got ${token.type}`,
        token.line,
        token.column
      );
      if (!this.check(TokenType.RBRACE) && !this.check(TokenType.COMMA) && !this.isAtEnd()) {
        this.advance(); // Skip invalid token
      }
      return null;
    }

    this.advance(); // consume '['

    // Parse X coordinate
    if (!this.check(TokenType.NUMBER)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected number for X coordinate, got ${token.type}`,
        token.line,
        token.column
      );
      this.skipUntilBracketOrComma();
      return null;
    }

    const xToken = this.advance();
    const x = parseFloat(xToken.value);

    if (isNaN(x)) {
      this.addError(
        'syntax',
        `Invalid X coordinate '${xToken.value}', expected a number`,
        xToken.line,
        xToken.column
      );
      this.skipUntilBracketOrComma();
      return null;
    }

    if (x < 0.0 || x > 1.0) {
      this.addError(
        'constraint',
        `X coordinate ${x} is out of range, expected 0.0-1.0`,
        xToken.line,
        xToken.column
      );
    }

    if (!this.check(TokenType.COMMA)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected ',' between coordinates, got ${token.type}`,
        token.line,
        token.column
      );
      this.skipUntilBracketOrComma();
      return null;
    }

    this.advance(); // consume ','

    // Parse Y coordinate
    if (!this.check(TokenType.NUMBER)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected number for Y coordinate, got ${token.type}`,
        token.line,
        token.column
      );
      this.skipUntilBracketOrComma();
      return null;
    }

    const yToken = this.advance();
    const y = parseFloat(yToken.value);

    if (isNaN(y)) {
      this.addError(
        'syntax',
        `Invalid Y coordinate '${yToken.value}', expected a number`,
        yToken.line,
        yToken.column
      );
      this.skipUntilBracketOrComma();
      return null;
    }

    if (y < 0.0 || y > 1.0) {
      this.addError(
        'constraint',
        `Y coordinate ${y} is out of range, expected 0.0-1.0`,
        yToken.line,
        yToken.column
      );
    }

    if (!this.check(TokenType.RBRACKET)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected ']' to close coordinate array, got ${token.type}`,
        token.line,
        token.column
      );
      this.skipUntilBracketOrComma();
      return null;
    }

    this.advance(); // consume ']'

    return [x, y];
  }

  private expectToken(type: TokenType): Token | null {
    if (!this.check(type)) {
      const token = this.peek();
      this.addError(
        'syntax',
        `Expected ${type}, got ${token.type}`,
        token.line,
        token.column
      );
      return null;
    }
    return this.advance();
  }

  private check(type: TokenType): boolean {
    if (this.isAtEnd()) return false;
    return this.peek().type === type;
  }

  private advance(): Token {
    if (!this.isAtEnd()) {
      this.current++;
    }
    return this.previous();
  }

  private peek(): Token {
    return this.tokens[this.current];
  }

  private peekNext(): Token | null {
    if (this.current + 1 >= this.tokens.length) {
      return null;
    }
    return this.tokens[this.current + 1];
  }

  private previous(): Token {
    return this.tokens[this.current - 1];
  }

  private isAtEnd(): boolean {
    return this.current >= this.tokens.length || this.peek().type === TokenType.EOF;
  }

  private addError(type: 'syntax' | 'reference' | 'constraint', message: string, line: number, column: number): void {
    this.errors.push({
      type,
      message,
      line,
      column,
    });
  }

  private skipUntilNodeOrEOF(): void {
    while (!this.isAtEnd() && !this.check(TokenType.IDENTIFIER) && !this.check(TokenType.EOF)) {
      this.advance();
    }
  }

  private skipUntilRecovery(): void {
    while (!this.isAtEnd() && 
           !this.check(TokenType.COLON) && 
           !this.check(TokenType.IDENTIFIER) && 
           !this.check(TokenType.RBRACE) && 
           !this.check(TokenType.EOF)) {
      this.advance();
    }
  }

  private skipUntilBracketOrComma(): void {
    while (!this.isAtEnd() && 
           !this.check(TokenType.RBRACKET) && 
           !this.check(TokenType.COMMA) && 
           !this.check(TokenType.RBRACE) && 
           !this.check(TokenType.EOF)) {
      this.advance();
    }
  }
}