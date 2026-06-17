package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeArk(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "diagram.ark")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp .ark: %v", err)
	}
	return p
}

func TestRunVersion(t *testing.T) {
	if code := run([]string{"--version"}); code != 0 {
		t.Errorf("run(--version) = %d, want 0", code)
	}
}

func TestRunMissingInput(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Errorf("run() with no input = %d, want 2", code)
	}
}

func TestRunFileNotFound(t *testing.T) {
	if code := run([]string{"--validate-only", "/no/such/file.ark"}); code != 2 {
		t.Errorf("run on missing file = %d, want 2", code)
	}
}

// Flags must work whether they come before or after the positional argument.
func TestRunValidateOnlyInterspersed(t *testing.T) {
	p := writeArk(t, `a { label: "x" }`)
	if code := run([]string{p, "--validate-only"}); code != 0 {
		t.Errorf("run(file --validate-only) = %d, want 0", code)
	}
	if code := run([]string{"--validate-only", p}); code != 0 {
		t.Errorf("run(--validate-only file) = %d, want 0", code)
	}
}

func TestRunValidationErrorExit1(t *testing.T) {
	p := writeArk(t, "a {")
	if code := run([]string{"--validate-only", p}); code != 1 {
		t.Errorf("run on syntactically invalid input = %d, want 1", code)
	}
}
