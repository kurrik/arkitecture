#!/usr/bin/env node

/**
 * Script to generate golden test outputs
 * Usage: npm run golden:generate
 */

import fs from 'fs';
import path from 'path';
import arkitectureToSVG from '../src/arkitecture';
import { Options } from '../src/types';

const goldenDir = path.join(__dirname, '..', '..', 'tests', 'golden', 'examples');

function generateGoldenOutputs() {
  if (!fs.existsSync(goldenDir)) {
    console.error(`Golden test directory not found: ${goldenDir}`);
    process.exit(1);
  }

  // Find all .ark files
  const arkFiles = fs.readdirSync(goldenDir)
    .filter(file => file.endsWith('.ark'))
    .map(file => path.basename(file, '.ark'));

  console.log(`Found ${arkFiles.length} test cases to process...`);

  let generated = 0;
  let errors = 0;

  arkFiles.forEach(testCase => {
    console.log(`\nProcessing: ${testCase}`);
    
    const arkFile = path.join(goldenDir, `${testCase}.ark`);
    const configFile = path.join(goldenDir, `${testCase}.json`);
    const svgFile = path.join(goldenDir, `${testCase}.svg`);
    const errorFile = path.join(goldenDir, `${testCase}.error`);

    try {
      // Read input files
      const arkContent = fs.readFileSync(arkFile, 'utf-8');
      
      let config: Options = {};
      if (fs.existsSync(configFile)) {
        const configContent = fs.readFileSync(configFile, 'utf-8');
        config = JSON.parse(configContent);
      }

      // Generate result
      console.log(`  Generating output for ${testCase}...`);
      const result = arkitectureToSVG(arkContent, config);

      if (result.success && result.svg) {
        // Write SVG output
        fs.writeFileSync(svgFile, result.svg);
        console.log(`  ✓ Generated SVG: ${path.basename(svgFile)}`);
        
        // Remove error file if it exists (this is now a success case)
        if (fs.existsSync(errorFile)) {
          fs.unlinkSync(errorFile);
          console.log(`  ✓ Removed old error file: ${path.basename(errorFile)}`);
        }
        
        generated++;
      } else {
        // Write error output
        const errorOutput = {
          type: result.errors[0]?.type || 'unknown',
          line: result.errors[0]?.line,
          column: result.errors[0]?.column,
          messageContains: extractKeyMessage(result.errors[0]?.message || ''),
        };
        
        fs.writeFileSync(errorFile, JSON.stringify(errorOutput, null, 2));
        console.log(`  ✓ Generated error file: ${path.basename(errorFile)}`);
        
        // Remove SVG file if it exists (this is now an error case)
        if (fs.existsSync(svgFile)) {
          fs.unlinkSync(svgFile);
          console.log(`  ✓ Removed old SVG file: ${path.basename(svgFile)}`);
        }
        
        errors++;
      }
    } catch (error) {
      console.error(`  ✗ Failed to process ${testCase}:`, error);
    }
  });

  console.log(`\n=== Summary ===`);
  console.log(`Test cases processed: ${arkFiles.length}`);
  console.log(`SVG files generated: ${generated}`);
  console.log(`Error files generated: ${errors}`);
  console.log(`\nGolden files updated successfully!`);
}

/**
 * Extract key part of error message for comparison
 * This helps make tests more stable by focusing on the important part
 */
function extractKeyMessage(message: string): string {
  // Remove file paths and line numbers that might vary
  return message
    .replace(/at line \d+/g, 'at line X')
    .replace(/column \d+/g, 'column X')
    .replace(/Internal error: .*/, 'Internal error')
    .trim();
}

if (require.main === module) {
  generateGoldenOutputs();
}