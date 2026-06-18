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

func TestToSVGReuseAndKind(t *testing.T) {
	const in = `app {
  grp { a {} b {} }
  ext { kind: invisible c {} }
}
@layout {
  @block row { box: none; direction: horizontal }
  app     { direction: vertical }
  app.grp { @use row }
}`
	res := arkitecture.ToSVG(in, nil)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res.Errors)
	}
	// app.grp (via @use row) and app.ext (via kind: invisible) are box:none and
	// draw no rect; app and the three leaves a/b/c do. Confirms both the @use
	// import and the kind hook reached the generator.
	if got := strings.Count(res.SVG, "<rect"); got != 4 {
		t.Errorf("rect count = %d, want 4 (app + a + b + c; grp and ext are box:none)", got)
	}
}

func TestToSVGUndefinedUseErrors(t *testing.T) {
	res := arkitecture.ToSVG("a {}\n@layout { a { @use ghost } }", nil)
	if res.Success {
		t.Fatal("expected failure for @use of an undefined block")
	}
	if len(res.Errors) == 0 || res.Errors[0].Type != arkitecture.ErrorReference {
		t.Fatalf("expected a reference error, got %+v", res.Errors)
	}
	if !strings.Contains(res.Errors[0].Message, "Layout block 'ghost' is not defined") {
		t.Errorf("unexpected message: %q", res.Errors[0].Message)
	}
}

func TestToSVGUnknownKindSucceeds(t *testing.T) {
	// An unknown kind is a semantic tag, not an error.
	res := arkitecture.ToSVG(`a { kind: database label: "DB" }`, nil)
	if !res.Success {
		t.Fatalf("unknown kind should render, got %+v", res.Errors)
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
