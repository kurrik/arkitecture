/**
 * Validator module exports
 */

export { Validator } from './validator';
export * from './validator';

import { Validator } from './validator';
import { Document, ValidationError } from '../types';

/**
 * Convenience function to validate a document
 */
export function validate(document: Document): ValidationError[] {
  const validator = new Validator(document);
  return validator.validate();
}