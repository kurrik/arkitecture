# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- **Document default margin** (2026-06-19): a bare `margin: N` at the root of an
  `@layout` sheet sets the document-wide default margin — the fallback spacing for
  every node that declares none, replacing the built-in 8. It's a single global
  baseline (not a cascade): nodes still override it directly, and there's no
  inheritance or selector specificity. Stored as `Document.DefaultMargin` and
  threaded into the generator's `marginOf`. The one knob for "space the whole
  diagram out" (the `pipeline` site sample now uses it). A `default-margin` golden
  locks it in. See the ADR in [decisions.md](decisions.md).
- **Group label bands** (2026-06-19): a parent with a label now reserves a strip
  for it — sized like a leaf box holding the label — instead of centring the label
  over (and obscuring) its children. A new `@layout` property `label: top | bottom`
  (default top) places the strip, and the box widens to fit the label. In a
  bordered parent the strip's inner edge is a wall the children's facing margin
  collapses against; a `box: none` parent reserves the same strip and packs its
  children flush below it. A new `group-label` golden locks both positions and the
  `box: none` case in; the four labelled-parent goldens and the `nesting`/`contexts`
  site samples were regenerated. See the ADR in [decisions.md](decisions.md).
- **M5 — `@layout` regrouping** (2026-06-18): an anonymous `@group { … }` inside a
  node's `@layout` block wraps sibling children into an invisible layout
  sub-container (its own `direction`/`size`/`margin`, no border, no path segment),
  and a node's children can be reordered by listing them. The arrangement is
  modelled on `Declarations` (a group is itself a nested `Declarations`) and is
  **direct-only** — never imported via `@use`/`kind`. The validator enforces
  same-parent (only the node's direct children, including those nested in groups)
  and completeness (each child referenced exactly once), keeping the layout tree a
  refinement of the semantic tree. A new `arrangement` golden locks it in. This
  completes the layered-authoring arc (split → reuse → regrouping). See the ADR in
  [decisions.md](decisions.md).
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
- **M4 — `@layout` reuse + `kind`** (2026-06-18): layout is now reusable.
  `@block <name> { decls }` (inside an `@layout` sheet) defines a parameterless,
  composable bundle; `@use <name>` (in a selector, inline block, or another
  block) imports it. Resolution gained a lower-precedence **imported** tier — the
  `kind` baseline first, then each `@use` in source order — under the existing
  **direct** tier, which overrides imports without conflict. A small built-in set
  ships (`invisible` → `box: none`); a user `@block` of the same name overrides a
  built-in. The validator flags an undefined `@use` block and `@use` cycles; an
  **unknown `kind` is a no-op**, not an error (it's a semantic tag, resolving the
  design open question — see the ADR in [decisions.md](decisions.md)). Generator
  unchanged; a new `kind-and-use` golden locks the behaviour in.
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

- (none) — `main` is Go and the full layered-authoring arc has shipped: M1 box
  model, M2 cardinal routing, M3 `@layout` split, M4 reuse + `kind`, and M5
  regrouping. The next work is from the lower-priority tracks below.

## Planned

The layered authoring model (M1–M5) is complete. What remains are the broader
tracks below; each is independently shippable. The *model* lives in
[design.md](design.md) and the *rationale* in [decisions.md](decisions.md).

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
