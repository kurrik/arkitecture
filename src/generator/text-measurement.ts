/**
 * Text measurement capabilities for layout calculations
 */

import stringWidth from 'string-width';

export interface FontConfig {
  family: string;
  size: number;
  lineHeight: number;
}

export interface TextDimensions {
  width: number;
  height: number;
}

export class TextMeasurement {
  private fontConfig: FontConfig;

  constructor(fontConfig?: Partial<FontConfig>) {
    this.fontConfig = {
      family: 'Arial',
      size: 12,
      lineHeight: 1.2,
      ...fontConfig,
    };
  }

  /**
   * Calculate the width of text in pixels
   */
  calculateTextWidth(text: string, fontSize?: number): number {
    if (!text || text.length === 0) {
      return 0;
    }

    const actualFontSize = fontSize || this.fontConfig.size;
    
    // Handle multi-line text by taking the maximum width of all lines
    const lines = text.split('\n');
    let maxWidth = 0;

    for (const line of lines) {
      // Use string-width to get character count considering unicode
      const charWidth = stringWidth(line);
      
      // Approximate pixel width based on font size
      // This is a rough approximation: average character width ≈ fontSize * 0.6
      const lineWidth = charWidth * actualFontSize * 0.6;
      
      maxWidth = Math.max(maxWidth, lineWidth);
    }

    return Math.round(maxWidth);
  }

  /**
   * Calculate the height of text in pixels
   */
  calculateTextHeight(text: string, fontSize?: number): number {
    if (!text || text.length === 0) {
      return 0;
    }

    const actualFontSize = fontSize || this.fontConfig.size;
    const lineHeight = actualFontSize * this.fontConfig.lineHeight;
    
    // Count number of lines
    const lines = text.split('\n');
    const totalHeight = lines.length * lineHeight;

    return Math.round(totalHeight);
  }

  /**
   * Get both width and height dimensions for text
   */
  getTextDimensions(text?: string, fontSize?: number): TextDimensions {
    const actualText = text || '';
    
    return {
      width: this.calculateTextWidth(actualText, fontSize),
      height: this.calculateTextHeight(actualText, fontSize),
    };
  }

  /**
   * Update font configuration
   */
  setFontConfig(fontConfig: Partial<FontConfig>): void {
    this.fontConfig = {
      ...this.fontConfig,
      ...fontConfig,
    };
  }

  /**
   * Get current font configuration
   */
  getFontConfig(): FontConfig {
    return { ...this.fontConfig };
  }

  /**
   * Calculate minimum dimensions for a node (when no text is provided)
   */
  getMinimumDimensions(): TextDimensions {
    // Minimum size is based on font size with some padding
    const minDimension = this.fontConfig.size * 2;
    return {
      width: minDimension,
      height: minDimension,
    };
  }

  /**
   * Handle special characters and normalize text for measurement
   */
  private normalizeText(text: string): string {
    // Remove any null characters or other control characters that might interfere
    return text.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '');
  }
}

// Default instance for convenience
export const defaultTextMeasurement = new TextMeasurement();

// Utility functions
export function calculateTextWidth(text: string, fontSize?: number): number {
  return defaultTextMeasurement.calculateTextWidth(text, fontSize);
}

export function calculateTextHeight(text: string, fontSize?: number): number {
  return defaultTextMeasurement.calculateTextHeight(text, fontSize);
}

export function getTextDimensions(text?: string, fontSize?: number): TextDimensions {
  return defaultTextMeasurement.getTextDimensions(text, fontSize);
}