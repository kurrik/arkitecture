package arkitecture_test

import (
	"strings"
	"testing"

	"github.com/kurrik/arkitecture"
)

func TestParseValid(t *testing.T) {
	r := arkitecture.Parse(`a { label: "x" }`)
	if !r.Success {
		t.Fatalf("Parse failed: %+v", r.Errors)
	}
	if r.Document == nil || len(r.Document.Nodes) != 1 {
		t.Fatalf("expected one node, got %+v", r.Document)
	}
}

func TestParseSyntaxError(t *testing.T) {
	r := arkitecture.Parse("a")
	if r.Success {
		t.Fatal("expected failure for incomplete node")
	}
	if len(r.Errors) == 0 || r.Errors[0].Type != arkitecture.ErrorSyntax {
		t.Fatalf("expected a syntax error, got %+v", r.Errors)
	}
}

func TestToSVGValidateOnly(t *testing.T) {
	res := arkitecture.ToSVG(`a { label: "x" }`, &arkitecture.Options{ValidateOnly: true})
	if !res.Success {
		t.Fatalf("validate-only failed: %+v", res.Errors)
	}
	if res.SVG != "" {
		t.Errorf("validate-only should not produce SVG, got %q", res.SVG)
	}
}

func TestToSVGParseErrorSurfaces(t *testing.T) {
	res := arkitecture.ToSVG("a {", nil)
	if res.Success {
		t.Fatal("expected failure for unterminated node")
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected errors to be reported")
	}
}

// TestToSVGGeneratorPending characterises the current state of the rewrite: the
// generator stage has not been ported yet, so a non-validate-only run wires
// through to the stub and surfaces its error. Replace this with real SVG
// assertions once generator.GenerateSVG is implemented.
func TestToSVGGeneratorPending(t *testing.T) {
	res := arkitecture.ToSVG(`a { label: "x" }`, nil)
	if res.Success {
		t.Skip("generator appears to be implemented; update this test with real SVG assertions")
	}
	if len(res.Errors) == 0 || !strings.Contains(res.Errors[0].Message, "not yet ported") {
		t.Fatalf("expected the generator-pending error, got %+v", res.Errors)
	}
}
