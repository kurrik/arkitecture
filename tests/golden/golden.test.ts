import fs from 'fs';
import path from 'path';
import arkitectureToSVG from '../../src/arkitecture';
import { Options } from '../../src/types';

describe('Golden Tests', () => {
  const goldenDir = path.join(__dirname, 'examples');
  
  if (!fs.existsSync(goldenDir)) {
    throw new Error(`Golden test directory not found: ${goldenDir}`);
  }

  // Find all .ark files in the golden directory
  const arkFiles = fs.readdirSync(goldenDir)
    .filter(file => file.endsWith('.ark'))
    .map(file => path.basename(file, '.ark'));

  arkFiles.forEach(testCase => {
    describe(`Test case: ${testCase}`, () => {
      const arkFile = path.join(goldenDir, `${testCase}.ark`);
      const configFile = path.join(goldenDir, `${testCase}.json`);
      const svgFile = path.join(goldenDir, `${testCase}.svg`);
      const errorFile = path.join(goldenDir, `${testCase}.error`);

      it('should match expected output or error', () => {
        // Read input files
        const arkContent = fs.readFileSync(arkFile, 'utf-8');
        
        let config: Options = {};
        if (fs.existsSync(configFile)) {
          const configContent = fs.readFileSync(configFile, 'utf-8');
          config = JSON.parse(configContent);
        }

        // Generate result
        const result = arkitectureToSVG(arkContent, config);

        // Check if this is an error test case
        if (fs.existsSync(errorFile)) {
          // This should be an error case
          expect(result.success).toBe(false);
          expect(result.errors).toBeDefined();
          expect(result.errors.length).toBeGreaterThan(0);

          // Read expected error output
          const expectedErrorContent = fs.readFileSync(errorFile, 'utf-8');
          const expectedError = JSON.parse(expectedErrorContent);
          
          // Compare error structure (be flexible about exact messages)
          expect(result.errors[0]).toMatchObject({
            type: expectedError.type,
            line: expect.any(Number),
            column: expect.any(Number),
            message: expect.any(String),
          });
          
          // If specific error details are provided, check them
          if (expectedError.line !== undefined) {
            expect(result.errors[0].line).toBe(expectedError.line);
          }
          if (expectedError.column !== undefined) {
            expect(result.errors[0].column).toBe(expectedError.column);
          }
          if (expectedError.messageContains) {
            expect(result.errors[0].message).toContain(expectedError.messageContains);
          }
        } else {
          // This should be a success case
          expect(result.success).toBe(true);
          expect(result.svg).toBeDefined();
          expect(result.errors).toEqual([]);

          if (fs.existsSync(svgFile)) {
            // Compare with golden SVG
            const expectedSvg = fs.readFileSync(svgFile, 'utf-8').trim();
            const actualSvg = result.svg!.trim();
            
            if (actualSvg !== expectedSvg) {
              // Provide detailed diff information
              const diffLines: string[] = [];
              const expectedLines = expectedSvg.split('\n');
              const actualLines = actualSvg.split('\n');
              const maxLines = Math.max(expectedLines.length, actualLines.length);
              
              for (let i = 0; i < maxLines; i++) {
                const expectedLine = expectedLines[i] || '';
                const actualLine = actualLines[i] || '';
                
                if (expectedLine !== actualLine) {
                  diffLines.push(`Line ${i + 1}:`);
                  diffLines.push(`  Expected: ${JSON.stringify(expectedLine)}`);
                  diffLines.push(`  Actual:   ${JSON.stringify(actualLine)}`);
                }
              }
              
              const message = [
                `SVG output does not match expected for test case: ${testCase}`,
                '',
                'Differences:',
                ...diffLines,
                '',
                'To update golden file, run: npm run golden:update',
              ].join('\n');
              
              throw new Error(message);
            }
          } else {
            throw new Error(
              `Golden SVG file not found: ${svgFile}\n` +
              `Run 'npm run golden:generate' to create golden files for new test cases.`
            );
          }
        }
      });
    });
  });

  // Test to ensure we have at least some test cases
  it('should have test cases', () => {
    expect(arkFiles.length).toBeGreaterThan(0);
  });
});