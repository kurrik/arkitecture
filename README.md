# Arkitecture

A Domain Specific Language (DSL) for generating SVG architecture diagrams with precise manual positioning control. Arkitecture is designed for creating high-level architecture diagrams such as Domain-Driven Design (DDD) domain boundaries and bounded context relationships.

Unlike traditional graphing tools that use automatic layout algorithms, Arkitecture provides fine-grained control over element positioning and sizing to create professional architectural diagrams.

## Features

- **Precise positioning control** with custom anchor points
- **Hierarchical node structures** with containers and groups
- **Manual layout control** via horizontal/vertical direction settings
- **TypeScript implementation** with full type safety
- **Command-line interface** for standalone usage
- **SDK/API** for programmatic integration
- **Cross-platform** Node.js and browser support

## DSL Features

The Arkitecture DSL provides a clean, intuitive syntax for describing architecture diagrams:

### Container Nodes

Container nodes are the primary building blocks with IDs, labels, and layout properties:

```arkitecture
# Basic node with label
api {
  label: "API Gateway"
  direction: "vertical"
}

# Node with size override and custom anchors
userService {
  label: "User Service"
  size: 0.75
  anchors: {
    db: [0.5, 1.0],
    api: [0.5, 0.0]
  }
}
```

### Groups

Groups provide layout organization without visual representation:

```arkitecture
services {
  label: "Microservices"
  direction: "horizontal"
  
  group {
    direction: "vertical"
    
    userService {
      label: "User Service"
    }
    
    orderService {
      label: "Order Service"
    }
  }
}
```

### Arrows

Connect nodes using arrow syntax with optional anchor references:

```arkitecture
# Simple arrow between nodes
api --> database

# Arrow with anchor points
api#south --> services#north

# Nested node references
services.userService#db --> database#north
```

### Properties

- **`label`**: Display text for the node
- **`direction`**: Layout direction (`"vertical"` or `"horizontal"`)
- **`size`**: Size override (0.0 to 1.0) for orthogonal dimension
- **`anchors`**: Custom anchor points as `{ anchorId: [x, y] }`

### Coordinate System

Anchors use relative coordinates within the node bounding box:

- `[0.0, 0.0]` = top-left corner
- `[0.5, 0.5]` = center (default anchor for all nodes)
- `[1.0, 1.0]` = bottom-right corner
- `[0.5, 0.0]` = top edge, horizontally centered

## Command Line Usage

Install arkitecture globally or use with npx:

```bash
npm install -g arkitecture
# or use with npx
npx arkitecture diagram.ark diagram.svg
```

### Basic Usage

```bash
# Generate SVG from DSL file
arkitecture input.ark output.svg

# Output file is optional (defaults to input with .svg extension)
arkitecture diagram.ark
```

### Command Line Options

```bash
# Validate DSL without generating SVG
arkitecture diagram.ark --validate-only

# Verbose output with processing details
arkitecture diagram.ark --verbose

# Override font settings
arkitecture diagram.ark --font-size 16 --font-family Helvetica

# Show help
arkitecture --help

# Show version
arkitecture --version
```

### Exit Codes

- `0`: Success
- `1`: Validation errors (syntax, references, constraints)
- `2`: File system errors (not found, permissions)

## SDK Usage

### Installation

```bash
npm install arkitecture
```

### Main API (Recommended)

```typescript
import arkitectureToSVG from 'arkitecture';

const dslContent = `
api {
  label: "API Gateway"
}

database {
  label: "Database"
}

api --> database
`;

const result = arkitectureToSVG(dslContent);

if (result.success) {
  console.log('Generated SVG:', result.svg);
} else {
  console.error('Errors:', result.errors);
}
```

### Options

```typescript
const result = arkitectureToSVG(dslContent, {
  validateOnly: true,    // Skip SVG generation
  fontSize: 14,          // Override font size
  fontFamily: 'Helvetica' // Override font family
});
```

### Advanced Usage

For more control, use individual functions:

```typescript
import { parseArkitecture, validate, generateSVG } from 'arkitecture';

// Parse DSL to AST
const parseResult = parseArkitecture(dslContent);
if (!parseResult.success) {
  console.error('Parse errors:', parseResult.errors);
  return;
}

// Validate references and constraints
const validationErrors = validate(parseResult.document);
if (validationErrors.length > 0) {
  console.error('Validation errors:', validationErrors);
  return;
}

// Generate SVG
const svg = generateSVG(parseResult.document, {
  fontSize: 14,
  fontFamily: 'Arial'
});
```

### Error Handling

All functions return structured error information:

```typescript
interface ValidationError {
  line: number;           // Line number in DSL
  column: number;         // Column number in DSL  
  message: string;        // Human-readable error message
  type: 'syntax' | 'reference' | 'constraint'; // Error category
}
```

## Development

### Prerequisites

- Node.js 18+
- TypeScript 5.0+

### Setup

```bash
# Clone and install dependencies
git clone <repository-url>
cd arkitecture
npm install

# Build the project
npm run build
```

### Testing

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run specific test suites
npm test -- tests/parser
npm test -- tests/golden
```

#### Test Selection Examples

```bash
# Run parser tests only
npm test -- --testPathPattern=parser

# Run specific test file
npm test -- tests/generator/layout.test.ts

# Run tests matching pattern
npm test -- --testNamePattern="anchor"

# Run tests with coverage
npm test -- --coverage
```

### Golden Tests

Golden tests validate output by comparing against reference files:

```bash
# Generate/update golden test outputs
npm run golden:generate

# Run only golden tests  
npm test -- tests/golden
```

The golden test system:

- Automatically discovers `.ark` test files in `tests/golden/examples/`
- Compares generated SVG against `.svg` reference files
- Validates error cases against `.error` reference files
- Provides detailed diff output when tests fail

### Development Commands

```bash
npm run build          # Compile TypeScript
npm run dev            # Watch mode compilation
npm run test           # Run test suite
npm run test:watch     # Watch mode testing
npm run lint           # Run ESLint
npm run format         # Format code with Prettier
npm run golden:generate # Update golden test files
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `npm test`
5. Update golden tests if needed: `npm run golden:generate`
6. Submit a pull request

## License

MIT
