/**
 * CLI interface for Arkitecture
 */

import fs from 'fs/promises';
import path from 'path';
import { Command } from 'commander';
import chalk from 'chalk';
import arkitectureToSVG from '../arkitecture';
import { ValidationError } from '../types';

export class CliApp {
  private program: Command;

  constructor() {
    this.program = new Command();
    this.setupProgram();
  }

  /**
   * Set up the command line program structure
   */
  private setupProgram(): void {
    this.program
      .name('arkitecture')
      .description('Generate SVG architecture diagrams from DSL files')
      .version('0.1.0')
      .argument('<input>', 'Input DSL file path')
      .argument('[output]', 'Output SVG file path (defaults to input with .svg extension)')
      .option('-v, --verbose', 'Show detailed processing information')
      .option('--validate-only', 'Parse and validate without generating SVG')
      .option('--font-size <size>', 'Override default font size (12px)', '12')
      .option('--font-family <family>', 'Override default font family (Arial)', 'Arial')
      .action(async (input: string, output?: string, options?: any) => {
        const exitCode = await this.processFiles(input, output, options);
        process.exit(exitCode);
      });

    this.program.on('--help', () => {
      console.log('');
      console.log('Examples:');
      console.log('  $ arkitecture diagram.ark diagram.svg');
      console.log('  $ arkitecture diagram.ark --validate-only');
      console.log('  $ arkitecture diagram.ark --verbose');
      console.log('  $ arkitecture diagram.ark --font-size 16 --font-family Helvetica');
    });
  }

  /**
   * Run the CLI with provided arguments
   */
  async run(args: string[]): Promise<number> {
    try {
      await this.program.parseAsync(args);
      return 0;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
      console.error(chalk.red(`Error: ${errorMessage}`));
      return 2;
    }
  }

  /**
   * Process input and output files
   */
  private async processFiles(
    inputPath: string,
    outputPath?: string,
    options?: {
      verbose?: boolean;
      validateOnly?: boolean;
      fontSize?: string;
      fontFamily?: string;
    }
  ): Promise<number> {
    try {
      // Resolve output path if not provided
      if (!outputPath) {
        const ext = path.extname(inputPath);
        outputPath = inputPath.replace(ext, '.svg');
      }

      if (options?.verbose) {
        console.log(chalk.blue(`Processing: ${inputPath} -> ${outputPath}`));
      }

      // Read input file
      let dslContent: string;
      try {
        dslContent = await fs.readFile(inputPath, 'utf-8');
      } catch (error) {
        if (error instanceof Error && 'code' in error && error.code === 'ENOENT') {
          console.error(chalk.red(`File not found: ${inputPath}`));
          return 2;
        }
        if (error instanceof Error && 'code' in error && error.code === 'EACCES') {
          console.error(chalk.red(`Permission denied: ${inputPath}`));
          return 2;
        }
        throw error;
      }

      if (options?.verbose) {
        console.log(chalk.blue(`Read ${dslContent.length} characters from ${inputPath}`));
      }

      // Process with arkitecture
      const result = arkitectureToSVG(dslContent, {
        validateOnly: options?.validateOnly,
        fontSize: options?.fontSize ? parseInt(options.fontSize, 10) : undefined,
        fontFamily: options?.fontFamily,
      });

      // Handle validation errors
      if (!result.success) {
        console.error(chalk.red('Validation errors:'));
        this.displayErrors(result.errors);
        return 1;
      }

      if (options?.validateOnly) {
        console.log(chalk.green('✓ DSL is valid'));
        return 0;
      }

      // Write output file
      if (result.svg) {
        try {
          await fs.writeFile(outputPath, result.svg, 'utf-8');
        } catch (error) {
          if (error instanceof Error && 'code' in error && error.code === 'EACCES') {
            console.error(chalk.red(`Permission denied writing to: ${outputPath}`));
            return 2;
          }
          throw error;
        }

        if (options?.verbose) {
          console.log(chalk.blue(`Wrote ${result.svg.length} characters to ${outputPath}`));
        }

        console.log(chalk.green(`✓ Generated SVG: ${outputPath}`));
      }

      return 0;

    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
      console.error(chalk.red(`Internal error: ${errorMessage}`));
      if (options?.verbose && error instanceof Error) {
        console.error(chalk.gray(error.stack));
      }
      return 2;
    }
  }

  /**
   * Display validation errors with proper formatting
   */
  private displayErrors(errors: ValidationError[]): void {
    for (const error of errors) {
      const location = error.line > 0 ? ` (line ${error.line}, column ${error.column})` : '';
      const typeColor = this.getErrorTypeColor(error.type);
      console.error(`  ${typeColor(error.type.toUpperCase())}${location}: ${error.message}`);
    }
  }

  /**
   * Get color for error type
   */
  private getErrorTypeColor(type: string): typeof chalk.red {
    switch (type) {
      case 'syntax':
        return chalk.red;
      case 'reference':
        return chalk.yellow;
      case 'constraint':
        return chalk.magenta;
      default:
        return chalk.gray;
    }
  }
}

// Default export for the CLI app
export default CliApp;