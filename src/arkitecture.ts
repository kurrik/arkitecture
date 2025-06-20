/**
 * Main API integration for Arkitecture
 */

import { Document, ValidationError, Result, Options } from './types';
import { parseArkitecture as parseArkitectureDSL } from './parser';
import { validate as validateDocument } from './validator';
import { calculateLayout } from './generator/layout';
import { generateSVG as generateSVGFromLayout, SvgGenerationOptions } from './generator/svg-generator';
import { TextMeasurement } from './generator/text-measurement';

/**
 * Main integrated function that converts DSL content to SVG
 */
export default function arkitectureToSVG(dslContent: string, options?: Options): Result {
  try {
    // Phase 1: Parse DSL content into AST
    const parseResult = parseArkitectureDSL(dslContent);
    
    if (!parseResult.success || !parseResult.document) {
      return {
        success: false,
        errors: parseResult.errors,
      };
    }

    // Phase 2: Validate the AST
    const validationErrors = validateDocument(parseResult.document);
    
    if (validationErrors.length > 0) {
      return {
        success: false,
        errors: validationErrors,
      };
    }

    // If validateOnly is true, stop here
    if (options?.validateOnly) {
      return {
        success: true,
        errors: [],
      };
    }

    // Phase 3: Calculate layout with optional text measurement customization
    const textMeasurement = createTextMeasurement(options);
    const layout = calculateLayout(parseResult.document, textMeasurement);

    // Phase 4: Generate SVG from layout
    const svgOptions: SvgGenerationOptions = {
      fontSize: options?.fontSize,
      fontFamily: options?.fontFamily,
    };
    
    const svg = generateSVGFromLayout(parseResult.document, layout, svgOptions);

    return {
      success: true,
      svg,
      errors: [],
    };

  } catch (error) {
    // Catch any unexpected errors and wrap them
    const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
    
    return {
      success: false,
      errors: [{
        line: 0,
        column: 0,
        message: `Internal error: ${errorMessage}`,
        type: 'syntax',
      }],
    };
  }
}

/**
 * Parse DSL content into AST
 */
export function parseArkitecture(dslContent: string) {
  return parseArkitectureDSL(dslContent);
}

/**
 * Validate a document AST
 */
export function validate(document: Document): ValidationError[] {
  return validateDocument(document);
}

/**
 * Generate SVG from document and optional generation options
 */
export function generateSVG(document: Document, options?: GenerationOptions): string {
  // Calculate layout first
  const textMeasurement = createTextMeasurement(options);
  const layout = calculateLayout(document, textMeasurement);
  
  // Generate SVG
  const svgOptions: SvgGenerationOptions = {
    fontSize: options?.fontSize,
    fontFamily: options?.fontFamily,
  };
  
  return generateSVGFromLayout(document, layout, svgOptions);
}

/**
 * Options for SVG generation (individual function)
 */
export interface GenerationOptions {
  fontSize?: number;
  fontFamily?: string;
}

/**
 * Create text measurement instance based on options
 */
function createTextMeasurement(options?: Options | GenerationOptions): TextMeasurement | undefined {
  if (!options?.fontSize && !options?.fontFamily) {
    return undefined; // Use default
  }
  
  return new TextMeasurement({
    size: options.fontSize,
    family: options.fontFamily,
  });
}