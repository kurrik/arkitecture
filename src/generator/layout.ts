/**
 * Layout algorithm for calculating node dimensions and positions
 */

import { Document, ContainerNode, GroupNode } from '../types';
import { TextMeasurement, defaultTextMeasurement } from './text-measurement';

export interface NodeDimensions {
  width: number;
  height: number;
  x: number;
  y: number;
}

export interface AnchorPosition {
  x: number;
  y: number;
  nodeId: string;
  anchorId: string;
}

export interface LayoutResult {
  nodeDimensions: Record<string, NodeDimensions>;
  anchorPositions: AnchorPosition[];
  canvasWidth: number;
  canvasHeight: number;
}

type LayoutNode = ContainerNode | GroupNode;

interface NodeLayout {
  node: LayoutNode;
  id: string; // For groups, this will be generated
  dimensions: NodeDimensions;
  children: NodeLayout[];
}

export class LayoutEngine {
  private textMeasurement: TextMeasurement;
  private borderWidth: number = 1; // 1px border for all nodes

  constructor(textMeasurement?: TextMeasurement) {
    this.textMeasurement = textMeasurement || defaultTextMeasurement;
  }

  /**
   * Calculate layout for the entire document
   */
  calculateLayout(document: Document): LayoutResult {
    const nodeDimensions: Record<string, NodeDimensions> = {};
    const rootLayouts: NodeLayout[] = [];

    // Phase 1: Build layout tree and calculate dimensions bottom-up
    for (const node of document.nodes) {
      const layout = this.buildLayoutTree(node);
      this.calculateDimensions(layout);
      rootLayouts.push(layout);
    }

    // Phase 2: Position nodes top-down
    let currentX = 0;
    for (const layout of rootLayouts) {
      this.positionNodes(layout, currentX, 0);
      this.collectNodeDimensions(layout, nodeDimensions);
      currentX += layout.dimensions.width;
    }

    // Phase 3: Calculate canvas size
    const { canvasWidth, canvasHeight } = this.calculateCanvasSize(rootLayouts);

    // Phase 4: Calculate anchor positions
    const anchorPositions = this.calculateAnchorPositions(document, nodeDimensions);

    return {
      nodeDimensions,
      anchorPositions,
      canvasWidth,
      canvasHeight,
    };
  }

  /**
   * Build the layout tree from the AST
   */
  private buildLayoutTree(node: LayoutNode, parentId?: string): NodeLayout {
    const isContainer = 'id' in node;
    const id = isContainer ? node.id : `${parentId || 'root'}_group_${Math.random().toString(36).substr(2, 9)}`;

    const children: NodeLayout[] = [];
    for (const child of node.children) {
      children.push(this.buildLayoutTree(child, id));
    }

    return {
      node,
      id,
      dimensions: { width: 0, height: 0, x: 0, y: 0 },
      children,
    };
  }

  /**
   * Calculate dimensions bottom-up
   */
  private calculateDimensions(layout: NodeLayout): void {
    // First, calculate dimensions for all children
    for (const child of layout.children) {
      this.calculateDimensions(child);
    }

    const node = layout.node;
    const isContainer = 'id' in node;
    const direction = node.direction || 'vertical';

    if (layout.children.length === 0) {
      // Leaf node - calculate based on text dimensions
      if (isContainer) {
        const textDims = this.textMeasurement.getTextDimensions(node.label);
        layout.dimensions.width = Math.max(textDims.width + 2 * this.borderWidth, this.textMeasurement.getMinimumDimensions().width);
        layout.dimensions.height = Math.max(textDims.height + 2 * this.borderWidth, this.textMeasurement.getMinimumDimensions().height);
      } else {
        // Groups with no children have no dimensions
        layout.dimensions.width = 0;
        layout.dimensions.height = 0;
      }
    } else {
      // Parent node - calculate based on children and layout direction
      if (direction === 'horizontal') {
        // Horizontal layout: width = sum of child widths, height = max child height
        layout.dimensions.width = layout.children.reduce((sum, child) => sum + child.dimensions.width, 0);
        layout.dimensions.height = Math.max(...layout.children.map(child => child.dimensions.height));

        // For container nodes, ensure children fit within the parent height
        if (isContainer) {
          for (const child of layout.children) {
            child.dimensions.height = layout.dimensions.height;
          }
        }
      } else {
        // Vertical layout: height = sum of child heights, width = max child width
        layout.dimensions.height = layout.children.reduce((sum, child) => sum + child.dimensions.height, 0);
        layout.dimensions.width = Math.max(...layout.children.map(child => child.dimensions.width));

        // For container nodes, ensure children fit within the parent width
        if (isContainer) {
          for (const child of layout.children) {
            child.dimensions.width = layout.dimensions.width;
          }
        }
      }

      // Apply size override if specified (only for container nodes)
      if (isContainer && node.size !== undefined) {
        if (direction === 'horizontal') {
          // Size affects height (orthogonal dimension)
          const originalHeight = layout.dimensions.height;
          layout.dimensions.height = originalHeight * node.size;
        } else {
          // Size affects width (orthogonal dimension)
          const originalWidth = layout.dimensions.width;
          layout.dimensions.width = originalWidth * node.size;
        }
      }
    }
  }

  /**
   * Position nodes top-down
   */
  private positionNodes(layout: NodeLayout, x: number, y: number): void {
    layout.dimensions.x = x;
    layout.dimensions.y = y;

    const node = layout.node;
    const direction = node.direction || 'vertical';

    // Position children
    let currentX = x;
    let currentY = y;

    for (const child of layout.children) {
      this.positionNodes(child, currentX, currentY);

      if (direction === 'horizontal') {
        currentX += child.dimensions.width;
      } else {
        currentY += child.dimensions.height;
      }
    }
  }

  /**
   * Collect node dimensions into the result map (only for container nodes)
   */
  private collectNodeDimensions(layout: NodeLayout, dimensions: Record<string, NodeDimensions>): void {
    const isContainer = 'id' in layout.node;
    
    if (isContainer) {
      const containerNode = layout.node as ContainerNode;
      dimensions[containerNode.id] = { ...layout.dimensions };
    }

    // Recursively collect from children
    for (const child of layout.children) {
      this.collectNodeDimensions(child, dimensions);
    }
  }

  /**
   * Calculate the overall canvas size
   */
  private calculateCanvasSize(rootLayouts: NodeLayout[]): { canvasWidth: number; canvasHeight: number } {
    if (rootLayouts.length === 0) {
      return { canvasWidth: 0, canvasHeight: 0 };
    }

    let canvasWidth = 0;
    let canvasHeight = 0;

    for (const layout of rootLayouts) {
      const rightEdge = layout.dimensions.x + layout.dimensions.width;
      const bottomEdge = layout.dimensions.y + layout.dimensions.height;
      
      canvasWidth = Math.max(canvasWidth, rightEdge);
      canvasHeight = Math.max(canvasHeight, bottomEdge);
    }

    return { canvasWidth, canvasHeight };
  }

  /**
   * Calculate absolute anchor positions for all nodes
   */
  private calculateAnchorPositions(document: Document, nodeDimensions: Record<string, NodeDimensions>): AnchorPosition[] {
    const anchorPositions: AnchorPosition[] = [];

    // Calculate anchors for all container nodes
    this.collectNodeAnchors(document.nodes, nodeDimensions, anchorPositions);

    return anchorPositions;
  }

  /**
   * Recursively collect anchor positions from nodes
   */
  private collectNodeAnchors(
    nodes: (ContainerNode | GroupNode)[],
    nodeDimensions: Record<string, NodeDimensions>,
    anchorPositions: AnchorPosition[]
  ): void {
    for (const node of nodes) {
      if ('id' in node) {
        // Container node - calculate its anchors
        const dimensions = nodeDimensions[node.id];
        if (dimensions) {
          const nodeAnchors = this.resolveNodeAnchors(node, dimensions);
          anchorPositions.push(...nodeAnchors);
        }
      }

      // Recursively process children
      this.collectNodeAnchors(node.children, nodeDimensions, anchorPositions);
    }
  }

  /**
   * Resolve anchor positions for a specific node
   */
  private resolveNodeAnchors(node: ContainerNode, dimensions: NodeDimensions): AnchorPosition[] {
    const anchors: AnchorPosition[] = [];

    // Always add implicit center anchor
    anchors.push({
      x: dimensions.x + dimensions.width * 0.5,
      y: dimensions.y + dimensions.height * 0.5,
      nodeId: node.id,
      anchorId: 'center',
    });

    // Add custom anchors if defined
    if (node.anchors) {
      for (const [anchorId, [relativeX, relativeY]] of Object.entries(node.anchors)) {
        // Validate anchor coordinates are in range [0.0, 1.0]
        if (relativeX < 0 || relativeX > 1 || relativeY < 0 || relativeY > 1) {
          continue; // Skip invalid anchors (validation should have caught this)
        }

        anchors.push({
          x: dimensions.x + dimensions.width * relativeX,
          y: dimensions.y + dimensions.height * relativeY,
          nodeId: node.id,
          anchorId,
        });
      }
    }

    return anchors;
  }
}

// Default instance for convenience
export const defaultLayoutEngine = new LayoutEngine();

// Utility functions
export function calculateLayout(document: Document, textMeasurement?: TextMeasurement): LayoutResult {
  const engine = textMeasurement ? new LayoutEngine(textMeasurement) : defaultLayoutEngine;
  return engine.calculateLayout(document);
}

/**
 * Find an anchor position by node path and anchor ID
 */
export function findAnchorPosition(
  anchorPositions: AnchorPosition[],
  nodePath: string,
  anchorId: string = 'center'
): AnchorPosition | null {
  return anchorPositions.find(
    anchor => anchor.nodeId === nodePath && anchor.anchorId === anchorId
  ) || null;
}

/**
 * Get all anchor positions for a specific node
 */
export function getNodeAnchors(
  anchorPositions: AnchorPosition[],
  nodeId: string
): AnchorPosition[] {
  return anchorPositions.filter(anchor => anchor.nodeId === nodeId);
}

/**
 * Resolve a node path to get the final node ID (handles nested paths like "parent.child")
 */
export function resolveNodePath(path: string): string {
  // For now, just return the path as-is
  // In the future, this could handle nested paths like "parent.child.grandchild"
  return path;
}