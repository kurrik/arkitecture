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

func TestToSVGGeneratesSVG(t *testing.T) {
	res := arkitecture.ToSVG(`a { label: "x" }`, nil)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res.Errors)
	}
	if !strings.HasPrefix(res.SVG, `<svg xmlns="http://www.w3.org/2000/svg"`) {
		t.Errorf("unexpected SVG prefix: %q", res.SVG)
	}
	if !strings.Contains(res.SVG, "</svg>") {
		t.Errorf("SVG is not closed: %q", res.SVG)
	}
}
