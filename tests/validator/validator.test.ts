import { Validator } from '../../src/validator/validator';
import { Document, ContainerNode, Arrow } from '../../src/types';

describe('Validator', () => {
  describe('Valid documents', () => {
    it('should return no errors for valid document with simple nodes', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: []
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: []
          }
        ],
        arrows: [
          {
            source: 'node1',
            target: 'node2'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should return no errors for valid nested document', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            children: [
              {
                id: 'child1',
                label: 'Child 1',
                children: []
              },
              {
                id: 'child2',
                label: 'Child 2',
                children: []
              }
            ]
          }
        ],
        arrows: [
          {
            source: 'parent.child1',
            target: 'parent.child2'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should return no errors for valid document with anchors', () => {
      const document: Document = {
        nodes: [
          {
            id: 'source',
            label: 'Source',
            children: []
          },
          {
            id: 'target',
            label: 'Target',
            anchors: {
              input: [0.0, 0.5],
              output: [1.0, 0.5]
            },
            children: []
          }
        ],
        arrows: [
          {
            source: 'source',
            target: 'target#input'
          },
          {
            source: 'target#output',
            target: 'source#center'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should return no errors for valid document with groups', () => {
      const document: Document = {
        nodes: [
          {
            id: 'container',
            label: 'Container',
            children: [
              {
                direction: 'horizontal',
                children: [
                  {
                    id: 'item1',
                    label: 'Item 1',
                    children: []
                  },
                  {
                    id: 'item2',
                    label: 'Item 2',
                    children: []
                  }
                ]
              }
            ]
          }
        ],
        arrows: [
          {
            source: 'container.group.item1',
            target: 'container.group.item2'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should return no errors for valid size constraints', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            size: 0.0,
            children: []
          },
          {
            id: 'node2',
            label: 'Node 2',
            size: 1.0,
            children: []
          },
          {
            id: 'node3',
            label: 'Node 3',
            size: 0.5,
            children: []
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should return no errors for valid anchor coordinates', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            anchors: {
              corner1: [0.0, 0.0],
              corner2: [1.0, 1.0],
              center: [0.5, 0.5],
              edge: [0.0, 0.5]
            },
            children: []
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });
  });

  describe('Node ID uniqueness validation', () => {
    it('should detect duplicate node IDs at root level', () => {
      const document: Document = {
        nodes: [
          {
            id: 'duplicate',
            label: 'First',
            children: []
          },
          {
            id: 'duplicate',
            label: 'Second',
            children: []
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Duplicate node ID 'duplicate'");
      expect(errors[0].message).toContain('root scope');
    });

    it('should detect duplicate node IDs within same parent', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            children: [
              {
                id: 'child',
                label: 'Child 1',
                children: []
              },
              {
                id: 'child',
                label: 'Child 2',
                children: []
              }
            ]
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Duplicate node ID 'child'");
      expect(errors[0].message).toContain('parent scope');
    });

    it('should allow same ID in different parent scopes', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent1',
            label: 'Parent 1',
            children: [
              {
                id: 'child',
                label: 'Child 1',
                children: []
              }
            ]
          },
          {
            id: 'parent2',
            label: 'Parent 2',
            children: [
              {
                id: 'child',
                label: 'Child 2',
                children: []
              }
            ]
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should detect duplicate IDs across group boundaries', () => {
      const document: Document = {
        nodes: [
          {
            id: 'container',
            label: 'Container',
            children: [
              {
                id: 'item',
                label: 'Item 1',
                children: []
              },
              {
                direction: 'horizontal',
                children: [
                  {
                    id: 'item',
                    label: 'Item 2',
                    children: []
                  }
                ]
              }
            ]
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Duplicate node ID 'item'");
    });
  });

  describe('Arrow reference validation', () => {
    it('should detect invalid source node reference', () => {
      const document: Document = {
        nodes: [
          {
            id: 'existing',
            label: 'Existing Node',
            children: []
          }
        ],
        arrows: [
          {
            source: 'nonexistent',
            target: 'existing'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Arrow source node 'nonexistent' does not exist");
    });

    it('should detect invalid target node reference', () => {
      const document: Document = {
        nodes: [
          {
            id: 'existing',
            label: 'Existing Node',
            children: []
          }
        ],
        arrows: [
          {
            source: 'existing',
            target: 'nonexistent'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Arrow target node 'nonexistent' does not exist");
    });

    it('should detect invalid nested node references', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            children: [
              {
                id: 'child',
                label: 'Child',
                children: []
              }
            ]
          }
        ],
        arrows: [
          {
            source: 'parent.nonexistent',
            target: 'parent.child'
          },
          {
            source: 'parent.child',
            target: 'nonexistent.child'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(2);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Arrow source node 'parent.nonexistent' does not exist");
      expect(errors[1].type).toBe('reference');
      expect(errors[1].message).toContain("Arrow target node 'nonexistent.child' does not exist");
    });

    it('should validate arrows with anchor references correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: []
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: []
          }
        ],
        arrows: [
          {
            source: 'node1#center',
            target: 'node2#center'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });
  });

  describe('Anchor reference validation', () => {
    it('should validate implicit center anchor', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            children: []
          },
          {
            id: 'node2',
            label: 'Node 2',
            children: []
          }
        ],
        arrows: [
          {
            source: 'node1',
            target: 'node2#center'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should detect invalid anchor references', () => {
      const document: Document = {
        nodes: [
          {
            id: 'source',
            label: 'Source',
            anchors: {
              output: [1.0, 0.5]
            },
            children: []
          },
          {
            id: 'target',
            label: 'Target',
            children: []
          }
        ],
        arrows: [
          {
            source: 'source#nonexistent',
            target: 'target#invalid'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(2);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Arrow source anchor 'nonexistent' does not exist on node 'source'");
      expect(errors[1].type).toBe('reference');
      expect(errors[1].message).toContain("Arrow target anchor 'invalid' does not exist on node 'target'");
    });

    it('should validate explicit anchors correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'source',
            label: 'Source',
            anchors: {
              output: [1.0, 0.5],
              api: [0.5, 0.0]
            },
            children: []
          },
          {
            id: 'target',
            label: 'Target',
            anchors: {
              input: [0.0, 0.5]
            },
            children: []
          }
        ],
        arrows: [
          {
            source: 'source#output',
            target: 'target#input'
          },
          {
            source: 'source#api',
            target: 'target#center'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });

    it('should handle nested node anchor references', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            children: [
              {
                id: 'child',
                label: 'Child',
                anchors: {
                  port: [1.0, 0.5]
                },
                children: []
              }
            ]
          },
          {
            id: 'external',
            label: 'External',
            children: []
          }
        ],
        arrows: [
          {
            source: 'parent.child#port',
            target: 'external'
          },
          {
            source: 'external',
            target: 'parent.child#nonexistent'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(1);
      expect(errors[0].type).toBe('reference');
      expect(errors[0].message).toContain("Arrow target anchor 'nonexistent' does not exist on node 'parent.child'");
    });
  });

  describe('Constraint validation', () => {
    it('should detect size values out of range', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            size: -0.1,
            children: []
          },
          {
            id: 'node2',
            label: 'Node 2',
            size: 1.5,
            children: []
          },
          {
            id: 'node3',
            label: 'Node 3',
            size: 2.0,
            children: []
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(3);
      expect(errors[0].type).toBe('constraint');
      expect(errors[0].message).toContain("Node 'node1' size -0.1 is out of range");
      expect(errors[1].type).toBe('constraint');
      expect(errors[1].message).toContain("Node 'node2' size 1.5 is out of range");
      expect(errors[2].type).toBe('constraint');
      expect(errors[2].message).toContain("Node 'node3' size 2 is out of range");
    });

    it('should detect anchor coordinate values out of range', () => {
      const document: Document = {
        nodes: [
          {
            id: 'node1',
            label: 'Node 1',
            anchors: {
              invalid1: [-0.1, 0.5],
              invalid2: [0.5, 1.5],
              invalid3: [2.0, -1.0],
              valid: [0.5, 0.5]
            },
            children: []
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(4);
      expect(errors[0].type).toBe('constraint');
      expect(errors[0].message).toContain("Node 'node1' anchor 'invalid1' X coordinate -0.1 is out of range");
      expect(errors[1].type).toBe('constraint');
      expect(errors[1].message).toContain("Node 'node1' anchor 'invalid2' Y coordinate 1.5 is out of range");
      expect(errors[2].type).toBe('constraint');
      expect(errors[2].message).toContain("Node 'node1' anchor 'invalid3' X coordinate 2 is out of range");
      expect(errors[3].type).toBe('constraint');
      expect(errors[3].message).toContain("Node 'node1' anchor 'invalid3' Y coordinate -1 is out of range");
    });

    it('should validate constraints in nested nodes', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            size: 1.1,
            children: [
              {
                id: 'child',
                label: 'Child',
                anchors: {
                  bad: [1.5, 0.5]
                },
                children: []
              }
            ]
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(2);
      expect(errors[0].type).toBe('constraint');
      expect(errors[0].message).toContain("Node 'parent' size 1.1 is out of range");
      expect(errors[1].type).toBe('constraint');
      expect(errors[1].message).toContain("Node 'child' anchor 'bad' X coordinate 1.5 is out of range");
    });
  });

  describe('Multiple error scenarios', () => {
    it('should collect all errors in a complex invalid document', () => {
      const document: Document = {
        nodes: [
          {
            id: 'duplicate',
            label: 'First',
            size: 1.5,
            anchors: {
              bad: [2.0, -1.0]
            },
            children: []
          },
          {
            id: 'duplicate',
            label: 'Second',
            size: -0.5,
            children: []
          }
        ],
        arrows: [
          {
            source: 'nonexistent1',
            target: 'nonexistent2'
          },
          {
            source: 'duplicate#invalid',
            target: 'duplicate#missing'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors.length).toBeGreaterThanOrEqual(6);
      
      // Should have reference, constraint, and reference errors
      const referenceErrors = errors.filter(e => e.type === 'reference');
      const constraintErrors = errors.filter(e => e.type === 'constraint');
      
      expect(referenceErrors.length).toBeGreaterThanOrEqual(3); // Duplicate ID + missing node refs + missing anchors
      expect(constraintErrors.length).toBeGreaterThanOrEqual(3); // Size + coordinate violations
    });

    it('should validate specification example correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'c1',
            label: 'Container 1',
            direction: 'vertical',
            anchors: {
              a1: [0.0, 0.5]
            },
            children: [
              {
                id: 'n2',
                label: 'Node 1',
                children: []
              },
              {
                direction: 'horizontal',
                children: [
                  {
                    id: 'n3',
                    label: 'Node 3',
                    anchors: {
                      a2: [1.0, 0.5]
                    },
                    children: []
                  },
                  {
                    id: 'n4',
                    label: 'Node 4',
                    children: []
                  }
                ]
              }
            ]
          },
          {
            id: 'c2',
            label: 'Container 2',
            anchors: {
              input: [0.0, 0.5]
            },
            children: []
          }
        ],
        arrows: [
          {
            source: 'c1.n2',
            target: 'c1.group.n3'
          },
          {
            source: 'c1.group.n3',
            target: 'c1.group.n4'
          },
          {
            source: 'c1.group.n3#a2',
            target: 'c2#input'
          },
          {
            source: 'c2',
            target: 'c1#a1'
          }
        ]
      };

      const validator = new Validator(document);
      const errors = validator.validate();

      expect(errors).toHaveLength(0);
    });
  });

  describe('Helper methods', () => {
    it('should resolve node paths correctly', () => {
      const document: Document = {
        nodes: [
          {
            id: 'parent',
            label: 'Parent',
            children: [
              {
                id: 'child',
                label: 'Child',
                children: []
              }
            ]
          }
        ],
        arrows: []
      };

      const validator = new Validator(document);
      validator.validate(); // Build node map

      expect(validator.resolveNodePath('parent')).toBeTruthy();
      expect(validator.resolveNodePath('parent.child')).toBeTruthy();
      expect(validator.resolveNodePath('nonexistent')).toBeNull();
    });
  });
});