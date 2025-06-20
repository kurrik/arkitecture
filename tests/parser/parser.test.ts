import { Parser, ParseError } from '../../src/parser/parser';
import { Tokenizer } from '../../src/parser/tokenizer';
import { parseArkitecture } from '../../src/parser';

describe('Parser', () => {
  describe('Simple node parsing', () => {
    it('should parse a node with just an ID', () => {
      const dsl = 'nodeId {}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document).toBeDefined();
      expect(result.document!.nodes).toHaveLength(1);
      expect(result.document!.nodes[0]).toEqual({
        id: 'nodeId',
        children: [],
      });
      expect(result.document!.arrows).toHaveLength(0);
    });

    it('should parse a node with a label', () => {
      const dsl = 'testNode {\n  label: "Test Label"\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      expect(result.document!.nodes[0]).toEqual({
        id: 'testNode',
        label: 'Test Label',
        children: [],
      });
    });

    it('should parse a node with direction', () => {
      const dsl = 'testNode {\n  direction: "horizontal"\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      expect(result.document!.nodes[0]).toEqual({
        id: 'testNode',
        direction: 'horizontal',
        children: [],
      });
    });

    it('should parse a node with both label and direction', () => {
      const dsl = `
        myNode {
          label: "My Node"
          direction: "vertical"
        }
      `;
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      expect(result.document!.nodes[0]).toEqual({
        id: 'myNode',
        label: 'My Node',
        direction: 'vertical',
        children: [],
      });
    });
  });

  describe('Multiple nodes', () => {
    it('should parse multiple top-level nodes', () => {
      const dsl = `
        node1 {
          label: "First Node"
        }
        
        node2 {
          label: "Second Node"
          direction: "horizontal"
        }
        
        node3 {}
      `;
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(3);
      
      expect(result.document!.nodes[0]).toEqual({
        id: 'node1',
        label: 'First Node',
        children: [],
      });
      
      expect(result.document!.nodes[1]).toEqual({
        id: 'node2',
        label: 'Second Node',
        direction: 'horizontal',
        children: [],
      });
      
      expect(result.document!.nodes[2]).toEqual({
        id: 'node3',
        children: [],
      });
    });
  });

  describe('Empty documents', () => {
    it('should handle empty documents', () => {
      const dsl = '';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(0);
      expect(result.document!.arrows).toHaveLength(0);
    });

    it('should handle documents with only comments and whitespace', () => {
      const dsl = `
        # This is a comment
        
        # Another comment
      `;
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(0);
    });
  });

  describe('Error handling', () => {
    it('should report error for missing opening brace', () => {
      const dsl = 'nodeId';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected '{'");
      expect(result.errors[0].line).toBe(1);
    });

    it('should report error for missing closing brace', () => {
      const dsl = 'nodeId {\n  label: "test"';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected '}'");
    });

    it('should report error for missing colon after property name', () => {
      const dsl = 'nodeId {\n  label "test"\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Expected ':'");
    });

    it('should report error for invalid property name', () => {
      const dsl = 'nodeId {\n  invalidProp: "test"\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Unknown property 'invalidProp'");
    });

    it('should report error for invalid direction value', () => {
      const dsl = 'nodeId {\n  direction: "diagonal"\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain("Invalid direction 'diagonal'");
    });

    it('should report error for non-string label value', () => {
      const dsl = 'nodeId {\n  label: 123\n}';
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain('Expected string value for label');
    });

    it('should report multiple errors and continue parsing', () => {
      const dsl = `
        node1 {
          label: 123
          direction: "invalid"
        }
        
        node2 {
          unknownProp: "test"
        }
      `;
      const tokenizer = new Tokenizer(dsl);
      const tokens = tokenizer.tokenize();
      const parser = new Parser(tokens);
      
      const result = parser.parseDocument();
      
      expect(result.success).toBe(false);
      expect(result.errors.length).toBeGreaterThan(1);
      
      // Should still have created nodes despite errors
      expect(result.document).toBeDefined();
      expect(result.document!.nodes).toHaveLength(2);
    });
  });

  describe('Integration with parseArkitecture', () => {
    it('should work through the full parsing pipeline', () => {
      const dsl = `
        container1 {
          label: "Main Container"
          direction: "vertical"
        }
        
        container2 {
          label: "Side Container"
          direction: "horizontal"
        }
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(2);
      expect(result.errors).toHaveLength(0);
      
      expect(result.document!.nodes[0]).toEqual({
        id: 'container1',
        label: 'Main Container',
        direction: 'vertical',
        children: [],
      });
      
      expect(result.document!.nodes[1]).toEqual({
        id: 'container2',
        label: 'Side Container',
        direction: 'horizontal',
        children: [],
      });
    });

    it('should handle tokenizer errors through parseArkitecture', () => {
      const dsl = 'nodeId { label: "unterminated string }';
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
    });

    it('should handle parser errors through parseArkitecture', () => {
      const dsl = 'nodeId { invalidProperty: "test" }';
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].type).toBe('syntax');
      expect(result.errors[0].message).toContain('Unknown property');
    });
  });

  describe('Edge cases', () => {
    it('should handle nodes with no properties', () => {
      const dsl = 'emptyNode {}';
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(1);
      expect(result.document!.nodes[0]).toEqual({
        id: 'emptyNode',
        children: [],
      });
    });

    it('should handle complex identifiers', () => {
      const dsl = `
        node_with_underscores {}
        nodeWithCamelCase {}
        node123 {}
      `;
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes).toHaveLength(3);
      expect(result.document!.nodes[0].id).toBe('node_with_underscores');
      expect(result.document!.nodes[1].id).toBe('nodeWithCamelCase');
      expect(result.document!.nodes[2].id).toBe('node123');
    });

    it('should handle strings with special characters', () => {
      const dsl = 'node { label: "Node with \\"quotes\\" and \\n newlines" }';
      
      const result = parseArkitecture(dsl);
      
      expect(result.success).toBe(true);
      expect(result.document!.nodes[0].label).toBe('Node with "quotes" and \n newlines');
    });
  });
});