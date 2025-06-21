/**
 * Validator for Arkitecture AST documents
 */

import { Document, ContainerNode, GroupNode, Arrow, ValidationError } from '../types';

export class Validator {
  private document: Document;
  private errors: ValidationError[];
  private nodeMap: Map<string, ContainerNode>;

  constructor(document: Document) {
    this.document = document;
    this.errors = [];
    this.nodeMap = new Map();
  }

  validate(): ValidationError[] {
    this.errors = [];
    this.nodeMap.clear();

    // Build node map for reference resolution
    this.buildNodeMap();

    // Run all validation checks
    this.validateNodeIdUniqueness();
    this.validateArrowReferences();
    this.validateAnchorReferences();
    this.validateConstraints();

    return this.errors;
  }

  private buildNodeMap(): void {
    // Build a flat map of all nodes with their full paths
    for (const node of this.document.nodes) {
      this.buildNodeMapRecursive(node, '');
    }
  }

  private buildNodeMapRecursive(node: ContainerNode | GroupNode, parentPath: string): void {
    if ('id' in node) {
      // Container node
      const fullPath = parentPath ? `${parentPath}.${node.id}` : node.id;
      this.nodeMap.set(fullPath, node);
      
      // Process children
      for (const child of node.children) {
        this.buildNodeMapRecursive(child, fullPath);
      }
    } else {
      // Group node - process children but don't add to map
      // Groups pass through their parent path directly - they don't create new path segments
      for (const child of node.children) {
        this.buildNodeMapRecursive(child, parentPath);
      }
    }
  }

  private validateNodeIdUniqueness(): void {
    // Validate ID uniqueness within parent scope
    this.validateNodeIdUniquenessRecursive(this.document.nodes, '');
  }

  private validateNodeIdUniquenessRecursive(
    nodes: (ContainerNode | GroupNode)[], 
    parentPath: string
  ): void {
    // Collect all IDs at this level (including those inside groups)
    const allIdsAtThisLevel = new Set<string>();
    
    // First pass: collect all IDs and check for duplicates
    this.collectIdsAtLevel(nodes, allIdsAtThisLevel, parentPath);
    
    // Second pass: recursively validate children
    for (const node of nodes) {
      if ('id' in node) {
        // Container node - validate children in their own scope
        const currentPath = parentPath ? `${parentPath}.${node.id}` : node.id;
        this.validateNodeIdUniquenessRecursive(node.children, currentPath);
      } else {
        // Group node - validate children recursively (they are already checked at this level)
        this.validateNodeIdUniquenessRecursive(node.children, parentPath);
      }
    }
  }
  
  private collectIdsAtLevel(
    nodes: (ContainerNode | GroupNode)[],
    seenIds: Set<string>,
    parentPath: string
  ): void {
    for (const node of nodes) {
      if ('id' in node) {
        // Container node
        if (seenIds.has(node.id)) {
          this.addError(
            'reference',
            `Duplicate node ID '${node.id}' within ${parentPath || 'root'} scope`,
            1,
            1
          );
        }
        seenIds.add(node.id);
      } else {
        // Group node - collect IDs from children at this same level
        this.collectIdsAtLevel(node.children, seenIds, parentPath);
      }
    }
  }

  private validateArrowReferences(): void {
    for (const arrow of this.document.arrows) {
      this.validateSingleArrow(arrow);
    }
  }

  private validateSingleArrow(arrow: Arrow): void {
    // Validate source node reference
    const sourceNodePath = this.extractNodePath(arrow.source);
    if (!this.nodeMap.has(sourceNodePath)) {
      this.addError(
        'reference',
        `Arrow source node '${sourceNodePath}' does not exist`,
        1,
        1
      );
    }

    // Validate target node reference
    const targetNodePath = this.extractNodePath(arrow.target);
    if (!this.nodeMap.has(targetNodePath)) {
      this.addError(
        'reference',
        `Arrow target node '${targetNodePath}' does not exist`,
        1,
        1
      );
    }
  }

  private validateAnchorReferences(): void {
    for (const arrow of this.document.arrows) {
      // Check source anchor if specified
      if (arrow.source.includes('#')) {
        const { nodePath, anchorId } = this.parseNodePathWithAnchor(arrow.source);
        const node = this.nodeMap.get(nodePath);
        if (node && !this.hasAnchor(node, anchorId)) {
          this.addError(
            'reference',
            `Arrow source anchor '${anchorId}' does not exist on node '${nodePath}'`,
            1,
            1
          );
        }
      }

      // Check target anchor if specified
      if (arrow.target.includes('#')) {
        const { nodePath, anchorId } = this.parseNodePathWithAnchor(arrow.target);
        const node = this.nodeMap.get(nodePath);
        if (node && !this.hasAnchor(node, anchorId)) {
          this.addError(
            'reference',
            `Arrow target anchor '${anchorId}' does not exist on node '${nodePath}'`,
            1,
            1
          );
        }
      }
    }
  }

  private validateConstraints(): void {
    // Validate all nodes recursively
    for (const node of this.document.nodes) {
      this.validateNodeConstraints(node);
    }
  }

  private validateNodeConstraints(node: ContainerNode | GroupNode): void {
    if ('id' in node) {
      // Container node - validate size and anchor constraints
      if (node.size !== undefined) {
        if (node.size < 0.0 || node.size > 1.0) {
          this.addError(
            'constraint',
            `Node '${node.id}' size ${node.size} is out of range, expected 0.0-1.0`,
            1,
            1
          );
        }
      }

      if (node.anchors) {
        for (const [anchorId, coordinates] of Object.entries(node.anchors)) {
          const [x, y] = coordinates;
          
          if (x < 0.0 || x > 1.0) {
            this.addError(
              'constraint',
              `Node '${node.id}' anchor '${anchorId}' X coordinate ${x} is out of range, expected 0.0-1.0`,
              1,
              1
            );
          }
          
          if (y < 0.0 || y > 1.0) {
            this.addError(
              'constraint',
              `Node '${node.id}' anchor '${anchorId}' Y coordinate ${y} is out of range, expected 0.0-1.0`,
              1,
              1
            );
          }
        }
      }
    }

    // Recursively validate children
    for (const child of node.children) {
      this.validateNodeConstraints(child);
    }
  }

  private extractNodePath(nodePathWithAnchor: string): string {
    const hashIndex = nodePathWithAnchor.indexOf('#');
    return hashIndex >= 0 ? nodePathWithAnchor.substring(0, hashIndex) : nodePathWithAnchor;
  }

  private parseNodePathWithAnchor(nodePathWithAnchor: string): { nodePath: string; anchorId: string } {
    const hashIndex = nodePathWithAnchor.indexOf('#');
    if (hashIndex >= 0) {
      return {
        nodePath: nodePathWithAnchor.substring(0, hashIndex),
        anchorId: nodePathWithAnchor.substring(hashIndex + 1)
      };
    }
    return {
      nodePath: nodePathWithAnchor,
      anchorId: ''
    };
  }

  private hasAnchor(node: ContainerNode, anchorId: string): boolean {
    // All nodes have an implicit 'center' anchor
    if (anchorId === 'center') {
      return true;
    }
    
    // Check explicit anchors
    return node.anchors ? anchorId in node.anchors : false;
  }

  private addError(
    type: 'syntax' | 'reference' | 'constraint', 
    message: string, 
    line: number, 
    column: number
  ): void {
    this.errors.push({
      type,
      message,
      line,
      column,
    });
  }

  // Helper method to resolve node path (for future use)
  resolveNodePath(path: string): ContainerNode | null {
    return this.nodeMap.get(path) || null;
  }
}