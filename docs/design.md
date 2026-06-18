# Design

What Arkitecture is, who it's for, and the workflow it supports. Update when the
high-level concept shifts.

## Concept

Arkitecture is a domain-specific language (DSL) and toolchain that turns a small
text file into an SVG architecture diagram. It targets *high-level* diagrams —
Domain-Driven Design bounded contexts, service boundaries, system-of-systems
overviews — where the author cares more about a clean, legible layout than about
capturing every implementation detail.

Its reason to exist is a deliberate rejection of automatic layout. Graphviz,
Mermaid, and friends decide where boxes go; the result is often "good enough" but
rarely exactly what you want, and you can't fix it without fighting the
algorithm. Arkitecture inverts that: the author describes the *structure* (what
contains what, what points at what) and controls the *layout* directly through
nesting, direction, and explicit sizing. The tool only measures text and packs
boxes — it never moves anything you didn't ask it to.

**Core principle — manual, deterministic layout.** The same input always produces
the same diagram, and every position is a consequence of a rule visible in the
source. There is no hidden layout engine to second-guess.

The design separates *semantic structure* from *presentation* into two authored
layers — modelled on HTML/CSS, but deliberately **without the cascade** — so
layout can be retuned, reused, or swapped without touching semantics while every
position stays traceable to a local rule. See *Semantic vs. layout* below.

## Target user & goals

Target user: an engineer or architect sketching system structure who wants the
diagram to look exactly the way they laid it out and to live in version control
as text next to the code it documents. Overriding goal: predictable, controllable
output — when a design call is between "more automatic" and "more controllable",
choose controllable.

## Vocabulary

- **DSL** — the `.ark` text format describing a diagram. Parsed into an AST; never
  hand-edited as SVG/XML.
- **Document** — a whole `.ark` file: a set of top-level nodes plus the arrows
  between them.
- **(Container) node** — a labelled box with an ID. The primary building block;
  may contain child nodes and groups. Sizes to its contents.
- **Group** — the `group` *keyword* was removed in M3 (a borderless **node** is
  now `box: none`, keeping an ID and a path segment). The presentational
  **`@group`** (M5) is its layout-layer successor: an anonymous, invisible wrapper
  inside a node's arrangement, with no ID and no path segment. See *Presentational
  regrouping* below.
- **Direction** — `vertical` or `horizontal`: how a node stacks its children, set
  in `@layout`. Defaults to `vertical`.
- **Anchor** — a named point on a node in relative `[x, y]` coordinates (`[0, 0]`
  top-left … `[1, 1]` bottom-right), used as an arrow endpoint. Every node has an
  implicit centre anchor at `[0.5, 0.5]`.
- **Arrow** — a directed connection `source --> target`, where each end is a
  (possibly dotted) node path with an optional `#anchor`.
- **Size** — an override in `[0, 1]` for a node's *orthogonal* dimension, as a
  fraction of what the parent would otherwise give it.
- **Semantic vs. layout layers** *(M3)* — structure (`id`, `label`, `kind`,
  anchor *names*, nesting, arrows) is authored separately from presentation
  (`@layout`: `direction`, `size`, anchor *positions*, `margin`, `box`, child
  arrangement). See the section below.
- **Kind** *(M4)* — an arbitrary semantic classification on a node
  (`kind: database`) that implicitly applies the layout block of the same name
  (`@use database`) as an overridable style baseline. An unknown kind (no
  matching block) is a harmless no-op, since `kind` is a semantic tag.
- **Block / use** *(M4)* — `@block <name> { decls }` defines a reusable,
  parameterless layout bundle inside an `@layout` sheet; `@use <name>` imports it
  into a node (or another block). Imports are explicit and opt-in — no cascade.
- **Margin** — layout space reserved around a node's border box (uniform,
  default 8; `margin: 0` packs flush). Counts inside a bordered parent (like
  padding) but collapses against an invisible (`box: none`) parent. See *Box
  model & margins*.
- **Box / border box / margin box** — `box: none` makes a node draw no border
  (an ID-bearing twin of the layout-only group); the **border box** is the
  visible rectangle and the **margin box** is the border box plus its margins.

## The workflow

1. Write a `.ark` file describing the diagram's nodes, nesting, and arrows.
2. Run `arkitecture diagram.ark diagram.svg` (the output name defaults to the
   input with a `.svg` extension).
3. Read the reported errors — syntax, bad references, out-of-range values — all at
   once, with line/column positions.
4. Adjust `direction`, nesting, and `size` to shape the layout; add `anchors` to
   steer arrows.
5. Re-run, or use `--watch` to regenerate on every save.
6. Commit the `.ark` source alongside the code it documents.

See [examples/annotated.ark](../examples/annotated.ark) for a fully commented
reference covering every feature.

## Layout model

Layout is bottom-up and deterministic:

- A leaf node sizes to its label text (measured with a fixed font).
- A **vertical** parent stacks children top-to-bottom: its width is the widest
  child; children span the full width unless they set `size`.
- A **horizontal** parent places children left-to-right: its height is the tallest
  child; children span the full height unless they set `size`.
- `size: f` scales only the orthogonal dimension to a fraction `f` of the parent;
  it does not affect the parent's own size.
- Each node reserves a uniform **`margin`** (default 8) around its border box.
  Margins **collapse** rather than stack: the channel between two adjacent
  siblings is the *larger* of their facing margins (not the sum), and a child's
  gap to its parent's wall is its own margin — so every channel is one uniform
  margin wide. `margin: 0` restores flush packing.
- A **box: none** group is *transparent*: it draws no border and adds no wall of
  its own, but it is not a barrier to margins. Its children's perimeter margins
  push straight through it to the nearest **bordered** ancestor (where they
  become padding inside that border). Only when there is no bordered ancestor —
  an invisible chain up to the document root, e.g. a top-level group — do those
  perimeter margins collapse to nothing, so the canvas gains no outer padding.
- `box: none` turns a node into such an invisible grouping while keeping its ID,
  label, and anchors. Its **effective margin** — the channel a parent reserves
  around it — is the *larger* of its own margin and its children's, so it never
  doubles a channel; `margin: 0` makes it contribute exactly its children's
  margins.
- The canvas fits the top-level content exactly: the document root is invisible,
  so the diagram gains no outer padding even though inner nodes are spaced.

## Semantic vs. layout (the `@layout` model)

> ✅ The full `@layout` model has shipped: the **split** (two-layer authoring,
> `@layout` blocks, exact-path selectors, no-cascade resolution) in M3, **reuse**
> (`@block`/`@use` and `kind` hooking a layout block) in M4, and presentational
> **`@group` regrouping** in M5. In M3 the inline `size`/`direction`/`anchors:{pos}`
> shorthand was **dropped**: a node body is purely semantic and all presentation
> lives in `@layout`.

The diagram is authored in two layers, like HTML + CSS — but **without CSS's
cascade**, because that cascade is exactly the action-at-a-distance the core
principle rejects.

- **Semantic layer** — *what* the components are and how they relate: a node's
  `id`, optional `label`, its `kind`, the named anchors it exposes, containment
  (nesting = "is part of"), and arrows (relations). The stable part.
- **Layout layer** (`@layout`) — *where and how* things are drawn: `direction`,
  `size`, anchor *positions*, whether a node draws a box, and how a node's
  children are arranged. The frequently-tweaked part, editable without touching
  semantics.

One semantic model can drive multiple layouts (e.g. a wide layout for slides, a
tall one for docs) — the headline reason the separation earns its complexity.

### One node type

Every component is a **node** (`id`, optional `label`, optional `kind`, named
anchors, children). The old layout-only "group" goes away: a node that should not
draw a border just sets `box: none` in layout — the spiritual twin of CSS
`display: contents`.

### What's semantic vs. what's layout

| Semantic (in the `.ark` structure)  | Layout (in `@layout`)            |
| ----------------------------------- | -------------------------------- |
| node `id`, `label`                  | `direction`, `size`              |
| `kind` (e.g. `database`)            | implicit `@use database` baseline |
| anchor **names** (`db`, `north`)    | anchor **positions** (`[x, y]`)  |
| containment (nesting)               | child **arrangement** + regroup  |
| arrows (`a#db --> b`)               | `box: none`, future styling      |

An arrow connects *named* anchors (semantic); where a named anchor sits on the box
is layout.

### Selectors and resolution — no cascade

Layout rules target nodes by **exact dotted path** (`services.userService`) — no
wildcards, no specificity, no inheritance. Resolution has just **two precedence
tiers**, which is the entire "specificity" model:

1. **Imported** — what a node receives via `kind` (the implicit baseline) and
   `@use <block>`. Within this tier, source order wins (a later `@use` beats an
   earlier one).
2. **Direct** — declarations that name the node itself, in its inline `@layout` or
   in a sheet selector for its exact path. Direct **overrides imported** for any
   property, with no conflict — the "redeclare for explicit control" path.

Within the **direct** tier a property may be set at most once: two separate direct
rules setting the same property on the same node is a **validation error**, never
a silent cascade. (If multi-sheet themes ever need it, this can relax to
import-order-wins — explicit, and still deterministic.)

### Reusable blocks (`@block` / `@use`)

Shared layout is opt-in and explicit — a named bundle you pull in, not a class
that matches at a distance:

```
@layout {
  @block service { size: 0.75 }

  services.userService  { @use service }
  services.orderService { @use service; size: 0.5 }   # local override (last wins)
}
```

Blocks are parameterless in v1 (no mixin arguments) and may compose (cycles are an
error).

### Kind — a semantic class that hooks layout

A node may declare a `kind` — an arbitrary semantic classification (`kind:
database`, `kind: external`). It is semantic (it says *what the component is*) and
it implicitly applies the layout block of the same name as a **style baseline**:
`kind: database` behaves as a leading `@use database`. This is the one good part
of CSS classes — a semantic hook for shared style — without the specificity,
because the binding is a 1:1 name match, not a selector contest.

```
external {                 # semantic
  label: "Payment Provider"
  kind: invisible
}

@layout {
  external { @use service }   # explicit layout still wins over the kind baseline
}
```

Rules:

- A node has **one** `kind` (v1); it expands to a single implicit `@use`.
- The kind baseline is the **lowest-precedence** layer: explicit layout (inline or
  sheet) overrides it without conflict — the "redeclare for explicit control"
  path. Conflicts *between explicit selectors* are still errors.
- A small set of **built-in kinds** ships (e.g. `invisible` → `box: none`); any
  kind can be (re)declared with `@block <kind> { … }` to take full control (a user
  block overrides the built-in of the same name).
- An **unknown kind** (no built-in and no `@block`) is a **no-op**, not an error:
  `kind` is a semantic tag, so a node may carry one without a matching layout
  block. An explicit **`@use` of an undefined block *is* an error**, because that
  is a layout import the author asked for that cannot be satisfied. (M4 decision.)
- v1 layout is structural (box, size, direction, anchors), so built-in kinds can
  only touch those for now. `kind` is the natural hook for visual styling (colour,
  fonts) if/when that layer lands — part of why the bridge is worth building now.

### Presentational regrouping (`@group`)

Inside a node's `@layout` block you may list its children to reorder them, and
wrap a run of them in an anonymous `@group` for purely visual nesting — the
layout-layer equivalent of an HTML wrapper `<div>`:

```
@layout {
  services {
    direction: horizontal
    @group { direction: vertical; userService; orderService }
    payments
  }
}
```

A bare identifier (no `:`) is a child reference; `@group { … }` is a wrapper with
its own `direction`/`size`/`margin` and nested arrangement. A `@group` is always
**invisible** (it renders as `box: none`) and **anonymous** — it has no id and
adds no path segment, so a child inside a group keeps its real dotted path and
arrows/anchors are unaffected. (v1: a group can't be bordered or labelled, and
`@use` is not allowed inside one.)

Two rules keep the picture honest (both enforced by the validator):

- **Same-parent only** — a `@group` may contain only direct semantic children of
  the enclosing node (and nested `@group`s thereof). This guarantees the **layout
  tree is always a refinement of the semantic tree**: a node never appears
  visually inside a box it isn't semantically part of.
- **Completeness** — once you arrange a node's children, reference each exactly
  once (no omissions, duplicates, or foreigners).

The arrangement is **direct-only**: it is authored on the node itself (inline or
sheet) and is never imported through `@use` or `kind` (child ids are node-specific,
so a reusable block carrying an arrangement makes no sense — and is an error).

### Both inline and standalone

`@layout { … }` may sit inside a node body (local presentation) or stand alone as
a sheet of selectors (separation / theming). Same declarations either way.

### Box model & margins

> ✅ Implemented. `margin`/`box` are `@layout` properties (relocated in M3);
> the geometry below is live. The original M1 model summed adjacent margins and
> collapsed *all* perimeter margins of a `box: none` node; both were revised to
> the collapsing, wall-seeking rules below (see [decisions.md](decisions.md)).

Spacing is expressed as a per-node **`margin`** (a layout property), implemented
with a real box model rather than a parent-level gap. Each node has a **border
box** (its visible rectangle — content plus a 1px border, or content only when
`box: none`) and a **margin box** (the border box plus its margins).

Two rules govern how margins consume space:

- **Channels collapse, they don't stack.** The gap between two adjacent siblings
  is the *larger* of their facing margins (not the sum), and a child's gap to its
  parent's wall is its own margin — so every channel is one uniform margin wide.
- **A child's margin lands at the nearest wall.** A **bordered** parent grows to
  contain each child's *margin* box (margins are padding inside the border). A
  **`box: none`** parent is transparent: it is not a wall, so its children's
  perimeter margins push *through* it to the nearest bordered ancestor and land
  there. Only with no bordered ancestor at all — an invisible chain up to the
  document root — do perimeter margins **collapse to zero**, so the canvas (and a
  top-level group) never gains phantom padding.

Anchors and arrows attach to the **border box**; margins are the empty space
around it — which is also what gives auto-routed arrows room to travel between
otherwise-touching boxes.

### Arrow endpoints — an auto-cardinal default

> ✅ Implemented (M2). Lives entirely in the generator's endpoint resolution and
> needed no parser change — a bare reference and an explicit `#center` are
> already distinct in the arrow string.

An arrow that names an anchor (`a#db --> b`) uses that anchor's resolved position.
An arrow that **doesn't** (`a --> b`) auto-routes: each end attaches to the
**cardinal side (N/E/S/W) facing the other node's centre**, giving a clean
edge-to-edge line instead of a centre-to-centre one through the boxes. The side is
the dominant axis of the centre-to-centre vector (`|dx| ≥ |dy|` → E/W by the sign
of `dx`, else N/S by the sign of `dy`; exact diagonals favour the horizontal
side) — fully deterministic, and always overridable by naming an explicit anchor.

This is auto-*routing*, not auto-*placement*: it never moves a box, only chooses
where a line attaches to boxes the author positioned — the one bounded "automatic"
behaviour in the tool. A **named** anchor stays a single fixed point (it can serve
many arrows, so it has no one "other node" to aim at); unpositioned, it defaults
to centre. This routing depends on margins for room — without a gap, edge
attachment between touching boxes degenerates to a zero-length arrow — but is
otherwise independent of the rest of the `@layout` model.

## Distribution

Arkitecture ships as a single self-contained binary (no runtime to install) and,
from the same library, as a WebAssembly module usable directly from
JavaScript/TypeScript in the browser or Node. The CLI and the WASM build are thin
wrappers over one library API, so both stay in lock-step. The docs site is the
first consumer of that WASM build: its Examples page lets you edit any example and
see it re-render live in the browser, with the static, CLI-rendered SVG as the
no-JavaScript fallback.

## Out of scope (v1)

- **Automatic layout / routing** — the entire point is manual control; no
  force-directed or hierarchical auto-placement.
- **CSS-style cascade** — selectors are exact-path; no wildcards, specificity, or
  inheritance. A deliberate non-feature (see *Semantic vs. layout*).
- **Parameterized layout blocks** — `@block`/`@use` is parameterless; no mixin
  arguments yet.
- **Visual styling / theming** — output is intentionally plain (white fill, 1px
  black border, one font). Colour and per-node fonts are not modelled; the
  `@layout` layer is structural (size, direction, anchors, box). `kind` is the
  intended hook for a future styling layer.
- **Interactivity** — static SVG only; no links, tooltips, or animation.
- **Curved / orthogonal arrow routing** — arrows are straight lines between
  anchors.
- **Output formats other than SVG** — no PNG/PDF; convert downstream if needed.

## Open questions

- Per-side margins (`margin-top` …), or keep the v1 uniform-only margin? (The
  default is 8, and adjacent siblings *collapse* to the larger facing margin —
  decided in [decisions.md](decisions.md).)
- Auto-cardinal routing ships with 4 sides (N/E/S/W); is 8 directions (incl.
  corners) ever worth the extra ambiguity? (4 is the v1 decision.)
- Should an unpositioned *named* anchor be an error rather than defaulting to
  centre?
- Should a node be allowed *multiple* kinds later, and if so how do their blocks
  combine?
- Cross-file layout sheets / an `@import` for sharing layout across diagrams?
- Should arrows support labels, and if so where do they sit in the layout?
- How should very long labels behave — wrap, truncate, or keep requiring explicit
  `\n` line breaks (current behaviour)?
