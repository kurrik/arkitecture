# Arkitecture - Architecture Specification

## Overview

Arkitecture is a Domain Specific Language (DSL) for generating SVG architecture diagrams with precise manual positioning control. It is designed primarily for creating high-level architecture diagrams such as Domain-Driven Design (DDD) domain boundaries and bounded context relationships.

Unlike traditional graphing tools that use automatic layout algorithms, Arkitecture provides fine-grained control over element positioning and sizing to create professional architectural diagrams.

## Core Requirements

### Functional Requirements

- Parse custom DSL syntax into an internal AST structure
- Generate SVG output from the AST
- Support hierarchical node structures with containers and groups
- Enable precise arrow positioning with anchor points
- Provide comprehensive validation with detailed error reporting
- Support both programmatic API usage and command-line interface

### Non-Functional Requirements

- Cross-platform compatibility (Node.js and browser environments)
- TypeScript implementation for type safety
- NPM package distribution
- Comprehensive error handling and validation
- Performance suitable for complex diagrams (100+ nodes)

## Architecture Overview

### Two-Pass Architecture

1. **Parse Pass**: DSL text → TypeScript AST
2. **Generation Pass**: AST → SVG output

### Component Structure

The arkitecture tool follows a three-stage pipeline:

1. **DSL Parser** → Converts DSL text into TypeScript AST
2. **Validator** → Validates references and constraints in the AST
3. **SVG Generator** → Renders the validated AST into SVG output

## DSL Specification

### Core Syntax Elements

#### Container Nodes

```text
nodeId {
  label: "Display Name"
  direction: "vertical" | "horizontal"
  size: 0.0..1.0  // Optional, affects orthogonal dimension
  anchors: {
    anchorId: [x, y]  // Coordinates between 0.0-1.0
  }
  // Child nodes and groups
}
```

#### Groups (Layout Only)

```text
group {
  direction: "vertical" | "horizontal"
  // Child nodes and groups
}
```

#### Arrows

```text
sourceNode.path --> targetNode.path#anchorId
```

### Validation Rules

- **Minimal Required Fields**: Only node IDs are required
- **ID Uniqueness**: IDs must be unique within parent scope
- **Reference Validation**: Arrow references validated after parsing
- **Value Constraints**:
  - `size` values: 0.0 ≤ value ≤ 1.0
  - Anchor coordinates: 0.0 ≤ x,y ≤ 1.0
- **No Structural Constraints**: No limits on nesting depth

## TypeScript AST Structure

### Core Interfaces

```typescript
interface Document {
  nodes: ContainerNode[];
  arrows: Arrow[];
}

interface ContainerNode {
  id: string;
  label?: string;
  direction?: 'vertical' | 'horizontal';
  size?: number;
  anchors?: Record<string, [number, number]>;
  children: (ContainerNode | GroupNode)[];
}

interface GroupNode {
  direction?: 'vertical' | 'horizontal';
  children: (ContainerNode | GroupNode)[];
}

interface Arrow {
  source: string; // e.g., "c1.n2"
  target: string; // e.g., "c1.n3#a1"
}

interface ParseResult {
  success: boolean;
  document?: Document;
  errors: ValidationError[];
}

interface ValidationError {
  line: number;
  column: number;
  message: string;
  type: 'syntax' | 'reference' | 'constraint';
}
```

## Sizing Algorithm

### Text Dimension Calculation

- **Default Font**: Arial 12px
- **Text Measurement**: Use `string-width` npm package for cross-platform text width calculation
- **Multi-line Support**: Labels are treated as single-line unless they contain explicit line breaks (`\n`)
- **Line Height**: 1.2x font size for multi-line labels

### Bottom-Up Dimension Calculation

1. **Leaf Node Resolution**: Start with nodes containing no children
2. **Text Dimension Calculation**: Compute rendered text bounds using specified font
3. **Parent Dimension Rules**:

#### Horizontal Parents

- Parent height = max(child text heights)
- Parent width = sum(child widths)
- Child height = parent height
- Child width = child text width

#### Vertical Parents

- Parent width = max(child text widths)
- Parent height = sum(child heights)
- Child width = parent width
- Child height = child text height

### Size Attribute Override

- Applies only to orthogonal dimension after parent size calculation
- `size: 0.5` = 50% of calculated parent dimension
- Does not affect parent size calculation

### Canvas Size Calculation

- **Auto-sizing**: SVG canvas dimensions calculated by running sizing algorithm on all top-level nodes
- **No Padding**: Canvas size matches the exact bounds of all content
- **Future Enhancement**: Scaling and size constraints can be added later

### Default Values

- **Direction**: `vertical` for both nodes and groups
- **Anchors**: All nodes have implicit `center` anchor at `[0.5, 0.5]`
- **Border Width**: 1px solid black border for all nodes
- **Groups**: No visual representation, no spacing/padding added

## API Design

### Primary Export (Integrated)

```typescript
export default function arkitectureToSVG(
  dslContent: string,
  options?: Options
): Result;

interface Result {
  success: boolean;
  svg?: string;
  errors: ValidationError[];
}
```

### Advanced Usage Exports

```typescript
export function parseArkitecture(dslContent: string): ParseResult;
export function validate(document: Document): ValidationError[];
export function generateSVG(document: Document): string;
```

## CLI Specification

### Basic Usage

```bash
arkitecture input.ark output.svg
```

### Flags

- `--verbose, -v`: Detailed error reporting and processing information
- `--watch, -w`: Watch input file for changes and regenerate
- `--validate-only`: Parse and validate without generating SVG
- `--help, -h`: Display usage information
- `--version`: Display version information

### Exit Codes

- `0`: Success
- `1`: Validation errors
- `2`: File system errors
- `3`: Invalid arguments

## Error Handling Strategy

### Parse Phase Errors

- Syntax errors with line/column information
- Malformed attribute values
- Invalid nesting structures

### Validation Phase Errors

- Undefined node references in arrows
- Invalid anchor references
- Constraint violations (size, coordinate ranges)
- Duplicate IDs within scope

### Generation Phase Errors

- SVG generation failures
- Dimension calculation errors
- Circular dependency detection

### Error Reporting

- Collect all errors before failing (don't fail-fast)
- Provide contextual information (line numbers, expected values)
- Group related errors for better UX
- Support both human-readable and machine-readable formats

## SVG Generation Specification

### Default Visual Styling

- **Nodes**: White fill, 1px solid black border, rectangular shape
- **Arrows**: Simple black lines with basic arrowheads
- **Text**: Black text, Arial 12px font
- **Groups**: No visual representation (layout only)

### Anchor Point Resolution

- **Coordinate System**: Anchors use relative positioning within node bounding box
- **Examples**:
  - `[0.5, 0.5]`: Center of node
  - `[0.5, 0.0]`: Top edge, horizontally centered
  - `[1.0, 1.0]`: Bottom-right corner
  - `[0.0, 0.5]`: Left edge, vertically centered
- **Arrow Connection**: Arrows connect to exact anchor coordinates, including border edges

### SVG Structure

```xml
<svg xmlns="http://www.w3.org/2000/svg" width="..." height="...">
  <defs>
    <!-- Arrowhead markers -->
  </defs>

  <!-- Node rectangles -->
  <rect x="..." y="..." width="..." height="..." fill="white" stroke="black"/>

  <!-- Node labels -->
  <text x="..." y="..." text-anchor="middle">Label</text>

  <!-- Arrows -->
  <line x1="..." y1="..." x2="..." y2="..." stroke="black" marker-end="url(#arrowhead)"/>
</svg>
```

## Testing Strategy

### Unit Tests

- **Parser Tests**
  - Valid syntax parsing
  - Invalid syntax error handling
  - Edge cases (empty files, comments only)
  - Complex nesting scenarios

- **Validator Tests**
  - Reference validation (valid/invalid references)
  - Constraint validation (size, coordinate ranges)
  - ID uniqueness within scopes
  - Comprehensive error collection

- **SVG Generation Tests**
  - Dimension calculation accuracy
  - Anchor positioning
  - Arrow path generation
  - Valid SVG output structure

### Integration Tests

- **End-to-End DSL Processing**
  - Complete DSL files � SVG output
  - Error propagation through pipeline
  - CLI functionality
  - Watch mode behavior

### Performance Tests

- Large document parsing (1000+ nodes)
- Deep nesting scenarios
- Memory usage validation
- SVG generation performance

### Browser Compatibility Tests

- Modern browser support verification
- Node.js environment testing
- Module loading in different environments

## Implementation Phases

### Phase 1: Core Parser

- Implement basic DSL parsing
- Create AST structure
- Build validation framework
- Unit tests for parsing

### Phase 2: SVG Generation

- Implement sizing algorithm
- Create SVG output generation
- Add anchor positioning
- Arrow path calculation

### Phase 3: CLI Tool

- Command-line interface
- File system operations
- Watch mode implementation
- Error reporting

### Phase 4: Package Distribution

- NPM package setup
- Documentation
- TypeScript declarations
- Build tooling

## Development Environment

### Prerequisites

- Node.js 18+
- TypeScript 5.0+
- Modern text editor with TypeScript support

### Recommended Libraries

- **Parser**: Consider PEG.js or hand-written recursive descent
- **Text Measurement**: `string-width` for cross-platform text width calculation
- **Testing**: Jest or Vitest
- **CLI**: Commander.js or similar
- **File Watching**: Chokidar
- **Build Tool**: Rollup or Webpack for bundling
