/**
 * Tests for CLI functionality
 */

import fs from 'fs/promises';
import path from 'path';
import { CliApp } from '../../src/cli/index';

// Mock fs module
jest.mock('fs/promises');
const mockFs = fs as jest.Mocked<typeof fs>;

// Mock chalk to prevent ANSI codes in test output
jest.mock('chalk', () => ({
  red: (text: string) => text,
  green: (text: string) => text,
  blue: (text: string) => text,
  yellow: (text: string) => text,
  magenta: (text: string) => text,
  gray: (text: string) => text,
}));

// Mock console methods
const originalConsoleLog = console.log;
const originalConsoleError = console.error;
const originalProcessExit = process.exit;

describe('CLI App', () => {
  let cliApp: CliApp;
  let mockConsoleLog: jest.SpyInstance;
  let mockConsoleError: jest.SpyInstance;
  let mockProcessExit: jest.SpyInstance;

  beforeEach(() => {
    cliApp = new CliApp();
    mockConsoleLog = jest.spyOn(console, 'log').mockImplementation(() => {});
    mockConsoleError = jest.spyOn(console, 'error').mockImplementation(() => {});
    mockProcessExit = jest.spyOn(process, 'exit').mockImplementation((code?: string | number | null) => {
      throw new Error(`Process exit called with code ${code}`);
    });
    
    // Reset mocks
    mockFs.readFile.mockReset();
    mockFs.writeFile.mockReset();
  });

  afterEach(() => {
    mockConsoleLog.mockRestore();
    mockConsoleError.mockRestore();
    mockProcessExit.mockRestore();
  });

  describe('basic file processing', () => {
    test('processes valid DSL file and generates SVG', async () => {
      const inputDsl = `
        node1 {
          label: "Test Node"
        }
      `;
      const expectedSvg = '<svg xmlns="http://www.w3.org/2000/svg"';

      mockFs.readFile.mockResolvedValue(inputDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark', 'output.svg']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockFs.readFile).toHaveBeenCalledWith('input.ark', 'utf-8');
      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'output.svg',
        expect.stringContaining(expectedSvg),
        'utf-8'
      );
      expect(mockConsoleLog).toHaveBeenCalledWith('✓ Generated SVG: output.svg');
      expect(mockProcessExit).toHaveBeenCalledWith(0);
    });

    test('defaults output filename when not provided', async () => {
      const inputDsl = `node1 { label: "Test" }`;

      mockFs.readFile.mockResolvedValue(inputDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'diagram.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'diagram.svg',
        expect.any(String),
        'utf-8'
      );
    });

    test('processes file with verbose flag', async () => {
      const inputDsl = `node1 { label: "Test" }`;

      mockFs.readFile.mockResolvedValue(inputDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark', 'output.svg', '--verbose']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleLog).toHaveBeenCalledWith('Processing: input.ark -> output.svg');
      expect(mockConsoleLog).toHaveBeenCalledWith(`Read ${inputDsl.length} characters from input.ark`);
      expect(mockConsoleLog).toHaveBeenCalledWith(expect.stringMatching(/Wrote \d+ characters to output.svg/));
    });
  });

  describe('error handling', () => {
    test('handles file not found error', async () => {
      const error = new Error('File not found') as any;
      error.code = 'ENOENT';
      mockFs.readFile.mockRejectedValue(error);

      try {
        await cliApp.run(['node', 'arkitecture', 'missing.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith('File not found: missing.ark');
      expect(mockProcessExit).toHaveBeenCalledWith(2);
    });

    test('handles file permission error', async () => {
      const error = new Error('Permission denied') as any;
      error.code = 'EACCES';
      mockFs.readFile.mockRejectedValue(error);

      try {
        await cliApp.run(['node', 'arkitecture', 'restricted.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith('Permission denied: restricted.ark');
      expect(mockProcessExit).toHaveBeenCalledWith(2);
    });

    test('handles output file permission error', async () => {
      const inputDsl = `node1 { label: "Test" }`;
      const writeError = new Error('Permission denied') as any;
      writeError.code = 'EACCES';

      mockFs.readFile.mockResolvedValue(inputDsl);
      mockFs.writeFile.mockRejectedValue(writeError);

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark', 'restricted.svg']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith('Permission denied writing to: restricted.svg');
      expect(mockProcessExit).toHaveBeenCalledWith(2);
    });

    test('handles invalid DSL syntax', async () => {
      const invalidDsl = `
        node1 {
          invalid syntax here
      `;

      mockFs.readFile.mockResolvedValue(invalidDsl);

      try {
        await cliApp.run(['node', 'arkitecture', 'invalid.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith('Validation errors:');
      expect(mockConsoleError).toHaveBeenCalledWith(expect.stringMatching(/SYNTAX.*:/));
      expect(mockProcessExit).toHaveBeenCalledWith(1);
    });

    test('handles validation errors with proper formatting', async () => {
      const dslWithValidationError = `
        node1 { label: "Node 1" }
        node1 --> nonexistent
      `;

      mockFs.readFile.mockResolvedValue(dslWithValidationError);

      try {
        await cliApp.run(['node', 'arkitecture', 'invalid.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith('Validation errors:');
      expect(mockConsoleError).toHaveBeenCalledWith(expect.stringMatching(/REFERENCE.*:/));
      expect(mockProcessExit).toHaveBeenCalledWith(1);
    });
  });

  describe('CLI flags', () => {
    test('processes validate-only flag', async () => {
      const validDsl = `node1 { label: "Valid Node" }`;

      mockFs.readFile.mockResolvedValue(validDsl);

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark', '--validate-only']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockFs.writeFile).not.toHaveBeenCalled();
      expect(mockConsoleLog).toHaveBeenCalledWith('✓ DSL is valid');
      expect(mockProcessExit).toHaveBeenCalledWith(0);
    });

    test('processes custom font options', async () => {
      const inputDsl = `node1 { label: "Custom Font" }`;

      mockFs.readFile.mockResolvedValue(inputDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark', 'output.svg', '--font-size', '16', '--font-family', 'Helvetica']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'output.svg',
        expect.stringContaining('font-size="16"'),
        'utf-8'
      );
      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'output.svg',
        expect.stringContaining('font-family="Helvetica"'),
        'utf-8'
      );
    });

    test('displays help information', async () => {
      try {
        await cliApp.run(['node', 'arkitecture', '--help']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleLog).toHaveBeenCalledWith(expect.stringContaining('Examples:'));
      expect(mockConsoleLog).toHaveBeenCalledWith(expect.stringContaining('arkitecture diagram.ark diagram.svg'));
    });

    test('displays version information', async () => {
      // Note: Commander.js handles --version internally, so we can't easily test the output
      // but we can verify it doesn't crash
      try {
        await cliApp.run(['node', 'arkitecture', '--version']);
      } catch (error) {
        // Expected due to process.exit mock
        expect((error as Error).message).toContain('Process exit called with code 0');
      }
    });
  });

  describe('exit codes', () => {
    test('returns 0 for successful processing', async () => {
      const validDsl = `node1 { label: "Test" }`;

      mockFs.readFile.mockResolvedValue(validDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockProcessExit).toHaveBeenCalledWith(0);
    });

    test('returns 1 for validation errors', async () => {
      const invalidDsl = `node1 { size: 1.5 }`;

      mockFs.readFile.mockResolvedValue(invalidDsl);

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockProcessExit).toHaveBeenCalledWith(1);
    });

    test('returns 2 for file system errors', async () => {
      const error = new Error('File not found') as any;
      error.code = 'ENOENT';
      mockFs.readFile.mockRejectedValue(error);

      try {
        await cliApp.run(['node', 'arkitecture', 'missing.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockProcessExit).toHaveBeenCalledWith(2);
    });

    test('returns 2 for internal errors', async () => {
      mockFs.readFile.mockRejectedValue(new Error('Unexpected error'));

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockProcessExit).toHaveBeenCalledWith(2);
    });
  });

  describe('error message formatting', () => {
    test('formats syntax errors correctly', async () => {
      const invalidDsl = `node1 { invalid }`;

      mockFs.readFile.mockResolvedValue(invalidDsl);

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith(expect.stringMatching(/SYNTAX.*line \d+, column \d+/));
    });

    test('formats reference errors correctly', async () => {
      const dslWithRefError = `
        node1 { label: "Node 1" }
        node1 --> nonexistent
      `;

      mockFs.readFile.mockResolvedValue(dslWithRefError);

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith(expect.stringMatching(/REFERENCE.*:/));
    });

    test('formats constraint errors correctly', async () => {
      const dslWithConstraintError = `
        node1 {
          label: "Node 1"
          size: 1.5
        }
      `;

      mockFs.readFile.mockResolvedValue(dslWithConstraintError);

      try {
        await cliApp.run(['node', 'arkitecture', 'input.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockConsoleError).toHaveBeenCalledWith(expect.stringMatching(/CONSTRAINT.*:/));
    });
  });

  describe('integration scenarios', () => {
    test('processes complex nested structure', async () => {
      const complexDsl = `
        api {
          label: "API Gateway"
          direction: "vertical"
          
          auth {
            label: "Authentication"
          }
          
          routing {
            label: "Request Routing"
          }
        }
        
        services {
          label: "Microservices"
          direction: "horizontal"
          
          userService {
            label: "User Service"
          }
          
          orderService {
            label: "Order Service"
          }
        }
        
        api --> services
      `;

      mockFs.readFile.mockResolvedValue(complexDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'complex.ark', 'complex.svg']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'complex.svg',
        expect.stringContaining('API Gateway'),
        'utf-8'
      );
      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'complex.svg',
        expect.stringContaining('Authentication'),
        'utf-8'
      );
      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'complex.svg',
        expect.stringContaining('User Service'),
        'utf-8'
      );
      expect(mockProcessExit).toHaveBeenCalledWith(0);
    });

    test('handles empty DSL file', async () => {
      mockFs.readFile.mockResolvedValue('');
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'empty.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'empty.svg',
        expect.stringContaining('<svg'),
        'utf-8'
      );
      expect(mockProcessExit).toHaveBeenCalledWith(0);
    });

    test('handles DSL with only comments', async () => {
      const commentOnlyDsl = `
        # This is a comment
        # Another comment
      `;

      mockFs.readFile.mockResolvedValue(commentOnlyDsl);
      mockFs.writeFile.mockResolvedValue();

      try {
        await cliApp.run(['node', 'arkitecture', 'comments.ark']);
      } catch (error) {
        // Expected due to process.exit mock
      }

      expect(mockProcessExit).toHaveBeenCalledWith(0);
    });
  });
});