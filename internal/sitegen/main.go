// Command sitegen keeps the docs site's shown example source in sync with the
// .ark files that actually render the diagrams, so there is a single canonical
// source per example. It replaces the body of every
//
//	<code … data-ark="PATH"> … </code>
//
// in site/*.html with the HTML-escaped contents of site/PATH (the .ark file).
// It is idempotent — re-running with no .ark change rewrites nothing — and is
// run by scripts/build-site.sh alongside the example SVG rendering, so a publish
// always reflects the current sources. Author examples by editing the .ark; the
// HTML blocks are generated.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// codeBlock matches a marked example source block. Group 1 is the opening tag
// (preserved verbatim), group 2 the .ark path relative to the site root, group 3
// the closing tag. The body in between is replaced.
var codeBlock = regexp.MustCompile(`(?s)(<code\b[^>]*\bdata-ark="([^"]+)"[^>]*>).*?(</code>)`)

func main() {
	root := "site"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	pages, err := filepath.Glob(filepath.Join(root, "*.html"))
	if err != nil {
		fail(err)
	}
	read := func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(root, rel)) }
	for _, page := range pages {
		src, err := os.ReadFile(page)
		if err != nil {
			fail(err)
		}
		out, err := inject(src, read)
		if err != nil {
			fail(fmt.Errorf("%s: %w", page, err))
		}
		if string(out) == string(src) {
			continue
		}
		if err := os.WriteFile(page, out, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("  updated %s\n", page)
	}
}

// inject replaces the body of every data-ark code block in htmlSrc with the
// HTML-escaped contents of the referenced .ark (fetched via read). The .ark's
// trailing newline is dropped so the block reads like the others (no blank line
// before </code>). A missing referenced file is an error.
func inject(htmlSrc []byte, read func(arkRel string) ([]byte, error)) ([]byte, error) {
	var firstErr error
	out := codeBlock.ReplaceAllFunc(htmlSrc, func(match []byte) []byte {
		g := codeBlock.FindSubmatch(match)
		openTag, arkRel, closeTag := g[1], string(g[2]), g[3]
		content, err := read(arkRel)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("data-ark %q: %w", arkRel, err)
			}
			return match
		}
		body := escapeHTML(strings.TrimRight(string(content), "\n"))
		return []byte(string(openTag) + body + string(closeTag))
	})
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

// escapeHTML escapes the three characters that matter in element text content,
// matching the site's hand-written style (quotes are left literal).
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "sitegen:", err)
	os.Exit(1)
}
