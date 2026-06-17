# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- Adopt the project-template docs structure — `CLAUDE.md` + `docs/` (2026-06-17).
- Go rewrite, part 1: module + library-first layout (`ast`, root `arkitecture`,
  `cmd/arkitecture`, `wasm/`), tokenizer + recursive-descent parser ported with
  tests, and Go CI (gofmt + vet + race tests + CLI/WASM builds) on Go 1.23/1.24
  (2026-06-17).
- *(Historical)* TypeScript v0.1 — full parser, validator, generator, and CLI.
  Removed in the Go rewrite; preserved in git history as the porting reference.

## In progress

- **Rewrite to Go (single switchover PR).** Port the remaining stages to reach
  parity with the removed TypeScript, then this one PR flips the project to Go.
  Done: tokenizer + parser. Remaining: validator, generator, CLI watch.

## Planned

Dependency-ordered. The goal is parity with the old TypeScript, then breadth.

### P0 — Reach parity (Go port)

- **Validator**: ID uniqueness within scope, arrow source/target + anchor
  reference resolution, range constraints; non-fail-fast.
- **Generator**: text measurement (a Go rune-width function replacing
  `string-width`), bottom-up layout with `size` overrides, absolute anchor
  resolution, and SVG emission.
- **Golden tests**: render `generator/testdata/golden/*.ark` and diff against the
  checked-in `.svg`/`.error` references, with a `-update` flag to regenerate.
- **CLI watch** (`--watch`): a debounced file-event loop (fsnotify or polling).

### P1 — Distribution

- Publish portable binary builds (per-OS/arch release artifacts).
- Wire the `wasm/` build into a usable JS/TS package plus a small example.

### P2 — Error & DX polish

- Richer diagnostics: stable error codes, fix suggestions, consistent formatting.
- Integration + performance tests (large and deeply-nested documents).

### P3 — Layout & rendering reach

- Optional spacing/padding controls (currently fixed at zero).
- Arrow labels and/or non-straight routing.
- Basic styling hooks (fill, stroke, per-node font) without giving up determinism.

## Ideas / parking lot

- Additional output targets (PNG/PDF) via downstream conversion.
- A web playground that renders `.ark` live in the browser (via the WASM build).
