# Golden Tests

Golden tests validate the output of the arkitecture DSL by comparing generated SVG output against known-good reference files.

## Directory Structure

```
tests/golden/examples/
├── test-case-name.ark         # Input DSL file
├── test-case-name.json        # Configuration options (optional)
├── test-case-name.svg         # Expected SVG output (for success cases)
└── test-case-name.error       # Expected error output (for error cases)
```

## Test Cases

- **test-case-one**: Basic single node test
- **simple-container**: Container with child nodes  
- **complex-layout**: Complex nested layout with arrows
- **test-case-with-error**: Syntax error test case
- **reference-error**: Reference error test case

## Commands

### Generate Golden Files

To generate/update all golden output files:

```bash
npm run golden:generate
# or
npm run golden:update
```

This will:
- Read all `.ark` files in `tests/golden/examples/`
- Generate SVG output or capture errors
- Create corresponding `.svg` or `.error` files

### Run Golden Tests

Golden tests run automatically as part of the main test suite:

```bash
npm test
```

To run only golden tests:

```bash
npm test -- tests/golden
```

## Adding New Test Cases

1. Create a new `.ark` file with your test input
2. Optionally create a `.json` file with configuration options
3. Run `npm run golden:generate` to create the expected output
4. Commit both input and output files

## Error Test Cases

For test cases that should produce errors:
- The test will expect `result.success` to be `false`
- Error details are captured in `.error` files with this structure:
```json
{
  "type": "syntax|reference|constraint",
  "line": 1,
  "column": 1,
  "messageContains": "Expected error message substring"
}
```

## Test Output

When tests fail, you'll get detailed diff information showing exactly which lines differ between expected and actual output, making debugging easier.