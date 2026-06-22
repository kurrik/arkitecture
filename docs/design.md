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
  in `@layout`. Defaults to `vertical`. It is exactly **sugar for a single-track
  grid**: `vertical` ≡ `cols: 1` (rows grow), `horizontal` ≡ `rows: 1` (columns
  grow). The two spellings are interchangeable — a stack and a grid are one model.
- **Anchor** — a named point on a node in relative `[x, y]` coordinates (`[0, 0]`
  top-left … `[1, 1]` bottom-right), used as an arrow endpoint. Every node has an
  implicit centre anchor at `[0.5, 0.5]`.
- **Arrow** — a directed connection `source --> target`, where each end is a
  (possibly dotted) node path with an optional `#anchor`.
- **Semantic vs. layout layers** *(M3)* — structure (`id`, `label`, `kind`,
  anchor *names*, nesting, arrows) is authored separately from presentation
  (`@layout`: `direction`, anchor *positions*, `margin`, `box`, child
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
  padding) but collapses against an invisible (`box: none`) parent. The default 8
  is overridable per document: a bare `margin: N` at an `@layout` sheet root sets
  the **default margin** for every node that declares none. See *Box model &
  margins*.
- **Default margin** — the document-wide fallback margin, set by `margin: N` at
  the root of an `@layout` sheet (not inside a selector). It replaces the built-in
  8 for any node that sets no margin of its own; nodes still override it directly.
  It is a single global baseline, *not* a cascade — there is no parent→child
  inheritance and no selector contest.
- **Route mode** — the document-wide arrow routing style, set by a bare
  `route: straight | orthogonal` at the root of an `@layout` sheet (mirroring the
  **default margin**). `straight` (the default) draws each arrow as the M2
  auto-cardinal line; `orthogonal` routes arrows as axis-aligned paths around the
  boxes between their endpoints. See *Auto edge routing — sized channels*.
- **Box / border box / margin box** — `box: none` makes a node draw no border
  (an ID-bearing twin of the layout-only group); the **border box** is the
  visible rectangle and the **margin box** is the border box plus its margins.
- **Label band** — the strip a parent reserves for its own label so it does not
  overlap the children. `label: top | bottom` (a `@layout` property, default top)
  chooses which end. In a bordered parent the band acts as an inner wall for
  margins; a `box: none` parent reserves the same strip but packs its children
  flush below it (it draws no border and adds no perimeter). A leaf needs no band
  (its box already fits its label). See *Box model & margins*.
- **Grid** — a third child-arrangement mode (`cols: N` plus optional `rows: M` in
  a node's `@layout`), the 2-D generalisation of `direction`'s 1-D packing.
  Children place themselves with `col`/`row` (1-based) and `colSpan`/`rowSpan`, or
  auto-fill the next free slot; tracks are sized jointly on both axes. See *Grid
  arrangement*.

## The workflow

1. Write a `.ark` file describing the diagram's nodes, nesting, and arrows.
2. Run `arkitecture diagram.ark diagram.svg` (the output name defaults to the
   input with a `.svg` extension).
3. Read the reported errors — syntax, bad references, out-of-range values — all at
   once, with line/column positions.
4. Adjust `direction`, nesting, and grid placement to shape the layout; add
   `anchors` to steer arrows.
5. Re-run, or use `--watch` to regenerate on every save.
6. Commit the `.ark` source alongside the code it documents.

See [examples/annotated.ark](../examples/annotated.ark) for a fully commented
reference covering every feature.

## Layout model

Layout is bottom-up and deterministic:

- A leaf node sizes to its label text (measured with a fixed font).
- A **parent with a label** reserves a strip — its **label band** — for that
  label, sized like a leaf box holding it, at the top (default) or bottom as set
  by `label: top | bottom` in `@layout`. The children lay out in the remaining
  area and the box grows at least as wide as the label, so the label is never
  obscured by the children. In a **bordered** parent the band's inner edge is a
  **wall** the children's facing margin collapses against; a **`box: none`**
  parent reserves the same strip but packs its children flush below it (it adds no
  perimeter of its own, consistent with how it packs them flush everywhere else).
- A **vertical** parent stacks children top-to-bottom: its width is the widest
  child; children span the full width.
- A **horizontal** parent places children left-to-right: its height is the tallest
  child; children span the full height.
- Each node reserves a uniform **`margin`** (default 8, or the document's
  **default margin** if set — see below) around its border box. Margins
  **collapse** rather than stack: the channel between two adjacent siblings is the
  *larger* of their facing margins (not the sum), and a child's gap to its
  parent's wall is its own margin — so every channel is one uniform margin wide.
  `margin: 0` restores flush packing.
- A bare **`margin: N`** at an `@layout` sheet root sets the document's **default
  margin** — the fallback for every node that declares none, replacing the
  built-in 8. It is a global baseline (not a cascade): a node still overrides it
  with its own `margin`, and there is no inheritance or selector specificity. It
  is the one knob for "space the whole diagram out".
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
  so the diagram gains no outer padding even though inner nodes are spaced. The one
  thing it does reserve is the *stroke* a border draws: an SVG stroke is centred on
  the box edge, so half its width sits outside the border box, and the canvas grows
  by that half-width wherever a border sits on the perimeter — otherwise the SVG
  viewport would clip an outer border to half its width. (Emitted as a `viewBox`
  offset, so element coordinates are unchanged; the effect is at most ~1px and is
  not author-visible spacing.)

## Semantic vs. layout (the `@layout` model)

> ✅ The full `@layout` model has shipped: the **split** (two-layer authoring,
> `@layout` blocks, exact-path selectors, no-cascade resolution) in M3, **reuse**
> (`@block`/`@use` and `kind` hooking a layout block) in M4, and presentational
> **`@group` regrouping** in M5. In M3 the inline `direction`/`anchors:{pos}`
> shorthand was **dropped**: a node body is purely semantic and all presentation
> lives in `@layout`.

The diagram is authored in two layers, like HTML + CSS — but **without CSS's
cascade**, because that cascade is exactly the action-at-a-distance the core
principle rejects.

- **Semantic layer** — *what* the components are and how they relate: a node's
  `id`, optional `label`, its `kind`, the named anchors it exposes, containment
  (nesting = "is part of"), and arrows (relations). The stable part.
- **Layout layer** (`@layout`) — *where and how* things are drawn: `direction`,
  anchor *positions*, whether a node draws a box, and how a node's
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

| Semantic (in the `.ark` structure)  | Layout (in `@layout`)             |
| ----------------------------------- | --------------------------------- |
| node `id`, `label` **text**         | `direction`, `label` **position**         |
| `kind` (e.g. `database`)            | implicit `@use database` baseline |
| anchor **names** (`db`, `north`)    | anchor **positions** (`[x, y]`)   |
| containment (nesting)               | child **arrangement** + regroup   |
| arrows (`a#db --> b`)               | `box: none`, **styling** (colour/width) |

An arrow connects *named* anchors (semantic); where a named anchor sits on the box
is layout. The same split applies to a node's label: the **text** is semantic (a
node-body `label: "…"`), while which end of a parent reserves the strip for it is
layout (an `@layout` `label: top | bottom`).

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
  @block service { margin: 16 }

  services.userService  { @use service }
  services.orderService { @use service; margin: 8 }   # local override (last wins)
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
- v1 layout is structural (box, direction, anchors), so built-in kinds can
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
its own `direction`/`margin` and nested arrangement. A `@group` is always
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

### Grid arrangement (`cols` / `rows`)

A node may arrange its children as a **2-D grid** instead of a 1-D stack — the
generalisation of `direction`. The grid is declared with two plain `@layout`
properties, `cols` (and optional `rows`), alongside `direction`/`margin`/… — a
node is a grid when it sets `cols`. Like `@group` regrouping, the track def is
**direct-only** (never imported via `@use`/`kind`):

```
@layout {
  board { cols: 3 }                 # 3 fixed columns; rows grow with content
  board.title { col: 1; row: 1; colSpan: 3 }
  board.a { col: 1; row: 2 }  board.b { col: 2; row: 2 }  board.c { col: 3; row: 2 }
}
```

- **Tracks.** `cols` is fixed; `rows` is optional and grows implicitly to fit the
  placed children when omitted (so you fix one axis and let the other extend —
  fixing columns and letting rows grow is the natural shape for a timeline).
- **Placement.** A child sets `col`/`row` (1-based grid lines) and optional
  `colSpan`/`rowSpan` (default 1). A child that sets neither **auto-places** into
  the next free slot, scanning left→right then top→bottom (sparse — the cursor
  only moves forward, never backfilling). A skipped slot therefore needs **no
  placeholder**: placement is sparse by construction.
- **Cell alignment.** A track is as large as its biggest cell, so a smaller child
  is positioned within its (possibly spanning) cell by `justify` (horizontal) and
  `align` (vertical), each `start | end | stretch` (default `stretch`, which fills
  the cell — and so centres an arrow's endpoint on the track).
- **Joint track sizing.** Unlike nested packing — which aligns the stacking axis
  but lets the orthogonal one drift per row — a grid sizes **both** axes together:
  each single-track cell grows its column to its width and its row to its height,
  then a spanning cell distributes any shortfall evenly across the tracks it
  covers. This is the capability packing cannot emulate without hand-padding every
  column to equal height.
- **Bounds are enforced.** A cell — or its `colSpan`/`rowSpan` extent — placed
  outside the declared tracks, or two cells overlapping, is a **constraint** error
  (the same no-silent-conflict stance as duplicate direct properties).

This stays inside *manual, deterministic layout*: every position is an explicit
per-node property, and auto-flow is a fixed, predictable, overridable rule — the
2-D form of the auto-packing `direction` already performs, not auto-*placement*
that moves the author's arrangement. The intended pattern for tabular diagrams
(e.g. a sequence diagram: columns = participants, rows = time) is to keep the
cells as **flat children of one grid node**, so the grid is a single-level
operation and no `@group` reparenting is needed — regrouping cannot cross the
semantic tree, but a flat grid never needs to.

> 🚧 **v1 limits** (tracked in the roadmap): an empty track collapses to zero
> (no min-size spacer rows yet); `stretch` resizes a cell child after its own
> subtree was sized, so stretching a *container* child can misalign its interior
> (leaves are fine); each inter-track channel is the *collapsed* (larger) facing
> margin of its adjacent children — the same box model as 1-D packing, but with no
> dedicated `gap` knob yet; and a grid ignores any `@group` arrangement on the same
> node.

The grid uses the **same margin-collapse box model as 1-D packing**: an
inter-track channel is the larger of the facing children's margins, a bordered
grid reserves a perimeter sized from its edge children's margins, and a
`box: none` grid carries its children's margins outward. The default cell
alignment follows the box model too — a **bordered** parent stretches a child to
fill its cross axis, a **`box: none`** parent leaves it at its natural size
(`start`) — uniformly with how stacks behave; an explicit `justify`/`align`
overrides it. As a result a single-track grid reproduces a `direction` stack
**exactly** (bordered *and* `box: none`), which is the property the unified engine
rests on, and `direction` is just sugar for `cols: 1` / `rows: 1`. (One deliberate
consequence: where heterogeneous per-child margins meet on the *cross* axis, a
grid collapses them to one track perimeter rather than insetting each child
individually — a grid keeps its tracks aligned. Another: a `box: none` grid no
longer stretches its cells, matching `box: none` stacks; author `justify: stretch`
for the old lane-filling look.)

Because a stack and a grid are one model, a node authored with `direction` (or
nothing) **gains grid placement for free**: a child may set `col`/`row`/spans to
place itself sparsely. There is **one layout engine** — every arranging node runs
through the grid path; the former 1-D packing code is gone. The grid carries the
orthogonal-route channel widening for a single-track stack, so `route: orthogonal`
through a stack is unchanged. (Channel widening through a *multi-track* grid is not
yet modelled — the router's channel graph is still 1-D per container — so
`route: orthogonal` across a true grid stays a follow-up; see the roadmap.)

### Both inline and standalone

`@layout { … }` may sit inside a node body (local presentation) or stand alone as
a sheet of selectors (separation / theming). Same declarations either way.

### Styling — colour & stroke width

Presentation now extends past structure to a small, deliberate set of visual
properties, authored in the same `@layout` layer (hex colours, plain numeric
widths). They split by *what they paint*:

- **A node's own box** — `borderWidth`, `borderColor`, `backgroundColor`.
- **The arrows that *start* at a node** — `pathWidth`, `pathColor`. An arrow is
  styled by its **source** node, so "make everything leaving the cache red" is one
  declaration on the cache. (An arrow's other end has no say; this keeps each
  arrow's style attributable to a single node.)

They obey the same no-cascade resolution as every other layout property, so each
can be set three ways with the usual two-tier precedence:

- **Per node** — `services.api { borderColor: #2563eb }` (direct).
- **Document-wide** — a bare `borderColor: #334155` at an `@layout` sheet root,
  exactly mirroring the default `margin`/`route`: the fallback for every node that
  sets none. (Imported `@use`/`kind` and direct per-node values both override it.)
- **In a `@block`** — bundle a look (`@block accent { borderColor: #2563eb;
  borderWidth: 2 }`) and `@use` it, or hook it to a `kind`.

Colours are `#rgb` / `#rgba` / `#rrggbb` / `#rrggbbaa` hex literals (a wrong length
or non-hex digit is a constraint error); widths are lengths `>= 0`. Everything
defaults to the plain look — white fill, 1px black border, 1px black arrows — so an
unstyled diagram is unchanged. A coloured arrow gets a colour-matched arrowhead.

This is the first visual layer over the structural one. It is deliberately *not* a
theme system: there is no inheritance or cascade (a child does not inherit its
parent's colour), no named palette, and no font control — those stay out of scope
(see below). `kind` remains the natural hook for sharing a look across many nodes.

### Consistent line rendering

Box borders and orthogonal arrow runs are emitted with
`shape-rendering="crispEdges"`. Without it, a 1px stroke whose coordinate does not
land on the half-pixel grid is anti-aliased into a fainter, ~2px smear, so
otherwise-identical boxes render at visibly different weights. `crispEdges` snaps
axis-aligned strokes to the device-pixel grid, giving every border and orthogonal
run a uniform width regardless of sub-pixel position. It is applied only to
axis-aligned strokes — a *diagonal* straight arrow keeps its anti-aliasing, since
snapping a diagonal to the pixel grid would make it a visible staircase.

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

A **parent with a label** also reserves a **label band** — a top (default) or
bottom strip (`label: top | bottom`) sized like a leaf box holding that label — so
the label sits in its own space instead of over the children, and the box widens
if needed to fit it. In a **bordered** parent the band's inner edge is an
additional wall: the children pack in the remaining area and their facing margin
collapses against the band just as it would against the border. A **`box: none`**
parent reserves the same strip but, being transparent (no border, no perimeter),
packs its children flush below it — the band is reserved space, not a wall.

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

### Auto edge routing — sized channels *(shipped)*

> ✅ **Shipped** in reviewable slices (see the ADRs in [decisions.md](decisions.md)).
> The opt-in mode is enabled with a document-level **`route: orthogonal`** at an
> `@layout` sheet root (default `straight` is the M2 line). It comprises: the
> **surface**; **clear-case orthogonal emission** (an unobstructed arrow draws as an
> axis-aligned elbow/Z); **explicit anchors** (met at the box border on the facing
> side, the line *entering the node* for an interior anchor); **break-out** across
> nesting levels; the **channel-graph router** — a blocked arrow detours *around* the
> boxes in its way (few-bend A* over a per-arrow channel grid) instead of falling
> back to the straight line; **edge-normal exits** — an anchor pinned to a box edge
> leaves/enters perpendicular to *that* edge (a bottom anchor drops straight down),
> the router then following the corridor it lands in; and **channel widening** — an
> arrow running along a channel (a between-children gap, a perimeter, or a cross-axis
> rail *along* a perimeter) reserves a lane there, so the channel widens and the
> boxes spread, with the line snapped to its lane — a **full margin** clear of the
> boxes (a wall in its own channel, not inside the node margin), half a margin
> between lanes — and **co-routed arrows spread to distinct lanes** so they never
> overlap. Parked refinements: crossing-minimised lane ordering and a per-arrow
> `route:` override (or configurable lane spacing). It extends M2 cardinal routing from "which edge
> does the line attach to" to "what path does the line take around the boxes in
> between" — the deliberate, ADR-backed reversal of the v1 "no orthogonal/auto
> routing" scope line, and only of *routing*: auto-*placement* (moving the author's
> arrangement) stays out.

The mode draws each arrow as an **orthogonal path routed around the boxes between
its endpoints**, instead of a straight line that may cut through them. Two ideas
carry the model:

- **Channel vs. margin.** A **channel** is a routing corridor between adjacent
  blocks that reserves *its own* width and **pushes the boxes apart** to hold the
  lines it carries. A **margin** stays aesthetic breathing room — the space that
  keeps a line from crowding the box it runs alongside. Routing therefore does not
  thread lines through the margin; it inserts a sized lane-bundle, with margin
  still framing it. A channel's width is `margin + lanes × laneSpacing`, where
  *lanes* counts the arrows running **along** that channel (a line merely crossing
  it perpendicularly needs a point, not a lane).
- **A channel graph, not a pixel grid.** Routing runs on a graph derived from the
  **arrangement tree** — its nodes are channel slots (between every pair of adjacent
  siblings, plus a **perimeter ring** inside each container), its edges topological
  adjacency — *before* any coordinates exist. Pathfinding (few-bend A*) assigns each
  arrow a sequence of slots; each slot's lane count is then known as pure topology,
  so layout runs **once** with the channels sized, with no routing↔position
  feedback loop. This is why channels (not a geometric visibility grid) are the
  substrate: they let "make room for lines" decouple from pixel positions.

**Breaking out of a container** is a single rule: for an arrow `a.b.c --> x.y`, the
obstacles are every box *except* the ancestors of the source and of the target. A
container's border blocks arrows passing by, but is passable to one that must
legitimately leave or enter it; the perimeter ring gives that arrow a lane to run
in before it crosses the border. Each arrow is its own routing problem with its own
obstacle set.

This stays inside *manual, deterministic layout*: widening a channel shifts
absolute positions but never the **arrangement** — exactly as a longer label
already grows a box and nudges its neighbours. The displacement is a visible
consequence of a local rule ("N arrows route here → this gap is N lanes wide"), not
hidden auto-placement. Open knobs (tracked below): the lane-spacing formula, and
whether the mode is a document-wide `route:` knob, per-arrow, or both — the latter
giving arrows their first presence in the `@layout` layer.

## Distribution

Arkitecture ships as a single self-contained binary (no runtime to install) and,
from the same library, as a WebAssembly module usable directly from
JavaScript/TypeScript in the browser or Node. The CLI and the WASM build are thin
wrappers over one library API, so both stay in lock-step. The docs site is the
first consumer of that WASM build: its Examples page lets you edit any example and
see it re-render live in the browser, with the static, CLI-rendered SVG as the
no-JavaScript fallback.

## Out of scope (v1)

- **Automatic *placement*** — the entire point is manual control; no
  force-directed or hierarchical auto-placement that moves the author's
  arrangement. (Auto *routing* — choosing arrow paths without disturbing the
  arrangement — is in scope: M2 cardinal endpoints today, sized-channel orthogonal
  routing designed above. The grid's `col`/`row` auto-flow is *not* this: it is a
  fixed, deterministic, overridable packing rule the author opts into — the 2-D
  form of `direction`'s existing auto-packing, not engine-decided placement.)
- **CSS-style cascade** — selectors are exact-path; no wildcards, specificity, or
  inheritance. A deliberate non-feature (see *Semantic vs. layout*).
- **Parameterized layout blocks** — `@block`/`@use` is parameterless; no mixin
  arguments yet.
- **Theming / cascade / fonts** — per-element styling (hex `borderWidth`/
  `borderColor`/`backgroundColor` and `pathWidth`/`pathColor`) *is* now modelled in
  the `@layout` layer (see *Styling* above), but a full theme system is not: no
  inheritance/cascade (a child never inherits a parent's colour), no named
  palettes or design tokens, and no per-node font control (one font for the whole
  diagram). `kind`/`@block` are the sharing mechanism instead of a cascade.
- **Interactivity** — static SVG only; no links, tooltips, or animation.
- **Curved arrow routing** — arrows are straight or orthogonal segments, never
  splines. (Orthogonal sized-channel routing is *planned*, designed above; curved
  paths stay out.)
- **Output formats other than SVG** — no PNG/PDF; convert downstream if needed.

## Open questions

- Per-side margins (`margin-top` …), or keep the v1 uniform-only margin? (The
  default is 8 — now overridable document-wide via a root `@layout { margin: N }`
  — and adjacent siblings *collapse* to the larger facing margin — decided in
  [decisions.md](decisions.md).)
- Auto-cardinal routing ships with 4 sides (N/E/S/W); is 8 directions (incl.
  corners) ever worth the extra ambiguity? (4 is the v1 decision.)
- For auto edge routing (sized channels, above): what is the lane-spacing formula —
  a fixed multiple of `margin`, or font-scaled? (The mode is settled as a
  document-wide `route: orthogonal` knob, now implemented; a per-arrow override —
  arrows' first foothold in the `@layout` layer — remains a possible follow-up.)
- Should an unpositioned *named* anchor be an error rather than defaulting to
  centre?
- Should a node be allowed *multiple* kinds later, and if so how do their blocks
  combine?
- Cross-file layout sheets / an `@import` for sharing layout across diagrams?
- Should arrows support labels, and if so where do they sit in the layout?
- How should very long labels behave — wrap, truncate, or keep requiring explicit
  `\n` line breaks (current behaviour)?
