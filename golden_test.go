package arkitecture_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kurrik/arkitecture"
)

var update = flag.Bool("update", false, "regenerate golden .svg files")

// goldenOptions mirrors the per-fixture <name>.json options.
type goldenOptions struct {
	FontSize   int    `json:"fontSize"`
	FontFamily string `json:"fontFamily"`
}

// goldenError mirrors the expected first diagnostic in a <name>.error file.
type goldenError struct {
	Type            string `json:"type"`
	Line            int    `json:"line"`
	Column          int    `json:"column"`
	MessageContains string `json:"messageContains"`
}

// TestGolden renders every generator/testdata/golden/*.ark fixture through the
// full pipeline and diffs against the checked-in .svg (success) or .error
// (failure) reference. Run with -update to regenerate the .svg files after an
// intentional output change, then review the diff.
func TestGolden(t *testing.T) {
	dir := filepath.Join("generator", "testdata", "golden")
	arks, err := filepath.Glob(filepath.Join(dir, "*.ark"))
	if err != nil {
		t.Fatal(err)
	}
	if len(arks) == 0 {
		t.Fatalf("no golden fixtures found in %s", dir)
	}

	for _, ark := range arks {
		base := strings.TrimSuffix(ark, ".ark")
		name := filepath.Base(base)
		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile(ark)
			if err != nil {
				t.Fatal(err)
			}

			opts := &arkitecture.Options{}
			if data, err := os.ReadFile(base + ".json"); err == nil {
				var o goldenOptions
				if err := json.Unmarshal(data, &o); err != nil {
					t.Fatalf("parsing %s.json: %v", name, err)
				}
				opts.FontSize, opts.FontFamily = o.FontSize, o.FontFamily
			}

			res := arkitecture.ToSVG(string(input), opts)

			if errPath := base + ".error"; fileExists(errPath) {
				assertErrorFixture(t, errPath, res)
				return
			}
			assertSVGFixture(t, base+".svg", res)
		})
	}
}

func assertErrorFixture(t *testing.T, path string, res arkitecture.Result) {
	t.Helper()
	if res.Success {
		t.Fatalf("expected failure, got success with SVG:\n%s", res.SVG)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want goldenError
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	got := res.Errors[0]
	if string(got.Type) != want.Type || got.Line != want.Line || got.Column != want.Column ||
		!strings.Contains(got.Message, want.MessageContains) {
		t.Errorf("first error = %+v,\nwant type=%s line=%d column=%d message~=%q",
			got, want.Type, want.Line, want.Column, want.MessageContains)
	}
}

func assertSVGFixture(t *testing.T, path string, res arkitecture.Result) {
	t.Helper()
	if !res.Success {
		t.Fatalf("expected success, got errors: %+v", res.Errors)
	}
	if *update {
		if err := os.WriteFile(path, []byte(res.SVG), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s (run with -update to create): %v", path, err)
	}
	if res.SVG != string(want) {
		t.Errorf("SVG mismatch.\n--- got ---\n%s\n--- want ---\n%s", res.SVG, string(want))
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
