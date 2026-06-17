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
- **Group** — a layout-only container with no ID, label, or border. Exists purely
  to arrange its children; invisible in the output.
- **Direction** — `vertical` or `horizontal`: how a node or group stacks its
  children. Defaults to `vertical`.
- **Anchor** — a named point on a node in relative `[x, y]` coordinates (`[0, 0]`
  top-left … `[1, 1]` bottom-right), used as an arrow endpoint. Every node has an
  implicit centre anchor at `[0.5, 0.5]`.
- **Arrow** — a directed connection `source --> target`, where each end is a
  (possibly dotted) node path with an optional `#anchor`.
- **Size** — an override in `[0, 1]` for a node's *orthogonal* dimension, as a
  fraction of what the parent would otherwise give it.

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
- Groups add no visual space — they only group children for direction.
- The canvas is sized to exactly fit all top-level content — no padding.

## Distribution

Arkitecture ships as a single self-contained binary (no runtime to install) and,
from the same library, as a WebAssembly module usable directly from
JavaScript/TypeScript in the browser or Node. The CLI and the WASM build are thin
wrappers over one library API, so both stay in lock-step.

## Out of scope (v1)

- **Automatic layout / routing** — the entire point is manual control; no
  force-directed or hierarchical auto-placement.
- **Styling and theming** — output is intentionally plain (white fill, 1px black
  border, one font). Colour, per-node fonts, and CSS are not modelled yet.
- **Interactivity** — static SVG only; no links, tooltips, or animation.
- **Curved / orthogonal arrow routing** — arrows are straight lines between
  anchors.
- **Output formats other than SVG** — no PNG/PDF; convert downstream if needed.

## Open questions

- Should spacing/padding (between siblings and inside nodes) become configurable,
  or stay at zero for predictability?
- Should arrows support labels, and if so where do they sit in the layout?
- How should very long labels behave — wrap, truncate, or keep requiring explicit
  `\n` line breaks (current behaviour)?
- Is there demand for size *constraints* (max canvas, scaling) beyond the current
  exact-fit canvas?
