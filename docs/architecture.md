# Architecture

How the code is laid out and how the runtime fits together. Update when the
structure changes — adding a package, introducing a stage, or establishing a new
pattern.

## App shape

Arkitecture is a Go library with two thin wrappers on top: a CLI
(`cmd/arkitecture`) and a WebAssembly shim (`wasm/`, built with
`GOOS=js GOARCH=wasm`). The library is pure and side-effect-free — given a DSL
string it returns a result — so it runs identically in a binary, in tests, and in
the browser via WASM. All file I/O lives in the CLI.

Processing is a one-way pipeline with no shared mutable state:

```
DSL text → Tokenizer → Parser → Validator → Resolver → Generator → SVG
            ([]Token)   (Document) (errors?)  (layout map) (layout+SVG)  (string)
```

The Document carries two layers: a semantic node tree and the layout layer (a
flat list of selector rules plus named `@block` definitions). The validator
checks both; the resolver merges the layout onto the tree by exact path —
applying each node's `kind` baseline and `@use` imports below its direct
declarations — into a per-node resolved layout the generator reads.

Each stage is a pure function of its input. Errors are *collected* as
`[]ast.Error`, never thrown across stage boundaries; the top-level `ToSVG`
recovers from any unexpected panic and returns it as one error, so a single run
reports every problem it finds.

## Layout

```
github.com/kurrik/arkitecture        (module)
  arkitecture.go     package arkitecture — public API (ToSVG/Parse/Validate/
                     GenerateSVG); re-exports the AST types as aliases
  ast/               package ast — Document, ContainerNode, Declarations,
                     LayoutRule, Use, Block, Arrow, Error, ParseResult, plus
                     BuiltinBlocks(); no deps, so every stage can import it
  parser/            tokenizer + recursive-descent parser → ast.Document
  validator/         Document → []ast.Error (references, ID uniqueness, layout
                     selectors/conflicts/ranges, anchor names)
  resolve/           Document → map[path]*Declarations (merge layout onto tree)
  generator/         Document + resolved layout → SVG string (text measurement,
                     layout, arrow routing, emit)
    route.go         arrow endpoint resolution + straight/orthogonal routing
    channel.go       channel-graph router: few-bend A* around an arrow's obstacles
    widen.go         channel widening: attribute runs to gaps, widen, snap to lanes
    grid.go          grid arrangement (cols/rows): joint two-axis track sizing + placement
    testdata/golden/ .ark fixtures + .svg/.error references for the golden test
  cmd/arkitecture/   package main — the CLI (flags, file I/O, watch); imports the library
  wasm/              package main — js,wasm shim exposing ToSVG to JS (+ host stub)
  internal/sitegen/  package main — build tool: inject each example's .ark source
                     into its <code data-ark> block in the site HTML (not shipped)
examples/            sample .ark inputs
site/                static docs site (GitHub Pages); examples.html is enhanced
                     into a live WASM editor by playground.js
scripts/build-site.sh  generation step: render site examples + inject sources + build WASM
```

Dependency direction is one-way: `cmd`/`wasm` → `arkitecture` → `{parser,
validator, resolve, generator}` → `ast` (the generator also reads the
`*ast.Declarations` the resolver produces). The `ast` package has no
dependencies, which is what lets the root package and the stage packages share
types without an import cycle. Nothing depends on the CLI. Keep packages flat
until a stage genuinely needs sub-packages.

## Domain model

The AST (`ast` package) is the contract every stage shares, split into a
semantic layer and a layout layer:

- **`Document`** — `{ Nodes []*ContainerNode; Layout []LayoutRule; Blocks []Block; Arrows []Arrow; DefaultMargin *float64; Route *RouteMode; Defaults *Declarations }`.
  Layout rules, blocks, and arrows are collected into flat lists (top-level
  statements may appear in any order), not attached to nodes. `DefaultMargin` (a
  bare `margin:` at a sheet root) is the document-wide fallback margin — the
  generator uses it in place of the built-in 8 for any node that sets none.
  `Route` is the document-wide routing mode (a bare `route:`). `Defaults` holds the
  document-wide **style** fallbacks (bare `borderColor:`/`pathWidth:`/… at a sheet
  root) — the same fallback model as `DefaultMargin`, reusing `Declarations` but
  with only its style fields populated.
- **`ContainerNode`** — the single node type: `ID`, optional `Label`, `Kind`,
  `Anchors` (declared anchor *names*), and `Children []*ContainerNode`. It carries
  no layout — `GroupNode` is gone; a borderless grouping is a `box: none` node.
- **`Declarations`** — a set of layout properties (`Direction`, `Margin`,
  `Box`, `LabelPos`, the style fields `BorderWidth`/`BorderColor`/`BackgroundColor`
  (the box) and `PathWidth`/`PathColor` (arrows starting at the node), and `Anchors`
  name→position) plus an optional `Arrangement` (the node's ordered child layout).
  Colours are hex strings; widths are floats. Each scalar is a pointer so "unset"
  stays distinguishable, which the conflict check relies on.
- **`ArrangementItem`** — one entry in a node's child arrangement: either a
  `ChildID` (a direct child reference) or a `Group *Declarations` (an anonymous
  `@group`). A group *is* a `Declarations` whose own `Arrangement` holds its nested
  items, so nodes and groups share one shape and nest recursively. The arrangement
  is direct-only — never imported via `@use`/`kind`.
- **`GridSpec` / grid placement** — `Declarations.Grid *GridSpec` ({`Cols`,
  `Rows`}) makes a node arrange its children as a 2-D grid (direct-only, like
  `Arrangement`); a child's own `Col`/`Row`/`ColSpan`/`RowSpan` and `Justify`/
  `Align` (all pointers) place and align it. `ast.PlaceGrid` is the pure,
  panic-free placement algorithm (explicit + sparse auto-flow, reporting
  out-of-bounds and overlap `GridProblem`s) shared by the validator's checks and
  the generator's geometry.
- **`LayoutRule`** — `{ Selector string; Decls *Declarations; Uses []Use; Line, Column }`.
  One per `@layout` selector block: the node's direct declarations plus any `@use`
  imports. An inline `@layout` is desugared by the parser into a rule whose selector
  is the enclosing node's full path, so inline and standalone layout are uniform.
- **`Use` / `Block`** — `Use` is a `@use <name>` import (with position); `Block` is
  a named `@block <name> { decls }` bundle (declarations plus its own `@use`s, so
  blocks compose). `ast.BuiltinBlocks()` supplies the built-in kinds (`invisible` →
  `box: none`); a user `Block` of the same name overrides a built-in.
- **`Arrow`** — `Source` and `Target` strings, each a dotted node path with an
  optional `#anchor` suffix (e.g. `c1.n2 --> c1.n3#a1`). Resolved by the validator.
- **`Error`** — `Line`, `Column`, `Message`, and `Type` (`syntax` | `reference` |
  `constraint`). Every failure is one of these; it also satisfies `error`.

## Pipeline stages

- **Tokenizer** (`parser/tokenizer.go`) — hand-written rune scanner producing
  `Token`s with line/column info: identifiers, strings (with escapes), numbers,
  hex **colour** values (`#rrggbb`), structural punctuation, the `-->` arrow, `@`
  (directive), `#` (anchor vs colour vs comment), and newlines (`;` is a cosmetic
  separator skipped like whitespace). The `#` is disambiguated by what precedes it:
  glued to an identifier (`node#anchor`) it is an anchor hash; otherwise a complete
  hex run with a word boundary is a colour and anything else is a comment. Returns a
  `*TokenizerError` on an unexpected character or unterminated string.
- **Parser** (`parser/parser.go`) — recursive-descent build of the `Document`:
  semantic node bodies (`label`/`kind`/anchor names/children), inline and
  standalone `@layout` blocks (the declaration grammar and exact-path selectors),
  a bare `margin:` at a sheet root (the document default, distinguished from a
  selector by the `:`), `@block` definitions, `@use` imports, `@group` child
  arrangements inside `@layout` (a bare identifier with no `:` is a child
  reference), the `cols`/`rows` grid track properties and per-child `col`/`row`/
  `colSpan`/`rowSpan`/`justify`/`align` placement, and arrows. Nodes,
  `@layout` sheets, and arrows may appear in **any order** at the top level — each
  statement is dispatched by lookahead (an identifier reaching `-->` is an arrow),
  so an arrow can be colocated with the nodes it connects. An inline `@layout` is
  desugared into a path selector. Collects syntax errors with positions and
  recovers to keep going; range checks moved to the validator. `parser.Parse` wires
  tokenizer and parser.
- **Validator** (`validator/validator.go`) — semantic checks over a parsed
  `Document`: ID uniqueness within a scope, dangling layout selectors (reported at
  the selector position), duplicate **direct** layout properties on a node, layout
  ranges (`margin`/`borderWidth`/`pathWidth`/coords) and hex-colour format
  (`borderColor`/`backgroundColor`/`pathColor`, including the document defaults),
  anchor positions naming a declared anchor,
  undefined `@use` blocks and `@use` composition cycles (reported at the `@use` /
  block position), child-arrangement same-parent and completeness checks (each
  direct child referenced exactly once, no foreigners), grid placement bounds and
  overlap (via `ast.PlaceGrid` over a node's resolved-direct cells), and arrow
  source/target + anchor-name resolution (with the implicit `center`);
  non-fail-fast. An unknown
  `kind` is deliberately *not* an error (it's a semantic tag). Apart from the
  position-bearing cases above,
  diagnostics report at line 1, column 1 — the semantic AST carries no node
  positions.
- **Resolver** (`resolve/resolve.go`) — pure merge of the document's layout onto
  node paths, producing `map[path]*ast.Declarations`. It assumes a validated
  document (conflicts already rejected), so merging is a deterministic overlay
  with two precedence tiers: the **imported** tier (a node's `kind` baseline, then
  each `@use` in source order, each block expanded recursively as its own
  imports-then-decls) underneath the **direct** tier (declarations naming the
  node). `Grid` is carried direct-only, like the child `Arrangement` (never
  imported via a block or kind). A visiting set makes block composition cycle-safe
  even though the validator rejects cycles first (so `GenerateSVG`, which skips
  validation, can't
  loop). A node's child **arrangement** is carried direct-only — copied from the
  node's own rules, never imported through a block or kind.
- **Generator** (`generator/`) — takes the document plus the resolved layout.
  `text.go` measures labels with a deterministic, dependency-free rune-width
  approximation; `layout.go` builds a path-keyed tree (from a node's resolved
  `Arrangement` when present, otherwise semantic child order — a `@group` becomes a
  synthetic invisible node that adds no path segment, so its children keep their
  real paths), reads each node's resolved declarations, sizes bottom-up applying
  the 1-D vertical/horizontal packing rules for a dense stack, or `grid.go`'s joint
  two-axis track sizing + per-cell placement/alignment when the node declares
  `cols`/`rows` or a child opts into placement (`direction` is sugar for a
  single-track grid, so the two paths are byte-identical), the label band a labelled
  parent reserves (a top/bottom strip — a wall in a bordered parent, flush-packed
  reserved space in a `box: none` one) — falling back to the document's
  `DefaultMargin` (else 8) for any node with no margin — positions top-down, sizes
  the canvas (the content bounds grown to include each border's *stroke* — half a
  border width sits outside its box, emitted as a `viewBox` offset so a perimeter
  border isn't clipped to half), and resolves anchor coordinates (an unpositioned
  declared anchor defaults to centre);
  `route.go` resolves each arrow's endpoints (the M2 cardinal edge, or an explicit
  anchor) and turns them into the ordered points its line passes through: two
  points (the straight default) or, under a document-level `route: orthogonal`, an
  axis-aligned path between them. That path is the clear-case elbow/Z when it is
  clear of the arrow's obstacles (every box not on the source's or target's
  lineage); when the elbow is blocked, `channel.go`'s `routeAround` detours it on a
  per-arrow channel grid (lanes one inset outside each obstacle edge, navigated by
  deterministic few-bend A*); only if no orthogonal path exists does it fall back to
  the straight line. A positioned anchor is met at the box border on the facing
  side, with a tail segment entering the node to reach an interior anchor
  (zero-length for an edge anchor). In orthogonal mode `GenerateSVG` lays out
  **twice**: `widen.go` attributes each routed run to the container gap it follows
  and reserves `lanes × margin/2` there, then a second `computeLayout` spreads the
  boxes and the runs snap to their lane centres, so a line gets its own lane instead
  of sitting in a box's margin;
  `svg.go` walks the tree to emit `<rect>` + `<text>` per visible node (the label
  is centred in its reserved band when a parent has one; a `box: none` node and a
  `@group` render no rect) and, per arrow, a `<line>` (two points) or `<polyline>`
  (a bent route) with the arrowhead `<marker>`. Each rect takes its resolved
  `fill`/`stroke`/`stroke-width` (node value → document `Defaults` → built-in
  white/black/1px); each arrow takes its **source** node's resolved
  `pathColor`/`pathWidth` and a colour-matched arrowhead — `buildDefs` emits one
  `<marker>` per distinct path colour (the black `arrowhead` plus `arrowhead-<hex>`
  per colour). Axis-aligned strokes (every rect, orthogonal polylines, and
  horizontal/vertical lines) carry `shape-rendering="crispEdges"` for a consistent
  1px; a diagonal line omits it. The `<svg>` carries a `viewBox` offset by the
  border-stroke overflow (above) so perimeter borders render full-width. Output is
  byte-for-byte stable, and identical to the pre-styling output for an unstyled
  document except for that `viewBox`.

## Public API

`arkitecture.go` (package `arkitecture`) is the supported surface:

- `ToSVG(dsl, *Options) Result` — the whole pipeline (parse → validate → resolve →
  generate); honours `ValidateOnly`, `FontSize`, `FontFamily`. Never panics across
  stages.
- `Parse(dsl) ParseResult` — tokenize + parse only.
- `Validate(*Document) []Error` — semantic checks on an AST.
- `GenerateSVG(*Document, *Options) (string, []Error)` — resolve the layout layer,
  then lay out + emit SVG.
- The AST/diagnostic types are re-exported as aliases (`arkitecture.Document`,
  `arkitecture.Declarations`, `arkitecture.LayoutRule`, `arkitecture.Error`, …),
  so callers need not import `ast` directly.

## Persistence

None. Arkitecture is stateless: input is a `.ark` string/file, output is an SVG
string. Only the CLI touches the filesystem (read input, write SVG, poll the
input in watch mode). There is no database, config file, or cache.

## CLI

`cmd/arkitecture` parses arguments with the standard-library `flag` package
(flags may appear before or after the input/output paths), reads the input, runs
`arkitecture.ToSVG`, and writes the SVG (defaulting the output to the input name
with a `.svg` extension). Flags: `--validate-only`, `--verbose`, `--watch`,
`--font-size`, `--font-family`, `--version`. Exit codes: `0` success, `1`
validation/parse errors, `2` filesystem errors. `--watch` re-renders on change
using a stdlib modtime poller and stops on SIGINT/SIGTERM.

## Concurrency

Single-threaded and synchronous. The library has no goroutines. The only
asynchrony is the CLI watch loop — a `time.Ticker` polling the input's modtime,
plus a signal channel for shutdown — which re-runs the same synchronous pipeline
per change; runs never overlap.

## Build & test

`go build ./...` builds the library, CLI, and the host wasm stub;
`GOOS=js GOARCH=wasm go build ./wasm` builds the real WASM module. Tests use the
standard `testing` package (table-driven). The **golden** test renders
`generator/testdata/golden/*.ark` through the full pipeline and diffs against the
checked-in `.svg`/`.error` references, regenerated with `-update`. CI runs gofmt +
vet + `go test -race` + the CLI and WASM builds on Go 1.23 and 1.24.

## Docs site

`site/` is a hand-rolled static site (no framework, no bundler) published to
GitHub Pages by `.github/workflows/pages.yml`. Publishing is a **generation
step**, not a verbatim copy: the workflow runs `scripts/build-site.sh`, which

1. re-renders every `site/examples/*.ark` to `.svg` via the CLI (the committed
   SVGs are a refreshed-at-publish snapshot and the no-JS fallback),
2. injects each example's `.ark` source into the page HTML via
   `internal/sitegen` — replacing the body of every `<code data-ark="…">` block
   with the file's escaped contents — so the **shown source has one canonical
   home (the `.ark`)** and can never drift from the diagram it renders, and
3. builds `site/arkitecture.wasm` (`-ldflags="-s -w"`) and copies Go's
   `wasm_exec.js` beside it,

then uploads `site/`. The SVGs and the injected HTML are committed (and
idempotently refreshed here); the `.wasm`/`wasm_exec.js` are git-ignored. The
same script runs locally to preview the site as it ships. Pages triggers on
changes to `site/**`, the script, the workflow, or any Go source (the site embeds
the library). To add or edit an example, change only its `.ark`: the SVG and the
shown source both regenerate from it.

The Examples page is **progressively enhanced** by `site/playground.js`: it loads
the WASM (the same `arkitectureToSVG` the JS/TS interop uses), replaces each
example's read-only source with an editable textarea, and re-renders live in the
browser on input — with a Reset control and inline compile errors. The library
stays pure; the page is just another thin consumer of `wasm/`, like the CLI. With
JS or WASM unavailable the page is unchanged (static SVGs, read-only source).
