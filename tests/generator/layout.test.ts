/**
 * Tests for the layout algorithm
 */

import { LayoutEngine, calculateLayout } from '../../src/generator/layout';
import { Document, ContainerNode, GroupNode } from '../../src/types';
import { TextMeasurement } from '../../src/generator/text-measurement';

describe('LayoutEngine', () => {
  let layoutEngine: LayoutEngine;
  let textMeasurement: TextMeasurement;

  beforeEach(() => {
    // Use fixed font metrics for predictable testing
    textMeasurement = new TextMeasurement({ size: 12, lineHeight: 1.2 });
    layoutEngine = new LayoutEngine(textMeasurement);
  });

  describe('single node layout', () => {
    test('calculates dimensions for single leaf node with text', () => {
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

      const result = layoutEngine.calculateLayout(document);

      expect(result.nodeDimensions.node1).toBeDefined();
      const dims = result.nodeDimensions.node1;
      
      // Should have positive dimensions including text + border
      expect(dims.width).toBeGreaterThan(0);
      expect(dims.height).toBeGreaterThan(0);
      expect(dims.x).toBe(0);
      expect(dims.y).toBe(0);

      // Canvas should match node dimensions
      expect(result.canvasWidth).toBe(dims.width);
      expect(result.canvasHeight).toBe(dims.height);
    });

    test('handles node with no label', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);
      const dims = result.nodeDimensions.node1;

      // Should have minimum dimensions
      expect(dims.width).toBeGreaterThan(0);
      expect(dims.height).toBeGreaterThan(0);
    });

    test('applies size override correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test',
            size: 0.5,
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      // For a single node with vertical direction (default), size affects width
      const dims = result.nodeDimensions.node1;
      expect(dims.width).toBeGreaterThan(0);
      expect(dims.height).toBeGreaterThan(0);
    });
  });

  describe('simple parent-child vertical layout', () => {
    test('calculates vertical layout correctly', () => {
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

      const result = layoutEngine.calculateLayout(document);

      const parentDims = result.nodeDimensions.parent;
      const child1Dims = result.nodeDimensions.child1;
      const child2Dims = result.nodeDimensions.child2;

      // Vertical layout: parent height = sum of child heights
      expect(parentDims.height).toBe(child1Dims.height + child2Dims.height);
      
      // Vertical layout: parent width = max child width, children width = parent width
      expect(parentDims.width).toBe(Math.max(child1Dims.width, child2Dims.width));
      expect(child1Dims.width).toBe(parentDims.width);
      expect(child2Dims.width).toBe(parentDims.width);

      // Children should be positioned vertically
      expect(child1Dims.x).toBe(parentDims.x);
      expect(child1Dims.y).toBe(parentDims.y);
      expect(child2Dims.x).toBe(parentDims.x);
      expect(child2Dims.y).toBe(parentDims.y + child1Dims.height);
    });
  });

  describe('simple parent-child horizontal layout', () => {
    test('calculates horizontal layout correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
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
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      const parentDims = result.nodeDimensions.parent;
      const child1Dims = result.nodeDimensions.child1;
      const child2Dims = result.nodeDimensions.child2;

      // Horizontal layout: parent width = sum of child widths
      expect(parentDims.width).toBe(child1Dims.width + child2Dims.width);
      
      // Horizontal layout: parent height = max child height, children height = parent height
      expect(parentDims.height).toBe(Math.max(child1Dims.height, child2Dims.height));
      expect(child1Dims.height).toBe(parentDims.height);
      expect(child2Dims.height).toBe(parentDims.height);

      // Children should be positioned horizontally
      expect(child1Dims.x).toBe(parentDims.x);
      expect(child1Dims.y).toBe(parentDims.y);
      expect(child2Dims.x).toBe(parentDims.x + child1Dims.width);
      expect(child2Dims.y).toBe(parentDims.y);
    });
  });

  describe('size override behavior', () => {
    test('applies size override to orthogonal dimension in vertical layout', () => {
      const baseDocument: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'vertical',
            children: [
              {
                id: 'child',
                label: 'Child',
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const overrideDocument: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'vertical',
            size: 0.5,
            children: [
              {
                id: 'child',
                label: 'Child',
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const baseResult = layoutEngine.calculateLayout(baseDocument);
      const overrideResult = layoutEngine.calculateLayout(overrideDocument);

      // In vertical layout, size affects width (orthogonal dimension)
      expect(overrideResult.nodeDimensions.parent.width).toBeLessThan(
        baseResult.nodeDimensions.parent.width
      );
      
      // Height should remain the same
      expect(overrideResult.nodeDimensions.parent.height).toBe(
        baseResult.nodeDimensions.parent.height
      );
    });

    test('applies size override to orthogonal dimension in horizontal layout', () => {
      const baseDocument: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'horizontal',
            children: [
              {
                id: 'child',
                label: 'Child',
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const overrideDocument: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'horizontal',
            size: 0.5,
            children: [
              {
                id: 'child',
                label: 'Child',
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const baseResult = layoutEngine.calculateLayout(baseDocument);
      const overrideResult = layoutEngine.calculateLayout(overrideDocument);

      // In horizontal layout, size affects height (orthogonal dimension)
      expect(overrideResult.nodeDimensions.parent.height).toBeLessThan(
        baseResult.nodeDimensions.parent.height
      );
      
      // Width should remain the same
      expect(overrideResult.nodeDimensions.parent.width).toBe(
        baseResult.nodeDimensions.parent.width
      );
    });
  });

  describe('nested layout calculations', () => {
    test('handles deeply nested structures', () => {
      const document: Document = {
        nodes: [
          {
            id: 'root',
            label: 'Root',
            direction: 'vertical',
            children: [
              {
                id: 'level1',
                label: 'Level 1',
                direction: 'horizontal',
                children: [
                  {
                    id: 'level2a',
                    label: 'Level 2A',
                    children: [],
                  },
                  {
                    id: 'level2b',
                    label: 'Level 2B',
                    children: [],
                  },
                ],
              },
              {
                id: 'level1b',
                label: 'Level 1B',
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      // All nodes should have valid dimensions
      expect(result.nodeDimensions.root).toBeDefined();
      expect(result.nodeDimensions.level1).toBeDefined();
      expect(result.nodeDimensions.level1b).toBeDefined();
      expect(result.nodeDimensions.level2a).toBeDefined();
      expect(result.nodeDimensions.level2b).toBeDefined();

      // Root should contain all its children
      const rootDims = result.nodeDimensions.root;
      const level1Dims = result.nodeDimensions.level1;
      const level1bDims = result.nodeDimensions.level1b;

      expect(rootDims.height).toBe(level1Dims.height + level1bDims.height);
      expect(rootDims.width).toBe(Math.max(level1Dims.width, level1bDims.width));
    });
  });

  describe('group layout', () => {
    test('groups have no visual representation but affect layout', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'vertical',
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

      const result = layoutEngine.calculateLayout(document);

      // Only container nodes should appear in dimensions
      expect(result.nodeDimensions.parent).toBeDefined();
      expect(result.nodeDimensions.child1).toBeDefined();
      expect(result.nodeDimensions.child2).toBeDefined();

      // Children should be laid out horizontally within the parent
      const child1Dims = result.nodeDimensions.child1;
      const child2Dims = result.nodeDimensions.child2;

      expect(child2Dims.x).toBe(child1Dims.x + child1Dims.width);
      expect(child1Dims.y).toBe(child2Dims.y); // Same vertical position
    });
  });

  describe('multiple top-level nodes', () => {
    test('positions multiple root nodes horizontally', () => {
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

      const result = layoutEngine.calculateLayout(document);

      const node1Dims = result.nodeDimensions.node1;
      const node2Dims = result.nodeDimensions.node2;

      // First node at origin
      expect(node1Dims.x).toBe(0);
      expect(node1Dims.y).toBe(0);

      // Second node to the right of first
      expect(node2Dims.x).toBe(node1Dims.width);
      expect(node2Dims.y).toBe(0);

      // Canvas should contain both nodes
      expect(result.canvasWidth).toBe(node1Dims.width + node2Dims.width);
      expect(result.canvasHeight).toBe(Math.max(node1Dims.height, node2Dims.height));
    });
  });

  describe('utility function', () => {
    test('calculateLayout utility function works', () => {
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

      const result = calculateLayout(document);
      expect(result.nodeDimensions.test).toBeDefined();
    });

    test('calculateLayout with custom text measurement', () => {
      const customTextMeasurement = new TextMeasurement({ size: 16 });
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

      const result = calculateLayout(document, customTextMeasurement);
      expect(result.nodeDimensions.test).toBeDefined();
    });
  });

  describe('edge cases', () => {
    test('handles empty document', () => {
      const document: Document = {
        nodes: [],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      expect(result.nodeDimensions).toEqual({});
      expect(result.canvasWidth).toBe(0);
      expect(result.canvasHeight).toBe(0);
    });

    test('handles nodes with empty children arrays', () => {
      const document: Document = {
        nodes: [
          {
            id: 'empty',
            label: 'Empty Parent',
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);
      expect(result.nodeDimensions.empty).toBeDefined();
      expect(result.nodeDimensions.empty.width).toBeGreaterThan(0);
      expect(result.nodeDimensions.empty.height).toBeGreaterThan(0);
    });
  });
});