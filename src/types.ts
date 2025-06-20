/**
 * Core AST interfaces for Arkitecture DSL
 */

export interface Document {
  nodes: ContainerNode[];
  arrows: Arrow[];
}

export interface ContainerNode {
  id: string;
  label?: string;
  direction?: 'vertical' | 'horizontal';
  size?: number;
  anchors?: Record<string, [number, number]>;
  children: (ContainerNode | GroupNode)[];
}

export interface GroupNode {
  direction?: 'vertical' | 'horizontal';
  children: (ContainerNode | GroupNode)[];
}

export interface Arrow {
  source: string; // e.g., "c1.n2"
  target: string; // e.g., "c1.n3#a1"
}

export interface ParseResult {
  success: boolean;
  document?: Document;
  errors: ValidationError[];
}

export interface ValidationError {
  line: number;
  column: number;
  message: string;
  type: 'syntax' | 'reference' | 'constraint';
}

export interface Result {
  success: boolean;
  svg?: string;
  errors: ValidationError[];
}

export interface Options {
  validateOnly?: boolean;
  fontSize?: number;
  fontFamily?: string;
}
