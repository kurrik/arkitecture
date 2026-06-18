# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- **Generated docs site + live WASM playground** (2026-06-18): the GitHub Pages
  publish is now a generation step (`scripts/build-site.sh`, run by `pages.yml`)
  that re-renders every `site/examples/*.ark` to `.svg` via the CLI and builds
  `site/arkitecture.wasm` (+ `wasm_exec.js`) before upload, so a publish always
  reflects the current library. The Examples page is progressively enhanced
  (`site/playground.js`): with JS on, each example's source becomes an editable
  textarea that re-renders live in the browser through the `wasm/` build, with a
  Reset control and inline compile errors; with JS off it is byte-for-byte the
  old static page. Artifacts stay un-committed (`.wasm` git-ignored, `wasm_exec.js`
  added to `.gitignore`). See the ADR in [decisions.md](decisions.md).
- **M3 — `@layout`, the split** (2026-06-18): semantics and presentation are now
  separate layers. A node body holds only semantics (`label`, `kind`, anchor
  *names*, children); all presentation — `direction`, `size`, `margin`, `box`,
  anchor *positions* — moves into `@layout` blocks, either inline on a node or as
  a standalone sheet of exact-path selectors. The tokenizer recognises `@`
  directives; the `group` keyword is gone (a `box: none` node replaces it and now
  carries a path segment); a new pure **resolve** stage merges layout onto the
  tree by path; the validator gained dangling-selector, duplicate-direct-property,
  anchor-name, and (relocated) range checks. The split is structural, not visual:
  every golden and site SVG renders byte-for-byte identically from the rewritten
  fixtures. Inline layout shorthand was dropped (no bare `direction:` on a node) —
  see the ADR in [decisions.md](decisions.md).
- **M2 — Cardinal arrow routing** (2026-06-18): anchorless arrows (`a --> b`) now
  attach to the nearest cardinal edge (N/E/S/W) of each box facing the other
  node's centre, instead of cutting centre-to-centre. Naming an anchor — including
  the implicit `#center` — pins a fixed point and always wins. Generator-only (no
  parser change was needed); builds on M1's margin gap. See the ADR in
  [decisions.md](decisions.md).
- **M1 — Box model + margins** (2026-06-18): a real border-box / margin-box
  layout model. Every node takes a uniform `margin` (default 8; `margin: 0`
  packs flush as before) and a `box: none|default` to drop its border. Bordered
  parents inset children like padding; invisible parents (`box: none`, groups,
  the document root) collapse perimeter margins and keep only the inter-sibling
  gaps. Anchors stay on the border box. Inline properties for now — they move
  into `@layout` at M3. All goldens and the site sample SVGs were regenerated;
  see the ADR in [decisions.md](decisions.md).
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
- Design captured for the layered `@layout` model, the margin box model, and
  auto-cardinal arrow routing — see [design.md](design.md) and the ADRs in
  [decisions.md](decisions.md) (2026-06-17).
- *(Historical)* TypeScript v0.1 — removed in the rewrite; in git history as the
  porting reference.

## In progress

- (none) — `main` is Go; the layout foundation (M1 box model, M2 cardinal
  routing) and the M3 `@layout` split have shipped, so M4 (reuse + `kind`) is
  unblocked and implementation-ready.

## Planned

The near-term arc is the layered authoring model; M3 (`@layout` split) is done,
leaving M4–M5. The detail below is meant to be implementation-ready; the *model*
lives in [design.md](design.md) and the *rationale* in
[decisions.md](decisions.md). Each milestone is independently shippable.

### Order & dependencies

```
M1 box model + margins (done) ──▶ M2 cardinal routing (done)
M3 @layout split (done) ─▶ M4 reuse + kind ─▶ M5 regrouping
```

M4–M5 build on the resolve stage and `@layout` grammar M3 introduced: M3 collapsed
`group` into a `box: none` node and moved `direction`/`size`/`margin`/`box`/anchor
positions into `@layout`. The `kind` property already parses (it records a name but
applies nothing) — M4 makes it hook a layout block.

### M4 — `@layout` reuse + `kind` *(phase 2)*

- **parser:** `@block name { decls }` and `@use name`.
- **resolve:** expand `@use` in place (source order, last-write-wins); `kind`
  expands to an implicit lowest-precedence `@use <kind>`; detect `@use` cycles.
- **built-in kinds:** ship a small set (`invisible` → `box: none`); any kind is
  redeclarable via `@block`.
- **validator:** undefined block/kind; cycles.
- **tests:** override precedence (direct beats imported), composition, cycle error,
  unknown-kind handling.

### M5 — `@layout` regrouping *(phase 3)*

- **parser:** anonymous `@group { … }` inside a node's arrangement, listing child
  ids and nested groups.
- **ast/resolve:** an arrangement tree per node (ordered children + anonymous group
  wrappers); groups arrange like invisible sub-containers (reuse M1's invisible
  box).
- **validator:** same-parent (only direct children of the enclosing node) and
  completeness (each child referenced exactly once).
- **generator:** lay out via the arrangement tree.
- **tests:** nesting, plus foreign/duplicate/missing-child errors.

### Other tracks (lower priority)

- **Distribution:** portable binary release builds (per-OS/arch); a usable JS/TS
  package around the `wasm/` build, with an example. *(The docs site is now a
  first consumer of the `wasm/` build — see the live Examples playground — but a
  published, versioned package is still open.)*
- **Diagnostics & DX:** stable error codes; source positions on validator errors
  (the AST carries none today); large/deep-document performance tests.
- **Rendering reach beyond the above:** arrow labels; non-straight routing; visual
  styling (fill/stroke/per-node font) layered on the `@layout`/`kind` machinery.

## Ideas / parking lot

- Multiple layout sheets / themes over one semantic model, and a cross-file
  `@import` for layout (the payoff that motivates the `@layout` epic).
- Migrating an arrow's *choice* of anchor into the layout layer (routing).
- Additional output targets (PNG/PDF) via downstream conversion.
- A fuller standalone web playground (the Examples page now does per-example live
  editing via the WASM build; a dedicated playground/share-URL page is the bigger
  version still parked here).
- Revisit text measurement if pixel-accurate fitting is ever needed.
