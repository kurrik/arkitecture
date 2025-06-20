/**
 * Tests for SVG generator
 */

import { SvgGenerator, generateSVG } from '../../src/generator/svg-generator';
import { Document, ContainerNode } from '../../src/types';
import { LayoutResult, NodeDimensions, AnchorPosition, calculateLayout } from '../../src/generator/layout';
import { TextMeasurement } from '../../src/generator/text-measurement';

describe('SvgGenerator', () => {
  let svgGenerator: SvgGenerator;

  beforeEach(() => {
    svgGenerator = new SvgGenerator();
  });

  describe('single node SVG generation', () => {
    test('generates SVG for single node with label', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test Node',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain SVG structure
      expect(svg).toContain('<svg xmlns="http://www.w3.org/2000/svg"');
      expect(svg).toContain('</svg>');

      // Should contain defs with arrowhead marker
      expect(svg).toContain('<defs>');
      expect(svg).toContain('<marker id="arrowhead"');
      expect(svg).toContain('</defs>');

      // Should contain rectangle for node
      expect(svg).toContain('<rect');
      expect(svg).toContain('fill="white"');
      expect(svg).toContain('stroke="black"');
      expect(svg).toContain('stroke-width="1"');

      // Should contain text for label
      expect(svg).toContain('<text');
      expect(svg).toContain('Test Node');
      expect(svg).toContain('text-anchor="middle"');
      expect(svg).toContain('font-family="Arial"');
      expect(svg).toContain('font-size="12"');
    });

    test('generates SVG for node without label', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain rectangle but no text
      expect(svg).toContain('<rect');
      expect(svg).not.toContain('<text');
    });

    test('handles multi-line labels', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Line 1\nLine 2\nLine 3',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should use tspan for multi-line text
      expect(svg).toContain('<text');
      expect(svg).toContain('<tspan');
      expect(svg).toContain('Line 1');
      expect(svg).toContain('Line 2');
      expect(svg).toContain('Line 3');
    });
  });

  describe('multiple nodes with proper positioning', () => {
    test('generates positioned rectangles for multiple nodes', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: [],
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain two rectangles with different x positions
      const rectMatches = svg.match(/<rect[^>]*>/g);
      expect(rectMatches).toHaveLength(2);

      // Should contain two text elements
      const textMatches = svg.match(/<text[^>]*>.*?<\/text>/g);
      expect(textMatches).toHaveLength(2);

      expect(svg).toContain('Node 1');
      expect(svg).toContain('Node 2');
    });

    test('handles nested node structures', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'vertical',
            children: [
              {
                id: 'child1',
                label: 'Child 1',
                children: [],
              },
              {
                id: 'child2',
                label: 'Child 2',
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain three rectangles (parent + 2 children)
      const rectMatches = svg.match(/<rect[^>]*>/g);
      expect(rectMatches).toHaveLength(3);

      // Should contain all labels
      expect(svg).toContain('Parent');
      expect(svg).toContain('Child 1');
      expect(svg).toContain('Child 2');
    });

    test('groups have no visual representation', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            children: [
              {
                direction: 'horizontal',
                children: [
                  {
                    id: 'child1',
                    label: 'Child 1',
                    children: [],
                  },
                  {
                    id: 'child2',
                    label: 'Child 2',
                    children: [],
                  },
                ],
              },
            ],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should only contain rectangles for container nodes (parent + 2 children = 3)
      const rectMatches = svg.match(/<rect[^>]*>/g);
      expect(rectMatches).toHaveLength(3);

      // Should contain labels for container nodes only
      expect(svg).toContain('Parent');
      expect(svg).toContain('Child 1');
      expect(svg).toContain('Child 2');
    });
  });

  describe('arrow generation between nodes', () => {
    test('generates arrow between two nodes', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: [],
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: [],
          },
        ],
        arrows: [
          {
            source: 'node1',
            target: 'node2',
          },
        ],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain line element for arrow
      expect(svg).toContain('<line');
      expect(svg).toContain('stroke="black"');
      expect(svg).toContain('marker-end="url(#arrowhead)"');

      // Should have x1, y1, x2, y2 coordinates
      expect(svg).toMatch(/x1="\d+(\.\d+)?"/);
      expect(svg).toMatch(/y1="\d+(\.\d+)?"/);
      expect(svg).toMatch(/x2="\d+(\.\d+)?"/);
      expect(svg).toMatch(/y2="\d+(\.\d+)?"/);
    });

    test('generates arrow with anchor reference', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            anchors: {
              right: [1.0, 0.5],
            },
            children: [],
          },
          {
            id: 'node2',
            label: 'Node 2',
            anchors: {
              left: [0.0, 0.5],
            },
            children: [],
          },
        ],
        arrows: [
          {
            source: 'node1',
            target: 'node2#left',
          },
        ],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain arrow line
      expect(svg).toContain('<line');
      expect(svg).toContain('marker-end="url(#arrowhead)"');
    });

    test('skips arrows with missing anchor references', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: [],
          },
        ],
        arrows: [
          {
            source: 'node1',
            target: 'nonexistent',
          },
        ],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should not contain any arrow lines
      expect(svg).not.toContain('<line');
    });
  });

  describe('SVG structure and valid XML', () => {
    test('generates valid XML structure', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should start and end with svg tags
      expect(svg).toMatch(/^<svg[^>]*>/);
      expect(svg).toMatch(/<\/svg>$/);

      // Should contain proper namespace
      expect(svg).toContain('xmlns="http://www.w3.org/2000/svg"');

      // Should have width and height attributes
      expect(svg).toMatch(/width="\d+"/);
      expect(svg).toMatch(/height="\d+"/);

      // Should be well-formed XML (basic check for balanced tags)
      const openTags = (svg.match(/</g) || []).length;
      const closeTags = (svg.match(/>/g) || []).length;
      expect(openTags).toBe(closeTags);
    });

    test('escapes XML special characters in text', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Text with <special> & "chars"',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should escape special characters
      expect(svg).toContain('&lt;special&gt;');
      expect(svg).toContain('&amp;');
      expect(svg).toContain('&quot;chars&quot;');
      
      // Should not contain unescaped characters
      expect(svg).not.toContain('Text with <special> & "chars"');
    });

    test('generates proper canvas dimensions', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Small',
            children: [],
          },
          {
            id: 'node2',
            label: 'Much Longer Label',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Extract width and height from SVG
      const widthMatch = svg.match(/width="(\d+)"/);
      const heightMatch = svg.match(/height="(\d+)"/);

      expect(widthMatch).toBeTruthy();
      expect(heightMatch).toBeTruthy();

      if (widthMatch && heightMatch) {
        const width = parseInt(widthMatch[1]);
        const height = parseInt(heightMatch[1]);

        // Should match layout canvas dimensions
        expect(width).toBe(layout.canvasWidth);
        expect(height).toBe(layout.canvasHeight);

        // Should be positive
        expect(width).toBeGreaterThan(0);
        expect(height).toBeGreaterThan(0);
      }
    });
  });

  describe('text rendering and positioning', () => {
    test('centers text in nodes', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Centered',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should use text-anchor="middle" and dominant-baseline="middle"
      expect(svg).toContain('text-anchor="middle"');
      expect(svg).toContain('dominant-baseline="middle"');

      // Text position should be center of the node
      const nodeDims = layout.nodeDimensions.node1;
      const expectedX = nodeDims.x + nodeDims.width / 2;
      const expectedY = nodeDims.y + nodeDims.height / 2;

      expect(svg).toContain(`x="${expectedX}"`);
      expect(svg).toContain(`y="${expectedY}"`);
    });

    test('uses correct font settings', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should use default Arial 12px
      expect(svg).toContain('font-family="Arial"');
      expect(svg).toContain('font-size="12"');
    });

    test('allows custom font settings', () => {
      const customGenerator = new SvgGenerator({
        fontSize: 16,
        fontFamily: 'Helvetica',
      });

      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Custom Font',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = customGenerator.generateSVG(document, layout);

      expect(svg).toContain('font-family="Helvetica"');
      expect(svg).toContain('font-size="16"');
    });
  });

  describe('arrowhead marker generation', () => {
    test('includes arrowhead marker definition', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: [],
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: [],
          },
        ],
        arrows: [
          {
            source: 'node1',
            target: 'node2',
          },
        ],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should contain marker definition
      expect(svg).toContain('<marker id="arrowhead"');
      expect(svg).toContain('markerWidth="10"');
      expect(svg).toContain('markerHeight="7"');
      expect(svg).toContain('orient="auto"');
      expect(svg).toContain('<polygon points="0 0, 10 3.5, 0 7"');
    });

    test('arrows reference arrowhead marker', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: [],
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: [],
          },
        ],
        arrows: [
          {
            source: 'node1',
            target: 'node2',
          },
        ],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Arrow should reference the marker
      expect(svg).toContain('marker-end="url(#arrowhead)"');
    });
  });

  describe('utility function', () => {
    test('generateSVG utility function works', () => {
      const document: Document = {
        nodes: [
          {
            id: 'test',
            label: 'Test',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = generateSVG(document, layout);

      expect(svg).toContain('<svg');
      expect(svg).toContain('Test');
    });

    test('generateSVG with custom options', () => {
      const document: Document = {
        nodes: [
          {
            id: 'test',
            label: 'Test',
            children: [],
          },
        ],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = generateSVG(document, layout, {
        fontSize: 20,
        fontFamily: 'Times',
      });

      expect(svg).toContain('font-size="20"');
      expect(svg).toContain('font-family="Times"');
    });
  });

  describe('edge cases', () => {
    test('handles empty document', () => {
      const document: Document = {
        nodes: [],
        arrows: [],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should still generate valid SVG structure
      expect(svg).toContain('<svg');
      expect(svg).toContain('</svg>');
      expect(svg).toContain('<defs>');

      // Should have zero dimensions
      expect(svg).toContain('width="0"');
      expect(svg).toContain('height="0"');
    });

    test('handles document with only arrows (no nodes)', () => {
      const document: Document = {
        nodes: [],
        arrows: [
          {
            source: 'nonexistent1',
            target: 'nonexistent2',
          },
        ],
      };

      const layout = calculateLayout(document);
      const svg = svgGenerator.generateSVG(document, layout);

      // Should not contain any arrows since nodes don't exist
      expect(svg).not.toContain('<line');
    });
  });
});