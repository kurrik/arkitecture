import {
  Document,
  ContainerNode,
  GroupNode,
  Arrow,
  ParseResult,
  ValidationError,
  Result,
  Options,
} from '../src/types';

describe('Type Definitions', () => {
  describe('Document', () => {
    it('should create a valid document with nodes and arrows', () => {
      const document: Document = {
        nodes: [],
        arrows: [],
      };
      expect(document).toBeDefined();
      expect(Array.isArray(document.nodes)).toBe(true);
      expect(Array.isArray(document.arrows)).toBe(true);
    });
  });

  describe('ContainerNode', () => {
    it('should create a minimal container node with only required fields', () => {
      const node: ContainerNode = {
        id: 'test-node',
        children: [],
      };
      expect(node.id).toBe('test-node');
      expect(Array.isArray(node.children)).toBe(true);
    });

    it('should create a full container node with all optional fields', () => {
      const node: ContainerNode = {
        id: 'full-node',
        label: 'Test Node',
        direction: 'vertical',
        size: 0.5,
        anchors: {
          top: [0.5, 0.0],
          center: [0.5, 0.5],
        },
        children: [],
      };
      expect(node.id).toBe('full-node');
      expect(node.label).toBe('Test Node');
      expect(node.direction).toBe('vertical');
      expect(node.size).toBe(0.5);
      expect(node.anchors?.top).toEqual([0.5, 0.0]);
    });

    it('should support horizontal direction', () => {
      const node: ContainerNode = {
        id: 'horizontal-node',
        direction: 'horizontal',
        children: [],
      };
      expect(node.direction).toBe('horizontal');
    });
  });

  describe('GroupNode', () => {
    it('should create a minimal group node', () => {
      const group: GroupNode = {
        children: [],
      };
      expect(Array.isArray(group.children)).toBe(true);
    });

    it('should create a group node with direction', () => {
      const group: GroupNode = {
        direction: 'horizontal',
        children: [],
      };
      expect(group.direction).toBe('horizontal');
    });
  });

  describe('Arrow', () => {
    it('should create an arrow with source and target', () => {
      const arrow: Arrow = {
        source: 'node1',
        target: 'node2',
      };
      expect(arrow.source).toBe('node1');
      expect(arrow.target).toBe('node2');
    });

    it('should create an arrow with complex paths and anchors', () => {
      const arrow: Arrow = {
        source: 'parent.child1',
        target: 'parent.child2#anchor1',
      };
      expect(arrow.source).toBe('parent.child1');
      expect(arrow.target).toBe('parent.child2#anchor1');
    });
  });

  describe('ParseResult', () => {
    it('should create a successful parse result', () => {
      const result: ParseResult = {
        success: true,
        document: {
          nodes: [],
          arrows: [],
        },
        errors: [],
      };
      expect(result.success).toBe(true);
      expect(result.document).toBeDefined();
      expect(Array.isArray(result.errors)).toBe(true);
    });

    it('should create a failed parse result', () => {
      const result: ParseResult = {
        success: false,
        errors: [
          {
            line: 1,
            column: 5,
            message: 'Unexpected token',
            type: 'syntax',
          },
        ],
      };
      expect(result.success).toBe(false);
      expect(result.document).toBeUndefined();
      expect(result.errors).toHaveLength(1);
    });
  });

  describe('ValidationError', () => {
    it('should create validation errors with all required fields', () => {
      const error: ValidationError = {
        line: 10,
        column: 15,
        message: 'Invalid reference',
        type: 'reference',
      };
      expect(error.line).toBe(10);
      expect(error.column).toBe(15);
      expect(error.message).toBe('Invalid reference');
      expect(error.type).toBe('reference');
    });

    it('should support all error types', () => {
      const syntaxError: ValidationError = {
        line: 1, column: 1, message: 'Syntax error', type: 'syntax'
      };
      const referenceError: ValidationError = {
        line: 2, column: 2, message: 'Reference error', type: 'reference'
      };
      const constraintError: ValidationError = {
        line: 3, column: 3, message: 'Constraint error', type: 'constraint'
      };
      
      expect(syntaxError.type).toBe('syntax');
      expect(referenceError.type).toBe('reference');
      expect(constraintError.type).toBe('constraint');
    });
  });

  describe('Result', () => {
    it('should create a successful result with SVG', () => {
      const result: Result = {
        success: true,
        svg: '<svg>test</svg>',
        errors: [],
      };
      expect(result.success).toBe(true);
      expect(result.svg).toBe('<svg>test</svg>');
      expect(result.errors).toHaveLength(0);
    });

    it('should create a failed result with errors', () => {
      const result: Result = {
        success: false,
        errors: [
          {
            line: 1,
            column: 1,
            message: 'Parse error',
            type: 'syntax',
          },
        ],
      };
      expect(result.success).toBe(false);
      expect(result.svg).toBeUndefined();
      expect(result.errors).toHaveLength(1);
    });
  });

  describe('Options', () => {
    it('should create empty options', () => {
      const options: Options = {};
      expect(options).toBeDefined();
    });

    it('should create options with all fields', () => {
      const options: Options = {
        validateOnly: true,
        fontSize: 14,
        fontFamily: 'Helvetica',
      };
      expect(options.validateOnly).toBe(true);
      expect(options.fontSize).toBe(14);
      expect(options.fontFamily).toBe('Helvetica');
    });
  });

  describe('Constraint validation', () => {
    it('should allow valid size values', () => {
      const node: ContainerNode = {
        id: 'test',
        size: 0.0,
        children: [],
      };
      expect(node.size).toBe(0.0);

      const node2: ContainerNode = {
        id: 'test2',
        size: 1.0,
        children: [],
      };
      expect(node2.size).toBe(1.0);

      const node3: ContainerNode = {
        id: 'test3',
        size: 0.5,
        children: [],
      };
      expect(node3.size).toBe(0.5);
    });

    it('should allow valid anchor coordinates', () => {
      const node: ContainerNode = {
        id: 'test',
        anchors: {
          topLeft: [0.0, 0.0],
          center: [0.5, 0.5],
          bottomRight: [1.0, 1.0],
        },
        children: [],
      };
      expect(node.anchors?.topLeft).toEqual([0.0, 0.0]);
      expect(node.anchors?.center).toEqual([0.5, 0.5]);
      expect(node.anchors?.bottomRight).toEqual([1.0, 1.0]);
    });
  });
});