# Roadmap

Lightweight tracker of what's done, in progress, and planned. The source of truth
for "where is this project at?". Move items between sections as work progresses:
**Planned → In progress → Done**.

## Done

- **`direction` unified with the grid engine** (2026-06-22): `direction` is now
  formally sugar for a single-track grid — `vertical` ≡ `cols: 1`, `horizontal` ≡
  `rows: 1` — proven byte-for-byte for bordered *and* `box: none`, both axes (a
  generator test covers all four). `PlaceGrid` gained column-major (row-primary)
  auto-flow via a transpose, so `rows: M` / horizontal grids work; the default cell
  alignment now follows the box model (bordered stretches, `box: none` does not),
  uniformly with stacks. A node authored with `direction` is routed through the grid
  engine the moment any child opts into placement (`col`/`row`/spans), so stacks
  gain **sparse placement for free**. Dense stacks stay on the 1-D packing path
  (byte-identical, and it carries the orthogonal-route widening the grid engine does
  not yet apply). One golden changed: `grid-sequence` (a `box: none` grid) no longer
  stretches its cells. Step (b) of the consolidation. See the ADR in
  [decisions.md](decisions.md).
- **Margin-collapse box model in the grid engine** (2026-06-22): `generator/grid.go`
  now sizes a grid with the *same* box model as 1-D packing instead of a flat
  uniform gap — each inter-track channel is the collapsed (larger) facing margin of
  its adjacent children, a bordered grid reserves a perimeter from its edge
  children's margins, and a `box: none` grid carries its children's margins outward
  as its effective margin. A new generator test proves a `cols: 1` grid renders
  **byte-for-byte identical** to a `direction: vertical` stack (label band,
  perimeter, collapsed channels, cross-axis stretch all matched); the two existing
  grid goldens are unchanged (uniform margins collapse to the old gap). This is the
  engine groundwork for routing `direction` through the grid — step (a) of the
  consolidation. Cross-axis heterogeneous margins now collapse to a track perimeter
  (a grid keeps tracks aligned) — a deliberate, documented semantic. See the ADR in
  [decisions.md](decisions.md).
- **`@grid` block → `cols`/`rows` properties** (2026-06-22): the grid track
  definition is now two plain `@layout` properties (`cols`, optional `rows`)
  instead of an `@grid { … }` block — the only block-shaped layout property, now
  uniform with `margin`/`direction`/… (its per-child `col`/`row`/span placement
  were already plain properties). `ast.Declarations` carries `Cols *int`/`Rows
  *int` (a node is a grid when `cols` is set); `GridSpec` is kept as the internal
  `PlaceGrid` input. Behaviour-preserving — the two grid goldens render
  byte-identically — and the canonical field shape that `direction` will desugar
  into next. Second step of the direction/grid consolidation. See the ADR in
  [decisions.md](decisions.md).
- **Removed the `size` property** (2026-06-22): the `size: f` layout override —
  scaling a node's *orthogonal* dimension to a fraction of what its parent would
  give it — has been dropped from the language (parser, validator, resolver,
  generator, and `ast.Declarations.Size`). It was implicit and hard to reason
  about ("orthogonal of what, exactly?"); explicit per-node sizing controls will
  replace it later (see *Planned*). A breaking change: a `.ark` using `size:` now
  reports `Unknown layout property 'size'`. No golden fixture used it, so SVG
  output is unchanged. First step of the direction/grid consolidation below. See
  the ADR in [decisions.md](decisions.md).
- **`@grid` arrangement (2-D layout)** (2026-06-22): a third child-arrangement
  mode beside `direction`, declared `@grid { cols: N; rows: M? }` in a node's
  `@layout` (direct-only, like `@group`). Children place themselves with `col`/
  `row` + `colSpan`/`rowSpan` or auto-fill the next free slot (sparse, l→r/t→b);
  `cols` is fixed, `rows` grows implicitly. Tracks are sized jointly on both axes
  (single-track cells set column width / row height; a spanning cell distributes
  the shortfall) — the thing nested packing can't do — and each child aligns in
  its cell via `justify`/`align` (`start`/`end`/`stretch`, default stretch). A
  cell outside the declared bounds (including its span extent) or overlapping
  another is a constraint error. The pure `ast.PlaceGrid` is shared by the
  validator and the generator (`generator/grid.go`); two goldens lock it in
  (`grid`, `grid-sequence` — a sequence diagram with sparse, placeholder-free
  placement). Prototyped to make conventional sequence/table diagrams expressible
  without the hand-padded-placeholder hack. Follow-ups below. See the ADR in
  [decisions.md](decisions.md).
- **Per-element styling + consistent line weight** (2026-06-21): `@layout` gained a
  visual layer — `borderWidth`/`borderColor`/`backgroundColor` (a node's box) and
  `pathWidth`/`pathColor` (the arrows that *start* at a node), as hex colours and
  plain widths, settable per-node, document-wide (`Document.Defaults`, a bare style
  property at a sheet root), or in a `@block`. Resolution reuses the no-cascade
  two-tier merge, so `@use`/`kind`/document-default work unchanged; the generator's
  accessors fall back node → document default → built-in plain look, so an unstyled
  diagram is byte-identical. The tokenizer learned a hex-colour token (disambiguating
  the overloaded `#` by what precedes it), and a coloured arrow gets a colour-matched
  arrowhead (one `<marker>` per distinct path colour). Separately, axis-aligned
  strokes now render with `shape-rendering="crispEdges"` so 1px borders/orthogonal
  runs stay a consistent width regardless of sub-pixel position (diagonal arrows keep
  anti-aliasing). A new `styling` golden locks the look in; the existing goldens
  gained the one crispEdges attribute. Theming/cascade/fonts stay out of scope. See
  the two ADRs in [decisions.md](decisions.md).
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
  sub-container (its own `direction`/`margin`, no border, no path segment),
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
  *names*, children); all presentation — `direction`, `margin`, `box`,
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
  - generator: deterministic text measurement, bottom-up layout + anchor
    resolution, byte-for-byte-stable SVG emission
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

_Nothing in progress._

## Recently shipped — Auto edge routing (sized channels)

- **The full epic** (2026-06-20 → 2026-06-21, complete). The designed orthogonal
  routing shipped in reviewable slices (ADRs in [decisions.md](decisions.md)):
  - ✅ **Surface** — `route: straight | orthogonal` as a document-level setting at
    the `@layout` sheet root (mirrors `margin: N`), stored on `Document.Route`.
  - ✅ **Orthogonal emission (clear case)** — in `route: orthogonal` mode an arrow
    is drawn as an axis-aligned elbow/Z between its M2 cardinal endpoints (a
    `<polyline>`; a straight two-point path still emits `<line>`), used only when
    clear of the arrow's obstacles and otherwise falling back to the straight
    line. An `orthogonal-arrows` golden locks it in.
  - ✅ **Positioned anchors** — orthogonal routing also applies to arrows attached
    to explicit anchors: the path meets the box border on the facing side and a
    tail enters the node to reach an interior anchor (an edge anchor's tail is
    zero-length). An `orthogonal-anchors` golden locks the edge and interior cases.
  - ✅ **Break out to the container channel** — when an endpoint is nested, the
    orthogonal gap-crossing runs between the endpoints' breakout containers (the
    outermost box holding one but not the other), so the run lands in the channel
    *between* containers instead of along a container border. An
    `orthogonal-breakout` golden locks it. (Routing *around* obstacles within that
    breakout is still below.)
  - ✅ **Edge-normal exits & channel-following** — an explicit anchor pinned to one
    of its box's own edges now leaves/enters **perpendicular to that edge** (a
    bottom anchor drops straight down), and the channel router follows the corridor
    it lands in and hops out. A non-facing edge anchor exits via a stub in the leaf
    channel with its own box walled off (so the route detours around it, not back
    through it); a facing-side edge anchor is met in-channel so it's approached
    along the normal. Only `orthogonal-anchors` moved (`source#out` now drops
    down). Residual: a *bare-reference* target can still be approached askew (pin
    it to an edge for the clean normal entry).
  - ✅ **Route around obstacles** — the channel-graph router (`generator/channel.go`,
    `routeAround`): a sparse per-arrow grid whose lanes sit one inset outside each
    obstacle edge (channel centrelines in the uniform case), navigated by
    deterministic few-bend A* over the arrow-relative obstacle set. A blocked elbow
    now detours instead of falling back; the elbow stays the fast path, so no
    existing golden moved. An `orthogonal-around` golden locks the up-and-over
    route. (Routing runs on the computed layout — sound until widening moves boxes.)
  - ✅ **Channel widening (gaps, rails, multi-lane)** — an arrow running along a
    channel reserves a lane there: a two-pass layout routes once, attributes each
    run to its container channel (clipping by container → exact lane counts), widens
    it, lays out again, and snaps runs to their lanes. Each lane sits a **full
    margin** clear of the boxes (a wall in its own channel, not inside the node
    margin), half a margin between lanes. Covers **main-axis gaps**, **cross-axis
    rails** (a run *along* a perimeter), and **multi-lane** distribution (co-routed
    arrows spread to distinct lanes, so they never overlap); bare references
    leave/enter through the channel (not along the box edge), so a detour follows
    the gaps. Threaded into `calcDimensions`/`positionNodes` (`gapExtra`/`railExtra`),
    deterministic, default-zero (un-widened docs byte-identical). The five
    orthogonal widening goldens lock it in. Parked refinements: crossing-minimised
    lane ordering, and a per-arrow `route:` override (or configurable lane spacing).

## Planned

The layered authoring model (M1–M5) is complete. What remains are the broader
tracks below; each is independently shippable. The *model* lives in
[design.md](design.md) and the *rationale* in [decisions.md](decisions.md).

Auto edge routing — sized channels has **shipped** (see *Recently shipped* above);
its only parked follow-ups are crossing-minimised lane ordering and a per-arrow
`route:` override.

### Unify `direction` and `@grid` into one arrangement engine

The agreed direction (2026-06-22): `direction: vertical | horizontal` are just
degenerate grids — a single column (rows grow) and a single row (cols grow). The
goal is **one arrangement engine** so a stack and a grid share placement
vocabulary, and direction'd nodes gain sparse `col`/`row` placement, spans, and
`justify`/`align` "for free". Concretely:

- ✅ **Dissolve the `@grid { … }` block** into plain `cols`/`rows` `@layout`
  properties — *done* (see *Done* above); `ast.Declarations` now carries
  `Cols`/`Rows`.
- **`direction` becomes sugar** for `cols: 1` / `rows: 1`, kept as the readable
  everyday spelling.
- ✅ **(a) Margin-collapse box model in the grid engine** — *done* (see *Done*
  above). The grid engine now uses collapsing channels, perimeters from edge
  children's margins, and `box: none` propagation, with a test proving `cols: 1`
  ≡ a vertical stack byte-for-byte. Remaining sub-steps:
- ✅ **(b) `direction` unified with the grid engine** — *done* (see *Done*
  above). `direction` is sugar for `cols: 1` / `rows: 1`; `PlaceGrid` auto-flows
  column-major (via a transpose) for the horizontal case; the cross-alignment
  default follows the box model. Stacks gain sparse placement, routed to the grid
  engine when a child opts in.
- **(c) Delete the 1-D path + 2-D channel widening.** Dense stacks still take the
  1-D `calcDimensions`/`positionNodes` packing path, kept *only* because it carries
  the orthogonal-route channel widening the grid engine cannot yet apply:
  `widen.go`'s `gapIndexAt`/`railSideAt`/`childrenCrossBand` assume children in a
  single line along one axis. Generalising the channel model from 1-D to grid
  tracks lets the grid engine widen too — at which point the 1-D path can be
  deleted and there is a single engine. Until then, `route: orthogonal` through a
  multi-track grid is unsupported. Stage carefully with golden review.
- **Sparse needs spacer tracks.** Sparse 1-D placement only yields *visible* gaps
  once empty tracks have a size — see *Min-size / spacer tracks* below; pair the
  two or "sparse" is a no-op in a stack.

Explicit **per-node sizing controls** (the replacement for the removed `size`)
land on top of this unified model rather than the old orthogonal-fraction hack.

### `@grid` follow-ups

- **Min-size / spacer tracks:** an empty grid track currently collapses to zero —
  add an optional minimum (or a `gap`/track-size knob) so an empty row can act as
  deliberate vertical spacing.
- **`stretch` on a container child:** stretch resizes a cell child *after* its own
  subtree was laid out, so a stretched *container* can misalign its interior
  (leaves are unaffected) — re-layout the subtree at the stretched size.
- **Dedicated `gap`:** the inter-track gap reuses the grid node's `margin`; a
  first-class `gap` (and possibly per-axis) would separate the two.
- **Grid + `@group`:** a grid currently ignores any `@group` arrangement on the
  same node — decide whether a group may occupy a cell.

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
