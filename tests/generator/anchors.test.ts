/**
 * Tests for anchor position calculation
 */

import {
  LayoutEngine,
  calculateLayout,
  findAnchorPosition,
  getNodeAnchors,
  resolveNodePath,
  AnchorPosition
} from '../../src/generator/layout';
import { Document, ContainerNode } from '../../src/types';
import { TextMeasurement } from '../../src/generator/text-measurement';

describe('Anchor Position Calculation', () => {
  let layoutEngine: LayoutEngine;
  let textMeasurement: TextMeasurement;

  beforeEach(() => {
    // Use fixed font metrics for predictable testing
    textMeasurement = new TextMeasurement({ size: 12, lineHeight: 1.2 });
    layoutEngine = new LayoutEngine(textMeasurement);
  });

  describe('implicit center anchor calculation', () => {
    test('calculates center anchor for single node', () => {
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

      // Should have center anchor for the node
      const centerAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'center'
      );

      expect(centerAnchor).toBeDefined();
      if (centerAnchor) {
        const nodeDims = result.nodeDimensions.node1;
        
        // Center anchor should be at the center of the node
        expect(centerAnchor.x).toBe(nodeDims.x + nodeDims.width * 0.5);
        expect(centerAnchor.y).toBe(nodeDims.y + nodeDims.height * 0.5);
      }
    });

    test('calculates center anchors for multiple nodes', () => {
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

      // Should have center anchors for both nodes
      const node1Center = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'center'
      );
      const node2Center = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node2' && anchor.anchorId === 'center'
      );

      expect(node1Center).toBeDefined();
      expect(node2Center).toBeDefined();

      // Nodes should be positioned horizontally, so x-coordinates should differ
      if (node1Center && node2Center) {
        expect(node2Center.x).toBeGreaterThan(node1Center.x);
        expect(node1Center.y).toBe(node2Center.y); // Same y-coordinate
      }
    });
  });

  describe('custom anchor position calculation', () => {
    test('calculates custom anchor positions correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test Node',
            anchors: {
              top: [0.5, 0.0],
              bottom: [0.5, 1.0],
              left: [0.0, 0.5],
              right: [1.0, 0.5],
              topLeft: [0.0, 0.0],
              bottomRight: [1.0, 1.0],
            },
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);
      const nodeDims = result.nodeDimensions.node1;

      // Test each custom anchor position
      const topAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'top'
      );
      expect(topAnchor).toBeDefined();
      if (topAnchor) {
        expect(topAnchor.x).toBe(nodeDims.x + nodeDims.width * 0.5);
        expect(topAnchor.y).toBe(nodeDims.y + nodeDims.height * 0.0);
      }

      const bottomAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'bottom'
      );
      expect(bottomAnchor).toBeDefined();
      if (bottomAnchor) {
        expect(bottomAnchor.x).toBe(nodeDims.x + nodeDims.width * 0.5);
        expect(bottomAnchor.y).toBe(nodeDims.y + nodeDims.height * 1.0);
      }

      const leftAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'left'
      );
      expect(leftAnchor).toBeDefined();
      if (leftAnchor) {
        expect(leftAnchor.x).toBe(nodeDims.x + nodeDims.width * 0.0);
        expect(leftAnchor.y).toBe(nodeDims.y + nodeDims.height * 0.5);
      }

      const rightAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'right'
      );
      expect(rightAnchor).toBeDefined();
      if (rightAnchor) {
        expect(rightAnchor.x).toBe(nodeDims.x + nodeDims.width * 1.0);
        expect(rightAnchor.y).toBe(nodeDims.y + nodeDims.height * 0.5);
      }

      const topLeftAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'topLeft'
      );
      expect(topLeftAnchor).toBeDefined();
      if (topLeftAnchor) {
        expect(topLeftAnchor.x).toBe(nodeDims.x + nodeDims.width * 0.0);
        expect(topLeftAnchor.y).toBe(nodeDims.y + nodeDims.height * 0.0);
      }

      const bottomRightAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'bottomRight'
      );
      expect(bottomRightAnchor).toBeDefined();
      if (bottomRightAnchor) {
        expect(bottomRightAnchor.x).toBe(nodeDims.x + nodeDims.width * 1.0);
        expect(bottomRightAnchor.y).toBe(nodeDims.y + nodeDims.height * 1.0);
      }
    });

    test('includes both center and custom anchors', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test Node',
            anchors: {
              custom: [0.25, 0.75],
            },
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      // Should have both center (implicit) and custom anchor
      const centerAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'center'
      );
      const customAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'node1' && anchor.anchorId === 'custom'
      );

      expect(centerAnchor).toBeDefined();
      expect(customAnchor).toBeDefined();

      // Should have exactly 2 anchors for this node
      const nodeAnchors = result.anchorPositions.filter(
        anchor => anchor.nodeId === 'node1'
      );
      expect(nodeAnchors).toHaveLength(2);
    });
  });

  describe('anchor positions on different sized nodes', () => {
    test('anchors scale with node size', () => {
      const smallDocument: Document = {
        nodes: [
          {
            id: 'small',
            label: 'S',
            anchors: { corner: [1.0, 1.0] },
            children: [],
          },
        ],
        arrows: [],
      };

      const largeDocument: Document = {
        nodes: [
          {
            id: 'large',
            label: 'This is a much longer label\nWith multiple lines\nTo make it taller',
            anchors: { corner: [1.0, 1.0] },
            children: [],
          },
        ],
        arrows: [],
      };

      const smallResult = layoutEngine.calculateLayout(smallDocument);
      const largeResult = layoutEngine.calculateLayout(largeDocument);

      const smallCorner = smallResult.anchorPositions.find(
        anchor => anchor.nodeId === 'small' && anchor.anchorId === 'corner'
      );
      const largeCorner = largeResult.anchorPositions.find(
        anchor => anchor.nodeId === 'large' && anchor.anchorId === 'corner'
      );

      expect(smallCorner).toBeDefined();
      expect(largeCorner).toBeDefined();

      if (smallCorner && largeCorner) {
        // Large node's corner anchor should be further from origin
        expect(largeCorner.x).toBeGreaterThan(smallCorner.x);
        expect(largeCorner.y).toBeGreaterThan(smallCorner.y);
      }
    });
  });

  describe('anchor positions with size overrides', () => {
    test('anchors adjust when size override is applied', () => {
      const baseDocument: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'vertical',
            anchors: { corner: [1.0, 1.0] },
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
            anchors: { corner: [1.0, 1.0] },
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

      const baseCorner = baseResult.anchorPositions.find(
        anchor => anchor.nodeId === 'parent' && anchor.anchorId === 'corner'
      );
      const overrideCorner = overrideResult.anchorPositions.find(
        anchor => anchor.nodeId === 'parent' && anchor.anchorId === 'corner'
      );

      expect(baseCorner).toBeDefined();
      expect(overrideCorner).toBeDefined();

      if (baseCorner && overrideCorner) {
        // Size override affects width in vertical layout, so x should be different
        expect(overrideCorner.x).not.toBe(baseCorner.x);
      }
    });
  });

  describe('nested node anchor positions', () => {
    test('calculates anchors for nested nodes correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            direction: 'vertical',
            anchors: { top: [0.5, 0.0] },
            children: [
              {
                id: 'child1',
                label: 'Child 1',
                anchors: { left: [0.0, 0.5] },
                children: [],
              },
              {
                id: 'child2',
                label: 'Child 2',
                anchors: { right: [1.0, 0.5] },
                children: [],
              },
            ],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      // Should have anchors for parent and both children
      const parentAnchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'parent' && anchor.anchorId === 'top'
      );
      const child1Anchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'child1' && anchor.anchorId === 'left'
      );
      const child2Anchor = result.anchorPositions.find(
        anchor => anchor.nodeId === 'child2' && anchor.anchorId === 'right'
      );

      expect(parentAnchor).toBeDefined();
      expect(child1Anchor).toBeDefined();
      expect(child2Anchor).toBeDefined();

      // Children should be positioned vertically within parent
      if (child1Anchor && child2Anchor) {
        expect(child2Anchor.y).toBeGreaterThan(child1Anchor.y);
        
        // The anchor positions might differ based on where the anchors are within each node
        // child1 has left anchor [0.0, 0.5], child2 has right anchor [1.0, 0.5]
        // So their x positions will be different even though the nodes have the same x position
        // Let's verify the anchors are at the correct relative positions within their nodes
        const child1Dims = result.nodeDimensions.child1;
        const child2Dims = result.nodeDimensions.child2;
        
        // child1 left anchor should be at left edge of child1 node
        expect(child1Anchor.x).toBe(child1Dims.x + child1Dims.width * 0.0);
        // child2 right anchor should be at right edge of child2 node  
        expect(child2Anchor.x).toBe(child2Dims.x + child2Dims.width * 1.0);
        
        // Both nodes should have the same x position (same left edge) in vertical layout
        expect(child1Dims.x).toBe(child2Dims.x);
      }
    });
  });

  describe('edge cases', () => {
    test('handles node with no custom anchors', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'No Anchors',
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      // Should only have center anchor
      const nodeAnchors = result.anchorPositions.filter(
        anchor => anchor.nodeId === 'node1'
      );

      expect(nodeAnchors).toHaveLength(1);
      expect(nodeAnchors[0].anchorId).toBe('center');
    });

    test('skips invalid anchor coordinates', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Invalid Anchors',
            anchors: {
              valid: [0.5, 0.5],
              tooHigh: [1.5, 0.5], // Invalid: > 1.0
              tooLow: [-0.5, 0.5], // Invalid: < 0.0
              validEdge: [1.0, 0.0], // Valid edge case
            },
            children: [],
          },
        ],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);
      const nodeAnchors = result.anchorPositions.filter(
        anchor => anchor.nodeId === 'node1'
      );

      // Should have center + valid + validEdge = 3 anchors (invalid ones skipped)
      expect(nodeAnchors).toHaveLength(3);
      expect(nodeAnchors.find(a => a.anchorId === 'valid')).toBeDefined();
      expect(nodeAnchors.find(a => a.anchorId === 'validEdge')).toBeDefined();
      expect(nodeAnchors.find(a => a.anchorId === 'tooHigh')).toBeUndefined();
      expect(nodeAnchors.find(a => a.anchorId === 'tooLow')).toBeUndefined();
    });

    test('handles empty document', () => {
      const document: Document = {
        nodes: [],
        arrows: [],
      };

      const result = layoutEngine.calculateLayout(document);

      expect(result.anchorPositions).toEqual([]);
    });
  });

  describe('utility functions', () => {
    test('findAnchorPosition finds correct anchor', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test',
            anchors: { custom: [0.25, 0.75] },
            children: [],
          },
        ],
        arrows: [],
      };

      const result = calculateLayout(document);

      // Find center anchor
      const centerAnchor = findAnchorPosition(result.anchorPositions, 'node1', 'center');
      expect(centerAnchor).toBeDefined();
      expect(centerAnchor?.anchorId).toBe('center');

      // Find custom anchor
      const customAnchor = findAnchorPosition(result.anchorPositions, 'node1', 'custom');
      expect(customAnchor).toBeDefined();
      expect(customAnchor?.anchorId).toBe('custom');

      // Default to center if anchor ID not specified
      const defaultAnchor = findAnchorPosition(result.anchorPositions, 'node1');
      expect(defaultAnchor).toBeDefined();
      expect(defaultAnchor?.anchorId).toBe('center');

      // Return null for non-existent anchor
      const nonExistent = findAnchorPosition(result.anchorPositions, 'node1', 'nonexistent');
      expect(nonExistent).toBeNull();

      // Return null for non-existent node
      const nonExistentNode = findAnchorPosition(result.anchorPositions, 'nonexistent', 'center');
      expect(nonExistentNode).toBeNull();
    });

    test('getNodeAnchors returns all anchors for a node', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Test',
            anchors: {
              top: [0.5, 0.0],
              bottom: [0.5, 1.0],
            },
            children: [],
          },
          {
            id: 'node2',
            label: 'Other',
            children: [],
          },
        ],
        arrows: [],
      };

      const result = calculateLayout(document);

      // Get all anchors for node1
      const node1Anchors = getNodeAnchors(result.anchorPositions, 'node1');
      expect(node1Anchors).toHaveLength(3); // center + top + bottom

      const anchorIds = node1Anchors.map(a => a.anchorId).sort();
      expect(anchorIds).toEqual(['bottom', 'center', 'top']);

      // Get all anchors for node2
      const node2Anchors = getNodeAnchors(result.anchorPositions, 'node2');
      expect(node2Anchors).toHaveLength(1); // just center

      // Non-existent node
      const nonExistentAnchors = getNodeAnchors(result.anchorPositions, 'nonexistent');
      expect(nonExistentAnchors).toEqual([]);
    });

    test('resolveNodePath returns path as-is', () => {
      // Current implementation just returns the path
      expect(resolveNodePath('node1')).toBe('node1');
      expect(resolveNodePath('parent.child')).toBe('parent.child');
    });
  });
});