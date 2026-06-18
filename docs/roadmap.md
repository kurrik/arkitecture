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
- Design captured for the layered `@layout` model, the margin box model, and
  auto-cardinal arrow routing — see [design.md](design.md) and the ADRs in
  [decisions.md](decisions.md) (2026-06-17).
- *(Historical)* TypeScript v0.1 — removed in the rewrite; in git history as the
  porting reference.

## In progress

- (none) — `main` is Go; the model below is designed and ready to implement.

## Planned

The near-term arc is the box-model work plus the layered authoring model. The
detail below is meant to be implementation-ready; the *model* lives in
[design.md](design.md) and the *rationale* in [decisions.md](decisions.md). Each
milestone is independently shippable.

### Order & dependencies

```
M1 box model + margins ─┬─▶ M2 cardinal routing
                        └─▶ M3 @layout split ─▶ M4 reuse + kind ─▶ M5 regrouping
```

M2 depends on M1 (edge-attached arrows need the margin gap, or they collapse to
zero length between flush-packed boxes). M3–M5 are the authoring epic (the earlier
"Phase 1/2/3"); `margin`/`box` are declared inline in M1 and *move into* `@layout`
at M3 — parser rework, but no geometry rework.

### M1 — Box model + margins *(foundational layout refactor)*

Goal: a real border-box / margin-box model so nodes can reserve space around
themselves, behaving correctly inside invisible vs bordered parents.

- **AST / properties:** add `margin` (uniform value, v1) and `box: none|default`.
  For now accept them in the current inline property syntax (`parser.parseProperty`);
  they relocate to `@layout` at M3.
- **generator/layout (the refactor):**
  - Track a node's **border box** (visible rect = content + 1px border, or content
    only when `box: none`) distinct from its **margin box** (border box + margins).
  - Sizing: a **bordered** parent's content = bounding box of children's **margin**
    boxes (outer margins count, like padding). An **invisible** (`box: none`)
    parent's content = children's **border** boxes + the inter-sibling gaps only;
    perimeter margins collapse to zero. The document root is invisible → no phantom
    canvas padding.
  - Positioning: offset children by their margins; lay out margin boxes.
  - Canvas: bounds of the top-level border boxes plus inter-sibling gaps.
- **anchors:** keep anchor positions on the **border box** (unchanged math; margin
  is outside it).
- **tests:** unit cases for {bordered, invisible} × {leaf, parent}, margin
  collapse, and nested invisible-in-bordered (the user's edge case). **Regenerate
  every golden** (positions shift once margins exist) and review the diff.
- **open:** adjacent-sibling collapse (sum vs max); per-side margins vs uniform;
  default margin (0 = today's look, >0 changes all output and is what M2 wants).

### M2 — Cardinal arrow routing

Goal: anchor-less arrows attach to the nearest cardinal edge instead of
centre-to-centre.

- **generator:** expose a `map[fullPath]box` from layout (alongside the existing
  anchor positions). For an arrow endpoint with no `#anchor`, attach to the
  N/E/S/W side of its border box facing the other node's centre (dominant axis of
  the centre-to-centre vector; ties favour horizontal). Named anchors and explicit
  `#anchor` unchanged; an explicit anchor always wins.
- **parser:** distinguish "no anchor" from explicit `#center` so the latter can
  still force centre-to-centre.
- **depends on M1:** without a margin gap, the two cardinal endpoints coincide on
  the shared edge → zero-length arrow.
- **tests + golden:** new arrow coordinates; add fixtures for each direction and
  the explicit-override path; regenerate and review.
- **open:** 4 vs 8 directions; unpositioned *named* anchor → centre or error.

### M3 — `@layout`, the split *(authoring epic, phase 1)*

Goal: author semantics and presentation as separate layers.

- **tokenizer:** recognise `@`-directives (`@layout`; later `@block`/`@use`/
  `@group`) — a new token or keyword handling.
- **ast:** split into a **semantic** tree (`ContainerNode`: `id`, `label`, `kind`,
  anchor **names**, children) and a **layout** model (selector → declaration list;
  declarations: `direction`, `size`, `box`, `margin`, `anchor <name>: [x,y]`).
  Anchors on the node become a name set; positions live in layout. Arrows stay
  string refs.
- **parser:** `@layout { selector { … } }` standalone and `@layout { … }` inline in
  a node body; exact dotted-path selectors; the declaration grammar; `kind: name`;
  anchor-name declarations (`anchors: [db, north]`). Decide the fate of today's
  inline `size`/`direction`/`anchors:{pos}` (drop, or keep as shorthand).
- **resolve stage (new, pure):** merge layout onto the semantic tree by exact path
  into a resolved per-node layout, with the two precedence tiers (imported <
  direct). Pipeline becomes parse → resolve → validate → generate (or validation
  spans both).
- **validator:** dangling selector (path matches no node); duplicate **direct**
  property on a node (conflict error); arrow anchor-name resolution against
  declared names; range checks for `size`/`margin` move here.
- **generator:** read the resolved layout (M1 geometry unchanged).
- **tests + golden:** rewrite fixtures in the new syntax; equivalent inputs must
  produce identical SVG (the split is structural, not visual).

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
  package around the `wasm/` build, with an example.
- **Diagnostics & DX:** stable error codes; source positions on validator errors
  (the AST carries none today); large/deep-document performance tests.
- **Rendering reach beyond the above:** arrow labels; non-straight routing; visual
  styling (fill/stroke/per-node font) layered on the `@layout`/`kind` machinery.

## Ideas / parking lot

- Multiple layout sheets / themes over one semantic model, and a cross-file
  `@import` for layout (the payoff that motivates the `@layout` epic).
- Migrating an arrow's *choice* of anchor into the layout layer (routing).
- Additional output targets (PNG/PDF) via downstream conversion.
- A web playground that renders `.ark` live in the browser (via the WASM build).
- Revisit text measurement if pixel-accurate fitting is ever needed.
