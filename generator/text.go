package generator

import (
	"math"
	"strings"
	"unicode"
)

// Text measurement estimates label box sizes before rendering. It mirrors the
// pre-rewrite TypeScript, which approximated width as the display-cell count
// times fontSize times 0.6, and height as the line count times fontSize times
// 1.2 — a deterministic, font-metric-free estimate that works headless and in
// WASM.

const (
	avgCharWidthRatio = 0.6
	lineHeightRatio   = 1.2
)

// textWidth returns the rounded pixel width of the widest line of text.
func textWidth(text string, fontSize float64) float64 {
	if text == "" {
		return 0
	}
	max := 0.0
	for _, line := range strings.Split(text, "\n") {
		if w := float64(displayWidth(line)) * fontSize * avgCharWidthRatio; w > max {
			max = w
		}
	}
	return math.Round(max)
}

// textHeight returns the rounded pixel height of the text block.
func textHeight(text string, fontSize float64) float64 {
	if text == "" {
		return 0
	}
	lines := strings.Count(text, "\n") + 1
	return math.Round(float64(lines) * fontSize * lineHeightRatio)
}

// displayWidth returns the number of terminal cells a string occupies, a
// dependency-free approximation of the npm `string-width` package: ASCII and
// most scripts are one cell, East-Asian wide / emoji runes are two, and
// combining/format marks are zero.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func runeWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) {
		return 0 // combining marks and zero-width format characters
	}
	if isWide(r) {
		return 2
	}
	return 1
}

func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0x303E, // CJK radicals, Kangxi
		r >= 0x3041 && r <= 0x33FF, // Hiragana .. CJK compatibility
		r >= 0x3400 && r <= 0x4DBF, // CJK Extension A
		r >= 0x4E00 && r <= 0x9FFF, // CJK Unified Ideographs
		r >= 0xA000 && r <= 0xA4CF, // Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul Syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK Compatibility Ideographs
		r >= 0xFE30 && r <= 0xFE4F, // CJK Compatibility Forms
		r >= 0xFF00 && r <= 0xFF60, // Fullwidth Forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1FAFF, // emoji and symbols
		r >= 0x20000 && r <= 0x3FFFD: // CJK Extension B and beyond
		return true
	}
	return false
}
