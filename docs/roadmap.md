# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- Adopt the project-template docs structure — `CLAUDE.md` + `docs/` (design,
  architecture, roadmap, decisions) — replacing the old `specs/` + `.claude`
  scaffolding (2026-06-17).
- Tokenizer + recursive-descent parser: container nodes, nested children,
  layout-only groups, `size`/`anchors`, and arrows.
- Validator: ID uniqueness, arrow + anchor reference resolution, range
  constraints, non-fail-fast error collection.
- Layout engine: bottom-up sizing with vertical/horizontal rules, `size`
  overrides, absolute anchor resolution; `string-width` text measurement.
- SVG generator: node rects + labels, groups as layout-only, straight arrows with
  arrowhead markers, exact-fit canvas.
- Main API (`arkitectureToSVG` + `parseArkitecture` / `validate` / `generateSVG`)
  and CLI with `--validate-only`, `--verbose`, `--watch`, font overrides, and exit
  codes.
- Golden-file suite and Jest unit tests across all stages; CI on Node 20/22 with
  coverage.

## In progress

- (none)

## Planned

### P0 — Restart housekeeping

- Audit the public API and CLI against this doc set; fix any drift between the code
  and the new `docs/`.
- Confirm the golden fixtures still match current output; regenerate if an
  intentional change has landed.

### P1 — Error & DX polish

- Richer diagnostics: stable error codes, fix suggestions, and consistent message
  formatting across stages.
- Expand integration + performance tests (large and deeply-nested documents).

### P2 — Layout & rendering reach

- Optional spacing/padding controls (currently fixed at zero).
- Arrow labels and/or non-straight routing.
- Basic styling hooks (fill, stroke, per-node font) without giving up deterministic
  output.

## Ideas / parking lot

Long-term, not yet scheduled:

- Additional output targets (PNG/PDF) via downstream conversion.
- A web playground that renders `.ark` live in the browser (the library already
  runs there).
- Library distribution polish: published types, usage examples, versioning policy.
