import { parseArkitecture } from '../../src/parser';

describe('Arrow Parser', () => {
  describe('Simple arrow parsing', () => {
    it('should parse simple arrow between two nodes', () => {
      const dsl = `
        node1 {
          label: "First Node"
        }
        
        node2 {
          label: "Second Node"
        }
        
        node1 --> node2
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(2);
      expect(result.document!.arrows).toHaveLength(1);
      
      const arrow = result.document!.arrows[0];
      expect(arrow.source).toBe('node1');
      expect(arrow.target).toBe('node2');
    });

    it('should parse multiple arrows', () => {
      const dsl = `
        node1 {}
        node2 {}
        node3 {}
        
        node1 --> node2
        node2 --> node3
        node1 --> node3
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(3);
      
      expect(result.document!.arrows[0]).toEqual({
        source: 'node1',
        target: 'node2'
      });
      expect(result.document!.arrows[1]).toEqual({
        source: 'node2',
        target: 'node3'
      });
      expect(result.document!.arrows[2]).toEqual({
        source: 'node1',
        target: 'node3'
      });
    });

    it('should handle empty document with no arrows', () => {
      const dsl = `
        node1 {}
        node2 {}
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(0);
    });
  });

  describe('Arrows with anchor references', () => {
    it('should parse arrow with target anchor', () => {
      const dsl = `
        node1 {}
        node2 {
          anchors: {
            top: [0.5, 0.0]
          }
        }
        
        node1 --> node2#top
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(1);
      
      const arrow = result.document!.arrows[0];
      expect(arrow.source).toBe('node1');
      expect(arrow.target).toBe('node2#top');
    });

    it('should parse arrows with various anchor names', () => {
      const dsl = `
        source {}
        target {
          anchors: {
            top: [0.5, 0.0],
            bottom: [0.5, 1.0],
            left: [0.0, 0.5],
            right: [1.0, 0.5],
            center: [0.5, 0.5]
          }
        }
        
        source --> target#top
        source --> target#bottom
        source --> target#left
        source --> target#right
        source --> target#center
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(5);
      
      expect(result.document!.arrows[0].target).toBe('target#top');
      expect(result.document!.arrows[1].target).toBe('target#bottom');
      expect(result.document!.arrows[2].target).toBe('target#left');
      expect(result.document!.arrows[3].target).toBe('target#right');
      expect(result.document!.arrows[4].target).toBe('target#center');
    });
  });

  describe('Arrows with nested node paths', () => {
    it('should parse arrow with dot-separated source path', () => {
      const dsl = `
        parent {
          child {
            label: "Child Node"
          }
        }
        
        target {}
        
        parent.child --> target
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(1);
      
      const arrow = result.document!.arrows[0];
      expect(arrow.source).toBe('parent.child');
      expect(arrow.target).toBe('target');
    });

    it('should parse arrow with dot-separated target path', () => {
      const dsl = `
        source {}
        
        parent {
          child {
            label: "Child Node"
          }
        }
        
        source --> parent.child
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(1);
      
      const arrow = result.document!.arrows[0];
      expect(arrow.source).toBe('source');
      expect(arrow.target).toBe('parent.child');
    });

    it('should parse deeply nested node paths', () => {
      const dsl = `
        level1 {
          level2 {
            level3 {
              level4 {}
            }
          }
        }
        
        other {}
        
        level1.level2.level3.level4 --> other
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(1);
      
      const arrow = result.document!.arrows[0];
      expect(arrow.source).toBe('level1.level2.level3.level4');
      expect(arrow.target).toBe('other');
    });

    it('should parse nested paths with anchor references', () => {
      const dsl = `
        parent {
          group {
            child {
              anchors: {
                output: [1.0, 0.5]
              }
            }
          }
        }
        
        target {}
        
        parent.group.child --> target
        target --> parent.group.child#output
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(2);
      
      expect(result.document!.arrows[0]).toEqual({
        source: 'parent.group.child',
        target: 'target'
      });
      expect(result.document!.arrows[1]).toEqual({
        source: 'target',
        target: 'parent.group.child#output'
      });
    });
  });

  describe('Complex arrow scenarios', () => {
    it('should parse arrows combined with nested node structures', () => {
      const dsl = `
        container1 {
          label: "Container 1"
          direction: "vertical"
          
          service1 {
            label: "Service 1"
            anchors: {
              api: [1.0, 0.5]
            }
          }
          
          service2 {
            label: "Service 2"
          }
        }
        
        container2 {
          label: "Container 2"
          
          database {
            label: "Database"
            anchors: {
              connection: [0.0, 0.5]
            }
          }
        }
        
        container1.service1 --> container2.database#connection
        container1.service2 --> container2.database
        container2.database --> container1.service1#api
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(2);
      expect(result.document!.arrows).toHaveLength(3);
      
      expect(result.document!.arrows[0]).toEqual({
        source: 'container1.service1',
        target: 'container2.database#connection'
      });
      expect(result.document!.arrows[1]).toEqual({
        source: 'container1.service2',
        target: 'container2.database'
      });
      expect(result.document!.arrows[2]).toEqual({
        source: 'container2.database',
        target: 'container1.service1#api'
      });
    });

    it('should handle arrows with groups in node paths', () => {
      const dsl = `
        parent {
          group {
            direction: "horizontal"
            
            child1 {}
            child2 {}
          }
          
          standalone {}
        }
        
        external {}
        
        parent.group.child1 --> parent.standalone
        parent.group.child2 --> external
        external --> parent.group.child1
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(3);
      
      expect(result.document!.arrows[0].source).toBe('parent.group.child1');
      expect(result.document!.arrows[0].target).toBe('parent.standalone');
      
      expect(result.document!.arrows[1].source).toBe('parent.group.child2');
      expect(result.document!.arrows[1].target).toBe('external');
      
      expect(result.document!.arrows[2].source).toBe('external');
      expect(result.document!.arrows[2].target).toBe('parent.group.child1');
    });
  });

  describe('Error handling', () => {
    it('should report error for malformed arrow syntax - missing arrow operator', () => {
      const dsl = `
        node1 {}
        node2 {}
        
        node1 node2
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      // This could be caught either as a malformed node or arrow syntax
      expect(result.errors[0].message).toMatch(/Expected.*('{' after node id|'-->' arrow operator)/);
    });

    it('should report error for missing target after arrow operator', () => {
      const dsl = `
        node1 {}
        
        node1 -->
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected arrow target identifier after '-->'");
    });

    it('should report error for invalid node path format', () => {
      const dsl = `
        node1 {}
        node2 {}
        
        node1. --> node2
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      // This could be caught as a malformed node or arrow path error
      expect(result.errors[0].message).toMatch(/Expected.*('{' after node id|identifier after '.')/);
    });

    it('should report error for missing anchor identifier after hash', () => {
      const dsl = `
        node1 {}
        node2 {}
        
        node1 --> node2#
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected anchor identifier after '#'");
    });

    it('should continue parsing after arrow errors', () => {
      const dsl = `
        node1 {}
        node2 {}
        node3 {}
        
        node1 node2
        node2 --> node3
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.document!.arrows).toHaveLength(1);
      expect(result.document!.arrows[0]).toEqual({
        source: 'node2',
        target: 'node3'
      });
    });

    it('should handle invalid arrow source gracefully', () => {
      const dsl = `
        node1 {}
        
        123 --> node1
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      // This could be caught as invalid node or arrow source
      expect(result.errors[0].message).toMatch(/Expected.*(node identifier|arrow source identifier)/);
    });
  });

  describe('Integration with specification example', () => {
    it('should parse arrows from the DSL specification example', () => {
      const dsl = `
        c1 {
          label: "Container 1"
          direction: "vertical"
          anchors: {
            a1: [0.0, 0.5]
          }

          n2 {
            label: "Node 1"
          }

          group {
            direction: "horizontal"

            n3 {
              label: "Node 3"
              anchors: {
                a2: [1.0, 0.5]
              }
            }

            n4 {
              label: "Node 4"
            }
          }
        }
        
        c2 {
          label: "Container 2"
          anchors: {
            input: [0.0, 0.5]
          }
        }
        
        c1.n2 --> c1.group.n3
        c1.group.n3 --> c1.group.n4
        c1.group.n3#a2 --> c2#input
        c2 --> c1#a1
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.arrows).toHaveLength(4);
      
      expect(result.document!.arrows[0]).toEqual({
        source: 'c1.n2',
        target: 'c1.group.n3'
      });
      expect(result.document!.arrows[1]).toEqual({
        source: 'c1.group.n3',
        target: 'c1.group.n4'
      });
      expect(result.document!.arrows[2]).toEqual({
        source: 'c1.group.n3#a2',
        target: 'c2#input'
      });
      expect(result.document!.arrows[3]).toEqual({
        source: 'c2',
        target: 'c1#a1'
      });
    });
  });

  describe('Backward compatibility', () => {
    it('should still parse documents with only nodes (no arrows)', () => {
      const dsl = `
        node1 {
          label: "First Node"
          direction: "horizontal"
        }
        
        node2 {
          label: "Second Node"
          size: 0.6
          anchors: {
            center: [0.5, 0.5]
          }
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(2);
      expect(result.document!.arrows).toHaveLength(0);
      
      // Verify nodes are still parsed correctly
      expect(result.document!.nodes[0].label).toBe('First Node');
      expect(result.document!.nodes[0].direction).toBe('horizontal');
      expect(result.document!.nodes[1].size).toBe(0.6);
      expect(result.document!.nodes[1].anchors!.center).toEqual([0.5, 0.5]);
    });

    it('should work with all existing nested and property features', () => {
      const dsl = `
        parent {
          label: "Parent Container"
          direction: "vertical"
          size: 0.8
          
          child1 {
            label: "Child 1"
            anchors: {
              output: [1.0, 0.5]
            }
          }
          
          group {
            direction: "horizontal"
            
            child2 {
              label: "Child 2"
            }
            
            child3 {
              label: "Child 3"
              anchors: {
                input: [0.0, 0.5]
              }
            }
          }
        }
        
        external {
          label: "External Service"
          anchors: {
            api: [0.0, 0.5]
          }
        }
        
        parent.child1 --> parent.group.child3#input
        parent.group.child2 --> external#api
        external --> parent.child1#output
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(2);
      expect(result.document!.arrows).toHaveLength(3);
      
      // Verify complex nested structure with all properties still works
      const parent = result.document!.nodes[0];
      expect(parent.size).toBe(0.8);
      expect(parent.children).toHaveLength(2);
      
      const child1 = parent.children[0] as any;
      expect(child1.anchors!.output).toEqual([1.0, 0.5]);
      
      const group = parent.children[1];
      expect(group.children).toHaveLength(2);
      
      // Verify arrows reference the correct nested paths
      expect(result.document!.arrows[0].source).toBe('parent.child1');
      expect(result.document!.arrows[0].target).toBe('parent.group.child3#input');
    });
  });
});