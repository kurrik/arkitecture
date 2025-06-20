import { 
  TextMeasurement, 
  FontConfig, 
  defaultTextMeasurement,
  calculateTextWidth,
  calculateTextHeight,
  getTextDimensions 
} from '../../src/generator/text-measurement';

describe('TextMeasurement', () => {
  describe('Basic text measurement', () => {
    it('should calculate width for simple text', () => {
      const textMeasurement = new TextMeasurement();
      const width = textMeasurement.calculateTextWidth('Hello World');
      
      expect(width).toBeGreaterThan(0);
      expect(typeof width).toBe('number');
      expect(Number.isInteger(width)).toBe(true);
    });

    it('should calculate height for simple text', () => {
      const textMeasurement = new TextMeasurement();
      const height = textMeasurement.calculateTextHeight('Hello World');
      
      expect(height).toBeGreaterThan(0);
      expect(typeof height).toBe('number');
      expect(Number.isInteger(height)).toBe(true);
    });

    it('should return both dimensions from getTextDimensions', () => {
      const textMeasurement = new TextMeasurement();
      const dimensions = textMeasurement.getTextDimensions('Hello World');
      
      expect(dimensions).toHaveProperty('width');
      expect(dimensions).toHaveProperty('height');
      expect(dimensions.width).toBeGreaterThan(0);
      expect(dimensions.height).toBeGreaterThan(0);
    });

    it('should handle empty string gracefully', () => {
      const textMeasurement = new TextMeasurement();
      
      expect(textMeasurement.calculateTextWidth('')).toBe(0);
      expect(textMeasurement.calculateTextHeight('')).toBe(0);
      
      const dimensions = textMeasurement.getTextDimensions('');
      expect(dimensions.width).toBe(0);
      expect(dimensions.height).toBe(0);
    });

    it('should handle null/undefined text gracefully', () => {
      const textMeasurement = new TextMeasurement();
      
      const dimensions = textMeasurement.getTextDimensions();
      expect(dimensions.width).toBe(0);
      expect(dimensions.height).toBe(0);
      
      const dimensionsUndefined = textMeasurement.getTextDimensions(undefined);
      expect(dimensionsUndefined.width).toBe(0);
      expect(dimensionsUndefined.height).toBe(0);
    });
  });

  describe('Multi-line text handling', () => {
    it('should handle multi-line text with newlines', () => {
      const textMeasurement = new TextMeasurement();
      const singleLine = textMeasurement.getTextDimensions('Hello');
      const multiLine = textMeasurement.getTextDimensions('Hello\nWorld');
      
      // Multi-line should be taller
      expect(multiLine.height).toBeGreaterThan(singleLine.height);
      
      // Width should be the maximum of all lines
      expect(multiLine.width).toBeGreaterThan(0);
    });

    it('should calculate height correctly for multiple lines', () => {
      const textMeasurement = new TextMeasurement();
      const oneLine = textMeasurement.calculateTextHeight('Single');
      const twoLines = textMeasurement.calculateTextHeight('Line 1\nLine 2');
      const threeLines = textMeasurement.calculateTextHeight('Line 1\nLine 2\nLine 3');
      
      expect(twoLines).toBeGreaterThan(oneLine);
      expect(threeLines).toBeGreaterThan(twoLines);
      
      // Should be approximately double and triple (allow for rounding)
      // Due to rounding in height calculations, allow up to 2 pixels difference
      expect(Math.abs(twoLines - (oneLine * 2))).toBeLessThanOrEqual(2);
      expect(Math.abs(threeLines - (oneLine * 3))).toBeLessThanOrEqual(3);
    });

    it('should handle width correctly for lines of different lengths', () => {
      const textMeasurement = new TextMeasurement();
      
      // Test with short and long lines
      const text = 'Short\nThis is a much longer line of text';
      const width = textMeasurement.calculateTextWidth(text);
      
      const shortLineWidth = textMeasurement.calculateTextWidth('Short');
      const longLineWidth = textMeasurement.calculateTextWidth('This is a much longer line of text');
      
      // Multi-line width should equal the longest line
      expect(width).toBe(longLineWidth);
      expect(width).toBeGreaterThan(shortLineWidth);
    });
  });

  describe('Font size variations', () => {
    it('should scale width with font size', () => {
      const textMeasurement = new TextMeasurement();
      const text = 'Sample Text';
      
      const width12 = textMeasurement.calculateTextWidth(text, 12);
      const width24 = textMeasurement.calculateTextWidth(text, 24);
      
      expect(width24).toBeGreaterThan(width12);
      expect(width24).toBeCloseTo(width12 * 2, -1); // Approximately double
    });

    it('should scale height with font size', () => {
      const textMeasurement = new TextMeasurement();
      const text = 'Sample Text';
      
      const height12 = textMeasurement.calculateTextHeight(text, 12);
      const height24 = textMeasurement.calculateTextHeight(text, 24);
      
      expect(height24).toBeGreaterThan(height12);
      expect(height24).toBeCloseTo(height12 * 2, -1); // Approximately double
    });

    it('should use default font size when not specified', () => {
      const textMeasurement = new TextMeasurement({ size: 16 });
      const text = 'Sample Text';
      
      const widthDefault = textMeasurement.calculateTextWidth(text);
      const widthExplicit = textMeasurement.calculateTextWidth(text, 16);
      
      expect(widthDefault).toBe(widthExplicit);
    });
  });

  describe('Font configuration', () => {
    it('should use default font configuration', () => {
      const textMeasurement = new TextMeasurement();
      const config = textMeasurement.getFontConfig();
      
      expect(config.family).toBe('Arial');
      expect(config.size).toBe(12);
      expect(config.lineHeight).toBe(1.2);
    });

    it('should allow custom font configuration', () => {
      const customConfig: Partial<FontConfig> = {
        family: 'Helvetica',
        size: 14,
        lineHeight: 1.5,
      };
      
      const textMeasurement = new TextMeasurement(customConfig);
      const config = textMeasurement.getFontConfig();
      
      expect(config.family).toBe('Helvetica');
      expect(config.size).toBe(14);
      expect(config.lineHeight).toBe(1.5);
    });

    it('should allow partial font configuration updates', () => {
      const textMeasurement = new TextMeasurement();
      
      textMeasurement.setFontConfig({ size: 18 });
      const config = textMeasurement.getFontConfig();
      
      expect(config.family).toBe('Arial'); // Should keep default
      expect(config.size).toBe(18); // Should be updated
      expect(config.lineHeight).toBe(1.2); // Should keep default
    });

    it('should affect height calculations when line height changes', () => {
      const text = 'Line 1\nLine 2';
      
      const textMeasurement1 = new TextMeasurement({ lineHeight: 1.0 });
      const textMeasurement2 = new TextMeasurement({ lineHeight: 2.0 });
      
      const height1 = textMeasurement1.calculateTextHeight(text);
      const height2 = textMeasurement2.calculateTextHeight(text);
      
      expect(height2).toBeGreaterThan(height1);
      expect(height2).toBeCloseTo(height1 * 2, -1);
    });
  });

  describe('Special character handling', () => {
    it('should handle unicode characters', () => {
      const textMeasurement = new TextMeasurement();
      
      const ascii = textMeasurement.calculateTextWidth('Hello');
      const unicode = textMeasurement.calculateTextWidth('Héllø');
      const emoji = textMeasurement.calculateTextWidth('Hello 😀');
      
      expect(ascii).toBeGreaterThan(0);
      expect(unicode).toBeGreaterThan(0);
      expect(emoji).toBeGreaterThan(ascii);
    });

    it('should handle wide characters correctly', () => {
      const textMeasurement = new TextMeasurement();
      
      const narrow = textMeasurement.calculateTextWidth('Hello');
      const wide = textMeasurement.calculateTextWidth('你好世界'); // Chinese characters
      
      expect(narrow).toBeGreaterThan(0);
      expect(wide).toBeGreaterThan(0);
      // Wide characters should generally take more space
      expect(wide).toBeGreaterThan(narrow * 0.8); // Reasonable expectation
    });

    it('should handle mixed character types', () => {
      const textMeasurement = new TextMeasurement();
      
      const mixed = 'Hello 世界 😀';
      const dimensions = textMeasurement.getTextDimensions(mixed);
      
      expect(dimensions.width).toBeGreaterThan(0);
      expect(dimensions.height).toBeGreaterThan(0);
    });
  });

  describe('Minimum dimensions', () => {
    it('should provide minimum dimensions', () => {
      const textMeasurement = new TextMeasurement();
      const minDimensions = textMeasurement.getMinimumDimensions();
      
      expect(minDimensions.width).toBeGreaterThan(0);
      expect(minDimensions.height).toBeGreaterThan(0);
      expect(minDimensions.width).toBe(minDimensions.height); // Should be square
    });

    it('should scale minimum dimensions with font size', () => {
      const small = new TextMeasurement({ size: 10 });
      const large = new TextMeasurement({ size: 20 });
      
      const smallMin = small.getMinimumDimensions();
      const largeMin = large.getMinimumDimensions();
      
      expect(largeMin.width).toBeGreaterThan(smallMin.width);
      expect(largeMin.height).toBeGreaterThan(smallMin.height);
    });
  });

  describe('Default instance and utility functions', () => {
    it('should provide working default instance', () => {
      expect(defaultTextMeasurement).toBeInstanceOf(TextMeasurement);
      
      const config = defaultTextMeasurement.getFontConfig();
      expect(config.family).toBe('Arial');
      expect(config.size).toBe(12);
    });

    it('should provide utility functions that work', () => {
      const text = 'Test Text';
      
      const width = calculateTextWidth(text);
      const height = calculateTextHeight(text);
      const dimensions = getTextDimensions(text);
      
      expect(width).toBeGreaterThan(0);
      expect(height).toBeGreaterThan(0);
      expect(dimensions.width).toBe(width);
      expect(dimensions.height).toBe(height);
    });

    it('should allow custom font size in utility functions', () => {
      const text = 'Test Text';
      
      const width12 = calculateTextWidth(text, 12);
      const width24 = calculateTextWidth(text, 24);
      const height12 = calculateTextHeight(text, 12);
      const height24 = calculateTextHeight(text, 24);
      
      expect(width24).toBeGreaterThan(width12);
      expect(height24).toBeGreaterThan(height12);
    });
  });

  describe('Edge cases and error handling', () => {
    it('should handle very long text', () => {
      const textMeasurement = new TextMeasurement();
      const longText = 'A'.repeat(1000);
      
      const dimensions = textMeasurement.getTextDimensions(longText);
      expect(dimensions.width).toBeGreaterThan(0);
      expect(dimensions.height).toBeGreaterThan(0);
    });

    it('should handle text with only whitespace', () => {
      const textMeasurement = new TextMeasurement();
      
      const spaces = textMeasurement.getTextDimensions('   ');
      const newlines = textMeasurement.getTextDimensions('\n\n\n');
      
      expect(spaces.width).toBeGreaterThan(0);
      expect(spaces.height).toBeGreaterThan(0);
      expect(newlines.width).toBe(0); // Newlines alone have no width
      expect(newlines.height).toBeGreaterThan(0); // But they do have height
    });

    it('should handle text with many newlines', () => {
      const textMeasurement = new TextMeasurement();
      const manyLines = 'Line\n'.repeat(10);
      
      const dimensions = textMeasurement.getTextDimensions(manyLines);
      expect(dimensions.width).toBeGreaterThan(0);
      expect(dimensions.height).toBeGreaterThan(0);
      
      // Should be much taller than a single line
      const singleLine = textMeasurement.getTextDimensions('Line');
      expect(dimensions.height).toBeGreaterThan(singleLine.height * 5);
    });
  });

  describe('Cross-platform consistency', () => {
    it('should return consistent integer pixel values', () => {
      const textMeasurement = new TextMeasurement();
      
      const width = textMeasurement.calculateTextWidth('Test');
      const height = textMeasurement.calculateTextHeight('Test');
      
      expect(Number.isInteger(width)).toBe(true);
      expect(Number.isInteger(height)).toBe(true);
      expect(width).toBeGreaterThan(0);
      expect(height).toBeGreaterThan(0);
    });

    it('should handle different font sizes consistently', () => {
      const textMeasurement = new TextMeasurement();
      const text = 'Consistent Test';
      
      const sizes = [8, 10, 12, 14, 16, 18, 20, 24];
      const results = sizes.map(size => ({
        size,
        width: textMeasurement.calculateTextWidth(text, size),
        height: textMeasurement.calculateTextHeight(text, size),
      }));
      
      // Results should be in ascending order
      for (let i = 1; i < results.length; i++) {
        expect(results[i].width).toBeGreaterThan(results[i - 1].width);
        expect(results[i].height).toBeGreaterThan(results[i - 1].height);
      }
    });
  });
});