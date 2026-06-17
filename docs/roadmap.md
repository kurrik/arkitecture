# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- **Go rewrite at parity with TypeScript v0.1** — the full pipeline ported and
  building on one switchover branch (2026-06-17):
  - module + library-first layout (`ast`, root `arkitecture`, `cmd/arkitecture`,
    `wasm/`); tokenizer + recursive-descent parser
  - validator: scoped ID uniqueness, arrow/anchor reference resolution, range
    constraints (non-fail-fast)
  - generator: deterministic text measurement, bottom-up layout with `size`
    overrides + anchor resolution, byte-for-byte-stable SVG emission
  - CLI (parse/validate/generate + `--watch` via a stdlib polling watcher) and a
    `GOOS=js GOARCH=wasm` shim, both thin wrappers over the library
  - golden tests reproducing the TypeScript SVG/error fixtures exactly; Go CI
    (gofmt + vet + race tests + CLI/WASM builds) on Go 1.23/1.24
- Adopt the project-template docs structure — `CLAUDE.md` + `docs/` (2026-06-17).
- *(Historical)* TypeScript v0.1 — removed in the rewrite; in git history as the
  porting reference.

## In progress

- (none) — the rewrite PR is ready for review; merging it flips `main` to Go.

## Planned

### P1 — Distribution

- Publish portable binary builds (per-OS/arch release artifacts).
- Wire the `wasm/` build into a usable JS/TS package plus a small example.

### P2 — Error & DX polish

- Richer diagnostics: stable error codes, fix suggestions, consistent formatting.
- Attach source positions to validator errors (the AST carries none today, so
  they all report at line 1, column 1).
- Integration + performance tests (large and deeply-nested documents).

### P3 — Layout & rendering reach

- Optional spacing/padding controls (currently fixed at zero).
- Arrow labels and/or non-straight routing.
- Basic styling hooks (fill, stroke, per-node font) without giving up determinism.

## Ideas / parking lot

- Additional output targets (PNG/PDF) via downstream conversion.
- A web playground that renders `.ark` live in the browser (via the WASM build).
- Revisit text measurement if pixel-accurate fitting is ever needed (currently a
  rune-width approximation, not true font metrics).
