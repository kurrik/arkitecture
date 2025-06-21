/**
 * SVG generation from layout results
 */

import { Document, ContainerNode, Arrow } from '../types';
import { LayoutResult, NodeDimensions, findAnchorPosition } from './layout';

export interface SvgGenerationOptions {
  fontSize?: number;
  fontFamily?: string;
}

export class SvgGenerator {
  private fontSize: number;
  private fontFamily: string;

  constructor(options?: SvgGenerationOptions) {
    this.fontSize = options?.fontSize || 12;
    this.fontFamily = options?.fontFamily || 'Arial';
  }

  /**
   * Generate SVG from document and layout results
   */
  generateSVG(document: Document, layout: LayoutResult): string {
    const defs = this.generateDefs();
    const nodes = this.generateNodes(document, layout);
    const arrows = this.generateArrows(document.arrows, layout);

    return this.assembleDocument(layout.canvasWidth, layout.canvasHeight, defs, nodes, arrows);
  }

  /**
   * Generate SVG defs section with arrowhead markers
   */
  private generateDefs(): string {
    return `  <defs>
    <marker id="arrowhead" markerWidth="10" markerHeight="7" 
            refX="9" refY="3.5" orient="auto" markerUnits="strokeWidth">
      <polygon points="0 0, 10 3.5, 0 7" fill="black" />
    </marker>
  </defs>`;
  }

  /**
   * Generate rect and text elements for all nodes
   */
  private generateNodes(document: Document, layout: LayoutResult): string {
    const elements: string[] = [];

    // Process all nodes recursively
    this.collectNodeElements(document.nodes, layout, elements);

    return elements.join('\n');
  }

  /**
   * Recursively collect SVG elements for nodes
   */
  private collectNodeElements(
    nodes: (ContainerNode | import('../types').GroupNode)[],
    layout: LayoutResult,
    elements: string[]
  ): void {
    for (const node of nodes) {
      if ('id' in node) {
        // Container node - render it
        const dimensions = layout.nodeDimensions[node.id];
        if (dimensions) {
          elements.push(this.generateNodeRect(node.id, dimensions));
          if (node.label) {
            elements.push(this.generateNodeText(node.label, dimensions));
          }
        }
      }

      // Recursively process children (for both container nodes and groups)
      this.collectNodeElements(node.children, layout, elements);
    }
  }

  /**
   * Generate rectangle element for a node
   */
  private generateNodeRect(_nodeId: string, dimensions: NodeDimensions): string {
    return `  <rect x="${dimensions.x}" y="${dimensions.y}" ` +
           `width="${dimensions.width}" height="${dimensions.height}" ` +
           `fill="white" stroke="black" stroke-width="1" />`;
  }

  /**
   * Generate text element for a node label
   */
  private generateNodeText(label: string, dimensions: NodeDimensions): string {
    const centerX = dimensions.x + dimensions.width / 2;
    const centerY = dimensions.y + dimensions.height / 2;

    // Handle multi-line text
    const lines = label.split('\n');
    
    if (lines.length === 1) {
      // Single line text
      return `  <text x="${centerX}" y="${centerY}" text-anchor="middle" ` +
             `dominant-baseline="middle" font-family="${this.fontFamily}" ` +
             `font-size="${this.fontSize}">${this.escapeXml(label)}</text>`;
    } else {
      // Multi-line text using tspan
      const lineHeight = this.fontSize * 1.2;
      const totalHeight = (lines.length - 1) * lineHeight;
      const startY = centerY - totalHeight / 2;

      let result = `  <text x="${centerX}" y="${startY}" text-anchor="middle" ` +
                   `dominant-baseline="middle" font-family="${this.fontFamily}" ` +
                   `font-size="${this.fontSize}">`;

      for (let i = 0; i < lines.length; i++) {
        const y = i === 0 ? 0 : lineHeight;
        result += `\n    <tspan x="${centerX}" dy="${y}">${this.escapeXml(lines[i])}</tspan>`;
      }

      result += '\n  </text>';
      return result;
    }
  }

  /**
   * Generate line elements with arrowhead markers for all arrows
   */
  private generateArrows(arrows: Arrow[], layout: LayoutResult): string {
    const elements: string[] = [];

    for (const arrow of arrows) {
      const arrowElement = this.generateArrow(arrow, layout);
      if (arrowElement) {
        elements.push(arrowElement);
      }
    }

    return elements.join('\n');
  }

  /**
   * Generate a single arrow line element
   */
  private generateArrow(arrow: Arrow, layout: LayoutResult): string | null {
    // Parse source and target (both can have anchor references)
    const [sourceNodePath, sourceAnchorId] = this.parseTarget(arrow.source);
    const [targetNodePath, targetAnchorId] = this.parseTarget(arrow.target);

    // Find anchor positions
    const sourceAnchor = findAnchorPosition(layout.anchorPositions, sourceNodePath, sourceAnchorId);
    const targetAnchor = findAnchorPosition(layout.anchorPositions, targetNodePath, targetAnchorId);

    if (!sourceAnchor || !targetAnchor) {
      // Skip arrows with missing anchors (should have been caught by validation)
      return null;
    }

    return `  <line x1="${sourceAnchor.x}" y1="${sourceAnchor.y}" ` +
           `x2="${targetAnchor.x}" y2="${targetAnchor.y}" ` +
           `stroke="black" stroke-width="1" marker-end="url(#arrowhead)" />`;
  }

  /**
   * Parse arrow target to extract node path and anchor ID
   */
  private parseTarget(target: string): [string, string] {
    const anchorSeparatorIndex = target.indexOf('#');
    
    if (anchorSeparatorIndex === -1) {
      // No anchor specified, use center
      return [target, 'center'];
    }

    const nodePath = target.substring(0, anchorSeparatorIndex);
    const anchorId = target.substring(anchorSeparatorIndex + 1);
    
    return [nodePath, anchorId];
  }

  /**
   * Escape XML special characters
   */
  private escapeXml(text: string): string {
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  /**
   * Assemble complete SVG document
   */
  private assembleDocument(
    width: number,
    height: number,
    defs: string,
    nodes: string,
    arrows: string
  ): string {
    const elements: string[] = [];

    elements.push(`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}">`);
    elements.push(defs);
    
    if (nodes) {
      elements.push('');
      elements.push('  <!-- Node rectangles and labels -->');
      elements.push(nodes);
    }
    
    if (arrows) {
      elements.push('');
      elements.push('  <!-- Arrows -->');
      elements.push(arrows);
    }
    
    elements.push('</svg>');

    return elements.join('\n');
  }
}

// Default instance for convenience
export const defaultSvgGenerator = new SvgGenerator();

// Utility function
export function generateSVG(
  document: Document,
  layout: LayoutResult,
  options?: SvgGenerationOptions
): string {
  const generator = options ? new SvgGenerator(options) : defaultSvgGenerator;
  return generator.generateSVG(document, layout);
}