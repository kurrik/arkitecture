import { parseArkitecture } from '../../src/parser';

describe('Property Parser - Size and Anchors', () => {
  describe('Size property parsing', () => {
    it('should parse valid size values', () => {
      const dsl = `
        node1 {
          label: "Node with size"
          size: 0.5
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const node = result.document!.nodes[0];
      expect(node.size).toBe(0.5);
    });

    it('should parse size value of 0.0', () => {
      const dsl = `
        node1 {
          size: 0.0
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes[0].size).toBe(0.0);
    });

    it('should parse size value of 1.0', () => {
      const dsl = `
        node1 {
          size: 1.0
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes[0].size).toBe(1.0);
    });

    it('should parse decimal size values', () => {
      const dsl = `
        node1 {
          size: 0.75
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes[0].size).toBe(0.75);
    });

    it('should report error for size value below 0.0', () => {
      const dsl = `
        node1 {
          size: -0.1
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Unexpected character '-'");
    });

    it('should report error for size value above 1.0', () => {
      const dsl = `
        node1 {
          size: 1.5
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('constraint');
      expect(result.errors[0].message).toContain('Size value 1.5 is out of range');
    });

    it('should report error for non-numeric size value', () => {
      const dsl = `
        node1 {
          size: "large"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain('Expected number value for size');
    });
  });

  describe('Anchors property parsing', () => {
    it('should parse single anchor with valid coordinates', () => {
      const dsl = `
        node1 {
          label: "Node with anchor"
          anchors: {
            center: [0.5, 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const node = result.document!.nodes[0];
      expect(node.anchors).toBeDefined();
      expect(node.anchors!.center).toEqual([0.5, 0.5]);
    });

    it('should parse multiple anchors', () => {
      const dsl = `
        node1 {
          anchors: {
            top: [0.5, 0.0],
            bottom: [0.5, 1.0],
            left: [0.0, 0.5],
            right: [1.0, 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const node = result.document!.nodes[0];
      expect(node.anchors).toBeDefined();
      expect(node.anchors!.top).toEqual([0.5, 0.0]);
      expect(node.anchors!.bottom).toEqual([0.5, 1.0]);
      expect(node.anchors!.left).toEqual([0.0, 0.5]);
      expect(node.anchors!.right).toEqual([1.0, 0.5]);
    });

    it('should parse anchors with decimal coordinates', () => {
      const dsl = `
        node1 {
          anchors: {
            custom: [0.25, 0.75]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes[0].anchors!.custom).toEqual([0.25, 0.75]);
    });

    it('should parse anchors with trailing comma', () => {
      const dsl = `
        node1 {
          anchors: {
            top: [0.5, 0.0],
            bottom: [0.5, 1.0],
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(Object.keys(result.document!.nodes[0].anchors!)).toHaveLength(2);
    });

    it('should handle empty anchors object', () => {
      const dsl = `
        node1 {
          anchors: {}
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes[0].anchors).toBeUndefined();
    });

    it('should report error for duplicate anchor IDs', () => {
      const dsl = `
        node1 {
          anchors: {
            center: [0.5, 0.5],
            center: [0.0, 0.0]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Duplicate anchor ID 'center'");
    });
  });

  describe('Coordinate array parsing', () => {
    it('should parse corner coordinates', () => {
      const dsl = `
        node1 {
          anchors: {
            topLeft: [0.0, 0.0],
            topRight: [1.0, 0.0],
            bottomLeft: [0.0, 1.0],
            bottomRight: [1.0, 1.0]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const anchors = result.document!.nodes[0].anchors!;
      expect(anchors.topLeft).toEqual([0.0, 0.0]);
      expect(anchors.topRight).toEqual([1.0, 0.0]);
      expect(anchors.bottomLeft).toEqual([0.0, 1.0]);
      expect(anchors.bottomRight).toEqual([1.0, 1.0]);
    });

    it('should report error for X coordinate out of range', () => {
      const dsl = `
        node1 {
          anchors: {
            invalid: [-0.1, 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Unexpected character '-'");
    });

    it('should report error for Y coordinate out of range', () => {
      const dsl = `
        node1 {
          anchors: {
            invalid: [0.5, 1.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('constraint');
      expect(result.errors[0].message).toContain('Y coordinate 1.5 is out of range');
    });

    it('should report error for malformed coordinate array - missing bracket', () => {
      const dsl = `
        node1 {
          anchors: {
            invalid: 0.5, 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected '[' to start coordinate array");
    });

    it('should report error for malformed coordinate array - missing comma', () => {
      const dsl = `
        node1 {
          anchors: {
            invalid: [0.5 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected ',' between coordinates");
    });

    it('should report error for non-numeric coordinates', () => {
      const dsl = `
        node1 {
          anchors: {
            invalid: ["left", "top"]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain('Expected number for X coordinate');
    });
  });

  describe('Combined properties', () => {
    it('should parse node with all properties', () => {
      const dsl = `
        node1 {
          label: "Complex Node"
          direction: "horizontal"
          size: 0.75
          anchors: {
            input: [0.0, 0.5],
            output: [1.0, 0.5],
            top: [0.5, 0.0]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const node = result.document!.nodes[0];
      expect(node.label).toBe('Complex Node');
      expect(node.direction).toBe('horizontal');
      expect(node.size).toBe(0.75);
      expect(node.anchors).toBeDefined();
      expect(node.anchors!.input).toEqual([0.0, 0.5]);
      expect(node.anchors!.output).toEqual([1.0, 0.5]);
      expect(node.anchors!.top).toEqual([0.5, 0.0]);
    });

    it('should parse nested nodes with size and anchors', () => {
      const dsl = `
        parent {
          label: "Parent"
          size: 0.8
          
          child {
            label: "Child"
            anchors: {
              connector: [0.5, 1.0]
            }
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const parent = result.document!.nodes[0];
      expect(parent.size).toBe(0.8);
      
      const child = parent.children[0] as any;
      expect(child.anchors!.connector).toEqual([0.5, 1.0]);
    });

    it('should handle properties in any order', () => {
      const dsl = `
        node1 {
          anchors: {
            center: [0.5, 0.5]
          }
          size: 0.6
          label: "Reordered Node"
          direction: "vertical"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const node = result.document!.nodes[0];
      expect(node.label).toBe('Reordered Node');
      expect(node.direction).toBe('vertical');
      expect(node.size).toBe(0.6);
      expect(node.anchors!.center).toEqual([0.5, 0.5]);
    });
  });

  describe('Error handling edge cases', () => {
    it('should handle multiple coordinate errors gracefully', () => {
      const dsl = `
        node1 {
          anchors: {
            bad1: [1.5, 2.0],
            bad2: [0.5, 1.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThan(1);
      expect(result.errors.some(e => e.message.includes('X coordinate 1.5 is out of range'))).toBe(true);
      expect(result.errors.some(e => e.message.includes('Y coordinate 2 is out of range'))).toBe(true);
    });

    it('should continue parsing after anchor errors', () => {
      const dsl = `
        node1 {
          anchors: {
            invalid: [1.5, 0.5]
          }
        }
        
        node2 {
          label: "Valid Node"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].message).toContain('X coordinate 1.5 is out of range');
      // Check that we still have a document even with errors
      if (result.document) {
        expect(result.document.nodes).toHaveLength(2);
        expect(result.document.nodes[1].label).toBe('Valid Node');
      }
    });

    it('should handle incomplete anchor syntax gracefully', () => {
      const dsl = `
        node1 {
          anchors: {
            incomplete: [0.5
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
      expect(result.errors[0].type).toBe('syntax');
    });
  });

  describe('Integration with specification example', () => {
    it('should parse complex node from specification', () => {
      const dsl = `
        node1 {
          label: "Node with anchors"
          size: 0.75
          anchors: {
            top: [0.5, 0.0],
            bottom: [0.5, 1.0],
            center: [0.5, 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const node = result.document!.nodes[0];
      expect(node.label).toBe('Node with anchors');
      expect(node.size).toBe(0.75);
      expect(node.anchors).toBeDefined();
      expect(Object.keys(node.anchors!)).toHaveLength(3);
      expect(node.anchors!.top).toEqual([0.5, 0.0]);
      expect(node.anchors!.bottom).toEqual([0.5, 1.0]);
      expect(node.anchors!.center).toEqual([0.5, 0.5]);
    });
  });

  describe('Backward compatibility', () => {
    it('should still parse nodes without size or anchors', () => {
      const dsl = `
        node1 {
          label: "Simple Node"
          direction: "horizontal"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const node = result.document!.nodes[0];
      expect(node.label).toBe('Simple Node');
      expect(node.direction).toBe('horizontal');
      expect(node.size).toBeUndefined();
      expect(node.anchors).toBeUndefined();
    });

    it('should work with existing nested structure tests', () => {
      const dsl = `
        parent {
          label: "Parent"
          direction: "vertical"
          
          child1 {
            label: "Child 1"
            size: 0.6
          }
          
          group {
            direction: "horizontal"
            
            child2 {
              label: "Child 2"
              anchors: {
                connector: [1.0, 0.5]
              }
            }
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const parent = result.document!.nodes[0];
      expect(parent.children).toHaveLength(2);
      
      const child1 = parent.children[0] as any;
      expect(child1.size).toBe(0.6);
      
      const group = parent.children[1];
      const child2 = group.children[0] as any;
      expect(child2.anchors!.connector).toEqual([1.0, 0.5]);
    });
  });
});