/**
 * Arkitecture - A DSL for generating SVG architecture diagrams
 */

export * from './types';
export * from './parser';
export * from './validator';
export * from './generator';

// Default export will be implemented in later steps
export default function arkitectureToSVG(): never {
  throw new Error('Not implemented yet');
}
