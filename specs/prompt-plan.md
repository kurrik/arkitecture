# Arkitecture Implementation Plan

## Overview

This document provides a step-by-step implementation plan for building the Arkitecture DSL. Each step is designed to be implemented incrementally with comprehensive testing, building on previous steps without any orphaned code.

## Implementation Strategy

- **Test-Driven Development**: Each step starts with tests
- **Incremental Progress**: Small, safe iterations that build on each other
- **No Orphaned Code**: Every feature is immediately integrated
- **Early Validation**: Working functionality at each step

---

## Step 1: Project Foundation & TypeScript Setup ✅ COMPLETED

### Context

Establish the basic project structure with TypeScript, testing framework, and core AST type definitions. This creates the foundation for all subsequent development.

### Implementation Prompt

```
Create a new TypeScript project for arkitecture with the following requirements:

1. Initialize a TypeScript project with:
   - package.json with arkitecture name and description
   - TypeScript 5.0+ configuration
   - Jest testing framework setup
   - ESLint and Prettier configuration
   - src/ directory structure (parser/, validator/, generator/, cli/)

2. Define core AST interfaces in src/types.ts:
   - Document interface (nodes, arrows)
   - ContainerNode interface (id, label?, direction?, size?, anchors?, children)
   - GroupNode interface (direction?, children)
   - Arrow interface (source, target)
   - ParseResult interface (success, document?, errors)
   - ValidationError interface (line, column, message, type)

3. Create comprehensive unit tests for the type definitions in tests/types.test.ts:
   - Test that interfaces can be instantiated correctly
   - Test optional vs required fields
   - Test that constraint types (direction enum, size range) work as expected

4. Set up build scripts in package.json:
   - npm run build (TypeScript compilation)
   - npm run test (Jest test runner)
   - npm run lint (ESLint)
   - npm run format (Prettier)

5. Create a basic index.ts that exports the core types

Ensure all tests pass and the project builds successfully.
```

---

## Step 2: Basic DSL Tokenizer ✅ COMPLETED

### Context

Create a simple tokenizer that can break DSL text into meaningful tokens. This is the foundation of the parser and handles the most basic level of syntax analysis.

### Implementation Prompt

```
Build a tokenizer for the arkitecture DSL that handles basic syntax elements:

1. Create src/parser/tokenizer.ts with:
   - Token interface (type, value, line, column)
   - TokenType enum (IDENTIFIER, STRING, NUMBER, LBRACE, RBRACE, COLON, ARROW, etc.)
   - Tokenizer class with tokenize(input: string): Token[] method

2. The tokenizer should handle:
   - Identifiers (node IDs, property names)
   - String literals with quotes
   - Numbers (for anchor coordinates, size values)
   - Structural tokens: { } : [ ] ,
   - Arrow operator: -->
   - Comments (# to end of line)
   - Whitespace (track for line/column positions)

3. Implement comprehensive error handling:
   - Track line and column numbers for each token
   - Handle unterminated strings
   - Handle invalid characters
   - Provide meaningful error messages

4. Create tests in tests/parser/tokenizer.test.ts:
   - Test basic token recognition
   - Test string parsing with escapes
   - Test number parsing (integers and decimals)
   - Test comment handling
   - Test error cases (unterminated strings, invalid chars)
   - Test line/column tracking accuracy

5. Integration: Export tokenizer from src/parser/index.ts and re-export from main index.ts

The tokenizer should successfully process the example DSL from the specification.
```

---

## Step 3: Simple Node Parser ✅ COMPLETED

### Context

Build a parser that can handle basic container nodes with labels and direction. This establishes the core parsing patterns and AST building without the complexity of nested structures.

### Implementation Prompt

```
Create a parser that can handle simple container nodes with basic properties:

1. Create src/parser/parser.ts with:
   - Parser class that takes tokens from the tokenizer
   - parseDocument(): ParseResult method
   - Helper methods: parseNode(), parseNodeProperties(), expectToken()

2. Support parsing:
   - Simple container nodes: nodeId { label: "text", direction: "vertical" }
   - Basic property parsing (label, direction)
   - Proper error reporting with line/column information
   - Multiple top-level nodes in a document

3. Handle parsing errors gracefully:
   - Unexpected tokens
   - Missing required syntax (braces, colons)
   - Invalid property values
   - Collect multiple errors before failing

4. Create comprehensive tests in tests/parser/parser.test.ts:
   - Test simple node parsing
   - Test property parsing (label, direction)
   - Test multiple nodes in document
   - Test error cases with proper line/column reporting
   - Test empty documents
   - Test nodes with no properties

5. Integration steps:
   - Create parseArkitecture(dslContent: string): ParseResult in src/parser/index.ts
   - This function combines tokenizer + parser
   - Export from main index.ts
   - Add integration test that uses the full parsing pipeline

No nested nodes yet - just flat container nodes with basic properties.
```

---

## Step 4: Nested Node Structure ✅ COMPLETED

### Context

Extend the parser to handle nested container nodes and groups. This adds the hierarchical structure that makes arkitecture useful for representing complex architectures.

### Implementation Prompt

```
Extend the parser to support nested nodes and groups:

1. Enhance src/parser/parser.ts to support:
   - Nested container nodes within other nodes
   - Group nodes (keyword "group" instead of identifier)
   - Recursive parsing of child nodes
   - Proper parent-child relationships in AST

2. Parser improvements:
   - Add parseGroup() method for group-specific parsing
   - Enhance parseNode() to handle nested children
   - Add parseChildren() helper for parsing child lists
   - Maintain proper scoping for nested structures

3. Support the following syntax:
   ```

   parent {
     label: "Parent Node"
     direction: "vertical"

     child1 {
       label: "Child 1"
     }

     group {
       direction: "horizontal"

       child2 {
         label: "Child 2"
       }
     }
   }

   ```

4. Error handling improvements:
   - Detect unclosed braces in nested structures
   - Validate that groups don't have IDs or labels
   - Ensure proper nesting depth tracking
   - Report errors with correct parent context

5. Create tests in tests/parser/nested.test.ts:
   - Test simple parent-child relationships
   - Test groups with children
   - Test deeply nested structures (3+ levels)
   - Test mixed container nodes and groups
   - Test error cases (unclosed braces, invalid group properties)

6. Integration:
   - Update the main parseArkitecture function to handle nested structures
   - Add integration tests with the example from the specification
   - Ensure backward compatibility with flat structures from Step 3

The parser should now handle the hierarchical node structure from the DSL specification.
```

---

## Step 5: Property Parsing - Size and Anchors ✅ COMPLETED

### Context

Add support for the remaining node properties: size values and anchor coordinates. This completes the node property parsing before moving to arrows.

### Implementation Prompt

```
Add support for size and anchors properties in node parsing:

1. Enhance src/parser/parser.ts to parse:
   - size property: size: 0.5 (decimal values between 0.0-1.0)
   - anchors property: anchors: { anchorId: [x, y], another: [0.0, 1.0] }
   - Coordinate arrays with two numeric values
   - Multiple anchors per node

2. Add parsing methods:
   - parseSize(): number - validates range 0.0-1.0
   - parseAnchors(): Record<string, [number, number]>
   - parseCoordinate(): [number, number] - validates coordinate pairs
   - Add validation for coordinate ranges (0.0-1.0)

3. Support this syntax:
   ```

   node1 {
     label: "Node with anchors"
     size: 0.75
     anchors: {
       top: [0.5, 0.0],
       bottom: [0.5, 1.0],
       center: [0.5, 0.5]
     }
   }

   ```

4. Validation and error handling:
   - Ensure size values are in range 0.0-1.0
   - Ensure coordinate values are in range 0.0-1.0
   - Validate anchor coordinate array format
   - Prevent duplicate anchor IDs within a node
   - Proper error messages for invalid values

5. Create tests in tests/parser/properties.test.ts:
   - Test size parsing with valid/invalid values
   - Test anchor parsing with various coordinate combinations
   - Test nodes with all properties combined
   - Test error cases (out of range values, malformed coordinates)
   - Test coordinate array parsing edge cases

6. Integration:
   - Update type definitions if needed for anchor coordinate validation
   - Ensure all properties work together in nested structures
   - Add integration tests combining all node features
   - Test with the full DSL specification example

Parser should now handle all node properties defined in the specification.
```

---

## Step 6: Arrow Parsing ✅ COMPLETED

### Context

Add arrow parsing to connect nodes with the `-->` syntax. This completes the core DSL parsing functionality by handling relationships between nodes.

### Implementation Prompt

```
Implement arrow parsing to handle node connections:

1. Enhance src/parser/parser.ts to parse arrows:
   - Arrow syntax: source.path --> target.path#anchor
   - Node path parsing: nodeId.childId.grandchildId
   - Anchor references: nodePath#anchorId
   - Multiple arrows in a document

2. Add parsing methods:
   - parseArrows(): Arrow[] - parse all arrows after nodes
   - parseArrow(): Arrow - parse single arrow statement
   - parseNodePath(): string - parse dot-separated node references
   - parseTargetWithAnchor(): string - handle target#anchor syntax

3. Support these arrow formats:
   ```

# Simple arrows

   node1 --> node2

# With anchor references

   node1 --> node2#top

# Nested node paths

   parent.child1 --> parent.group.child2#center

   ```

4. Parser architecture changes:
   - Parse document in two phases: nodes first, then arrows
   - Store arrows separately in Document.arrows array
   - Maintain position tracking for arrow error reporting

5. Error handling:
   - Detect malformed arrow syntax
   - Handle missing --> operator
   - Report errors for invalid node path format
   - Validate basic syntax (don't validate references yet - that's for validator)

6. Create tests in tests/parser/arrows.test.ts:
   - Test simple arrow parsing
   - Test arrows with anchor references
   - Test arrows with nested node paths
   - Test multiple arrows in document
   - Test arrow parsing error cases
   - Test arrows combined with nested node structures

7. Integration:
   - Update Document interface to include arrows array
   - Modify parseDocument() to handle two-phase parsing
   - Add comprehensive integration tests with full DSL examples
   - Test the complete specification example with nodes and arrows

Parser should now handle complete DSL documents with nodes and arrows.
```

---

## Step 7: Basic Validation Engine ✅ COMPLETED

### Context

Create a validation system that checks references and constraints after parsing. This ensures parsed ASTs are semantically correct before SVG generation.

### Implementation Prompt

```
Build a validation engine for parsed AST documents:

1. Create src/validator/validator.ts with:
   - Validator class with validate(document: Document): ValidationError[] method
   - Reference validation for arrows pointing to existing nodes
   - Anchor reference validation
   - Constraint validation (size, coordinate ranges)

2. Implement validation rules:
   - Node ID uniqueness within parent scope
   - Arrow source/target node path validation
   - Anchor reference validation (ensure referenced anchors exist)
   - Size value constraints (0.0 ≤ value ≤ 1.0)
   - Coordinate constraints for anchor positions

3. Create validation methods:
   - validateNodeReferences(): ValidationError[]
   - validateAnchorReferences(): ValidationError[]
   - validateConstraints(): ValidationError[]
   - validateNodeIdUniqueness(): ValidationError[]
   - Helper: resolveNodePath(path: string): ContainerNode | null

4. Error reporting:
   - Generate meaningful error messages with context
   - Include suggestions for fixing common issues
   - Group related errors together
   - Use appropriate error types (reference, constraint, syntax)

5. Create comprehensive tests in tests/validator/validator.test.ts:
   - Test valid documents (should return no errors)
   - Test invalid node references in arrows
   - Test invalid anchor references
   - Test constraint violations (size, coordinates out of range)
   - Test duplicate node ID detection
   - Test nested reference resolution

6. Integration:
   - Create validate() function in src/validator/index.ts
   - Export from main index.ts
   - Update parseArkitecture to optionally run validation
   - Add integration tests combining parsing and validation

7. Error collection strategy:
   - Don't fail-fast - collect all validation errors
   - Provide line/column information when possible
   - Return comprehensive ValidationError[] array

Validator should catch all semantic errors defined in the specification.
```

---

## Step 8: Text Measurement Foundation ✅ COMPLETED

### Context

Implement text dimension calculation using the string-width library. This is essential for the sizing algorithm that follows.

### Implementation Prompt

```
Create text measurement capabilities for layout calculations:

1. Install and configure string-width:
   - Add string-width dependency to package.json
   - Create src/generator/text-measurement.ts module

2. Implement TextMeasurement class:
   - calculateTextWidth(text: string, fontSize: number): number
   - calculateTextHeight(text: string, fontSize: number): number
   - Support for multi-line text (split on \n)
   - Default font: Arial 12px
   - Line height: 1.2x font size for multi-line

3. Text measurement methods:
   - getTextDimensions(text: string, fontSize?: number): { width: number, height: number }
   - Support for empty/null text handling
   - Handle special characters and unicode
   - Cross-platform consistency

4. Configuration support:
   - FontConfig interface (family, size, lineHeight)
   - Default font configuration
   - Override capability for different fonts

5. Create tests in tests/generator/text-measurement.test.ts:
   - Test single-line text measurement
   - Test multi-line text with \n characters
   - Test empty string handling
   - Test unicode and special character handling
   - Test different font sizes
   - Test line height calculations for multi-line text

6. Integration:
   - Export TextMeasurement from src/generator/index.ts
   - Create default text measurement instance
   - Add to main index.ts exports
   - Create utility functions for common use cases

7. Cross-platform considerations:
   - Ensure consistent results between Node.js and browser environments
   - Handle fallbacks for missing font families
   - Document any platform-specific limitations

Text measurement should provide accurate, consistent results for layout calculations.
```

---

## Step 9: Basic Layout Algorithm ✅ COMPLETED

### Context

Implement the bottom-up sizing algorithm for calculating node dimensions. This is the core of the layout system before SVG generation.

### Implementation Prompt

```
Implement the bottom-up layout algorithm for node sizing:

1. Create src/generator/layout.ts with:
   - LayoutEngine class with calculateLayout(document: Document): LayoutResult
   - NodeDimensions interface (width, height, x, y)
   - LayoutResult interface mapping node IDs to dimensions

2. Implement sizing algorithm:
   - calculateNodeDimensions(node: ContainerNode | GroupNode): NodeDimensions
   - Start with leaf nodes (no children)
   - Work bottom-up through the hierarchy
   - Apply horizontal/vertical layout rules from specification

3. Layout rules implementation:
   - Horizontal parents: parent width = sum of child widths, parent height = max child height
   - Vertical parents: parent height = sum of child heights, parent width = max child width
   - Child sizing in orthogonal dimension (100% of parent unless size override)
   - Size attribute override (affects orthogonal dimension only)

4. Key algorithm steps:
   - Phase 1: Calculate text dimensions for all leaf nodes
   - Phase 2: Bottom-up dimension calculation
   - Phase 3: Top-down positioning (x, y coordinates)
   - Phase 4: Apply size overrides

5. Create tests in tests/generator/layout.test.ts:
   - Test single node layout (leaf node with text)
   - Test simple parent-child vertical layout
   - Test simple parent-child horizontal layout
   - Test size override behavior
   - Test nested layout calculations
   - Test group layout (groups don't add visual space)

6. Integration with text measurement:
   - Use TextMeasurement for calculating base text dimensions
   - Handle empty labels (use default minimum dimensions)
   - Consider border width (1px) in calculations

7. Canvas size calculation:
   - Calculate overall canvas dimensions from top-level nodes
   - No padding - canvas matches content bounds exactly
   - Return canvas dimensions as part of LayoutResult

Layout engine should correctly size all nodes according to the specification.
```

---

## Step 10: Anchor Position Calculation ✅ COMPLETED

### Context

Add anchor position calculation to the layout system. This enables precise arrow positioning by resolving anchor coordinates to absolute positions.

### Implementation Prompt

```
Extend the layout system to calculate absolute anchor positions:

1. Enhance src/generator/layout.ts with anchor support:
   - AnchorPosition interface (x, y, nodeId, anchorId)
   - Add anchor position calculation to LayoutResult
   - calculateAnchorPositions(layoutResult: LayoutResult, document: Document): AnchorPosition[]

2. Anchor position calculation:
   - Convert relative anchor coordinates to absolute positions
   - Use node bounding box (including borders) for calculation
   - Handle implicit center anchor [0.5, 0.5] for all nodes
   - Support custom anchors defined in node.anchors

3. Coordinate system:
   - [0.0, 0.0] = top-left corner of node (including border)
   - [1.0, 1.0] = bottom-right corner of node (including border)
   - [0.5, 0.5] = center of node
   - [0.5, 0.0] = top edge, horizontally centered

4. Implementation details:
   - resolveNodeAnchors(node: ContainerNode, dimensions: NodeDimensions): AnchorPosition[]
   - Handle both custom anchors and implicit center anchor
   - Account for 1px border in position calculations
   - Validate anchor coordinates are in range [0.0, 1.0]

5. Create tests in tests/generator/anchors.test.ts:
   - Test implicit center anchor calculation
   - Test custom anchor position calculation
   - Test anchor positions on different sized nodes
   - Test anchor positions with size overrides
   - Test nested node anchor positions
   - Test edge cases (anchor at corners, edges)

6. Integration:
   - Enhance LayoutResult to include anchor positions
   - Update LayoutEngine.calculateLayout() to include anchors
   - Create helper functions for finding anchors by node path
   - Add anchor resolution utilities

7. Validation integration:
   - Ensure anchor positions are calculated for all nodes referenced in arrows
   - Handle missing anchor references gracefully
   - Provide clear error messages for anchor calculation failures

Anchor system should provide precise positioning for arrow connections.
```

---

## Step 11: Basic SVG Generation ✅ COMPLETED

### Context

Create SVG output generation using the layout results. This transforms the calculated layout into visual SVG representation.

### Implementation Prompt

```
Implement SVG generation from layout results:

1. Create src/generator/svg-generator.ts with:
   - SvgGenerator class with generateSVG(document: Document, layout: LayoutResult): string
   - Helper methods for creating SVG elements (rect, text, line)
   - SVG structure following the specification template

2. SVG generation components:
   - generateNodes(): Generate rect and text elements for all nodes
   - generateArrows(): Generate line elements with arrowhead markers
   - generateDefs(): Create arrowhead marker definitions
   - assembleDocument(): Combine all elements into complete SVG

3. Node rendering:
   - Rectangle with white fill, 1px black border
   - Text labels centered in nodes
   - Arial 12px font for text
   - Groups have no visual representation

4. Arrow rendering:
   - Simple black lines between anchor positions
   - Basic arrowhead markers at target end
   - Use anchor positions from layout results

5. SVG structure:
   xml
   <svg xmlns="http://www.w3.org/2000/svg" width="..." height="...">
     <defs>
       <marker id="arrowhead">...</marker>
     </defs>
     <!-- Node rectangles -->
     <!-- Node labels -->
     <!-- Arrows -->
   </svg>


6. Create tests in tests/generator/svg-generator.test.ts:
   - Test single node SVG generation
   - Test multiple nodes with proper positioning
   - Test arrow generation between nodes
   - Test SVG structure and valid XML
   - Test text rendering and positioning
   - Test arrowhead marker generation

7. Integration:
   - Create generateSVG() function in src/generator/index.ts
   - Combine layout engine and SVG generator
   - Export from main index.ts
   - Add end-to-end integration tests

8. SVG validation:
   - Ensure valid XML structure
   - Test that coordinates are numeric and positive
   - Verify proper namespace declarations
   - Test SVG can be parsed by standard tools

SVG generator should produce valid, renderable SVG from layout results.
```

---

## Step 12: Main API Integration ✅ COMPLETED

### Context

Create the main arkitectureToSVG function that ties together parsing, validation, layout, and SVG generation. This provides the primary API for library users.

### Implementation Prompt

```
Create the main integrated API function and individual step exports:

1. Create src/arkitecture.ts with the main integration:
   - arkitectureToSVG(dslContent: string, options?: Options): Result
   - Options interface (validateOnly?, fontSize?, fontFamily?)
   - Result interface (success, svg?, errors)

2. Implementation flow:
   - Parse DSL content into AST
   - Validate the AST (collect all errors)
   - If validation fails, return errors without generating SVG
   - Calculate layout with text measurement
   - Generate SVG from layout
   - Return success result with SVG string

3. Error handling strategy:
   - Catch and wrap parsing errors
   - Include validation errors in result
   - Handle layout calculation errors
   - Provide meaningful error messages for each phase
   - Never throw exceptions - always return Result object

4. Options support:
   - validateOnly: boolean - skip layout/SVG generation
   - fontSize: number - override default 12px font
   - fontFamily: string - override default Arial font

5. Individual function exports:
   - parseArkitecture(dslContent: string): ParseResult
   - validate(document: Document): ValidationError[]
   - generateSVG(document: Document, options?: GenerationOptions): string

6. Create tests in tests/arkitecture.test.ts:
   - Test successful end-to-end processing
   - Test error handling at each phase
   - Test options handling (validateOnly, font settings)
   - Test individual function exports
   - Test with specification example DSL

7. Update main index.ts:
   - Default export: arkitectureToSVG
   - Named exports: parseArkitecture, validate, generateSVG
   - Export all types and interfaces
   - Clean, documented API surface

8. Integration validation:
   - Test complete pipeline with valid DSL
   - Test error propagation through pipeline
   - Test that each phase builds correctly on previous phases
   - Verify no orphaned code or unused functions

Main API should provide both convenience and flexibility for different use cases.
```

---

## Step 13: CLI Foundation ✅ COMPLETED

### Context

Create a basic command-line interface that uses the library to process DSL files. This makes arkitecture usable as a standalone tool.

### Implementation Prompt

```
Build the basic CLI foundation for arkitecture:

1. Install CLI dependencies:
   - Add commander.js for argument parsing
   - Add chalk for colored output (optional but helpful)
   - Create src/cli/index.ts as CLI entry point

2. Create basic CLI structure:
   - CliApp class with run(args: string[]): Promise<number> method
   - Support basic usage: arkitecture input.ark output.svg
   - Return appropriate exit codes (0=success, 1=validation error, 2=file error)

3. Implement core CLI features:
   - Read input DSL file
   - Process with arkitectureToSVG()
   - Write SVG output file
   - Display errors to stderr with proper formatting
   - Show success/failure messages

4. Error handling:
   - Handle file not found errors
   - Handle file permission errors
   - Handle invalid DSL syntax
   - Display validation errors with line/column info
   - Proper exit codes for different error types

5. Add CLI flags:
   - --verbose, -v: Show detailed processing information
   - --validate-only: Parse and validate without generating SVG
   - --help, -h: Display usage information
   - --version: Display version information

6. Create tests in tests/cli/cli.test.ts:
   - Test basic file processing
   - Test error handling (missing files, invalid DSL)
   - Test CLI flags (verbose, validate-only, help)
   - Test exit codes for different scenarios
   - Mock file system operations for testing

7. Add package.json bin configuration:
   - Add "bin" field pointing to CLI entry
   - Create executable script that calls the CLI
   - Add shebang line for Node.js execution

8. Integration:
   - Create simple test DSL files for CLI testing
   - Test CLI with specification example
   - Verify output SVG files are created correctly
   - Test error scenarios produce helpful messages

CLI should provide a user-friendly interface to arkitecture functionality.
```

---

## Step 14: CLI Watch Mode ✅ COMPLETED

### Context

Add watch mode functionality to the CLI that monitors input files for changes and automatically regenerates output. This improves the development experience.

### Implementation Prompt

```
Add watch mode functionality to the CLI:

1. Install watch dependencies:
   - Add chokidar for cross-platform file watching
   - Enhance CLI with watch mode support

2. Extend src/cli/index.ts with watch functionality:
   - Add --watch, -w flag
   - WatchMode class for managing file monitoring
   - Debounced regeneration to handle rapid file changes

3. Watch mode implementation:
   - Monitor input DSL file for changes
   - Automatically regenerate SVG when DSL file changes
   - Display status messages (watching, regenerating, success/error)
   - Handle file deletion and recreation
   - Graceful shutdown on Ctrl+C

4. User experience improvements:
   - Clear, informative console output
   - Timestamp for each regeneration
   - Success/error indicators with colors (if chalk is available)
   - Show watching status and file paths

5. Error handling in watch mode:
   - Continue watching even after errors
   - Display errors without stopping watch mode
   - Clear previous errors on successful regeneration
   - Handle file system errors (permissions, disk full)

6. Create tests in tests/cli/watch.test.ts:
   - Test watch mode activation
   - Test file change detection and regeneration
   - Test error handling during watch mode
   - Test graceful shutdown
   - Mock file system and chokidar for testing

7. CLI integration:
   - Combine watch mode with other flags (verbose, validate-only)
   - Ensure watch mode works with all CLI options
   - Add watch mode to help text
   - Handle conflicts (watch mode with multiple files)

8. Development workflow testing:
   - Test rapid file changes (debouncing)
   - Test with large DSL files
   - Test watch mode startup and shutdown
   - Verify memory usage stays stable during long-running watch

Watch mode should provide smooth, responsive development experience.
```

---

## Step 15: Error Enhancement & Final Integration

### Context

Polish error handling, add comprehensive integration tests, and ensure the entire system works together seamlessly. This is the final step before packaging.

### Implementation Prompt

```
Enhance error handling and create comprehensive integration tests:

1. Improve error reporting across all components:
   - Enhance ValidationError with better context information
   - Add error codes for programmatic error handling
   - Improve error messages with suggestions for fixes
   - Add error recovery suggestions where possible

2. Create comprehensive integration tests in tests/integration/:
   - full-pipeline.test.ts: End-to-end testing with real DSL files
   - specification-example.test.ts: Test the specification example
   - error-scenarios.test.ts: Test various error conditions
   - performance.test.ts: Basic performance testing

3. Error handling improvements:
   - Standardize error message formats
   - Add line/column information to more error types
   - Create error categorization (syntax, semantic, generation)
   - Improve error aggregation and reporting

4. CLI error experience:
   - Better formatting for validation errors
   - Helpful suggestions for common mistakes
   - Clear distinction between different error types
   - Improved verbose mode output

5. Documentation and examples:
   - Create examples/ directory with sample DSL files
   - Add examples showing different features
   - Include common error scenarios and fixes
   - Document the complete API

6. Final integration testing:
   - Test all CLI flags in combination
   - Test library API with various inputs
   - Test error propagation through all phases
   - Verify no memory leaks or performance issues

7. Package preparation:
   - Ensure all exports are properly defined
   - Verify TypeScript declarations are generated
   - Test installation and usage in external project
   - Add README with usage examples

8. Quality assurance:
   - Run full test suite
   - Check code coverage
   - Verify all lint rules pass
   - Test in both Node.js and browser environments (if applicable)

The project should be complete, well-tested, and ready for distribution.
```

---

## Summary

This implementation plan provides 15 incremental steps to build arkitecture from foundation to completion. Each step:

- **Builds incrementally** on previous work
- **Includes comprehensive testing** to ensure quality
- **Integrates immediately** to avoid orphaned code
- **Has appropriate scope** - large enough to make progress, small enough to implement safely
- **Follows TDD principles** with tests driving implementation

The plan progresses logically from basic infrastructure through core functionality to user-facing features, ensuring a solid foundation at each step.
