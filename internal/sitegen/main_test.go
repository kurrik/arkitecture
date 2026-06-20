package main

import (
	"fmt"
	"testing"
)

func fakeReader(files map[string]string) func(string) ([]byte, error) {
	return func(rel string) ([]byte, error) {
		s, ok := files[rel]
		if !ok {
			return nil, fmt.Errorf("missing %s", rel)
		}
		return []byte(s), nil
	}
}

func TestInjectReplacesMarkedBlock(t *testing.T) {
	read := fakeReader(map[string]string{
		// Trailing newline trimmed; &, <, > escaped; quotes left literal.
		"examples/x.ark": "a {\n  label: \"A & B\"\n}\nx --> y\n",
	})
	in := `<pre><code data-ark="examples/x.ark">STALE CONTENT</code></pre>`
	want := `<pre><code data-ark="examples/x.ark">a {
  label: "A &amp; B"
}
x --&gt; y</code></pre>`

	got, err := inject([]byte(in), read)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("inject =\n%q\nwant\n%q", got, want)
	}

	// Idempotent: a second pass over the result changes nothing.
	again, err := inject(got, read)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != want {
		t.Errorf("second pass changed output:\n%q", again)
	}
}

func TestInjectLeavesUnmarkedBlocksAlone(t *testing.T) {
	read := fakeReader(map[string]string{"examples/x.ark": "z {}\n"})
	in := `<pre><code>go run ./cmd/arkitecture in.ark out.svg</code></pre>` +
		`<pre><code data-ark="examples/x.ark">old</code></pre>`
	want := `<pre><code>go run ./cmd/arkitecture in.ark out.svg</code></pre>` +
		`<pre><code data-ark="examples/x.ark">z {}</code></pre>`

	got, err := inject([]byte(in), read)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("inject =\n%q\nwant\n%q", got, want)
	}
}

func TestInjectErrorsOnMissingArk(t *testing.T) {
	read := fakeReader(map[string]string{})
	in := `<pre><code data-ark="examples/missing.ark">x</code></pre>`
	if _, err := inject([]byte(in), read); err == nil {
		t.Fatal("expected an error for a missing .ark, got nil")
	}
}

func TestEscapeHTML(t *testing.T) {
	if got := escapeHTML(`a --> b & <c>`); got != `a --&gt; b &amp; &lt;c&gt;` {
		t.Errorf("escapeHTML = %q", got)
	}
}
