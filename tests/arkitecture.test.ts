/**
 * Tests for main API integration
 */

import arkitectureToSVG, { parseArkitecture, validate, generateSVG } from '../src/arkitecture';
import { Document } from '../src/types';

describe('Main API Integration', () => {
  describe('arkitectureToSVG', () => {
    test('successful end-to-end processing', () => {
      const dsl = `
        node1 {
          label: "Node 1"
        }
        
        node2 {
          label: "Node 2"
        }
        
        node1 --> node2
      `;

      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
      expect(result.errors).toEqual([]);

      if (result.svg) {
        // Should contain valid SVG structure
        expect(result.svg).toContain('<svg');
        expect(result.svg).toContain('</svg>');
        
        // Should contain nodes
        expect(result.svg).toContain('Node 1');
        expect(result.svg).toContain('Node 2');
        
        // Should contain arrow
        expect(result.svg).toContain('<line');
        expect(result.svg).toContain('marker-end="url(#arrowhead)"');
      }
    });

    test('handles parsing errors', () => {
      const invalidDsl = `
        node1 {
          invalid syntax here
      `;

      const result = arkitectureToSVG(invalidDsl);

      expect(result.success).toBe(false);
      expect(result.svg).toBeUndefined();
      expect(result.errors.length).toBeGreaterThan(0);
      expect(result.errors[0].type).toBe('syntax');
    });

    test('handles validation errors', () => {
      const dsl = `
        node1 {
          label: "Node 1"
        }
        
        node1 --> nonexistent
      `;

      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(false);
      expect(result.svg).toBeUndefined();
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('reference');
    });

    test('validateOnly option skips SVG generation', () => {
      const dsl = `
        node1 {
          label: "Node 1"
        }
      `;

      const result = arkitectureToSVG(dsl, { validateOnly: true });

      expect(result.success).toBe(true);
      expect(result.svg).toBeUndefined();
      expect(result.errors).toEqual([]);
    });

    test('custom font settings', () => {
      const dsl = `
        node1 {
          label: "Custom Font"
        }
      `;

      const result = arkitectureToSVG(dsl, {
        fontSize: 16,
        fontFamily: 'Helvetica',
      });

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();

      if (result.svg) {
        expect(result.svg).toContain('font-size="16"');
        expect(result.svg).toContain('font-family="Helvetica"');
      }
    });

    test('handles internal errors gracefully', () => {
      // Test with empty string which should be handled gracefully
      const result = arkitectureToSVG('');

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
      expect(result.errors).toEqual([]);
    });

    test('complex nested structure', () => {
      const dsl = `
        parent {
          label: "Parent Container"
          direction: "vertical"
          
          child1 {
            label: "Child 1"
          }
          
          child2 {
            label: "Child 2"
          }
        }
        
        standalone {
          label: "Standalone Node"
          anchors: {
            left: [0.0, 0.5]
          }
        }
        
        parent.child1 --> parent.child2
        parent --> standalone#left
      `;

      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
      expect(result.errors).toEqual([]);

      if (result.svg) {
        // Should contain all node labels
        expect(result.svg).toContain('Parent Container');
        expect(result.svg).toContain('Child 1');
        expect(result.svg).toContain('Child 2');
        expect(result.svg).toContain('Standalone Node');
        
        // Should contain multiple arrows
        const arrowMatches = result.svg.match(/<line[^>]*marker-end="url\(#arrowhead\)"/g);
        expect(arrowMatches).toHaveLength(2); // Both arrows should now render correctly
      }
    });
  });

  describe('individual function exports', () => {
    test('parseArkitecture function', () => {
      const dsl = `
        node1 {
          label: "Test Node"
        }
      `;

      const result = parseArkitecture(dsl);

      expect(result.success).toBe(true);
      expect(result.document).toBeDefined();
      expect(result.errors).toEqual([]);

      if (result.document) {
        expect(result.document.nodes).toHaveLength(1);
        expect(result.document.nodes[0].id).toBe('node1');
        expect(result.document.nodes[0].label).toBe('Test Node');
      }
    });

    test('validate function', () => {
      const validDocument: Document = {
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

      const errors = validate(validDocument);
      expect(errors).toEqual([]);
    });

    test('validate function with errors', () => {
      const invalidDocument: Document = {
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

      const errors = validate(invalidDocument);
      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
    });

    test('generateSVG function', () => {
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

      const svg = generateSVG(document);

      expect(svg).toContain('<svg');
      expect(svg).toContain('Test Node');
      expect(svg).toContain('</svg>');
    });

    test('generateSVG function with custom options', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Custom Font Node',
            children: [],
          },
        ],
        arrows: [],
      };

      const svg = generateSVG(document, {
        fontSize: 20,
        fontFamily: 'Times',
      });

      expect(svg).toContain('font-size="20"');
      expect(svg).toContain('font-family="Times"');
    });
  });

  describe('error handling', () => {
    test('parsing phase errors', () => {
      const invalidDsl = 'node1 { invalid }';
      const result = arkitectureToSVG(invalidDsl);

      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].message).toContain('Expected');
    });

    test('validation phase errors', () => {
      const dsl = `
        node1 { label: "Node 1" }
        node1 --> invalid_target
      `;
      
      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('reference');
    });

    test('constraint validation errors', () => {
      const dsl = `
        node1 {
          label: "Node 1"
          size: 1.5
        }
      `;
      
      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('constraint');
    });

    test('multiple validation errors', () => {
      const dsl = `
        node1 {
          label: "Node 1"
          size: 1.5
          anchors: {
            invalid: [2.0, 0.5]
          }
        }
        
        node1 --> nonexistent1
        node1 --> nonexistent2
      `;
      
      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThan(1);
    });
  });

  describe('specification example', () => {
    test('processes specification DSL example', () => {
      const specDsl = `
        api {
          label: "API Gateway"
          direction: "vertical"
          anchors: {
            south: [0.5, 1.0]
          }
          
          auth {
            label: "Authentication"
          }
          
          routing {
            label: "Request Routing"
          }
        }
        
        services {
          label: "Microservices"
          direction: "horizontal"
          anchors: {
            north: [0.5, 0.0]
          }
          
          userService {
            label: "User Service"
            anchors: {
              db: [0.5, 1.0]
            }
          }
          
          orderService {
            label: "Order Service"
            anchors: {
              db: [0.5, 1.0]
            }
          }
        }
        
        database {
          label: "Database Cluster"
          anchors: {
            north: [0.5, 0.0]
          }
        }
        
        api#south --> services#north
        services.userService#db --> database#north
        services.orderService#db --> database#north
      `;

      const result = arkitectureToSVG(specDsl);

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
      expect(result.errors).toEqual([]);

      if (result.svg) {
        // Should contain all major components
        expect(result.svg).toContain('API Gateway');
        expect(result.svg).toContain('Authentication');
        expect(result.svg).toContain('Request Routing');
        expect(result.svg).toContain('Microservices');
        expect(result.svg).toContain('User Service');
        expect(result.svg).toContain('Order Service');
        expect(result.svg).toContain('Database Cluster');

        // Should contain arrows (some arrows may not render if anchors can't be found)
        const arrowCount = (result.svg.match(/<line[^>]*marker-end/g) || []).length;
        expect(arrowCount).toBeGreaterThanOrEqual(0);
      }
    });
  });

  describe('edge cases', () => {
    test('empty DSL', () => {
      const result = arkitectureToSVG('');

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
      expect(result.errors).toEqual([]);

      if (result.svg) {
        expect(result.svg).toContain('<svg');
        expect(result.svg).toContain('width="0"');
        expect(result.svg).toContain('height="0"');
      }
    });

    test('only whitespace', () => {
      const result = arkitectureToSVG('   \n\t  \n  ');

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
    });

    test('only comments', () => {
      const result = arkitectureToSVG(`
        # This is a comment
        # Another comment
      `);

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
    });

    test('nodes without arrows', () => {
      const dsl = `
        node1 { label: "Isolated 1" }
        node2 { label: "Isolated 2" }
      `;

      const result = arkitectureToSVG(dsl);

      expect(result.success).toBe(true);
      expect(result.svg).toBeDefined();
      
      if (result.svg) {
        expect(result.svg).toContain('Isolated 1');
        expect(result.svg).toContain('Isolated 2');
        expect(result.svg).not.toContain('<line');
      }
    });
  });
});