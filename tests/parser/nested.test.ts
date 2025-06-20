import { parseArkitecture } from '../../src/parser';

describe('Nested Node Parser', () => {
  describe('Simple parent-child relationships', () => {
    it('should parse a parent node with a single child', () => {
      const dsl = `
        parent {
          label: "Parent Node"
          
          child {
            label: "Child Node"
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const parent = result.document!.nodes[0];
      expect(parent.id).toBe('parent');
      expect(parent.label).toBe('Parent Node');
      expect(parent.children).toHaveLength(1);
      
      const child = parent.children[0];
      expect(child).toHaveProperty('id', 'child');
      expect(child).toHaveProperty('label', 'Child Node');
      expect((child as any).children).toHaveLength(0);
    });

    it('should parse a parent node with multiple children', () => {
      const dsl = `
        parent {
          direction: "horizontal"
          
          child1 {
            label: "First Child"
          }
          
          child2 {
            label: "Second Child"
          }
          
          child3 {}
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const parent = result.document!.nodes[0];
      expect(parent.direction).toBe('horizontal');
      expect(parent.children).toHaveLength(3);
      
      expect((parent.children[0] as any).id).toBe('child1');
      expect((parent.children[0] as any).label).toBe('First Child');
      
      expect((parent.children[1] as any).id).toBe('child2');
      expect((parent.children[1] as any).label).toBe('Second Child');
      
      expect((parent.children[2] as any).id).toBe('child3');
      expect((parent.children[2] as any).label).toBeUndefined();
    });
  });

  describe('Group nodes', () => {
    it('should parse a simple group with children', () => {
      const dsl = `
        container {
          label: "Container"
          
          group {
            direction: "horizontal"
            
            item1 {
              label: "Item 1"
            }
            
            item2 {
              label: "Item 2"
            }
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const container = result.document!.nodes[0];
      expect(container.children).toHaveLength(1);
      
      const group = container.children[0];
      expect(group).not.toHaveProperty('id');
      expect(group).toHaveProperty('direction', 'horizontal');
      expect(group.children).toHaveLength(2);
      
      expect((group.children[0] as any).id).toBe('item1');
      expect((group.children[1] as any).id).toBe('item2');
    });

    it('should parse a group without direction property', () => {
      const dsl = `
        container {
          group {
            child1 {}
            child2 {}
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const container = result.document!.nodes[0];
      const group = container.children[0];
      
      expect(group).not.toHaveProperty('direction');
      expect(group.children).toHaveLength(2);
    });

    it('should parse nested groups', () => {
      const dsl = `
        root {
          group {
            direction: "vertical"
            
            child1 {}
            
            group {
              direction: "horizontal"
              
              child2 {}
              child3 {}
            }
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const root = result.document!.nodes[0];
      const outerGroup = root.children[0];
      
      expect(outerGroup).toHaveProperty('direction', 'vertical');
      expect(outerGroup.children).toHaveLength(2);
      
      const innerGroup = outerGroup.children[1];
      expect(innerGroup).toHaveProperty('direction', 'horizontal');
      expect(innerGroup.children).toHaveLength(2);
    });
  });

  describe('Deeply nested structures', () => {
    it('should parse 3+ levels of nesting', () => {
      const dsl = `
        level1 {
          label: "Level 1"
          
          level2 {
            label: "Level 2"
            
            level3 {
              label: "Level 3"
              
              level4 {
                label: "Level 4"
              }
            }
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      let current = result.document!.nodes[0];
      expect(current.id).toBe('level1');
      expect(current.label).toBe('Level 1');
      
      current = current.children[0] as any;
      expect(current.id).toBe('level2');
      expect(current.label).toBe('Level 2');
      
      current = current.children[0] as any;
      expect(current.id).toBe('level3');
      expect(current.label).toBe('Level 3');
      
      current = current.children[0] as any;
      expect(current.id).toBe('level4');
      expect(current.label).toBe('Level 4');
      expect(current.children).toHaveLength(0);
    });

    it('should parse complex mixed structures', () => {
      const dsl = `
        parent {
          label: "Parent Node"
          direction: "vertical"
          
          child1 {
            label: "Child 1"
          }
          
          group {
            direction: "horizontal"
            
            child2 {
              label: "Child 2"
              
              grandchild {
                label: "Grandchild"
              }
            }
            
            child3 {
              label: "Child 3"
            }
          }
          
          child4 {
            label: "Child 4"
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      
      const parent = result.document!.nodes[0];
      expect(parent.children).toHaveLength(3);
      
      expect((parent.children[0] as any).id).toBe('child1');
      
      const group = parent.children[1];
      expect(group).toHaveProperty('direction', 'horizontal');
      expect(group.children).toHaveLength(2);
      
      const child2 = group.children[0] as any;
      expect(child2.id).toBe('child2');
      expect(child2.children).toHaveLength(1);
      expect(child2.children[0].id).toBe('grandchild');
      
      expect((parent.children[2] as any).id).toBe('child4');
    });
  });

  describe('Error handling', () => {
    it('should report error for unclosed braces in nested structures', () => {
      const dsl = `
        parent {
          child {
            label: "Missing close brace"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected '}'");
    });

    it('should validate that groups cannot have ID or label properties', () => {
      const dsl = `
        container {
          group {
            label: "Groups cannot have labels"
            direction: "horizontal"
            
            child {}
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Groups can only have 'direction' property");
    });

    it('should handle invalid direction in groups', () => {
      const dsl = `
        container {
          group {
            direction: "diagonal"
            child {}
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Invalid direction 'diagonal'");
    });

    it('should report proper context for nested errors', () => {
      const dsl = `
        parent {
          child {
            invalidProperty: "test"
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Unknown property 'invalidProperty'");
      expect(result.errors[0].line).toBe(4); // Should point to the line with the error
    });
  });

  describe('Integration with specification example', () => {
    it('should parse the DSL specification example structure', () => {
      const dsl = `
        c1 {
          label: "Container 1"
          direction: "vertical"

          n2 {
            label: "Node 1"
          }

          group {
            direction: "horizontal"

            n3 {
              label: "Node 3"
            }

            n4 {
              label: "Node 4"
            }
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      
      const c1 = result.document!.nodes[0];
      expect(c1.id).toBe('c1');
      expect(c1.label).toBe('Container 1');
      expect(c1.direction).toBe('vertical');
      expect(c1.children).toHaveLength(2);
      
      const n2 = c1.children[0] as any;
      expect(n2.id).toBe('n2');
      expect(n2.label).toBe('Node 1');
      
      const group = c1.children[1];
      expect(group).toHaveProperty('direction', 'horizontal');
      expect(group.children).toHaveLength(2);
      
      const n3 = group.children[0] as any;
      expect(n3.id).toBe('n3');
      expect(n3.label).toBe('Node 3');
      
      const n4 = group.children[1] as any;
      expect(n4.id).toBe('n4');
      expect(n4.label).toBe('Node 4');
    });
  });

  describe('Backward compatibility', () => {
    it('should still handle flat structures from Step 3', () => {
      const dsl = `
        node1 {
          label: "First Node"
          direction: "horizontal"
        }
        
        node2 {
          label: "Second Node"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(2);
      
      expect(result.document!.nodes[0]).toEqual({
        id: 'node1',
        label: 'First Node',
        direction: 'horizontal',
        children: [],
      });
      
      expect(result.document!.nodes[1]).toEqual({
        id: 'node2',
        label: 'Second Node',
        children: [],
      });
    });
  });
});