# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- **Go rewrite at parity with TypeScript v0.1** — full pipeline ported; merged to
  `main` ([#2](https://github.com/kurrik/arkitecture/pull/2), 2026-06-17):
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

- (none) — `main` is now Go. The `@layout` model below is designed (see
  [design.md](design.md) + the ADR in [decisions.md](decisions.md)); ready to
  implement.

## Planned

### Layered semantics/layout model (`@layout`) — next major epic

Separate semantic structure from presentation so layout can be retuned, reused,
and swapped without touching semantics, while staying deterministic (exact-path
selectors, no cascade). The full model is in *Semantic vs. layout* in
[design.md](design.md), with rationale in [decisions.md](decisions.md). Built in
phases, each independently useful and shippable:

- **Phase 1 — the split.** Introduce `@layout` (inline block + exact-path selector
  sheet), one node type with `box: none`, and the anchor name/position split. Move
  `direction`/`size`/anchor-positions into the layout layer; add a **resolve**
  stage and selector/conflict validation. Delivers "edit layout without touching
  semantics".
- **Phase 2 — reuse.** `@block`/`@use` with last-write-wins override and cycle
  detection, plus the semantic `kind` property (implicit lowest-precedence
  `@use`) and a small set of built-in kinds (e.g. `invisible`).
- **Phase 3 — presentational regrouping.** Anonymous `@group` arrangements with
  the same-parent and completeness checks.

Sequencing note: each phase is a parser + AST + resolve/validator change; the
generator keeps consuming a resolved layout, so its core is largely untouched.

### Distribution

- Publish portable binary builds (per-OS/arch release artifacts).
- Wire the `wasm/` build into a usable JS/TS package plus a small example.

### Diagnostics & DX

- Richer diagnostics: stable error codes, fix suggestions, consistent formatting.
- Attach source positions to validator errors (the AST carries none today, so
  they all report at line 1, column 1).
- Integration + performance tests (large and deeply-nested documents).

### Rendering reach

- **Auto-cardinal arrow endpoints** for anchor-less arrows — a deterministic
  "nearest side facing the target" default (see the ADR). Separable from the
  `@layout` epic; could ship on the current pipeline as a near-term win.
- Optional spacing/padding controls (currently fixed at zero).
- Arrow labels and/or non-straight routing.
- Visual styling (fill, stroke, per-node font) — the natural payoff of `kind` and
  the `@layout` layer — without giving up determinism.

## Ideas / parking lot

- Multiple layout sheets / themes over one semantic model, and a cross-file
  `@import` for layout (the payoff that motivates the `@layout` epic).
- Migrating an arrow's *choice* of anchor into the layout layer (routing).
- Additional output targets (PNG/PDF) via downstream conversion.
- A web playground that renders `.ark` live in the browser (via the WASM build).
- Revisit text measurement if pixel-accurate fitting is ever needed.
