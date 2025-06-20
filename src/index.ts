/**
 * Arkitecture - A DSL for generating SVG architecture diagrams
 */

export * from './types';
export * from './parser';
export * from './validator';
export * from './generator';

// Main API exports
export { default } from './arkitecture';
export { 
  parseArkitecture, 
  validate, 
  generateSVG,
  GenerationOptions
} from './arkitecture';
