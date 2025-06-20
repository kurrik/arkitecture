/**
 * Parser module exports
 */

export * from './tokenizer';
export * from './parser';

import { Tokenizer, TokenizerError } from './tokenizer';
import { Parser } from './parser';
import { ParseResult } from '../types';

export function parseArkitecture(dslContent: string): ParseResult {
  try {
    // Tokenize the input
    const tokenizer = new Tokenizer(dslContent);
    const tokens = tokenizer.tokenize();

    // Parse the tokens into AST
    const parser = new Parser(tokens);
    return parser.parseDocument();
  } catch (error) {
    // Handle tokenizer errors
    if (error instanceof TokenizerError) {
      return {
        success: false,
        errors: [
          {
            type: 'syntax',
            message: error.message,
            line: error.line,
            column: error.column,
          },
        ],
      };
    }

    // Handle other errors
    return {
      success: false,
      errors: [
        {
          type: 'syntax',
          message: (error as Error).message || 'Unknown parsing error',
          line: 1,
          column: 1,
        },
      ],
    };
  }
}